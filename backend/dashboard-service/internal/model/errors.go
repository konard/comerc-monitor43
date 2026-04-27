package model

import "fmt"

type DomainError struct {
	code    string
	message string
}

func NewDomainError(message string) *DomainError {
	return &DomainError{code: "DOMAIN_ERROR", message: message}
}

func (e *DomainError) Error() string                     { return e.message }
func (e *DomainError) Code() string                      { return e.code }
func (e *DomainError) WithCode(code string) *DomainError { e.code = code; return e }
func (e *DomainError) Wrap(message string) error {
	return &DomainError{code: e.code, message: fmt.Sprintf("%s: %s", message, e.message)}
}

var (
	ErrMonitorNotFound    = NewDomainError("monitor not found")
	ErrIncidentNotFound   = NewDomainError("incident not found")
	ErrInvalidFilter      = NewDomainError("invalid filter parameter").WithCode("INVALID_FILTER_PARAMETER")
	ErrInvalidPageSize    = NewDomainError("page size exceeds maximum").WithCode("INVALID_PAGE_SIZE")
	ErrInvalidDateRange   = NewDomainError("start date cannot be greater than end date").WithCode("INVALID_DATE_RANGE")
	ErrInvalidSearchQuery = NewDomainError("search query contains invalid characters").WithCode("INVALID_SEARCH_QUERY")
	ErrSearchQueryTooLong = NewDomainError("search query exceeds maximum length").WithCode("SEARCH_QUERY_TOO_LONG")
	ErrFilterTooLong      = NewDomainError("filter exceeds maximum length").WithCode("FILTER_TOO_LONG")
	ErrExportFailed       = NewDomainError("failed to generate export").WithCode("EXPORT_FAILED")
)
