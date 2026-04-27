package model

import (
	"context"
	"errors"
	"strings"
	"time"
)

type DeliveryErrorCategory string

const (
	DeliveryErrorTransient DeliveryErrorCategory = "transient"
	DeliveryErrorPermanent DeliveryErrorCategory = "permanent"
	DeliveryErrorRateLimit DeliveryErrorCategory = "rate_limit"
	DeliveryErrorTimeout   DeliveryErrorCategory = "timeout"
	DeliveryErrorAuth      DeliveryErrorCategory = "auth"
	DeliveryErrorSSLCert   DeliveryErrorCategory = "ssl_certificate"
	DeliveryErrorUnknown   DeliveryErrorCategory = "unknown"
)

func (c DeliveryErrorCategory) String() string { return string(c) }

type DeliveryError struct {
	Category   DeliveryErrorCategory
	StatusCode int
	Message    string
	Retryable  bool
	MaxRetries int
	BaseDelay  time.Duration
}

func ClassifyHTTPErr(statusCode int, body string) *DeliveryError {
	switch {
	case statusCode == 429:
		return &DeliveryError{
			Category:   DeliveryErrorRateLimit,
			StatusCode: statusCode,
			Message:    "rate limit exceeded",
			Retryable:  true,
			MaxRetries: 5,
			BaseDelay:  5 * time.Minute,
		}
	case statusCode == 401, statusCode == 403:
		return &DeliveryError{
			Category:   DeliveryErrorAuth,
			StatusCode: statusCode,
			Message:    body,
			Retryable:  false,
		}
	case statusCode >= 400 && statusCode < 500:
		return &DeliveryError{
			Category:   DeliveryErrorPermanent,
			StatusCode: statusCode,
			Message:    body,
			Retryable:  false,
		}
	case statusCode >= 500:
		return &DeliveryError{
			Category:   DeliveryErrorTransient,
			StatusCode: statusCode,
			Message:    body,
			Retryable:  true,
			MaxRetries: 3,
			BaseDelay:  1 * time.Minute,
		}
	default:
		return &DeliveryError{
			Category:   DeliveryErrorUnknown,
			StatusCode: statusCode,
			Message:    body,
			Retryable:  false,
		}
	}
}

func ClassifyNetworkErr(err error) *DeliveryError {
	if errors.Is(err, context.DeadlineExceeded) {
		return &DeliveryError{
			Category:   DeliveryErrorTimeout,
			Message:    "connection timeout",
			Retryable:  true,
			MaxRetries: 3,
			BaseDelay:  1 * time.Minute,
		}
	}
	if strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "TLS") {
		return &DeliveryError{
			Category:  DeliveryErrorSSLCert,
			Message:   err.Error(),
			Retryable: false,
		}
	}
	return &DeliveryError{
		Category:   DeliveryErrorTransient,
		Message:    err.Error(),
		Retryable:  true,
		MaxRetries: 3,
		BaseDelay:  1 * time.Minute,
	}
}
