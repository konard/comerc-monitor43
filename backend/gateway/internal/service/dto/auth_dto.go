package dto

import "encoding/json"

// ValidateTokenRequest represents a token validation request DTO
type ValidateTokenRequest struct {
	AccessToken string `json:"access_token"`
}

// ValidateTokenResponse represents a token validation response DTO
type ValidateTokenResponse struct {
	Valid   bool   `json:"valid"`
	UserID  string `json:"user_id,omitempty"`
	Email   string `json:"email,omitempty"`
	Tier    string `json:"tier,omitempty"`
	Message string `json:"message,omitempty"`
}

// ErrorDTO represents a structured error response
type ErrorDTO struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// NewValidateTokenResponse creates a new validation response
func NewValidateTokenResponse(valid bool, userID, email, tier string) *ValidateTokenResponse {
	return &ValidateTokenResponse{
		Valid:  valid,
		UserID: userID,
		Email:  email,
		Tier:   tier,
	}
}

// NewErrorDTO creates a new error DTO
func NewErrorDTO(err string) *ErrorDTO {
	return &ErrorDTO{
		Error: err,
	}
}

// ToJSON converts the DTO to JSON bytes
func (d *ErrorDTO) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}
