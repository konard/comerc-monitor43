package dto

// AlertRequest представляет DTO для создания алерта
type AlertRequest struct {
	MonitorID string         `json:"monitor_id"`
	RuleID    string         `json:"rule_id"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// AlertResponse представляет DTO для ответа с алертом
type AlertResponse struct {
	ID        string         `json:"id"`
	MonitorID string         `json:"monitor_id"`
	RuleID    string         `json:"rule_id"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at"`
}
