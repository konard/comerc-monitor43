package model

// ProxyRequest represents a proxy request to a downstream service
type ProxyRequest struct {
	ServiceName string
	Path        string
	Method      string
	Headers     map[string]string
	Body        []byte
}

// ProxyResponse represents the response from a downstream service
type ProxyResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Error      error
}

// NewProxyResponse creates a new proxy response
func NewProxyResponse(statusCode int, body []byte) *ProxyResponse {
	return &ProxyResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    make(map[string]string),
	}
}

// NewProxyErrorResponse creates a proxy response with an error
func NewProxyErrorResponse(err error) *ProxyResponse {
	return &ProxyResponse{
		Error: err,
	}
}
