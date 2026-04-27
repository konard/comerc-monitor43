package http_checker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHTTPChecker тестирует создание HTTPChecker.
func TestNewHTTPChecker(t *testing.T) {
	t.Parallel()
	t.Run("creates with default config", func(t *testing.T) {
		checker := NewHTTPChecker(Config{})

		assert.NotNil(t, checker)
		assert.NotNil(t, checker.client)
		assert.Equal(t, 30*time.Second, checker.client.Timeout)
	})

	t.Run("creates with custom config", func(t *testing.T) {
		config := Config{
			Timeout:         10 * time.Second,
			MaxRedirects:    5,
			KeepAlive:       60 * time.Second,
			MaxIdleConns:    50,
			IdleConnTimeout: 30 * time.Second,
		}

		checker := NewHTTPChecker(config)

		assert.NotNil(t, checker)
		assert.NotNil(t, checker.client)
		assert.Equal(t, 10*time.Second, checker.client.Timeout)
	})
}

// TestHTTPChecker_Check_Success тестирует успешную проверку.
func TestHTTPChecker_Check_Success(t *testing.T) {
	t.Parallel()
	// Создаём тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	checker := NewHTTPChecker(Config{})
	ctx := context.Background()

	result := checker.Check(ctx, server.URL)

	assert.NotNil(t, result)
	assert.Nil(t, result.Error)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.True(t, result.Success)
	assert.Greater(t, result.ResponseTime, time.Duration(0))
	assert.Equal(t, int64(2), result.ContentLength)
}

// TestHTTPChecker_Check_StatusCodes тестирует разные статусы ответа.
func TestHTTPChecker_Check_StatusCodes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name            string
		statusCode      int
		expectedSuccess bool
	}{
		{
			name:            "200 OK",
			statusCode:      http.StatusOK,
			expectedSuccess: true,
		},
		{
			name:            "201 Created",
			statusCode:      http.StatusCreated,
			expectedSuccess: true,
		},
		{
			name:            "204 No Content",
			statusCode:      http.StatusNoContent,
			expectedSuccess: true,
		},
		{
			name:            "301 Moved Permanently",
			statusCode:      http.StatusMovedPermanently,
			expectedSuccess: true,
		},
		{
			name:            "302 Found",
			statusCode:      http.StatusFound,
			expectedSuccess: true,
		},
		{
			name:            "304 Not Modified",
			statusCode:      http.StatusNotModified,
			expectedSuccess: true,
		},
		{
			name:            "400 Bad Request",
			statusCode:      http.StatusBadRequest,
			expectedSuccess: false,
		},
		{
			name:            "401 Unauthorized",
			statusCode:      http.StatusUnauthorized,
			expectedSuccess: false,
		},
		{
			name:            "403 Forbidden",
			statusCode:      http.StatusForbidden,
			expectedSuccess: false,
		},
		{
			name:            "404 Not Found",
			statusCode:      http.StatusNotFound,
			expectedSuccess: false,
		},
		{
			name:            "500 Internal Server Error",
			statusCode:      http.StatusInternalServerError,
			expectedSuccess: false,
		},
		{
			name:            "502 Bad Gateway",
			statusCode:      http.StatusBadGateway,
			expectedSuccess: false,
		},
		{
			name:            "503 Service Unavailable",
			statusCode:      http.StatusServiceUnavailable,
			expectedSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			t.Cleanup(server.Close)

			checker := NewHTTPChecker(Config{})
			ctx := context.Background()

			result := checker.Check(ctx, server.URL)

			assert.NotNil(t, result)
			assert.Nil(t, result.Error)
			assert.Equal(t, tc.statusCode, result.StatusCode)
			assert.Equal(t, tc.expectedSuccess, result.Success)
		})
	}
}

// TestHTTPChecker_Check_Timeout тестирует timeout.
func TestHTTPChecker_Check_Timeout(t *testing.T) {
	t.Parallel()
	// Создаём сервер, который отвечает очень медленно
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	config := Config{
		Timeout: 100 * time.Millisecond,
	}
	checker := NewHTTPChecker(config)
	ctx := context.Background()

	result := checker.Check(ctx, server.URL)

	assert.NotNil(t, result)
	assert.NotNil(t, result.Error)
	// Timeout error may contain "deadline exceeded" or "Timeout"
	errMsg := result.Error.Error()
	assert.True(t, containsIgnoreCase(errMsg, "timeout") || containsIgnoreCase(errMsg, "deadline exceeded"))
	assert.False(t, result.Success)
}

// TestHTTPChecker_Check_InvalidURL тестирует обработку невалидного URL.
func TestHTTPChecker_Check_InvalidURL(t *testing.T) {
	t.Parallel()
	checker := NewHTTPChecker(Config{})
	ctx := context.Background()

	result := checker.Check(ctx, "://invalid-url")

	assert.NotNil(t, result)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "failed to create request")
	assert.False(t, result.Success)
}

// TestHTTPChecker_Check_ConnectionRefused тестирует обработку отказа в соединении.
func TestHTTPChecker_Check_ConnectionRefused(t *testing.T) {
	t.Parallel()
	checker := NewHTTPChecker(Config{})
	ctx := context.Background()

	// Используем адрес, который скорее всего не слушает
	result := checker.Check(ctx, "http://localhost:59999")

	assert.NotNil(t, result)
	assert.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "failed to execute request")
	assert.False(t, result.Success)
}

