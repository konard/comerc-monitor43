package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/raul/monitor/backend/monitor-service/internal/model"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// BenchmarkMonitorService_CreateMonitor benchmarks creating a monitor.
func BenchmarkMonitorService_CreateMonitor(b *testing.B) {
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New().String()
	req := &dto.CreateMonitorRequest{
		UserID:          userID,
		Name:            "Benchmark Monitor",
		URL:             "https://example.com",
		IntervalSeconds: 60,
		Tier:            "Pro",
	}

	// Mock: проверяем, что монитор с таким именем не существует
	mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("ExistsByName", mock.Anything, mock.Anything, "Benchmark Monitor").Return(false, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.CreateMonitor(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMonitorService_GetMonitor benchmarks getting a monitor by ID.
func BenchmarkMonitorService_GetMonitor(b *testing.B) {
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	monitor := mustNewMonitor(b, userID, "Test", "https://example.com", 60)

	// Mock
	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.GetMonitor(context.Background(), monitor.ID.String(), userID.String()); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMonitorService_ListMonitors benchmarks listing monitors.
func BenchmarkMonitorService_ListMonitors(b *testing.B) {
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	monitor1 := mustNewMonitor(b, userID, "Monitor 1", "https://example1.com", 60)
	monitor2 := mustNewMonitor(b, userID, "Monitor 2", "https://example2.com", 60)

	monitors := []*domain.Monitor{monitor1, monitor2}

	req := &dto.ListMonitorsRequest{
		UserID: userID.String(),
		Limit:  10,
		Offset: 0,
	}

	// Mock
	mockRepo.On("ListByUserID", mock.Anything, userID, 10, 0).Return(monitors, nil)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(2, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.ListMonitors(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMonitorService_ListMonitors_Large benchmarks listing many monitors.
func BenchmarkMonitorService_ListMonitors_Large(b *testing.B) {
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	monitors := make([]*domain.Monitor, 100)
	for i := 0; i < 100; i++ {
		monitors[i] = mustNewMonitor(b, userID, "Monitor", "https://example.com", 60)
	}

	req := &dto.ListMonitorsRequest{
		UserID: userID.String(),
		Limit:  100,
		Offset: 0,
	}

	// Mock
	mockRepo.On("ListByUserID", mock.Anything, userID, 100, 0).Return(monitors, nil)
	mockRepo.On("CountByUserID", mock.Anything, userID).Return(100, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.ListMonitors(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMonitorService_UpdateMonitor benchmarks updating a monitor.
func BenchmarkMonitorService_UpdateMonitor(b *testing.B) {
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	monitor := mustNewMonitor(b, userID, "Test", "https://example.com", 60)

	updatedName := "Updated Name"
	updatedURL := "https://updated.com"

	req := &dto.UpdateMonitorRequest{
		ID:     monitor.ID.String(),
		UserID: userID.String(),
		Name:   &updatedName,
		URL:    &updatedURL,
	}

	// Mock
	mockRepo.On("GetByID", mock.Anything, mock.Anything).Return(monitor, nil)
	mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.UpdateMonitor(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMonitorService_DeleteMonitor benchmarks deleting a monitor.
func BenchmarkMonitorService_DeleteMonitor(b *testing.B) {
	mockRepo := new(MockMonitorRepository)
	mockAudit := new(MockAuditRepository)
	service := NewMonitorService(mockRepo, mockAudit)

	userID := uuid.New()
	monitor := mustNewMonitor(b, userID, "Test", "https://example.com", 60)

	// Mock
	mockRepo.On("GetByID", mock.Anything, monitor.ID).Return(monitor, nil)
	mockAudit.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Delete", mock.Anything, monitor.ID).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := service.DeleteMonitor(context.Background(), monitor.ID.String(), userID.String()); err != nil {
			b.Fatal(err)
		}
	}
}
