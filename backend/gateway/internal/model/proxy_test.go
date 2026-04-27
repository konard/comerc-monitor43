package model

import (
	"errors"
	"testing"
)

func TestNewProxyResponse(t *testing.T) {
	body := []byte("response body")
	resp := NewProxyResponse(200, body)

	if resp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode)
	}

	if string(resp.Body) != string(body) {
		t.Errorf("expected body '%s', got '%s'", string(body), string(resp.Body))
	}

	if resp.Headers == nil {
		t.Error("expected headers to be initialized")
	}
}

func TestNewProxyErrorResponse(t *testing.T) {
	err := errors.New("test error")
	resp := NewProxyErrorResponse(err)

	if resp.StatusCode != 0 {
		t.Errorf("expected status code 0, got %d", resp.StatusCode)
	}

	if resp.Error == nil {
		t.Error("expected error to be set")
	}

	if resp.Error != err {
		t.Errorf("expected error '%v', got '%v'", err, resp.Error)
	}
}

func TestProxyRequest(t *testing.T) {
	req := ProxyRequest{
		ServiceName: "monitor",
		Path:        "/api/v1/monitors",
		Method:      "GET",
		Headers:     map[string]string{"Content-Type": "application/json"},
		Body:        []byte("request body"),
	}

	if req.ServiceName != "monitor" {
		t.Errorf("expected service name 'monitor', got '%s'", req.ServiceName)
	}

	if req.Path != "/api/v1/monitors" {
		t.Errorf("expected path '/api/v1/monitors', got '%s'", req.Path)
	}

	if req.Method != "GET" {
		t.Errorf("expected method 'GET', got '%s'", req.Method)
	}

	if req.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header 'application/json', got '%s'", req.Headers["Content-Type"])
	}

	if string(req.Body) != "request body" {
		t.Errorf("expected body 'request body', got '%s'", string(req.Body))
	}
}

func TestProxyResponse(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		body := []byte("success")
		resp := &ProxyResponse{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       body,
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected status code 200, got %d", resp.StatusCode)
		}

		if string(resp.Body) != string(body) {
			t.Errorf("expected body '%s', got '%s'", string(body), string(resp.Body))
		}

		if resp.Error != nil {
			t.Error("expected no error")
		}
	})

	t.Run("error response", func(t *testing.T) {
		err := errors.New("proxy error")
		resp := &ProxyResponse{
			Error: err,
		}

		if resp.Error == nil {
			t.Error("expected error to be set")
		}

		if resp.Error != err {
			t.Errorf("expected error '%v', got '%v'", err, resp.Error)
		}
	})
}
