package errors

import (
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(InvalidRequest, "test message", http.StatusBadRequest)

	if err.Code != InvalidRequest {
		t.Errorf("expected code %s, got %s", InvalidRequest, err.Code)
	}

	if err.Message != "test message" {
		t.Errorf("expected message 'test message', got '%s'", err.Message)
	}

	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status code %d, got %d", http.StatusBadRequest, err.StatusCode)
	}
}

func TestWrap(t *testing.T) {
	original := New(InternalError, "original error", http.StatusInternalServerError)
	wrapped := Wrap(InvalidRequest, "wrapper message", http.StatusBadRequest, original)

	if wrapped.Code != InvalidRequest {
		t.Errorf("expected code %s, got %s", InvalidRequest, wrapped.Code)
	}

	if wrapped.Err == nil {
		t.Error("expected wrapped error to have underlying error")
	}

	if wrapped.Unwrap() != original {
		t.Error("expected Unwrap to return original error")
	}
}

func TestError(t *testing.T) {
	t.Run("error without underlying", func(t *testing.T) {
		err := New(InternalError, "test error", http.StatusInternalServerError)
		msg := err.Error()
		if msg != "test error" {
			t.Errorf("expected 'test error', got '%s'", msg)
		}
	})

	t.Run("error with underlying", func(t *testing.T) {
		original := New(InternalError, "original", http.StatusInternalServerError)
		wrapped := Wrap(InvalidRequest, "wrapper", http.StatusBadRequest, original)
		msg := wrapped.Error()
		expected := "wrapper: original"
		if msg != expected {
			t.Errorf("expected '%s', got '%s'", expected, msg)
		}
	})
}

func TestIsAppError(t *testing.T) {
	t.Run("is app error", func(t *testing.T) {
		err := New(InternalError, "test", http.StatusInternalServerError)
		if !IsAppError(err) {
			t.Error("expected IsAppError to return true")
		}
	})

	t.Run("is not app error", func(t *testing.T) {
		var err error = nil
		if IsAppError(err) {
			t.Error("expected IsAppError to return false for nil")
		}
	})
}

func TestGetStatusCode(t *testing.T) {
	t.Run("app error", func(t *testing.T) {
		err := New(NotFound, "not found", http.StatusNotFound)
		code := GetStatusCode(err)
		if code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, code)
		}
	})

	t.Run("non-app error", func(t *testing.T) {
		code := GetStatusCode(nil)
		if code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, code)
		}
	})
}

func TestGetErrorCode(t *testing.T) {
	t.Run("app error", func(t *testing.T) {
		err := New(NotFound, "not found", http.StatusNotFound)
		code := GetErrorCode(err)
		if code != string(NotFound) {
			t.Errorf("expected code %s, got %s", NotFound, code)
		}
	})

	t.Run("non-app error", func(t *testing.T) {
		code := GetErrorCode(nil)
		if code != string(InternalError) {
			t.Errorf("expected code %s, got %s", InternalError, code)
		}
	})
}

func TestWithRequestID(t *testing.T) {
	err := New(InternalError, "test error", http.StatusInternalServerError)
	errWithID := err.WithRequestID("req-123")

	if errWithID.RequestID != "req-123" {
		t.Errorf("expected request ID 'req-123', got '%s'", errWithID.RequestID)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        *AppError
		wantCode   ErrorCode
		wantStatus int
	}{
		{"ErrInternal", ErrInternal, InternalError, http.StatusInternalServerError},
		{"ErrInvalidRequest", ErrInvalidRequest, InvalidRequest, http.StatusBadRequest},
		{"ErrUnauthorized", ErrUnauthorized, Unauthorized, http.StatusUnauthorized},
		{"ErrNotFound", ErrNotFound, NotFound, http.StatusNotFound},
		{"ErrServiceUnavailable", ErrServiceUnavailable, ServiceUnavailable, http.StatusServiceUnavailable},
		{"ErrTimeout", ErrTimeout, Timeout, http.StatusGatewayTimeout},
		{"ErrForbidden", ErrForbidden, Forbidden, http.StatusForbidden},
		{"ErrInvalidToken", ErrInvalidToken, InvalidToken, http.StatusUnauthorized},
		{"ErrExpiredToken", ErrExpiredToken, ExpiredToken, http.StatusUnauthorized},
		{"ErrInvalidCredentials", ErrInvalidCredentials, InvalidCredentials, http.StatusUnauthorized},
		{"ErrAccountLocked", ErrAccountLocked, AccountLocked, http.StatusForbidden},
		{"ErrUserNotFound", ErrUserNotFound, UserNotFound, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("expected code %s, got %s", tt.wantCode, tt.err.Code)
			}
			if tt.err.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, tt.err.StatusCode)
			}
		})
	}
}
