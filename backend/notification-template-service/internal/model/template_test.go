package model

import (
	"testing"
	"time"
)

func TestTemplateValidate(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    *Template
		wantErr error
	}{
		{
			name: "valid template",
			tmpl: &Template{
				ID:      "123e4567-e89b-12d3-a456-426614174000",
				Name:    "Test Template",
				Body:    "Monitor {{ .monitor_name }} is {{ .status }}",
				Channel: ChannelEmail,
				Type:    TypeMonitorDown,
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			tmpl: &Template{
				Body:    "Monitor {{ .monitor_name }} is {{ .status }}",
				Channel: ChannelEmail,
				Type:    TypeMonitorDown,
			},
			wantErr: ErrInvalidTemplateName,
		},
		{
			name: "empty body",
			tmpl: &Template{
				Name:    "Test Template",
				Channel: ChannelEmail,
				Type:    TypeMonitorDown,
			},
			wantErr: ErrInvalidTemplateBody,
		},
		{
			name: "invalid channel",
			tmpl: &Template{
				Name:    "Test Template",
				Body:    "Monitor {{ .monitor_name }} is {{ .status }}",
				Channel: TemplateChannel("invalid"),
				Type:    TypeMonitorDown,
			},
			wantErr: ErrInvalidChannel,
		},
		{
			name: "invalid type",
			tmpl: &Template{
				Name:    "Test Template",
				Body:    "Monitor {{ .monitor_name }} is {{ .status }}",
				Channel: ChannelEmail,
				Type:    TemplateType("invalid"),
			},
			wantErr: ErrInvalidType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tmpl.Validate()
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestTemplateCanDelete(t *testing.T) {
	tests := []struct {
		name     string
		isSystem bool
		want     bool
	}{
		{
			name:     "user template can be deleted",
			isSystem: false,
			want:     true,
		},
		{
			name:     "system template cannot be deleted",
			isSystem: true,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := &Template{IsSystem: tt.isSystem}
			if got := tmpl.CanDelete(); got != tt.want {
				t.Errorf("CanDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateClone(t *testing.T) {
	original := &Template{
		ID:        "123e4567-e89b-12d3-a456-426614174000",
		UserID:    "user-1",
		Name:      "Original Template",
		Channel:   ChannelEmail,
		Type:      TypeMonitorDown,
		Body:      "Monitor {{ .monitor_name }} is {{ .status }}",
		Format:    FormatHTML,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   2,
	}

	clone := original.Clone("Cloned Template")

	if clone.Name != "Cloned Template" {
		t.Errorf("Clone() name = %v, want %v", clone.Name, "Cloned Template")
	}

	if clone.ParentID == nil || *clone.ParentID != original.ID {
		t.Errorf("Clone() parentID = %v, want %v", clone.ParentID, original.ID)
	}

	if clone.Version != 1 {
		t.Errorf("Clone() version = %v, want %v", clone.Version, 1)
	}

	if clone.IsDefault {
		t.Error("Clone() should not be default")
	}

	if clone.IsSystem {
		t.Error("Clone() should not be system")
	}

	if clone.Body != original.Body {
		t.Error("Clone() body should match original")
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []string
	}{
		{
			name:     "simple variables",
			template: "Monitor {{ .monitor_name }} is {{ .status }}",
			want:     []string{"monitor_name", "status"},
		},
		{
			name:     "variable with space",
			template: "Monitor {{ .monitor_name }} is DOWN",
			want:     []string{"monitor_name"},
		},
		{
			name:     "no variables",
			template: "Monitor is DOWN",
			want:     nil,
		},
		{
			name:     "nested fields",
			template: "{{ .monitor.name }} - {{ .monitor.url }}",
			want:     []string{"monitor.name", "monitor.url"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVariables(tt.template)

			// Проверяем длину
			if len(got) != len(tt.want) {
				t.Errorf("ExtractVariables() = %v, want %v", got, tt.want)
				return
			}

			// Проверяем содержимое (для простоты - без учёта порядка)
			gotMap := make(map[string]bool)
			for _, v := range got {
				gotMap[v] = true
			}

			for _, wantVar := range tt.want {
				if !gotMap[wantVar] {
					t.Errorf("ExtractVariables() missing variable %v", wantVar)
				}
			}
		})
	}
}
