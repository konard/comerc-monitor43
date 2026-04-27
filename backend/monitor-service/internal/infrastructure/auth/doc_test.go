package auth

import "testing"

// TestDocCompile проверяет, что пакет компилируется.
func TestDocCompile(t *testing.T) {
	_ = &AuthClient{}
	_ = NewAuthClient
	_ = (*AuthClient).ValidateToken
	_ = (*AuthClient).Close
}
