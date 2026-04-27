package model

type DashboardFilter struct {
	Statuses  []MonitorStatus
	Tags      []string
	Search    string
	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}

const (
	MaxPageSize     = 200
	MaxSearchLength = 200
	MaxFilterLength = 500
	DefaultPageSize = 50
	DefaultPage     = 1
)
