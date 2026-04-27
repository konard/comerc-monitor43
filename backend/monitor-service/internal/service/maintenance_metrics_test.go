package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewMaintenanceMetrics тестирует создание нового коллектора метрик.
func TestNewMaintenanceMetrics(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()

	assert.NotNil(t, m)
	assert.Equal(t, int64(0), m.createdTotal)
	assert.Equal(t, int64(0), m.updatedTotal)
	assert.Equal(t, int64(0), m.deletedTotal)
	assert.Equal(t, int64(0), m.cancelledTotal)
}

// TestMaintenanceMetrics_RecordCreation_Success тестирует запись метрики создания.
func TestMaintenanceMetrics_RecordCreation_Success(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()
	ctx := context.Background()

	m.RecordCreation(ctx, 100*time.Millisecond, true)

	assert.Equal(t, int64(1), m.createdTotal)
	assert.Equal(t, 100*time.Millisecond, m.creationDuration)
}

// TestMaintenanceMetrics_RecordCreation_Failure тестирует запись метрики при неуспешном создании.
func TestMaintenanceMetrics_RecordCreation_Failure(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()
	ctx := context.Background()

	m.RecordCreation(ctx, 50*time.Millisecond, false)

	assert.Equal(t, int64(1), m.createdTotal)
	// При failure duration не прибавляется
	assert.Equal(t, time.Duration(0), m.creationDuration)
}

// TestMaintenanceMetrics_RecordCreation_Multiple тестирует накопление метрик создания.
func TestMaintenanceMetrics_RecordCreation_Multiple(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()
	ctx := context.Background()

	m.RecordCreation(ctx, 100*time.Millisecond, true)
	m.RecordCreation(ctx, 200*time.Millisecond, true)
	m.RecordCreation(ctx, 50*time.Millisecond, false)

	assert.Equal(t, int64(3), m.createdTotal)
	assert.Equal(t, 300*time.Millisecond, m.creationDuration)
}

// TestMaintenanceMetrics_RecordUpdate_Success тестирует запись метрики обновления.
func TestMaintenanceMetrics_RecordUpdate_Success(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()
	ctx := context.Background()

	m.RecordUpdate(ctx, 75*time.Millisecond, true)

	assert.Equal(t, int64(1), m.updatedTotal)
	assert.Equal(t, 75*time.Millisecond, m.updateDuration)
}

// TestMaintenanceMetrics_RecordUpdate_Failure тестирует запись метрики при неуспешном обновлении.
func TestMaintenanceMetrics_RecordUpdate_Failure(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()
	ctx := context.Background()

	m.RecordUpdate(ctx, 75*time.Millisecond, false)

	assert.Equal(t, int64(1), m.updatedTotal)
	assert.Equal(t, time.Duration(0), m.updateDuration)
}

// TestMaintenanceMetrics_RecordDeletion тестирует запись метрики удаления.
func TestMaintenanceMetrics_RecordDeletion(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()
	ctx := context.Background()

	m.RecordDeletion(ctx)
	m.RecordDeletion(ctx)

	assert.Equal(t, int64(2), m.deletedTotal)
}

// TestMaintenanceMetrics_RecordCancellation тестирует запись метрики отмены.
func TestMaintenanceMetrics_RecordCancellation(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()
	ctx := context.Background()

	m.RecordCancellation(ctx)
	m.RecordCancellation(ctx)
	m.RecordCancellation(ctx)

	assert.Equal(t, int64(3), m.cancelledTotal)
}

// TestMaintenanceMetrics_SetActiveWindows тестирует установку количества активных окон.
func TestMaintenanceMetrics_SetActiveWindows(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()

	m.SetActiveWindows(42)

	assert.Equal(t, int64(42), m.activeCurrent)
}

// TestMaintenanceMetrics_SetActiveWindows_Override тестирует перезапись количества активных окон.
func TestMaintenanceMetrics_SetActiveWindows_Override(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()

	m.SetActiveWindows(10)
	m.SetActiveWindows(5)

	assert.Equal(t, int64(5), m.activeCurrent)
}

// TestMaintenanceMetrics_GetMetrics тестирует получение всех метрик.
func TestMaintenanceMetrics_GetMetrics(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()
	ctx := context.Background()

	m.RecordCreation(ctx, 100*time.Millisecond, true)
	m.RecordUpdate(ctx, 50*time.Millisecond, true)
	m.RecordDeletion(ctx)
	m.RecordCancellation(ctx)
	m.SetActiveWindows(3)

	metrics := m.GetMetrics()

	assert.Equal(t, int64(1), metrics["created_total"])
	assert.Equal(t, int64(1), metrics["updated_total"])
	assert.Equal(t, int64(1), metrics["deleted_total"])
	assert.Equal(t, int64(1), metrics["cancelled_total"])
	assert.Equal(t, int64(3), metrics["active_current"])
	assert.Equal(t, int64(100), metrics["creation_duration_ms"])
	assert.Equal(t, int64(50), metrics["update_duration_ms"])
}

// TestMaintenanceMetrics_GetMetrics_Empty тестирует получение пустых метрик.
func TestMaintenanceMetrics_GetMetrics_Empty(t *testing.T) {
	t.Parallel()

	m := NewMaintenanceMetrics()

	metrics := m.GetMetrics()

	assert.Equal(t, int64(0), metrics["created_total"])
	assert.Equal(t, int64(0), metrics["updated_total"])
	assert.Equal(t, int64(0), metrics["deleted_total"])
	assert.Equal(t, int64(0), metrics["cancelled_total"])
	assert.Equal(t, int64(0), metrics["active_current"])
}
