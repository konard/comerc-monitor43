package dto

import "time"

// CreateMaintenanceWindowRequest - запрос на создание окна обслуживания.
type CreateMaintenanceWindowRequest struct {
	UserID          string    `json:"user_id"`
	Name            string    `json:"name"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Recurrence      string    `json:"recurrence"` // ONCE, DAILY, WEEKLY, MONTHLY
	IsGlobal        bool      `json:"is_global"`
	PauseMonitoring bool      `json:"pause_monitoring"`
	SuppressAlerts  bool      `json:"suppress_alerts"`
	SafeMode        bool      `json:"safe_mode"`
	MonitorIDs      []string  `json:"monitor_ids"`
	UserRole        string    `json:"user_role"` // USER, ADMIN
	UserTier        string    `json:"user_tier"` // Free, Pro, Enterprise
}

// UpdateMaintenanceWindowRequest - запрос на обновление окна обслуживания.
type UpdateMaintenanceWindowRequest struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Version   int       `json:"version"`
	UserID    string    `json:"user_id"`
}

// ListMaintenanceWindowsRequest - запрос на список окон обслуживания.
type ListMaintenanceWindowsRequest struct {
	UserID    string     `json:"user_id"`
	Status    string     `json:"status"` // Optional filter
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
}

// GetMaintenanceWindowHistoryRequest - запрос на историю окон.
type GetMaintenanceWindowHistoryRequest struct {
	UserID     string    `json:"user_id"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	MonitorIDs []string  `json:"monitor_ids"` // Optional filter
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
}

// CancelMaintenanceWindowRequest - запрос на отмену окна.
type CancelMaintenanceWindowRequest struct {
	ID                 string `json:"id"`
	CancellationReason string `json:"cancellation_reason"`
	UserID             string `json:"user_id"`
}

// MaintenanceWindowResponse - ответ с окном обслуживания.
type MaintenanceWindowResponse struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Name            string     `json:"name"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         time.Time  `json:"end_time"`
	Status          string     `json:"status"`
	Recurrence      string     `json:"recurrence"`
	IsGlobal        bool       `json:"is_global"`
	PauseMonitoring bool       `json:"pause_monitoring"`
	SuppressAlerts  bool       `json:"suppress_alerts"`
	SafeMode        bool       `json:"safe_mode"`
	MonitorIDs      []string   `json:"monitor_ids"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ActivatedAt     *time.Time `json:"activated_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	Version         int        `json:"version"`
}

// ListMaintenanceWindowsResponse - ответ со списком окон.
type ListMaintenanceWindowsResponse struct {
	Windows  []*MaintenanceWindowResponse `json:"windows"`
	Total    int                          `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}
