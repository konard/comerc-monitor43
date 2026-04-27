package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTracer(t *testing.T) {
	t.Parallel()

	tracer, err := NewTracer("test-service", "http://localhost:4318")
	require.NoError(t, err)
	require.NotNil(t, tracer)
	assert.NotNil(t, tracer.GetTracer())
}

func TestTracer_Shutdown(t *testing.T) {
	t.Parallel()

	tracer, err := NewTracer("test-service", "http://localhost:4318")
	require.NoError(t, err)

	err = tracer.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestTracer_StartSpan(t *testing.T) {
	t.Parallel()

	tracer, err := NewTracer("test-service", "http://localhost:4318")
	require.NoError(t, err)

	ctx, span := tracer.StartSpan(context.Background(), "test-operation")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()
}

func TestNewMetrics(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service")
	require.NoError(t, err)
	require.NotNil(t, m)
}

func TestMetrics_RecordCheck(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service")
	require.NoError(t, err)

	m.RecordCheck(context.Background(), "mon-1", true, 42.5)
	m.RecordCheck(context.Background(), "mon-2", false, 100.0)
}

func TestMetrics_RecordHeartbeat(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service")
	require.NoError(t, err)

	m.RecordHeartbeat(context.Background(), true)
	m.RecordHeartbeat(context.Background(), false)
}
