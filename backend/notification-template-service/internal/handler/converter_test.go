package handler

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestTimestampToProto проверяет конвертацию time.Time в protobuf Timestamp.
func TestTimestampToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   time.Time
		wantNil bool
	}{
		{
			name:    "zero time returns nil",
			input:   time.Time{},
			wantNil: true,
		},
		{
			name:    "non-zero time returns timestamp",
			input:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			wantNil: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := timestampToProto(tc.input)

			// Assert
			if tc.wantNil && result != nil {
				t.Errorf("timestampToProto() = %v, want nil", result)
			}
			if !tc.wantNil && result == nil {
				t.Error("timestampToProto() = nil, want non-nil")
			}
			if !tc.wantNil && result != nil {
				got := result.AsTime().UTC()
				if !got.Equal(tc.input.UTC()) {
					t.Errorf("timestampToProto() time = %v, want %v", got, tc.input.UTC())
				}
			}
		})
	}
}

// TestProtoToTimestamp проверяет конвертацию protobuf Timestamp в time.Time.
func TestProtoToTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *timestamppb.Timestamp
		want  time.Time
	}{
		{
			name:  "nil timestamp returns zero time",
			input: nil,
			want:  time.Time{},
		},
		{
			name:  "valid timestamp returns time",
			input: timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			want:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := protoToTimestamp(tc.input)

			// Assert
			if !result.UTC().Equal(tc.want.UTC()) {
				t.Errorf("protoToTimestamp() = %v, want %v", result.UTC(), tc.want.UTC())
			}
		})
	}
}
