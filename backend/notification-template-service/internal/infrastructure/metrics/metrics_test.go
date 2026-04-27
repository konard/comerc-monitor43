package metrics

import (
	"testing"
)

// TestRecordTemplateCreate проверяет запись метрики создания шаблона.
func TestRecordTemplateCreate(t *testing.T) {
	t.Parallel()

	// Act & Assert — не должно паниковать
	RecordTemplateCreate("email", "monitor_down", "user-1")
	RecordTemplateCreate("telegram", "monitor_up", "user-2")
}

// TestRecordTemplateRender проверяет запись метрики рендеринга шаблона.
func TestRecordTemplateRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		channel  string
		tmplType string
		success  bool
		duration float64
	}{
		{
			name:     "success render",
			channel:  "email",
			tmplType: "monitor_down",
			success:  true,
			duration: 0.001,
		},
		{
			name:     "failed render",
			channel:  "telegram",
			tmplType: "monitor_up",
			success:  false,
			duration: 0.002,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act & Assert — не должно паниковать
			RecordTemplateRender(tc.channel, tc.tmplType, tc.success, tc.duration)
		})
	}
}

// TestRecordValidationError проверяет запись метрики ошибки валидации.
func TestRecordValidationError(t *testing.T) {
	t.Parallel()

	// Act & Assert — не должно паниковать
	RecordValidationError("syntax_error")
	RecordValidationError("missing_variable")
}

// TestRecordTemplateDelete проверяет запись метрики удаления шаблона.
func TestRecordTemplateDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		isSystem bool
		channel  string
		tmplType string
	}{
		{
			name:     "delete user template",
			isSystem: false,
			channel:  "email",
			tmplType: "monitor_down",
		},
		{
			name:     "delete system template",
			isSystem: true,
			channel:  "telegram",
			tmplType: "monitor_up",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act & Assert — не должно паниковать
			RecordTemplateDelete(tc.isSystem, tc.channel, tc.tmplType)
		})
	}
}

// TestRecordTemplateUpdate проверяет запись метрики обновления шаблона.
func TestRecordTemplateUpdate(t *testing.T) {
	t.Parallel()

	// Act & Assert — не должно паниковать
	RecordTemplateUpdate("email", "monitor_down")
	RecordTemplateUpdate("slack", "certificate_expiry")
}

// TestRecordTemplateClone проверяет запись метрики клонирования шаблона.
func TestRecordTemplateClone(t *testing.T) {
	t.Parallel()

	// Act & Assert — не должно паниковать
	RecordTemplateClone()
}