// TestHTTPChecker_Check_ContextCancellation тестирует отмену контекста.
func TestHTTPChecker_Check_ContextCancellation(t *testing.T) {
	t.Parallel()
	// Создаём сервер, который отвечает очень медленно
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	checker := NewHTTPChecker(Config{})
	ctx, cancel := context.WithCancel(context.Background())

	// Отменяем контекст сразу после начала проверки
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result := checker.Check(ctx, server.URL)

	assert.NotNil(t, result)
	// Ошибка может быть разной в зависимости от тайминга
	assert.NotNil(t, result.Error)
	assert.False(t, result.Success)
}

// TestHTTPChecker_Check_ResponseTime тестирует измерение времени ответа.
func TestHTTPChecker_Check_ResponseTime(t *testing.T) {
	t.Parallel()
	// Создаём сервер с фиксированной задержкой
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	checker := NewHTTPChecker(Config{})
	ctx := context.Background()

	result := checker.Check(ctx, server.URL)

	assert.NotNil(t, result)
	assert.Nil(t, result.Error)
	assert.GreaterOrEqual(t, result.ResponseTime, 100*time.Millisecond)
	assert.Less(t, result.ResponseTime, 500*time.Millisecond)
}

// TestHTTPChecker_Check_ContentLength тестирует измерение размера контента.
func TestHTTPChecker_Check_ContentLength(t *testing.T) {
	t.Parallel()
	testContent := "Hello, World! This is a test response."

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(testContent))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	checker := NewHTTPChecker(Config{})
	ctx := context.Background()

	result := checker.Check(ctx, server.URL)

	assert.NotNil(t, result)
	assert.Nil(t, result.Error)
	assert.Equal(t, int64(len(testContent)), result.ContentLength)
}

// TestHTTPChecker_Check_Redirect тестирует обработку редиректов.
func TestHTTPChecker_Check_Redirect(t *testing.T) {
	t.Parallel()
	t.Run("follows redirect", func(t *testing.T) {
		// Создаём цепочку редиректов
		finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Final destination"))
			require.NoError(t, err)
		}))
		t.Cleanup(finalServer.Close)

		redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, finalServer.URL, http.StatusFound)
		}))
		t.Cleanup(redirectServer.Close)

		checker := NewHTTPChecker(Config{})
		ctx := context.Background()

		result := checker.Check(ctx, redirectServer.URL)

		assert.NotNil(t, result)
		assert.Nil(t, result.Error)
		// После редиректа должен быть статус 200
		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.True(t, result.Success)
	})

	t.Run("too many redirects", func(t *testing.T) {
		config := Config{
			MaxRedirects: 2,
		}

		// Создаём цепочку редиректов
		var server *httptest.Server
		redirectCount := 0
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			redirectCount++
			if redirectCount > 3 {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Redirect(w, r, server.URL, http.StatusFound)
		}))
		t.Cleanup(server.Close)

		checker := NewHTTPChecker(config)
		ctx := context.Background()

		result := checker.Check(ctx, server.URL)

		assert.NotNil(t, result)
		assert.NotNil(t, result.Error)
		assert.Contains(t, result.Error.Error(), "too many redirects")
		assert.False(t, result.Success)
	})
}

// TestHTTPChecker_Check_Headers тестирует отправку headers.
func TestHTTPChecker_Check_Headers(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем User-Agent (должен содержать "Go-http-client")
		userAgent := r.Header.Get("User-Agent")
		assert.Contains(t, userAgent, "Go-http-client")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	checker := NewHTTPChecker(Config{})
	ctx := context.Background()

	result := checker.Check(ctx, server.URL)

	assert.NotNil(t, result)
	assert.Nil(t, result.Error)
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

// TestHTTPChecker_Check_ConcurrentRequests тестирует конкурентные запросы.
func TestHTTPChecker_Check_ConcurrentRequests(t *testing.T) {
	t.Parallel()
	var requestCount int64 = 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	checker := NewHTTPChecker(Config{})
	ctx := context.Background()

	// Выполняем 10 concurrent запросов
	results := make(chan *CheckResult, 10)
	for i := 0; i < 10; i++ {
		go func() {
			results <- checker.Check(ctx, server.URL)
		}()
	}

	// Собираем результаты
	for i := 0; i < 10; i++ {
		result := <-results
		assert.NotNil(t, result)
		assert.Nil(t, result.Error)
		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.True(t, result.Success)
	}

	assert.Equal(t, int64(10), atomic.LoadInt64(&requestCount))
}

// TestHTTPChecker_Check_POSTRequest тестирует POST запрос.
func TestHTTPChecker_Check_POSTRequest(t *testing.T) {
	t.Parallel()
	// Примечание: текущая реализация Check() выполняет только GET запросы
	// Этот тест демонстрирует текущее поведение
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что это GET запрос
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	checker := NewHTTPChecker(Config{})
	ctx := context.Background()

	result := checker.Check(ctx, server.URL)

	assert.NotNil(t, result)
	assert.Nil(t, result.Error)
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

// containsIgnoreCase проверяет, что подстрока содержится в строке без учёта регистра.
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstringIgnoreCase(s, substr)))
}

func containsSubstringIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	return len(sLower) >= len(substrLower) && indexOf(sLower, substrLower) >= 0
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
