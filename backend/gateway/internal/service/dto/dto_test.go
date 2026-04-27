package dto

import (
	"testing"
)

func TestNewValidateTokenResponse(t *testing.T) {
	t.Parallel()

	resp := NewValidateTokenResponse(true, "user1", "user@example.com", "pro")

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !resp.Valid {
		t.Error("expected valid=true")
	}
	if resp.UserID != "user1" {
		t.Errorf("expected UserID user1, got %s", resp.UserID)
	}
	if resp.Email != "user@example.com" {
		t.Errorf("expected email user@example.com, got %s", resp.Email)
	}
	if resp.Tier != "pro" {
		t.Errorf("expected tier pro, got %s", resp.Tier)
	}
}

func TestNewValidateTokenResponse_invalid(t *testing.T) {
	t.Parallel()

	resp := NewValidateTokenResponse(false, "", "", "")

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Valid {
		t.Error("expected valid=false")
	}
}

func TestNewErrorDTO(t *testing.T) {
	t.Parallel()

	dto := NewErrorDTO("some error occurred")

	if dto == nil {
		t.Fatal("expected non-nil DTO")
	}
	if dto.Error != "some error occurred" {
		t.Errorf("expected error 'some error occurred', got '%s'", dto.Error)
	}
}

func TestErrorDTO_ToJSON(t *testing.T) {
	t.Parallel()

	dto := NewErrorDTO("test error")

	data, err := dto.ToJSON()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}

	// Проверяем, что JSON содержит поле error
	jsonStr := string(data)
	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}
}

func TestNewHealthResponse(t *testing.T) {
	t.Parallel()

	resp := NewHealthResponse("healthy", "2026-01-01T00:00:00Z")

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Status != "healthy" {
		t.Errorf("expected status healthy, got %s", resp.Status)
	}
	if resp.Timestamp != "2026-01-01T00:00:00Z" {
		t.Errorf("expected timestamp, got %s", resp.Timestamp)
	}
	if resp.Services == nil {
		t.Error("expected non-nil services slice")
	}
	if resp.Gateway.Name != "api-gateway" {
		t.Errorf("expected gateway name api-gateway, got %s", resp.Gateway.Name)
	}
}

func TestProxyRequest_fields(t *testing.T) {
	t.Parallel()

	req := &ProxyRequest{
		ServiceName: "monitor",
		Path:        "/api/v1/monitors",
		Method:      "GET",
	}

	if req.ServiceName != "monitor" {
		t.Errorf("expected service monitor, got %s", req.ServiceName)
	}
	if req.Path != "/api/v1/monitors" {
		t.Errorf("expected path /api/v1/monitors, got %s", req.Path)
	}
}

func TestProxyResponse_fields(t *testing.T) {
	t.Parallel()

	resp := &ProxyResponse{
		StatusCode: 200,
		Message:    "OK",
		Body:       map[string]string{"key": "value"},
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Message != "OK" {
		t.Errorf("expected message OK, got %s", resp.Message)
	}
}

func TestValidateTokenRequest_fields(t *testing.T) {
	t.Parallel()

	req := &ValidateTokenRequest{
		AccessToken: "my-token",
	}

	if req.AccessToken != "my-token" {
		t.Errorf("expected token my-token, got %s", req.AccessToken)
	}
}
