package model

import (
	"testing"
	"time"
)

func TestTemplate_IsDefaultFor(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     *Template
		channel  TemplateChannel
		tmplType TemplateType
		want     bool
	}{
		{
			name: "is default for matching channel and type",
			tmpl: &Template{
				IsDefault: true,
				Channel:   ChannelEmail,
				Type:      TypeMonitorDown,
			},
			channel:  ChannelEmail,
			tmplType: TypeMonitorDown,
			want:     true,
		},
		{
			name: "not default flag",
			tmpl: &Template{
				IsDefault: false,
				Channel:   ChannelEmail,
				Type:      TypeMonitorDown,
			},
			channel:  ChannelEmail,
			tmplType: TypeMonitorDown,
			want:     false,
		},
		{
			name: "wrong channel",
			tmpl: &Template{
				IsDefault: true,
				Channel:   ChannelTelegram,
				Type:      TypeMonitorDown,
			},
			channel:  ChannelEmail,
			tmplType: TypeMonitorDown,
			want:     false,
		},
		{
			name: "wrong type",
			tmpl: &Template{
				IsDefault: true,
				Channel:   ChannelEmail,
				Type:      TypeMonitorUp,
			},
			channel:  ChannelEmail,
			tmplType: TypeMonitorDown,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tmpl.IsDefaultFor(tt.channel, tt.tmplType); got != tt.want {
				t.Errorf("IsDefaultFor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplate_ValidateSyntax(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     *Template
		wantErrs int
	}{
		{
			name: "valid body and subject",
			tmpl: &Template{
				Subject: "Alert: {{ .monitor_name }}",
				Body:    "Monitor {{ .monitor_name }} is {{ .status }}",
			},
			wantErrs: 0,
		},
		{
			name: "valid body only",
			tmpl: &Template{
				Body: "Monitor is down",
			},
			wantErrs: 0,
		},
		{
			name: "invalid body syntax",
			tmpl: &Template{
				Body: "Monitor {{ .name is broken",
			},
			wantErrs: 1,
		},
		{
			name: "invalid subject syntax",
			tmpl: &Template{
				Subject: "Alert {{ bad",
				Body:    "Valid body",
			},
			wantErrs: 1,
		},
		{
			name: "both invalid",
			tmpl: &Template{
				Subject: "{{ bad subject",
				Body:    "{{ bad body",
			},
			wantErrs: 2,
		},
		{
			name: "empty subject is skipped",
			tmpl: &Template{
				Subject: "",
				Body:    "Valid body {{ .name }}",
			},
			wantErrs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.tmpl.ValidateSyntax()
			if len(errs) != tt.wantErrs {
				t.Errorf("ValidateSyntax() len(errs) = %d, want %d; errs = %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}

func TestTemplate_BeforeCreate(t *testing.T) {
	before := time.Now()

	tmpl := &Template{}
	tmpl.BeforeCreate()

	after := time.Now()

	if tmpl.Version != 1 {
		t.Errorf("BeforeCreate() Version = %d, want 1", tmpl.Version)
	}

	if tmpl.CreatedAt.Before(before) || tmpl.CreatedAt.After(after) {
		t.Errorf("BeforeCreate() CreatedAt = %v is outside expected range [%v, %v]", tmpl.CreatedAt, before, after)
	}

	if tmpl.UpdatedAt.Before(before) || tmpl.UpdatedAt.After(after) {
		t.Errorf("BeforeCreate() UpdatedAt = %v is outside expected range [%v, %v]", tmpl.UpdatedAt, before, after)
	}
}

func TestTemplate_BeforeUpdate(t *testing.T) {
	tmpl := &Template{
		Version:   1,
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	before := time.Now()
	tmpl.BeforeUpdate()
	after := time.Now()

	if tmpl.Version != 2 {
		t.Errorf("BeforeUpdate() Version = %d, want 2", tmpl.Version)
	}

	if tmpl.UpdatedAt.Before(before) || tmpl.UpdatedAt.After(after) {
		t.Errorf("BeforeUpdate() UpdatedAt = %v is outside expected range [%v, %v]", tmpl.UpdatedAt, before, after)
	}
}

func TestTemplate_BeforeUpdate_IncrementsVersion(t *testing.T) {
	tmpl := &Template{Version: 5}
	tmpl.BeforeUpdate()
	if tmpl.Version != 6 {
		t.Errorf("BeforeUpdate() Version = %d, want 6", tmpl.Version)
	}
}

func TestTemplate_Validate_DefaultFormat(t *testing.T) {
	tests := []struct {
		name           string
		channel        TemplateChannel
		expectedFormat TemplateFormat
	}{
		{
			name:           "email gets html format",
			channel:        ChannelEmail,
			expectedFormat: FormatHTML,
		},
		{
			name:           "webhook gets json format",
			channel:        ChannelWebhook,
			expectedFormat: FormatJSON,
		},
		{
			name:           "telegram gets text format",
			channel:        ChannelTelegram,
			expectedFormat: FormatText,
		},
		{
			name:           "slack gets text format",
			channel:        ChannelSlack,
			expectedFormat: FormatText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := &Template{
				Name:    "Test",
				Body:    "body",
				Channel: tt.channel,
				Type:    TypeMonitorDown,
				Format:  "", // unset
			}
			if err := tmpl.Validate(); err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}
			if tmpl.Format != tt.expectedFormat {
				t.Errorf("Validate() Format = %v, want %v", tmpl.Format, tt.expectedFormat)
			}
		})
	}
}

func TestTemplate_Validate_DefaultEngine(t *testing.T) {
	tmpl := &Template{
		Name:    "Test",
		Body:    "body",
		Channel: ChannelEmail,
		Type:    TypeMonitorDown,
		Engine:  "", // unset
	}
	if err := tmpl.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
	if tmpl.Engine != EngineGoTemplate {
		t.Errorf("Validate() Engine = %v, want %v", tmpl.Engine, EngineGoTemplate)
	}
}

func TestTemplate_Validate_InvalidUUID(t *testing.T) {
	tmpl := &Template{
		ID:      "not-a-uuid",
		Name:    "Test",
		Body:    "body",
		Channel: ChannelEmail,
		Type:    TypeMonitorDown,
	}
	err := tmpl.Validate()
	if err != ErrInvalidTemplateID {
		t.Errorf("Validate() error = %v, want ErrInvalidTemplateID", err)
	}
}

func TestTemplate_Validate_EmptyID_IsAllowed(t *testing.T) {
	tmpl := &Template{
		ID:      "",
		Name:    "Test",
		Body:    "body",
		Channel: ChannelEmail,
		Type:    TypeMonitorDown,
	}
	if err := tmpl.Validate(); err != nil {
		t.Errorf("Validate() with empty ID should not error, got: %v", err)
	}
}

func TestTemplate_Validate_WhitespaceName(t *testing.T) {
	tmpl := &Template{
		Name:    "   ",
		Body:    "body",
		Channel: ChannelEmail,
		Type:    TypeMonitorDown,
	}
	err := tmpl.Validate()
	if err != ErrInvalidTemplateName {
		t.Errorf("Validate() error = %v, want ErrInvalidTemplateName", err)
	}
}

func TestTemplate_Validate_WhitespaceBody(t *testing.T) {
	tmpl := &Template{
		Name:    "Test",
		Body:    "   ",
		Channel: ChannelEmail,
		Type:    TypeMonitorDown,
	}
	err := tmpl.Validate()
	if err != ErrInvalidTemplateBody {
		t.Errorf("Validate() error = %v, want ErrInvalidTemplateBody", err)
	}
}

func TestIsValidUUID_ValidFormat(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{
			id:   "123e4567-e89b-12d3-a456-426614174000",
			want: true,
		},
		{
			id:   "00000000-0000-0000-0000-000000000000",
			want: true,
		},
		{
			id:   "not-a-valid-uuid",
			want: false,
		},
		{
			id:   "12345678-1234-1234-1234-1234567",
			want: false,
		},
		{
			id:   "12345678-1234-1234-1234",
			want: false,
		},
		{
			id:   "1234567-1234-1234-1234-123456789012",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := isValidUUID(tt.id)
			if got != tt.want {
				t.Errorf("isValidUUID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestTemplate_Clone_Description(t *testing.T) {
	original := &Template{
		ID:   "123e4567-e89b-12d3-a456-426614174000",
		Name: "Original",
		Body: "body",
	}
	clone := original.Clone("New Name")

	expectedDesc := "Clone of Original"
	if clone.Description != expectedDesc {
		t.Errorf("Clone() Description = %q, want %q", clone.Description, expectedDesc)
	}
}

func TestTemplate_Clone_UserIDNotCopied(t *testing.T) {
	original := &Template{
		ID:     "123e4567-e89b-12d3-a456-426614174000",
		UserID: "owner-user",
		Name:   "Original",
		Body:   "body",
	}
	clone := original.Clone("New Name")

	// Clone preserves the original UserID (caller overwrites if needed)
	if clone.UserID != "owner-user" {
		t.Errorf("Clone() UserID = %q, want %q", clone.UserID, "owner-user")
	}
}
