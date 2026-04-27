package executor

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"
)

type CheckRequest struct {
	MonitorID              string
	URL                    string
	Method                 string // GET, HEAD, etc.
	ExpectedStatusCode     string
	ExpectedBodyPattern    string
	Timeout                time.Duration
	SSLVerify              bool
	Headers                map[string]string
	BasicAuthUsername      string
	BasicAuthPassword      string
	FollowRedirects        bool
	MaxRedirects           int
	MaxResponseSize        int64 // bytes
	DegradedResponseTimeMs float64
}

type CheckResponse struct {
	Success         bool
	StatusCode      int
	ResponseTimeMs  float64
	ErrorCode       ErrorCode
	ErrorMessage    string
	Warning         string
	ResponseBody    []byte
	ResponseHeaders map[string]string
	FinalURL        string
	DetailedMetrics *DetailedMetrics
	CheckedAt       time.Time
}

type CheckExecutor struct {
	defaultTimeout     time.Duration
	classifier         *ErrorClassifier
	maintenanceChecker *MaintenanceChecker
}

func NewCheckExecutor(defaultTimeout time.Duration) *CheckExecutor {
	return &CheckExecutor{
		defaultTimeout:     defaultTimeout,
		classifier:         NewErrorClassifier(),
		maintenanceChecker: nil, // Will be set later if needed
	}
}

// SetMaintenanceChecker устанавливает checker для maintenance windows
func (e *CheckExecutor) SetMaintenanceChecker(checker *MaintenanceChecker) {
	e.maintenanceChecker = checker
}

