package executor

import (
	"crypto/tls"
	"net/http/httptrace"
	"time"
)

// HTTPTracer собирает детальные метрики HTTP запросов
type HTTPTracer struct {
	dnsStart          time.Time
	dnsDone           time.Time
	connectStart      time.Time
	connectDone       time.Time
	tlsHandshakeStart time.Time
	tlsHandshakeDone  time.Time
	wroteRequest      time.Time
	gotFirstByte      time.Time
	end               time.Time
}

// NewHTTPTracer создаёт новый tracer
func NewHTTPTracer() *HTTPTracer {
	return &HTTPTracer{}
}

// Trace возвращает httptrace.ClientTrace для сбора метрик
func (t *HTTPTracer) Trace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			t.dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			t.dnsDone = time.Now()
		},
		ConnectStart: func(_ string, _ string) {
			t.connectStart = time.Now()
		},
		ConnectDone: func(_ string, _ string, _ error) {
			t.connectDone = time.Now()
		},
		TLSHandshakeStart: func() {
			t.tlsHandshakeStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			t.tlsHandshakeDone = time.Now()
		},
		WroteRequest: func(_ httptrace.WroteRequestInfo) {
			t.wroteRequest = time.Now()
		},
		GotFirstResponseByte: func() {
			t.gotFirstByte = time.Now()
		},
	}
}

// GetMetrics возвращает собранные метрики
func (t *HTTPTracer) GetMetrics() DetailedMetrics {
	metrics := DetailedMetrics{}

	// DNS lookup time
	if !t.dnsStart.IsZero() && !t.dnsDone.IsZero() {
		metrics.DNSLookupMs = float64(t.dnsDone.Sub(t.dnsStart).Milliseconds())
	}

	// TCP connect time
	if !t.connectStart.IsZero() && !t.connectDone.IsZero() {
		metrics.TCPConnectMs = float64(t.connectDone.Sub(t.connectStart).Milliseconds())
	}

	// TLS handshake time
	if !t.tlsHandshakeStart.IsZero() && !t.tlsHandshakeDone.IsZero() {
		metrics.TLSHandshakeMs = float64(t.tlsHandshakeDone.Sub(t.tlsHandshakeStart).Milliseconds())
	}

	// Time to First Byte (TTFB)
	if !t.wroteRequest.IsZero() && !t.gotFirstByte.IsZero() {
		metrics.TTFBMs = float64(t.gotFirstByte.Sub(t.wroteRequest).Milliseconds())
	}

	// Total time
	if !t.dnsStart.IsZero() && !t.gotFirstByte.IsZero() {
		metrics.TotalMs = float64(t.gotFirstByte.Sub(t.dnsStart).Milliseconds())
	}

	return metrics
}

// RecordEnd записывает время окончания запроса
func (t *HTTPTracer) RecordEnd() {
	t.end = time.Now()
}

// GetTotalTime возвращает общее время запроса
func (t *HTTPTracer) GetTotalTime() time.Duration {
	if t.end.IsZero() {
		return 0
	}
	if !t.dnsStart.IsZero() {
		return t.end.Sub(t.dnsStart)
	}
	return 0
}
