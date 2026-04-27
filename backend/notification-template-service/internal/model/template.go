package model

import (
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"
)

var (
	ErrInvalidTemplateID    = errors.New("invalid template id")
	ErrInvalidTemplateName  = errors.New("template name cannot be empty")
	ErrInvalidTemplateBody  = errors.New("template body cannot be empty")
	ErrInvalidChannel       = errors.New("invalid channel")
	ErrInvalidType          = errors.New("invalid type")
	ErrSystemTemplateDelete = errors.New("system templates cannot be deleted")
	ErrRenderFailed         = errors.New("template rendering failed")
)

// TemplateChannel представляет канал уведомления.
type TemplateChannel string

const (
	ChannelUnspecified TemplateChannel = ""
	ChannelEmail       TemplateChannel = "email"
	ChannelTelegram    TemplateChannel = "telegram"
	ChannelWebhook     TemplateChannel = "webhook"
	ChannelSlack       TemplateChannel = "slack"
	ChannelDiscord     TemplateChannel = "discord"
	ChannelSMS         TemplateChannel = "sms"
)

// TemplateType представляет тип уведомления.
type TemplateType string

const (
	TypeUnspecified        TemplateType = ""
	TypeMonitorUp          TemplateType = "monitor_up"
	TypeMonitorDown        TemplateType = "monitor_down"
	TypeMonitorDegraded    TemplateType = "monitor_degraded"
	TypeCertificateExpiry  TemplateType = "certificate_expiry"
	TypeFlappingDetected   TemplateType = "flapping_detected"
	TypeIncidentCreated    TemplateType = "incident_created"
	TypeIncidentResolved   TemplateType = "incident_resolved"
	TypeMaintenanceStarted TemplateType = "maintenance_started"
	TypeMaintenanceEnded   TemplateType = "maintenance_ended"
	TypeCustom             TemplateType = "custom"
)

// TemplateEngine представляет движок шаблонов.
type TemplateEngine string

const (
	EngineUnspecified TemplateEngine = ""
	EngineGoTemplate  TemplateEngine = "gotemplate"
	EngineJinja2      TemplateEngine = "jinja2"
	EngineHandlebars  TemplateEngine = "handlebars"
)

// TemplateFormat представляет формат вывода.
type TemplateFormat string

const (
	FormatText     TemplateFormat = "text"
	FormatHTML     TemplateFormat = "html"
	FormatMarkdown TemplateFormat = "markdown"
	FormatJSON     TemplateFormat = "json"
)

// Template представляет шаблон уведомления.
type Template struct {
	ID          string          `db:"id"`
	UserID      string          `db:"user_id"`
	Name        string          `db:"name"`
	Description string          `db:"description"`
	Channel     TemplateChannel `db:"channel"`
	Type        TemplateType    `db:"type"`
	Engine      TemplateEngine  `db:"engine"`

	// Template content
	Subject string         `db:"subject"`
	Body    string         `db:"body"`
	Format  TemplateFormat `db:"format"`

	// Metadata
	IsDefault bool      `db:"is_default"`
	IsSystem  bool      `db:"is_system"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	// Versioning
	Version  int     `db:"version"`
	ParentID *string `db:"parent_id"`
}

// Validate проверяет корректность шаблона.
func (t *Template) Validate() error {
	if t.ID != "" && !isValidUUID(t.ID) {
		return ErrInvalidTemplateID
	}

	if strings.TrimSpace(t.Name) == "" {
		return ErrInvalidTemplateName
	}

	if strings.TrimSpace(t.Body) == "" {
		return ErrInvalidTemplateBody
	}

	if !isValidChannel(t.Channel) {
		return ErrInvalidChannel
	}

	if !isValidType(t.Type) {
		return ErrInvalidType
	}

	if t.Engine == "" {
		t.Engine = EngineGoTemplate // Default
	}

	if t.Format == "" {
		switch t.Channel {
		case ChannelEmail:
			t.Format = FormatHTML
		case ChannelWebhook:
			t.Format = FormatJSON
		default:
			t.Format = FormatText
		}
	}

	return nil
}

// CanDelete проверяет можно ли удалить шаблон.
func (t *Template) CanDelete() bool {
	return !t.IsSystem
}

// IsDefaultFor проверяет является ли шаблон дефолтным для канала и типа.
func (t *Template) IsDefaultFor(channel TemplateChannel, tmplType TemplateType) bool {
	return t.IsDefault && t.Channel == channel && t.Type == tmplType
}

// Clone создает копию шаблона с новым именем.
func (t *Template) Clone(newName string) *Template {
	return &Template{
		UserID:      t.UserID,
		Name:        newName,
		Description: fmt.Sprintf("Clone of %s", t.Name),
		Channel:     t.Channel,
		Type:        t.Type,
		Engine:      t.Engine,
		Subject:     t.Subject,
		Body:        t.Body,
		Format:      t.Format,
		IsDefault:   false,
		IsSystem:    false,
		Version:     1,
		ParentID:    &t.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ValidateSyntax проверяет синтаксис шаблона.
func (t *Template) ValidateSyntax() []error {
	var errs []error

	// Validate subject template (if present)
	if t.Subject != "" {
		if _, err := template.New("subject").Parse(t.Subject); err != nil {
			errs = append(errs, fmt.Errorf("subject syntax error: %w", err))
		}
	}

	// Validate body template
	if _, err := template.New("body").Parse(t.Body); err != nil {
		errs = append(errs, fmt.Errorf("body syntax error: %w", err))
	}

	return errs
}

// BeforeCreate выполняет действия перед созданием.
func (t *Template) BeforeCreate() {
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	t.Version = 1
	if t.Engine == "" {
		t.Engine = EngineGoTemplate
	}
	if t.Format == "" {
		switch t.Channel {
		case ChannelEmail:
			t.Format = FormatHTML
		case ChannelWebhook:
			t.Format = FormatJSON
		default:
			t.Format = FormatText
		}
	}
}

// BeforeUpdate выполняет действия перед обновлением.
func (t *Template) BeforeUpdate() {
	t.UpdatedAt = time.Now()
	t.Version++
}

// isValidChannel проверяет валидность канала.
func isValidChannel(channel TemplateChannel) bool {
	switch channel {
	case ChannelEmail, ChannelTelegram, ChannelWebhook, ChannelSlack, ChannelDiscord, ChannelSMS:
		return true
	default:
		return false
	}
}

// isValidType проверяет валидность типа.
func isValidType(tmplType TemplateType) bool {
	switch tmplType {
	case TypeMonitorUp, TypeMonitorDown, TypeMonitorDegraded,
		TypeCertificateExpiry, TypeFlappingDetected,
		TypeIncidentCreated, TypeIncidentResolved,
		TypeMaintenanceStarted, TypeMaintenanceEnded, TypeCustom:
		return true
	default:
		return false
	}
}

// isValidUUID проверяет валидность UUID (базовая проверка).
func isValidUUID(id string) bool {
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		return false
	}
	// Базовая проверка длины каждой части
	lengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != lengths[i] {
			return false
		}
	}
	return true
}
