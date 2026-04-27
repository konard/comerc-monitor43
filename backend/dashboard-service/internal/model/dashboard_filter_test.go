package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDashboardFilter_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 200, MaxPageSize)
	assert.Equal(t, 200, MaxSearchLength)
	assert.Equal(t, 50, DefaultPageSize)
	assert.Equal(t, 1, DefaultPage)
}

func TestDashboardFilter_MaxFilterLength(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 500, MaxFilterLength)
}

func TestDashboardFilter_Fields(t *testing.T) {
	t.Parallel()

	filter := DashboardFilter{
		Statuses:  []MonitorStatus{MonitorStatusUP, MonitorStatusDOWN},
		Tags:      []string{"prod", "web"},
		Search:    "example",
		SortBy:    "name",
		SortOrder: "asc",
		Page:      2,
		PageSize:  25,
	}

	assert.Equal(t, []MonitorStatus{MonitorStatusUP, MonitorStatusDOWN}, filter.Statuses)
	assert.Equal(t, []string{"prod", "web"}, filter.Tags)
	assert.Equal(t, "example", filter.Search)
	assert.Equal(t, "name", filter.SortBy)
	assert.Equal(t, "asc", filter.SortOrder)
	assert.Equal(t, 2, filter.Page)
	assert.Equal(t, 25, filter.PageSize)
}
