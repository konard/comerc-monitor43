package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/raul/monitor/gateway/internal/service/dto"
)

// GatewayService defines the interface for gateway operations
type GatewayService interface {
	CheckHealth(ctx context.Context) *dto.HealthResponse
	CheckAuthHealth(ctx context.Context) (bool, time.Duration, error)
	ProxyRequest(ctx context.Context, serviceName, path string, body []byte) (*dto.ProxyResponse, error)
}

// GatewayHandler handles gateway-related HTTP requests
type GatewayHandler struct {
	gatewayService GatewayService
}

// NewGatewayHandler creates a new gateway handler
func NewGatewayHandler(gatewayService GatewayService) *GatewayHandler {
	return &GatewayHandler{
		gatewayService: gatewayService,
	}
}

// Health handles GET /health
func (h *GatewayHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := h.gatewayService.CheckHealth(r.Context())
	h.respondJSON(w, resp, http.StatusOK)
}

// Readiness handles GET /ready
func (h *GatewayHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := h.gatewayService.CheckHealth(r.Context())

	// Only return 200 if all services are healthy
	if resp.Status != "healthy" {
		h.respondError(w, "Service not ready", http.StatusServiceUnavailable)
		return
	}

	h.respondJSON(w, resp, http.StatusOK)
}

// Liveness handles GET /live
func (h *GatewayHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Liveness is simpler - just check if gateway is running
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		return
	}
}

// Proxy handles proxy requests to downstream services
func (h *GatewayHandler) Proxy(w http.ResponseWriter, r *http.Request) {
	serviceName := r.PathValue("service")
	if serviceName == "" {
		h.respondError(w, "Service name required", http.StatusBadRequest)
		return
	}

	resp, err := h.gatewayService.ProxyRequest(r.Context(), serviceName, r.URL.Path, nil)
	if err != nil {
		h.respondError(w, "Proxy failed", http.StatusBadGateway)
		return
	}

	h.respondJSON(w, resp, resp.StatusCode)
}

// respondJSON sends a JSON response
func (h *GatewayHandler) respondJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// respondError sends an error response
func (h *GatewayHandler) respondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
