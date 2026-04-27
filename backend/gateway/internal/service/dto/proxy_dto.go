package dto

// ProxyRequest represents a proxy request DTO
type ProxyRequest struct {
	ServiceName string `json:"service_name"`
	Path        string `json:"path"`
	Method      string `json:"method"`
}

// ProxyResponse represents a proxy response DTO
type ProxyResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       any               `json:"body,omitempty"`
	Message    string            `json:"message,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string             `json:"status"`
	Timestamp string             `json:"timestamp"`
	Services  []ServiceHealthDTO `json:"services"`
	Gateway   GatewayDTO         `json:"gateway"`
}

// ServiceHealthDTO represents a service health check result
type ServiceHealthDTO struct {
	Name      string `json:"name"`
	Healthy   bool   `json:"healthy"`
	LatencyMS int64  `json:"latency_ms"`
	Message   string `json:"message"`
}

// GatewayDTO represents gateway information
type GatewayDTO struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	StartedAt string `json:"started_at"`
}

// NewHealthResponse creates a new health response
func NewHealthResponse(status string, timestamp string) *HealthResponse {
	return &HealthResponse{
		Status:    status,
		Timestamp: timestamp,
		Services:  make([]ServiceHealthDTO, 0),
		Gateway: GatewayDTO{
			Name:    "api-gateway",
			Version: "1.0.0",
		},
	}
}
