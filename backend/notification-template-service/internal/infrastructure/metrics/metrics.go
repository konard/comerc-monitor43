package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// TemplatesCreated общее количество созданных шаблонов
	TemplatesCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_template_templates_created_total",
			Help: "Total number of templates created",
		},
		[]string{"channel", "type", "user_id"},
	)

	// TemplatesRendered общее количество рендерингов шаблонов
	TemplatesRendered = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_template_templates_rendered_total",
			Help: "Total number of template renderings",
		},
		[]string{"channel", "type", "status"}, // status: success/error
	)

	// TemplateRenderDuration гистограмма времени рендеринга
	TemplateRenderDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notification_template_render_duration_seconds",
			Help:    "Template rendering duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"channel", "type"},
	)

	// TemplateValidationErrors общее количество ошибок валидации
	TemplateValidationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_template_validation_errors_total",
			Help: "Total number of template validation errors",
		},
		[]string{"error_type"},
	)

	// TemplatesDeleted общее количество удалённых шаблонов
	TemplatesDeleted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_template_templates_deleted_total",
			Help: "Total number of templates deleted",
		},
		[]string{"is_system"},
	)

	// TemplatesUpdated общее количество обновлений шаблонов
	TemplatesUpdated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_template_templates_updated_total",
			Help: "Total number of template updates",
		},
		[]string{"channel", "type"},
	)

	// TemplatesCloned общее количество клонирований шаблонов
	TemplatesCloned = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "notification_template_templates_cloned_total",
			Help: "Total number of template clones",
		},
	)

	// ActiveTemplates общее количество активных шаблонов
	ActiveTemplates = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "notification_template_active_templates",
			Help: "Number of active templates by channel and type",
		},
		[]string{"channel", "type"},
	)
)

// RecordTemplateCreate записывает метрику создания шаблона.
func RecordTemplateCreate(channel, tmplType, userID string) {
	TemplatesCreated.WithLabelValues(channel, tmplType, userID).Inc()
	ActiveTemplates.WithLabelValues(channel, tmplType).Inc()
}

// RecordTemplateRender записывает метрику рендеринга шаблона.
func RecordTemplateRender(channel, tmplType string, success bool, duration float64) {
	status := "success"
	if !success {
		status = "error"
	}

	TemplatesRendered.WithLabelValues(channel, tmplType, status).Inc()
	TemplateRenderDuration.WithLabelValues(channel, tmplType).Observe(duration)
}

// RecordValidationError записывает метрику ошибки валидации.
func RecordValidationError(errorType string) {
	TemplateValidationErrors.WithLabelValues(errorType).Inc()
}

// RecordTemplateDelete записывает метрику удаления шаблона.
func RecordTemplateDelete(isSystem bool, channel, tmplType string) {
	systemFlag := "false"
	if isSystem {
		systemFlag = "true"
	}

	TemplatesDeleted.WithLabelValues(systemFlag).Inc()
	ActiveTemplates.WithLabelValues(channel, tmplType).Dec()
}

// RecordTemplateUpdate записывает метрику обновления шаблона.
func RecordTemplateUpdate(channel, tmplType string) {
	TemplatesUpdated.WithLabelValues(channel, tmplType).Inc()
}

// RecordTemplateClone записывает метрику клонирования шаблона.
func RecordTemplateClone() {
	TemplatesCloned.Inc()
	ActiveTemplates.WithLabelValues("", "").Inc() // Клон может быть любого типа
}
