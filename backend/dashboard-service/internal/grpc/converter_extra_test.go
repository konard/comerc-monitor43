package grpc

import (
	"testing"

	dashboardproto "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"

	domain "github.com/raul/monitor/backend/dashboard-service/internal/model"
)

func Test_checkStatusToProto_unknown_returns_unspecified(t *testing.T) {
	t.Parallel()

	result := checkStatusToProto(domain.CheckStatus("UNKNOWN"))

	assert.Equal(t, dashboardproto.HealthStatus_HEALTH_STATUS_UNSPECIFIED, result)
}

func Test_healthStatusToProto_unknown_returns_unspecified(t *testing.T) {
	t.Parallel()

	result := healthStatusToProto(domain.MonitorStatus("UNKNOWN"))

	assert.Equal(t, dashboardproto.HealthStatus_HEALTH_STATUS_UNSPECIFIED, result)
}
