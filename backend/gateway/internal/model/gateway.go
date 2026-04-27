package model

// Gateway represents the API Gateway service
type Gateway struct {
	Name      string
	Version   string
	StartedAt string
}

// ServiceHealth represents the health status of a downstream service
type ServiceHealth struct {
	Name      string
	Healthy   bool
	LatencyMS int64
	Message   string
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    string          `json:"status"`
	Timestamp string          `json:"timestamp"`
	Services  []ServiceHealth `json:"services"`
	Gateway   Gateway         `json:"gateway"`
}

// NewHealthStatus creates a new health status
func NewHealthStatus(status string) *HealthStatus {
	return &HealthStatus{
		Status:   status,
		Services: make([]ServiceHealth, 0),
		Gateway: Gateway{
			Name:      "api-gateway",
			Version:   "1.0.0",
			StartedAt: "",
		},
	}
}

// AddService adds a service health check
func (h *HealthStatus) AddService(name string, healthy bool, latencyMS int64, message string) {
	h.Services = append(h.Services, ServiceHealth{
		Name:      name,
		Healthy:   healthy,
		LatencyMS: latencyMS,
		Message:   message,
	})
}

// IsHealthy returns true if all services are healthy
func (h *HealthStatus) IsHealthy() bool {
	if h.Status != "healthy" {
		return false
	}
	for _, service := range h.Services {
		if !service.Healthy {
			return false
		}
	}
	return true
}
