package health

import (
	"context"
	"fmt"
	"net/http"
)

type HealthChecker struct {
	schedulerConn healthPinger
	monitorConn   healthPinger
}

type healthPinger interface {
	IsConnected() bool
}

func NewHealthChecker(schedulerConn, monitorConn healthPinger) *HealthChecker {
	return &HealthChecker{
		schedulerConn: schedulerConn,
		monitorConn:   monitorConn,
	}
}

func (h *HealthChecker) Check(ctx context.Context) error {
	if h.schedulerConn != nil {
		if !h.schedulerConn.IsConnected() {
			return fmt.Errorf("scheduler connection unhealthy")
		}
	}
	if h.monitorConn != nil {
		if !h.monitorConn.IsConnected() {
			return fmt.Errorf("monitor connection unhealthy")
		}
	}
	return nil
}

func (h *HealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*1000*1000*1000)
	defer cancel()

	if err := h.Check(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"unhealthy","error":"%s"}`, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"healthy"}`)); err != nil {
		return
	}
}
