package model

import "strings"

func MaskSecret(secret string) string {
	if len(secret) <= 8 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:7] + strings.Repeat("*", len(secret)-7)
}
