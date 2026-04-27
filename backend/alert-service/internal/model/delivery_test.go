package model

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClassifyHTTPErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		statusCode     int
		body           string
		wantCategory   DeliveryErrorCategory
		wantRetryable  bool
		wantMaxRetries int
		wantBaseDelay  time.Duration
	}{
		{
			name:           "rate_limit_429",
			statusCode:     429,
			wantCategory:   DeliveryErrorRateLimit,
			wantRetryable:  true,
			wantMaxRetries: 5,
			wantBaseDelay:  5 * time.Minute,
		},
		{
			name:          "auth_401",
			statusCode:    401,
			body:          "unauthorized",
			wantCategory:  DeliveryErrorAuth,
			wantRetryable: false,
		},
		{
			name:          "auth_403",
			statusCode:    403,
			body:          "forbidden",
			wantCategory:  DeliveryErrorAuth,
			wantRetryable: false,
		},
		{
			name:          "permanent_400",
			statusCode:    400,
			body:          "bad request",
			wantCategory:  DeliveryErrorPermanent,
			wantRetryable: false,
		},
		{
			name:          "permanent_404",
			statusCode:    404,
			body:          "not found",
			wantCategory:  DeliveryErrorPermanent,
			wantRetryable: false,
		},
		{
			name:           "transient_500",
			statusCode:     500,
			body:           "internal server error",
			wantCategory:   DeliveryErrorTransient,
			wantRetryable:  true,
			wantMaxRetries: 3,
			wantBaseDelay:  1 * time.Minute,
		},
		{
			name:           "transient_503",
			statusCode:     503,
			body:           "service unavailable",
			wantCategory:   DeliveryErrorTransient,
			wantRetryable:  true,
			wantMaxRetries: 3,
			wantBaseDelay:  1 * time.Minute,
		},
		{
			name:          "unknown_200",
			statusCode:    200,
			body:          "ok",
			wantCategory:  DeliveryErrorUnknown,
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyHTTPErr(tt.statusCode, tt.body)

			assert.Equal(t, tt.wantCategory, result.Category)
			assert.Equal(t, tt.wantRetryable, result.Retryable)
			assert.Equal(t, tt.statusCode, result.StatusCode)

			if tt.wantMaxRetries > 0 {
				assert.Equal(t, tt.wantMaxRetries, result.MaxRetries)
			}
			if tt.wantBaseDelay > 0 {
				assert.Equal(t, tt.wantBaseDelay, result.BaseDelay)
			}
		})
	}
}

func TestClassifyNetworkErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		err            error
		wantCategory   DeliveryErrorCategory
		wantRetryable  bool
		wantMaxRetries int
		wantBaseDelay  time.Duration
	}{
		{
			name:           "timeout",
			err:            context.DeadlineExceeded,
			wantCategory:   DeliveryErrorTimeout,
			wantRetryable:  true,
			wantMaxRetries: 3,
			wantBaseDelay:  1 * time.Minute,
		},
		{
			name:          "ssl_certificate_error",
			err:           errors.New("TLS handshake failure: certificate expired"),
			wantCategory:  DeliveryErrorSSLCert,
			wantRetryable: false,
		},
		{
			name:          "ssl_error_lowercase",
			err:           errors.New("certificate verify failed"),
			wantCategory:  DeliveryErrorSSLCert,
			wantRetryable: false,
		},
		{
			name:           "generic_network_error",
			err:            errors.New("connection refused"),
			wantCategory:   DeliveryErrorTransient,
			wantRetryable:  true,
			wantMaxRetries: 3,
			wantBaseDelay:  1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyNetworkErr(tt.err)

			assert.Equal(t, tt.wantCategory, result.Category)
			assert.Equal(t, tt.wantRetryable, result.Retryable)
			assert.NotEmpty(t, result.Message)

			if tt.wantMaxRetries > 0 {
				assert.Equal(t, tt.wantMaxRetries, result.MaxRetries)
			}
			if tt.wantBaseDelay > 0 {
				assert.Equal(t, tt.wantBaseDelay, result.BaseDelay)
			}
		})
	}
}
