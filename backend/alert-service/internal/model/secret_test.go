package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		secret string
		want   string
	}{
		{
			name:   "short_secret",
			secret: "abc",
			want:   "***",
		},
		{
			name:   "empty_string",
			secret: "",
			want:   "",
		},
		{
			name:   "exact_8_chars",
			secret: "abcdefgh",
			want:   "********",
		},
		{
			name:   "long_secret",
			secret: "my-super-secret-api-key-12345",
			want:   "my-supe**********************",
		},
		{
			name:   "single_char",
			secret: "x",
			want:   "*",
		},
		{
			name:   "two_chars",
			secret: "ab",
			want:   "**",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskSecret(tt.secret)

			assert.Equal(t, tt.want, got)
			assert.Len(t, got, len(tt.secret))
		})
	}
}
