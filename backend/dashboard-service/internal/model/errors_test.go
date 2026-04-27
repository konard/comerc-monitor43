package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainError_Error(t *testing.T) {
	t.Parallel()

	err := NewDomainError("something went wrong")

	assert.Equal(t, "something went wrong", err.Error())
}

func TestDomainError_Code(t *testing.T) {
	t.Parallel()

	err := NewDomainError("test").WithCode("CUSTOM_CODE")

	assert.Equal(t, "CUSTOM_CODE", err.Code())
}

func TestDomainError_Wrap(t *testing.T) {
	t.Parallel()

	inner := NewDomainError("monitor not found").WithCode("NOT_FOUND")
	wrapped := inner.Wrap("failed to get monitor")

	require.Error(t, wrapped)
	assert.Equal(t, "failed to get monitor: monitor not found", wrapped.Error())

	var domainErr *DomainError
	require.True(t, assert.ErrorAs(t, wrapped, &domainErr))
	assert.Equal(t, "NOT_FOUND", domainErr.Code())
}

func TestDomainError_SentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *DomainError
		wantMsg  string
		wantCode string
	}{
		{"monitor not found", ErrMonitorNotFound, "monitor not found", "DOMAIN_ERROR"},
		{"incident not found", ErrIncidentNotFound, "incident not found", "DOMAIN_ERROR"},
		{"invalid filter", ErrInvalidFilter, "invalid filter parameter", "INVALID_FILTER_PARAMETER"},
		{"invalid page size", ErrInvalidPageSize, "page size exceeds maximum", "INVALID_PAGE_SIZE"},
		{"invalid date range", ErrInvalidDateRange, "start date cannot be greater than end date", "INVALID_DATE_RANGE"},
		{"invalid search query", ErrInvalidSearchQuery, "search query contains invalid characters", "INVALID_SEARCH_QUERY"},
		{"search query too long", ErrSearchQueryTooLong, "search query exceeds maximum length", "SEARCH_QUERY_TOO_LONG"},
		{"filter too long", ErrFilterTooLong, "filter exceeds maximum length", "FILTER_TOO_LONG"},
		{"export failed", ErrExportFailed, "failed to generate export", "EXPORT_FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NotNil(t, tt.err)
			assert.Equal(t, tt.wantMsg, tt.err.Error())
			assert.Equal(t, tt.wantCode, tt.err.Code())
		})
	}
}

func TestDomainError_CodeChange(t *testing.T) {
	t.Parallel()

	err := NewDomainError("base error")

	originalPtr := err.WithCode("NEW_CODE")

	assert.Same(t, err, originalPtr)
	assert.Equal(t, "NEW_CODE", err.Code())
}