func (e *CheckExecutor) ExecuteCheck(ctx context.Context, req CheckRequest) (*CheckResponse, error) {
	start := time.Now()
	result := &CheckResponse{
		CheckedAt:       start,
		ResponseHeaders: make(map[string]string),
		DetailedMetrics: &DetailedMetrics{},
	}

	// Проверяем maintenance windows
	if e.maintenanceChecker != nil {
		shouldSkip, window, err := e.maintenanceChecker.ShouldSkipCheck(ctx, req.MonitorID)
		if err != nil {
			// Логируем ошибку, но не блокируем проверку
			// Если не удалось проверить maintenance, выполняем проверку
		} else if shouldSkip {
			result.Success = false
			result.ErrorCode = ErrorCodeMaintenanceSkipped
			result.ErrorMessage = "check skipped due to active maintenance window"
			result.ResponseTimeMs = float64(time.Since(start).Milliseconds())
			result.Warning = fmt.Sprintf("maintenance window: %s (until %s)",
				window.ID,
				window.EndTime.Format(time.RFC3339))
			return result, nil
		}
	}

	// Валидация запроса
	if err := ValidateRequest(req); err != nil {
		result.ErrorCode = ErrorCodeInvalidMonitorConfig
		result.ErrorMessage = e.classifier.GetErrorMessage(ErrorCodeInvalidMonitorConfig)
		result.ResponseTimeMs = float64(time.Since(start).Milliseconds())
		return result, nil
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = e.defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Создаём tracer для сбора детальных метрик
	tracer := NewHTTPTracer()
	ctx = httptrace.WithClientTrace(ctx, tracer.Trace())

	// Определяем метод
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, nil)
	if err != nil {
		result.ErrorCode = ErrorCodeInvalidMonitorConfig
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		result.ResponseTimeMs = float64(time.Since(start).Milliseconds())
		return result, nil
	}

	// Добавляем заголовки
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Добавляем Basic Auth
	if req.BasicAuthUsername != "" && req.BasicAuthPassword != "" {
		httpReq.SetBasicAuth(req.BasicAuthUsername, req.BasicAuthPassword)
	}

	// Настраиваем транспорт
	transport := &http.Transport{
		//nolint:gosec // Конфигурация монитора явно управляет проверкой SSL для диагностических проверок.
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !req.SSLVerify},
	}

	// Настраиваем клиент
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	// Настраиваем redirect handling
	if req.FollowRedirects {
		if req.MaxRedirects > 0 {
			maxRedirects := req.MaxRedirects // Сохраняем в локальную переменную
			client.CheckRedirect = func(redirectReq *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					result.ErrorCode = ErrorCodeTooManyRedirects
					result.ErrorMessage = e.classifier.GetErrorMessage(ErrorCodeTooManyRedirects)
					return fmt.Errorf("stopped after %d redirects", maxRedirects)
				}
				return nil
			}
		}
	} else {
		client.CheckRedirect = func(redirectReq *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Выполняем запрос
	resp, err := client.Do(httpReq)
	result.ResponseTimeMs = float64(time.Since(start).Milliseconds())

	if err != nil {
		// Классифицируем ошибку
		result.ErrorCode = e.classifier.ClassifyError(err)
		result.ErrorMessage = e.classifier.GetErrorMessage(result.ErrorCode)
		result.Success = false

		// Записываем метрики даже при ошибке
		tracer.RecordEnd()
		metrics := tracer.GetMetrics()
		result.DetailedMetrics = &metrics
		return result, nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			return
		}
	}()

	// Записываем метрики после успешного запроса
	tracer.RecordEnd()
	metrics := tracer.GetMetrics()
	result.DetailedMetrics = &metrics

	result.StatusCode = resp.StatusCode
	result.FinalURL = req.URL
	if resp.Request != nil && resp.Request.URL != nil {
		result.FinalURL = resp.Request.URL.String()
	}

	// Копируем заголовки ответа
	for k, v := range resp.Header {
		if len(v) > 0 {
			result.ResponseHeaders[k] = v[0]
		}
	}

	// Читаем тело ответа с ограничением размера
	maxSize := req.MaxResponseSize
	if maxSize <= 0 {
		maxSize = 1 * 1024 * 1024 // 1MB по умолчанию
	}

	body := make([]byte, 0, 1024*1024)
	buf := make([]byte, 4096)
	totalRead := int64(0)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			totalRead += int64(n)
			if totalRead > maxSize {
				result.ErrorCode = ErrorCodeResponseTooLarge
				result.ErrorMessage = e.classifier.GetErrorMessage(ErrorCodeResponseTooLarge)
				result.Success = false
				return result, nil
			}
			body = append(body, buf[:n]...)
		}
		if readErr != nil {
			break
		}
	}

	result.ResponseBody = body

	// Проверяем на пустой ответ
	if len(body) == 0 && method == http.MethodGet {
		result.Warning = e.classifier.GetErrorMessage(ErrorCodeEmptyResponse)
	}

	// Проверяем статус код
	if req.ExpectedStatusCode != "" && fmt.Sprintf("%d", resp.StatusCode) != req.ExpectedStatusCode {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("expected status %s, got %d", req.ExpectedStatusCode, resp.StatusCode)
		return result, nil
	}

	// Определяем базовый статус
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 400

	// Проверяем на DEGRADED статус
	if result.Success && req.DegradedResponseTimeMs > 0 {
		if IsDegraded(result.ResponseTimeMs, req.DegradedResponseTimeMs) {
			result.Success = false // DEGRADED считается как проблема
			result.Warning = e.classifier.GetErrorMessage(ErrorCodeSlowResponse)
		}
	}

	if !result.Success && result.ErrorMessage == "" {
		result.ErrorMessage = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
	}

	return result, nil
}

// ExtractErrorFromURL извлекает информацию об ошибке из URL для детальной диагностики
func (e *CheckExecutor) ExtractErrorFromURL(targetURL string) error {
	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Проверяем формат URL
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("empty host")
	}

	return nil
}

// IsSSLValid проверяет валидность SSL сертификата хоста
func (e *CheckExecutor) IsSSLValid(targetURL string) (bool, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return false, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "https" {
		return true, nil // SSL не требуется
	}

	// Создаем тестовое соединение для проверки сертификата
	conn, err := tls.Dial("tcp", u.Host, &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		return false, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			return
		}
	}()

	// Проверяем состояние соединения
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return false, fmt.Errorf("no peer certificates")
	}

	// Проверяем срок действия сертификата
	cert := state.PeerCertificates[0]
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return false, fmt.Errorf("certificate not yet valid")
	}
	if now.After(cert.NotAfter) {
		return false, fmt.Errorf("certificate expired")
	}

	return true, nil
}

// GetMetricsHeader извлекает значение из заголовка как float64
func GetMetricsHeader(headers map[string]string, key string) float64 {
	val := headers[key]
	if val == "" {
		return 0
	}

	// Пытаемся распарсить как float
	var f float64
	_, err := fmt.Sscanf(val, "%f", &f)
	if err != nil {
		return 0
	}

	return f
}
