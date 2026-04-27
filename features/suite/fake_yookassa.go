//go:build bdd

package suite

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
)

// FakeYookassaServer имитирует минимальный HTTP API YooKassa для BDD checkout-сценариев.
type FakeYookassaServer struct {
	server *httptest.Server
	seq    atomic.Uint64
}

// NewFakeYookassaServer создаёт и запускает fake YooKassa сервер.
func NewFakeYookassaServer() *FakeYookassaServer {
	f := &FakeYookassaServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/v3/payments", f.handlePayments)
	mux.HandleFunc("/v3/payments/", f.handlePayment)
	f.server = httptest.NewServer(mux)
	return f
}

// BaseURL возвращает URL API с префиксом версии.
func (f *FakeYookassaServer) BaseURL() string {
	return f.server.URL + "/v3"
}

// Stop останавливает fake YooKassa сервер.
func (f *FakeYookassaServer) Stop() {
	if f == nil || f.server == nil {
		return
	}
	f.server.Close()
}

func (f *FakeYookassaServer) handlePayments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := f.nextPaymentID()
	writeYookassaJSON(w, map[string]any{
		"id":     id,
		"status": "pending",
		"confirmation": map[string]string{
			"type":             "redirect",
			"confirmation_url": f.server.URL + "/checkout/" + id,
		},
		"amount": map[string]string{
			"value":    "299.00",
			"currency": "RUB",
		},
	})
}

func (f *FakeYookassaServer) handlePayment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v3/payments/")
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeYookassaJSON(w, map[string]any{
		"id":     id,
		"status": "succeeded",
		"amount": map[string]string{
			"value":    "299.00",
			"currency": "RUB",
		},
		"metadata":   map[string]string{},
		"created_at": "2026-01-01T00:00:00Z",
	})
}

func (f *FakeYookassaServer) nextPaymentID() string {
	return "bdd-yookassa-" + strconv.FormatUint(f.seq.Add(1), 10)
}

func writeYookassaJSON(w http.ResponseWriter, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}
