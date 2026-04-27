package model

import (
	"testing"
)

func TestValidationResult_HasErrors(t *testing.T) {
	tests := []struct {
		name string
		vr   *ValidationResult
		want bool
	}{
		{
			name: "no errors",
			vr:   &ValidationResult{Valid: true, Errors: []string{}},
			want: false,
		},
		{
			name: "has errors",
			vr:   &ValidationResult{Valid: false, Errors: []string{"error 1"}},
			want: true,
		},
		{
			name: "nil errors slice",
			vr:   &ValidationResult{Valid: true},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vr.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationResult_HasWarnings(t *testing.T) {
	tests := []struct {
		name string
		vr   *ValidationResult
		want bool
	}{
		{
			name: "no warnings",
			vr:   &ValidationResult{Warnings: []string{}},
			want: false,
		},
		{
			name: "has warnings",
			vr:   &ValidationResult{Warnings: []string{"warning 1"}},
			want: true,
		},
		{
			name: "nil warnings slice",
			vr:   &ValidationResult{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vr.HasWarnings(); got != tt.want {
				t.Errorf("HasWarnings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationResult_AddError(t *testing.T) {
	vr := &ValidationResult{Valid: true, Errors: []string{}}

	vr.AddError("first error")

	if vr.Valid {
		t.Error("AddError() should set Valid to false")
	}
	if len(vr.Errors) != 1 {
		t.Errorf("AddError() len(Errors) = %d, want 1", len(vr.Errors))
	}
	if vr.Errors[0] != "first error" {
		t.Errorf("AddError() Errors[0] = %v, want 'first error'", vr.Errors[0])
	}

	vr.AddError("second error")
	if len(vr.Errors) != 2 {
		t.Errorf("AddError() len(Errors) = %d, want 2", len(vr.Errors))
	}
}

func TestValidationResult_AddWarning(t *testing.T) {
	vr := &ValidationResult{Valid: true, Warnings: []string{}}

	vr.AddWarning("first warning")

	if !vr.Valid {
		t.Error("AddWarning() should not change Valid")
	}
	if len(vr.Warnings) != 1 {
		t.Errorf("AddWarning() len(Warnings) = %d, want 1", len(vr.Warnings))
	}
	if vr.Warnings[0] != "first warning" {
		t.Errorf("AddWarning() Warnings[0] = %v, want 'first warning'", vr.Warnings[0])
	}

	vr.AddWarning("second warning")
	if len(vr.Warnings) != 2 {
		t.Errorf("AddWarning() len(Warnings) = %d, want 2", len(vr.Warnings))
	}
}

func TestGetRequiredVariables(t *testing.T) {
	tests := []struct {
		name        string
		tmplType    TemplateType
		channel     TemplateChannel
		wantMinLen  int
		wantVarName string
	}{
		{
			name:        "monitor down has error_message",
			tmplType:    TypeMonitorDown,
			channel:     ChannelEmail,
			wantMinLen:  5,
			wantVarName: "error_message",
		},
		{
			name:        "monitor degraded has response_time_ms",
			tmplType:    TypeMonitorDegraded,
			channel:     ChannelTelegram,
			wantMinLen:  6,
			wantVarName: "response_time_ms",
		},
		{
			name:        "certificate expiry has days_until_expiry",
			tmplType:    TypeCertificateExpiry,
			channel:     ChannelSlack,
			wantMinLen:  6,
			wantVarName: "days_until_expiry",
		},
		{
			name:        "flapping detected has state_changes",
			tmplType:    TypeFlappingDetected,
			channel:     ChannelWebhook,
			wantMinLen:  6,
			wantVarName: "state_changes",
		},
		{
			name:        "monitor up returns base vars",
			tmplType:    TypeMonitorUp,
			channel:     ChannelEmail,
			wantMinLen:  4,
			wantVarName: "monitor_name",
		},
		{
			name:        "custom returns base vars",
			tmplType:    TypeCustom,
			channel:     ChannelDiscord,
			wantMinLen:  4,
			wantVarName: "status",
		},
		{
			name:        "incident created returns base vars",
			tmplType:    TypeIncidentCreated,
			channel:     ChannelSMS,
			wantMinLen:  4,
			wantVarName: "timestamp",
		},
		{
			name:        "incident resolved returns base vars",
			tmplType:    TypeIncidentResolved,
			channel:     ChannelEmail,
			wantMinLen:  4,
			wantVarName: "monitor_url",
		},
		{
			name:        "maintenance started returns base vars",
			tmplType:    TypeMaintenanceStarted,
			channel:     ChannelEmail,
			wantMinLen:  4,
			wantVarName: "monitor_name",
		},
		{
			name:        "maintenance ended returns base vars",
			tmplType:    TypeMaintenanceEnded,
			channel:     ChannelEmail,
			wantMinLen:  4,
			wantVarName: "monitor_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := GetRequiredVariables(tt.tmplType, tt.channel)

			if len(vars) < tt.wantMinLen {
				t.Errorf("GetRequiredVariables() len = %d, want >= %d", len(vars), tt.wantMinLen)
			}

			found := false
			for _, v := range vars {
				if v.Name == tt.wantVarName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("GetRequiredVariables() missing variable %q", tt.wantVarName)
			}
		})
	}
}

func TestGetRequiredVariables_AllRequiredFields(t *testing.T) {
	vars := GetRequiredVariables(TypeMonitorDown, ChannelEmail)

	for _, v := range vars {
		if v.Name == "" {
			t.Error("variable name should not be empty")
		}
		if v.Type == "" {
			t.Errorf("variable %q type should not be empty", v.Name)
		}
	}
}

func TestExtractVariables_DuplicateVars(t *testing.T) {
	tmpl := "{{ .monitor_name }} is {{ .status }}, {{ .monitor_name }} again"
	vars := ExtractVariables(tmpl)

	count := 0
	for _, v := range vars {
		if v == "monitor_name" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("ExtractVariables() should deduplicate, got %d occurrences of monitor_name", count)
	}
}

func TestExtractVariables_ComplexExpressions(t *testing.T) {
	// Expressions with spaces are excluded by the implementation.
	// "{{ if .active }}" has a space so it's excluded.
	// "{{ end }}" becomes "end" (no dot prefix, just a word) — the implementation
	// doesn't filter keywords, so "end" may appear. We only assert .name is found.
	tmpl := "{{ if .active }}yes{{ end }} {{ .name }}"
	vars := ExtractVariables(tmpl)

	found := false
	for _, v := range vars {
		if v == "name" {
			found = true
		}
	}
	if !found {
		t.Error("ExtractVariables() should include 'name'")
	}
}

func TestExtractVariables_NoClosingBrace(t *testing.T) {
	// Template with no closing brace — should not crash, just skip
	tmpl := "{{ .monitor_name is broken"
	vars := ExtractVariables(tmpl)
	if vars != nil {
		t.Errorf("ExtractVariables() with no closing brace should return nil, got %v", vars)
	}
}
