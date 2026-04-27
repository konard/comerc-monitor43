package executor

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestErrorClassifier_ClassifyError_Timeout(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name string
		err  error
		want ErrorCode
	}{
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			want: ErrorCodeConnectionTimeout,
		},
		{
			name: "net timeout error",
			err:  &net.OpError{Err: fmt.Errorf("timeout")},
			want: ErrorCodeConnectionTimeout,
		},
		{
			name: "os deadline exceeded",
			err:  os.ErrDeadlineExceeded,
			want: ErrorCodeConnectionTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.ClassifyError(tt.err)
			if got != tt.want {
				t.Errorf("ClassifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorClassifier_ClassifyError_DNS(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name string
		err  error
		want ErrorCode
	}{
		{
			name: "DNS resolution failed",
			err:  &net.DNSError{Err: "no such host"},
			want: ErrorCodeDNSResolutionFailed,
		},
		{
			name: "DNS not found",
			err:  &net.DNSError{Err: "no such host", IsNotFound: true},
			want: ErrorCodeDNSResolutionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.ClassifyError(tt.err)
			if got != tt.want {
				t.Errorf("ClassifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorClassifier_ClassifyError_ConnectionRefused(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name string
		err  error
		want ErrorCode
	}{
		{
			name: "syscall ECONNREFUSED",
			err:  &os.SyscallError{Err: syscall.ECONNREFUSED},
			want: ErrorCodeConnectionRefused,
		},
		{
			name: "net dial error",
			err:  &net.OpError{Op: "dial", Err: fmt.Errorf("connection refused")},
			want: ErrorCodeConnectionRefused,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.ClassifyError(tt.err)
			if got != tt.want {
				t.Errorf("ClassifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorClassifier_ClassifyError_SSL(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name string
		err  error
		want ErrorCode
	}{
		{
			name: "certificate expired",
			err:  &x509.CertificateInvalidError{},
			want: ErrorCodeSSLCertificateExpired,
		},
		{
			name: "unknown authority",
			err:  &x509.UnknownAuthorityError{},
			want: ErrorCodeSSLCertificateExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.ClassifyError(tt.err)
			if got != tt.want {
				t.Errorf("ClassifyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorClassifier_ClassifyError_Nil(t *testing.T) {
	classifier := NewErrorClassifier()
	got := classifier.ClassifyError(nil)
	if got != "" {
		t.Errorf("ClassifyError(nil) = %v, want empty string", got)
	}
}

func TestErrorClassifier_GetErrorMessage(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		code ErrorCode
		want string
	}{
		{ErrorCodeConnectionTimeout, "connection timeout"},
		{ErrorCodeDNSResolutionFailed, "dns resolution failed"},
		{ErrorCodeSSLCertificateExpired, "ssl certificate expired"},
		{ErrorCodeTooManyRedirects, "too many redirects"},
		{ErrorCodeConnectionRefused, "connection refused"},
		{ErrorCodeResponseTooLarge, "response too large"},
		{ErrorCodeSlowResponse, "slow response"},
		{ErrorCodeEmptyResponse, "empty response"},
		{ErrorCodeInvalidMonitorConfig, "invalid monitor configuration"},
		{"UNKNOWN", "unknown error"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			got := classifier.GetErrorMessage(tt.code)
			if got != tt.want {
				t.Errorf("GetErrorMessage(%v) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestIsDegraded(t *testing.T) {
	tests := []struct {
		name                string
		responseTimeMs      float64
		degradedThresholdMs float64
		want                bool
	}{
		{
			name:                "fast response",
			responseTimeMs:      100,
			degradedThresholdMs: 5000,
			want:                false,
		},
		{
			name:                "slow response",
			responseTimeMs:      6000,
			degradedThresholdMs: 5000,
			want:                true,
		},
		{
			name:                "exactly threshold",
			responseTimeMs:      5000,
			degradedThresholdMs: 5000,
			want:                false,
		},
		{
			name:                "no threshold configured",
			responseTimeMs:      10000,
			degradedThresholdMs: 0,
			want:                false,
		},
		{
			name:                "negative threshold",
			responseTimeMs:      1000,
			degradedThresholdMs: -1,
			want:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDegraded(tt.responseTimeMs, tt.degradedThresholdMs)
			if got != tt.want {
				t.Errorf("IsDegraded(%v, %v) = %v, want %v",
					tt.responseTimeMs, tt.degradedThresholdMs, got, tt.want)
			}
		})
	}
}

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     CheckRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: CheckRequest{
				URL:     "https://example.com",
				Timeout: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			req: CheckRequest{
				Timeout: 10 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "empty URL",
			req: CheckRequest{
				URL:     "",
				Timeout: 10 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want float64
	}{
		{
			name: "milliseconds",
			d:    150 * time.Millisecond,
			want: 150,
		},
		{
			name: "seconds",
			d:    2 * time.Second,
			want: 2000,
		},
		{
			name: "zero",
			d:    0,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.d)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %v, want %v", tt.d, got, tt.want)
			}
		})
	}
}

func TestDetailedMetrics(t *testing.T) {
	metrics := DetailedMetrics{
		DNSLookupMs:    15.5,
		TCPConnectMs:   30.2,
		TLSHandshakeMs: 45.8,
		TTFBMs:         120.0,
		TotalMs:        250.0,
	}

	if metrics.DNSLookupMs != 15.5 {
		t.Errorf("DNSLookupMs = %v, want 15.5", metrics.DNSLookupMs)
	}

	if metrics.TCPConnectMs != 30.2 {
		t.Errorf("TCPConnectMs = %v, want 30.2", metrics.TCPConnectMs)
	}

	if metrics.TLSHandshakeMs != 45.8 {
		t.Errorf("TLSHandshakeMs = %v, want 45.8", metrics.TLSHandshakeMs)
	}

	if metrics.TTFBMs != 120.0 {
		t.Errorf("TTFBMs = %v, want 120.0", metrics.TTFBMs)
	}

	if metrics.TotalMs != 250.0 {
		t.Errorf("TotalMs = %v, want 250.0", metrics.TotalMs)
	}
}
