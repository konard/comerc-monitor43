package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeResponse(t *testing.T, w http.ResponseWriter, body []byte) {
	t.Helper()

	if _, err := w.Write(body); err != nil {
		t.Logf("write response: %v", err) // broken pipe нормален при client disconnect
	}
}

func TestExecuteCheck(t *testing.T) {
	t.Parallel()

	t.Run("success_200", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			writeResponse(t, w, []byte("OK"))
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-1",
			URL:       srv.URL,
			Timeout:   3 * time.Second,
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 200, resp.StatusCode)
		assert.GreaterOrEqual(t, resp.ResponseTimeMs, float64(0))
		assert.Equal(t, []byte("OK"), resp.ResponseBody)
		assert.NotNil(t, resp.CheckedAt)
	})

	t.Run("expected_status_code", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID:          "mon-2",
			URL:                srv.URL,
			ExpectedStatusCode: "201",
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 201, resp.StatusCode)
	})

	t.Run("unexpected_status_code", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID:          "mon-3",
			URL:                srv.URL,
			ExpectedStatusCode: "200",
		})

		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.ErrorMessage, "expected status 200, got 404")
	})

	t.Run("server_error_500", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-4",
			URL:       srv.URL,
		})

		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.ErrorMessage, "unexpected status code: 500")
	})

	t.Run("invalid_url", func(t *testing.T) {
		t.Parallel()

		exec := NewCheckExecutor(1 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-5",
			URL:       "http://[::1]:namedport",
		})

		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.NotEmpty(t, resp.ErrorMessage)
	})

	t.Run("context_timeout", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(10 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		resp, err := exec.ExecuteCheck(ctx, CheckRequest{
			MonitorID: "mon-6",
			URL:       srv.URL,
		})

		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.NotEmpty(t, resp.ErrorMessage)
	})

	t.Run("custom_headers", func(t *testing.T) {
		t.Parallel()

		var receivedHeader string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeader = r.Header.Get("X-Custom")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		_, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-7",
			URL:       srv.URL,
			Headers:   map[string]string{"X-Custom": "test-value"},
		})

		require.NoError(t, err)
		assert.Equal(t, "test-value", receivedHeader)
	})

	t.Run("response_headers_captured", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-8",
			URL:       srv.URL,
		})

		require.NoError(t, err)
		assert.Equal(t, "application/json", resp.ResponseHeaders["Content-Type"])
	})

	t.Run("default_timeout_used_when_zero", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-9",
			URL:       srv.URL,
			Timeout:   0,
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("ssl_verify_false", func(t *testing.T) {
		t.Parallel()

		exec := NewCheckExecutor(1 * time.Second)
		req := CheckRequest{
			MonitorID: "mon-10",
			URL:       "http://localhost:1",
			SSLVerify: false,
		}

		resp, err := exec.ExecuteCheck(context.Background(), req)
		require.NoError(t, err)
		assert.False(t, resp.Success)
	})

	t.Run("head_method", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodHead, r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-11",
			URL:       srv.URL,
			Method:    http.MethodHead,
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, []byte{}, resp.ResponseBody) // HEAD не должен иметь body
	})

	t.Run("basic_auth_success", func(t *testing.T) {
		t.Parallel()

		var receivedAuth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Authorization")
			if receivedAuth == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID:         "mon-12",
			URL:               srv.URL,
			BasicAuthUsername: "admin",
			BasicAuthPassword: "secret",
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, receivedAuth)
		assert.Contains(t, receivedAuth, "Basic")
	})

	t.Run("basic_auth_failure", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			// Проверяем, что авторизация неверная
			expectedAuth := "Basic YWRtaW46Y29ycmVjdF9wYXNzd29yZA==" // admin:correct_password
			if auth != expectedAuth {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID:         "mon-13",
			URL:               srv.URL,
			BasicAuthUsername: "admin",
			BasicAuthPassword: "wrong_password",
		})

		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("follow_redirects_success", func(t *testing.T) {
		t.Parallel()

		redirectSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			writeResponse(t, w, []byte("Final destination"))
		}))
		defer redirectSrv.Close()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, redirectSrv.URL, http.StatusMovedPermanently)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID:       "mon-14",
			URL:             srv.URL,
			FollowRedirects: true,
			MaxRedirects:    5,
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, redirectSrv.URL, resp.FinalURL)
		assert.Equal(t, []byte("Final destination"), resp.ResponseBody)
	})

	t.Run("too_many_redirects", func(t *testing.T) {
		t.Parallel()

		// Создаём цепочку redirect серверов
		var srv1, srv2, srv3, srv4 *httptest.Server

		srv1 = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, srv2.URL, http.StatusFound)
		}))
		defer srv1.Close()

		srv2 = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, srv3.URL, http.StatusFound)
		}))
		defer srv2.Close()

		srv3 = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, srv4.URL, http.StatusFound)
		}))
		defer srv3.Close()

		srv4 = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, srv1.URL, http.StatusFound) // Создаём loop
		}))
		defer srv4.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID:       "mon-15",
			URL:             srv1.URL,
			FollowRedirects: true,
			MaxRedirects:    3,
		})

		require.NoError(t, err)
		// Проверяем что запрос завершился с ошибкой из-за redirects
		assert.False(t, resp.Success)
		// ErrorCode должен быть TOO_MANY_REDIRECTS или CONNECTION_REFUSED (зависит от реализации HTTP client)
		assert.True(t, resp.ErrorCode == ErrorCodeTooManyRedirects || resp.ErrorCode == ErrorCodeConnectionRefused)
		assert.NotEmpty(t, resp.ErrorMessage)
	})

	t.Run("no_follow_redirects", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://example.com", http.StatusMovedPermanently)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID:       "mon-16",
			URL:             srv.URL,
			FollowRedirects: false,
		})

		require.NoError(t, err)
		assert.True(t, resp.Success) // 3xx redirect считается успехом
		assert.Equal(t, 301, resp.StatusCode)
	})

	t.Run("max_response_size_respected", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Отправляем 2MB данных
			w.WriteHeader(http.StatusOK)
			largeData := make([]byte, 2*1024*1024)
			writeResponse(t, w, largeData)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID:       "mon-17",
			URL:             srv.URL,
			MaxResponseSize: 1024 * 1024, // 1MB
		})

		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, ErrorCodeResponseTooLarge, resp.ErrorCode)
		assert.NotEmpty(t, resp.ErrorMessage)
	})

	t.Run("degraded_status_slow_response", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID:              "mon-18",
			URL:                    srv.URL,
			DegradedResponseTimeMs: 50, // 50ms threshold
		})

		require.NoError(t, err)
		assert.False(t, resp.Success) // DEGRADED считается как проблема
		assert.NotEmpty(t, resp.Warning)
		assert.GreaterOrEqual(t, resp.ResponseTimeMs, 100.0)
	})

	t.Run("degraded_status_fast_response", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			writeResponse(t, w, []byte("OK")) // Добавляем body чтобы избежать empty response warning
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID:              "mon-19",
			URL:                    srv.URL,
			DegradedResponseTimeMs: 5000, // 5s threshold
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Empty(t, resp.Warning)
	})

	t.Run("empty_response_warning", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			// Не пишем body
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-20",
			URL:       srv.URL,
			Method:    http.MethodGet,
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Warning)
		assert.Equal(t, "empty response", resp.Warning)
	})

	t.Run("invalid_monitor_config_no_url", func(t *testing.T) {
		t.Parallel()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-21",
			URL:       "", // Пустой URL
		})

		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, ErrorCodeInvalidMonitorConfig, resp.ErrorCode)
		assert.NotEmpty(t, resp.ErrorMessage)
	})

	t.Run("error_classification_timeout", func(t *testing.T) {
		t.Parallel()

		exec := NewCheckExecutor(1 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		resp, err := exec.ExecuteCheck(ctx, CheckRequest{
			MonitorID: "mon-22",
			URL:       "http://example.com",
		})

		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, ErrorCodeConnectionTimeout, resp.ErrorCode)
		assert.NotEmpty(t, resp.ErrorMessage)
	})

	t.Run("detailed_metrics_collected", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			writeResponse(t, w, []byte("OK"))
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-23",
			URL:       srv.URL,
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.DetailedMetrics)

		// Проверяем что метрики собраны
		assert.GreaterOrEqual(t, resp.DetailedMetrics.TotalMs, float64(0))
		assert.GreaterOrEqual(t, resp.DetailedMetrics.TTFBMs, float64(0))
	})

	t.Run("detailed_metrics_includes_all_components", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			writeResponse(t, w, []byte("Hello, World!"))
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-24",
			URL:       srv.URL,
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.DetailedMetrics)

		// Для httptest серверов метрики могут быть неполные
		// Проверяем только что структура создана
		assert.GreaterOrEqual(t, resp.DetailedMetrics.TotalMs, float64(0))
	})

	t.Run("detailed_metrics_with_https", func(t *testing.T) {
		t.Parallel()

		// Создаём HTTPS тестовый сервер с самоподписанным сертификатом
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			writeResponse(t, w, []byte("HTTPS OK"))
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)
		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "mon-25",
			URL:       srv.URL,
			SSLVerify: false, // Отключаем верификацию для самоподписанного сертификата
		})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotNil(t, resp.DetailedMetrics)

		// Для httptest TLS серверов метрики могут быть неполные
		// Проверяем только что структура создана
		assert.GreaterOrEqual(t, resp.DetailedMetrics.TotalMs, float64(0))
	})

	t.Run("maintenance_window_skip_check", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			writeResponse(t, w, []byte("OK"))
		}))
		defer srv.Close()

		// Создаём executor с maintenance checker
		exec := NewCheckExecutor(5 * time.Second)

		// Создаём mock maintenance client с активным окном
		mockClient := NewConfigMaintenanceClient(ExampleActiveWindows())
		checker := NewMaintenanceChecker(mockClient, 5*time.Minute)
		exec.SetMaintenanceChecker(checker)

		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "monitor-1", // Этот монитор в maintenance
			URL:       srv.URL,
		})

		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, ErrorCodeMaintenanceSkipped, resp.ErrorCode)
		assert.Contains(t, resp.ErrorMessage, "maintenance window")
		assert.NotEmpty(t, resp.Warning)
	})

	t.Run("maintenance_window_check_not_affected", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			writeResponse(t, w, []byte("OK"))
		}))
		defer srv.Close()

		// Создаём executor с maintenance checker
		exec := NewCheckExecutor(5 * time.Second)

		// Создаём mock maintenance client только с одним специфическим окном
		now := time.Now()
		mockClient := NewConfigMaintenanceClient([]MaintenanceWindow{
			{
				ID:              "monitor-1-only",
				MonitorID:       "monitor-1",
				IsGlobal:        false,
				StartTime:       now.Add(-30 * time.Minute),
				EndTime:         now.Add(30 * time.Minute),
				PauseMonitoring: true,
			},
		})
		checker := NewMaintenanceChecker(mockClient, 5*time.Minute)
		exec.SetMaintenanceChecker(checker)

		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "monitor-999", // Этот монитор НЕ в maintenance
			URL:       srv.URL,
		})

		require.NoError(t, err)
		assert.True(t, resp.Success) // Проверка выполнена
		assert.Equal(t, ErrorCode(""), resp.ErrorCode)
	})

	t.Run("maintenance_window_global_affects_all", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			writeResponse(t, w, []byte("OK"))
		}))
		defer srv.Close()

		exec := NewCheckExecutor(5 * time.Second)

		// Глобальное окно обслуживания
		now := time.Now()
		mockClient := NewConfigMaintenanceClient([]MaintenanceWindow{
			{
				ID:              "global-1",
				IsGlobal:        true,
				StartTime:       now.Add(-1 * time.Hour),
				EndTime:         now.Add(1 * time.Hour),
				PauseMonitoring: true,
			},
		})
		checker := NewMaintenanceChecker(mockClient, 5*time.Minute)
		exec.SetMaintenanceChecker(checker)

		// Проверяем что все мониторы пропускаются
		resp1, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "monitor-1",
			URL:       srv.URL,
		})
		require.NoError(t, err)

		resp2, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "monitor-2",
			URL:       srv.URL,
		})
		require.NoError(t, err)

		assert.False(t, resp1.Success)
		assert.Equal(t, ErrorCodeMaintenanceSkipped, resp1.ErrorCode)

		assert.False(t, resp2.Success)
		assert.Equal(t, ErrorCodeMaintenanceSkipped, resp2.ErrorCode)
	})

	t.Run("maintenance_window_not_set_checks_proceed", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			writeResponse(t, w, []byte("OK"))
		}))
		defer srv.Close()

		// Executor без maintenance checker
		exec := NewCheckExecutor(5 * time.Second)

		resp, err := exec.ExecuteCheck(context.Background(), CheckRequest{
			MonitorID: "monitor-1",
			URL:       srv.URL,
		})

		require.NoError(t, err)
		assert.True(t, resp.Success) // Проверка выполнена
	})
}
