//go:build bdd

package suite

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"
)

// FakeWebhookTLSServer — HTTPS-вариант FakeWebhookServer с self-signed сертификатом.
// Используется в BDD-сценариях, проверяющих поведение alert-service при доставке на TLS endpoint.
type FakeWebhookTLSServer struct {
	server   *http.Server
	listener net.Listener
	url      string
	cert     []byte // PEM-encoded сертификат — доступен для проверки

	mu       sync.Mutex
	requests []ReceivedWebhook
}

// NewFakeWebhookTLSServer генерирует self-signed сертификат и поднимает HTTPS-сервер на случайном порту.
func NewFakeWebhookTLSServer() (*FakeWebhookTLSServer, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "bdd-fake-webhook-tls"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("tls listen: %w", err)
	}
	addr := ln.Addr().(*net.TCPAddr)

	s := &FakeWebhookTLSServer{
		listener: ln,
		url:      fmt.Sprintf("https://%s:%d/", addr.IP.String(), addr.Port),
		cert:     certPEM,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)

	s.server = &http.Server{
		Handler:           mux,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		//nolint:errcheck // ошибки сервера обрабатываются через Stop
		_ = s.server.Serve(ln)
	}()

	return s, nil
}

// URL возвращает базовый URL сервера (с финальным "/").
func (s *FakeWebhookTLSServer) URL() string { return s.url }

// CertPEM возвращает PEM-encoded self-signed сертификат (для тестов, которые должны добавить CA).
func (s *FakeWebhookTLSServer) CertPEM() []byte { return s.cert }

// Requests возвращает копию принятых запросов.
func (s *FakeWebhookTLSServer) Requests() []ReceivedWebhook {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ReceivedWebhook, len(s.requests))
	copy(out, s.requests)
	return out
}

// Reset очищает накопленные запросы.
func (s *FakeWebhookTLSServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = nil
}

// Stop останавливает HTTPS-сервер с graceful shutdown.
func (s *FakeWebhookTLSServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *FakeWebhookTLSServer) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	//nolint:errcheck // best-effort close
	_ = r.Body.Close()

	rw := ReceivedWebhook{
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   r.Header.Clone(),
		Body:      json.RawMessage(append([]byte(nil), body...)),
		Timestamp: time.Now(),
	}

	s.mu.Lock()
	s.requests = append(s.requests, rw)
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	//nolint:errcheck // best-effort write
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
