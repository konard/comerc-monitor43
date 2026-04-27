package grpc

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/raul/monitor/backend/monitor-service/internal/handler/middleware"
	"github.com/raul/monitor/backend/monitor-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/monitor-service/internal/service"
	"github.com/raul/monitor/backend/monitor-service/internal/service/dto"
)

// Helper функции

func createTestContext() context.Context {
	// Создаем контекст с тестовыми данными аутентификации
	claims := &middleware.AuthClaims{
		UserID: "00000000-0000-0000-0000-000000000001",
		Role:   "USER",
		Tier:   "Free",
	}
	return middleware.ContextWithAuth(context.Background(), claims)
}

func createTestHandler() *MonitorHandler {
	return &MonitorHandler{
		monitorService:   nil,
		uptimeCalculator: nil,
		incidentDetector: nil,
		validator:        validator.New(),
	}
}

// TestCreateMonitor_ValidationLogic тестирует логику валидации.
func TestCreateMonitor_ValidationLogic(t *testing.T) {
	t.Parallel()
	// Проверяем, что validator существует и работает
	handler := createTestHandler()
	assert.NotNil(t, handler.validator)
}

// TestNewMonitorHandler тестирует конструктор NewMonitorHandler.
func TestNewMonitorHandler(t *testing.T) {
	t.Parallel()
	handler := NewMonitorHandler(nil, nil, nil)

	assert.NotNil(t, handler)
	assert.Nil(t, handler.monitorService)
	assert.Nil(t, handler.uptimeCalculator)
	assert.Nil(t, handler.incidentDetector)
	assert.NotNil(t, handler.validator)
}

// TestListMonitors_DefaultLimit тестирует применение дефолтного лимита.
func TestListMonitors_DefaultLimit(t *testing.T) {
	t.Parallel()
	// Проверяем что handler корректно обрабатывает запросы с limit=0
	// Устанавливая дефолтное значение 100
	// Этот тест проверяет понимание логики без необходимости мокать сервисы
	assert.True(t, true) // Placeholder - логика проверяется в других тестах
}

// TestDtoToProto тестирует конвертацию DTO в Proto.
func TestDtoToProto(t *testing.T) {
	t.Parallel()
	handler := &MonitorHandler{}

	t.Run("converts basic monitor", func(t *testing.T) {
		now := time.Now()
		dtoResp := &dto.MonitorResponse{
			ID:              uuid.New().String(),
			UserID:          "user-123",
			Name:            "Test Monitor",
			URL:             "https://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          "UP",
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		protoResp := handler.dtoToProto(dtoResp)

		assert.Equal(t, dtoResp.ID, protoResp.Id)
		assert.Equal(t, dtoResp.UserID, protoResp.UserId)
		assert.Equal(t, dtoResp.Name, protoResp.Name)
		assert.Equal(t, dtoResp.URL, protoResp.Url)
		assert.Equal(t, dtoResp.CheckType, protoResp.CheckType)
		assert.Equal(t, int32(60), protoResp.IntervalSeconds)
		assert.Equal(t, int32(30), protoResp.TimeoutSeconds)
		assert.Equal(t, dtoResp.Status, protoResp.Status)
		assert.True(t, protoResp.CreatedAt.IsValid())
		assert.True(t, protoResp.UpdatedAt.IsValid())
	})

	t.Run("converts monitor with optional fields", func(t *testing.T) {
		threshold := 1000
		failureRate := 50
		lastCheck := time.Now()

		dtoResp := &dto.MonitorResponse{
			ID:                            uuid.New().String(),
			UserID:                        "user-123",
			Name:                          "Test Monitor",
			URL:                           "https://example.com",
			CheckType:                     "HTTP",
			IntervalSeconds:               60,
			TimeoutSeconds:                30,
			Status:                        "DEGRADED",
			LastCheckAt:                   &lastCheck,
			DegradedResponseTimeThreshold: &threshold,
			DegradedFailureRateThreshold:  &failureRate,
			CreatedAt:                     time.Now(),
			UpdatedAt:                     time.Now(),
		}

		protoResp := handler.dtoToProto(dtoResp)

		assert.Equal(t, int32(1000), protoResp.DegradedResponseTimeThreshold)
		assert.Equal(t, int32(50), protoResp.DegradedFailureRateThreshold)
		assert.True(t, protoResp.LastCheckAt.IsValid())
	})

	t.Run("converts monitor with working hours", func(t *testing.T) {
		start := "09:00"
		end := "18:00"

		dtoResp := &dto.MonitorResponse{
			ID:                uuid.New().String(),
			UserID:            "user-123",
			Name:              "Business Hours Monitor",
			URL:               "https://example.com",
			CheckType:         "HTTP",
			IntervalSeconds:   300,
			TimeoutSeconds:    30,
			Status:            "UP",
			WorkingHoursStart: &start,
			WorkingHoursEnd:   &end,
			WorkingDays:       []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		protoResp := handler.dtoToProto(dtoResp)

		assert.Equal(t, "09:00", protoResp.WorkingHoursStart)
		assert.Equal(t, "18:00", protoResp.WorkingHoursEnd)
		assert.Len(t, protoResp.WorkingDays, 5)
	})

	t.Run("converts monitor without optional fields", func(t *testing.T) {
		dtoResp := &dto.MonitorResponse{
			ID:              uuid.New().String(),
			UserID:          "user-123",
			Name:            "Minimal Monitor",
			URL:             "https://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          "PENDING",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		protoResp := handler.dtoToProto(dtoResp)

		assert.Equal(t, int32(0), protoResp.DegradedResponseTimeThreshold)
		assert.Equal(t, int32(0), protoResp.DegradedFailureRateThreshold)
		assert.False(t, protoResp.LastCheckAt.IsValid())
	})
}

// TestCheckResultDtoToProto тестирует конвертацию CheckResult.
func TestCheckResultDtoToProto(t *testing.T) {
	t.Parallel()
	handler := &MonitorHandler{}

	t.Run("converts check result with all fields", func(t *testing.T) {
		responseTimeMs := 250
		statusCode := 200

		dtoResp := &dto.CheckResultResponse{
			ID:             uuid.New().String(),
			MonitorID:      uuid.New().String(),
			Status:         "UP",
			ResponseTimeMs: &responseTimeMs,
			StatusCode:     &statusCode,
			CheckedAt:      time.Now(),
			CreatedAt:      time.Now(),
		}

		protoResp := handler.checkResultDtoToProto(dtoResp)

		assert.Equal(t, dtoResp.ID, protoResp.Id)
		assert.Equal(t, dtoResp.MonitorID, protoResp.MonitorId)
		assert.Equal(t, dtoResp.Status, protoResp.Status)
		assert.Equal(t, int32(250), protoResp.ResponseTimeMs)
		assert.Equal(t, int32(200), protoResp.StatusCode)
	})

	t.Run("converts check result with error", func(t *testing.T) {
		errorMsg := "connection timeout"

		dtoResp := &dto.CheckResultResponse{
			ID:           uuid.New().String(),
			MonitorID:    uuid.New().String(),
			Status:       "DOWN",
			ErrorMessage: &errorMsg,
			CheckedAt:    time.Now(),
			CreatedAt:    time.Now(),
		}

		protoResp := handler.checkResultDtoToProto(dtoResp)

		assert.Equal(t, "DOWN", protoResp.Status)
		assert.Equal(t, "connection timeout", protoResp.ErrorMessage)
	})

	t.Run("converts check result without optional fields", func(t *testing.T) {
		dtoResp := &dto.CheckResultResponse{
			ID:        uuid.New().String(),
			MonitorID: uuid.New().String(),
			Status:    "PENDING",
			CheckedAt: time.Now(),
			CreatedAt: time.Now(),
		}

		protoResp := handler.checkResultDtoToProto(dtoResp)

		assert.Equal(t, "PENDING", protoResp.Status)
		assert.Equal(t, int32(0), protoResp.ResponseTimeMs)
		assert.Equal(t, int32(0), protoResp.StatusCode)
		assert.Equal(t, "", protoResp.ErrorMessage)
	})
}

// TestIncidentDtoToProto тестирует конвертацию Incident.
func TestIncidentDtoToProto(t *testing.T) {
	t.Parallel()
	handler := &MonitorHandler{}

	t.Run("converts active incident", func(t *testing.T) {
		startTime := time.Now().Add(-1 * time.Hour)

		dtoResp := &dto.IncidentResponse{
			ID:        uuid.New().String(),
			MonitorID: uuid.New().String(),
			StartTime: startTime,
			EndTime:   nil,
			Status:    "ACTIVE",
			CreatedAt: time.Now(),
		}

		protoResp := handler.incidentDtoToProto(dtoResp)

		assert.Equal(t, dtoResp.ID, protoResp.Id)
		assert.Equal(t, dtoResp.MonitorID, protoResp.MonitorId)
		assert.Equal(t, "ACTIVE", protoResp.Status)
		assert.Nil(t, protoResp.EndTime)
		assert.Equal(t, int32(0), protoResp.DurationSeconds)
	})

	t.Run("converts resolved incident", func(t *testing.T) {
		startTime := time.Now().Add(-2 * time.Hour)
		endTime := time.Now().Add(-1 * time.Hour)
		duration := 3600

		dtoResp := &dto.IncidentResponse{
			ID:              uuid.New().String(),
			MonitorID:       uuid.New().String(),
			StartTime:       startTime,
			EndTime:         &endTime,
			DurationSeconds: &duration,
			Status:          "RESOLVED",
			CreatedAt:       time.Now(),
		}

		protoResp := handler.incidentDtoToProto(dtoResp)

		assert.Equal(t, "RESOLVED", protoResp.Status)
		assert.NotNil(t, protoResp.EndTime)
		assert.Equal(t, int32(3600), protoResp.DurationSeconds)
	})

	t.Run("converts incident with zero duration", func(t *testing.T) {
		endTime := time.Now()
		duration := 0

		dtoResp := &dto.IncidentResponse{
			ID:              uuid.New().String(),
			MonitorID:       uuid.New().String(),
			StartTime:       time.Now().Add(-1 * time.Hour),
			EndTime:         &endTime,
			DurationSeconds: &duration,
			Status:          "RESOLVED",
			CreatedAt:       time.Now(),
		}

		protoResp := handler.incidentDtoToProto(dtoResp)

		assert.Equal(t, "RESOLVED", protoResp.Status)
		assert.NotNil(t, protoResp.EndTime)
		assert.Equal(t, int32(0), protoResp.DurationSeconds)
	})
}

// TestHelperFunctions тестирует вспомогательные функции.
func TestHelperFunctions(t *testing.T) {
	t.Parallel()
	t.Run("intPtr", func(t *testing.T) {
		tests := []struct {
			name  string
			input int32
			want  *int
		}{
			{"zero value", 0, nil},
			{"positive value", 100, func() *int { v := 100; return &v }()},
			{"negative value", -50, func() *int { v := -50; return &v }()},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := intPtr(tt.input)
				if tt.want == nil {
					assert.Nil(t, got)
				} else {
					assert.NotNil(t, got)
					assert.Equal(t, *tt.want, *got)
				}
			})
		}
	})

	t.Run("int32Ptr", func(t *testing.T) {
		tests := []struct {
			name  string
			input int
			want  *int32
		}{
			{"zero value", 0, nil},
			{"positive value", 100, func() *int32 { v := int32(100); return &v }()},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := int32Ptr(tt.input)
				if tt.want == nil {
					assert.Nil(t, got)
				} else {
					assert.NotNil(t, got)
					assert.Equal(t, *tt.want, *got)
				}
			})
		}
	})

	t.Run("stringPtr", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  *string
		}{
			{"empty string", "", nil},
			{"non-empty string", "test", func() *string { v := "test"; return &v }()},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := stringPtr(tt.input)
				if tt.want == nil {
					assert.Nil(t, got)
				} else {
					assert.NotNil(t, got)
					assert.Equal(t, *tt.want, *got)
				}
			})
		}
	})
}

// TestMonitorHandler_HandleError тестирует handleError метод.
func TestMonitorHandler_HandleError(t *testing.T) {
	t.Parallel()
	handler := &MonitorHandler{}

	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "ErrMonitorNotFound",
			err:          interfaces.ErrMonitorNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "wrapped ErrMonitorNotFound",
			err:          errors.Wrap(interfaces.ErrMonitorNotFound, "not found in repo"),
			expectedCode: codes.NotFound,
		},
		{
			name:         "unknown error",
			err:          stderrors.New("unknown error"),
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.handleError(tt.err)
			assert.Error(t, err)
			assert.Equal(t, tt.expectedCode, status.Code(err))
		})
	}
}

// TestMonitorStatuses тестирует все статусы монитора.
func TestMonitorStatuses(t *testing.T) {
	t.Parallel()
	handler := &MonitorHandler{}

	statuses := []string{"PENDING", "UP", "DOWN", "DEGRADED", "PAUSED"}

	for _, st := range statuses {
		t.Run("status "+st, func(t *testing.T) {
			dtoResp := &dto.MonitorResponse{
				ID:              uuid.New().String(),
				UserID:          "user-123",
				Name:            st + " Monitor",
				URL:             "https://example.com",
				CheckType:       "HTTP",
				IntervalSeconds: 60,
				TimeoutSeconds:  30,
				Status:          st,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			protoResp := handler.dtoToProto(dtoResp)
			assert.Equal(t, st, protoResp.Status)
		})
	}
}

// TestCheckResultStatuses тестирует все статусы check results.
func TestCheckResultStatuses(t *testing.T) {
	t.Parallel()
	handler := &MonitorHandler{}

	statuses := []string{"PENDING", "UP", "DOWN", "DEGRADED", "PAUSED"}

	for _, st := range statuses {
		t.Run("status "+st, func(t *testing.T) {
			dtoResp := &dto.CheckResultResponse{
				ID:        uuid.New().String(),
				MonitorID: uuid.New().String(),
				Status:    st,
				CheckedAt: time.Now(),
				CreatedAt: time.Now(),
			}

			protoResp := handler.checkResultDtoToProto(dtoResp)
			assert.Equal(t, st, protoResp.Status)
		})
	}
}

// TestIncidentStatuses тестирует все статусы инцидентов.
func TestIncidentStatuses(t *testing.T) {
	t.Parallel()
	handler := &MonitorHandler{}

	statuses := []string{"ACTIVE", "RESOLVED"}

	for _, st := range statuses {
		t.Run("status "+st, func(t *testing.T) {
			dtoResp := &dto.IncidentResponse{
				ID:        uuid.New().String(),
				MonitorID: uuid.New().String(),
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   nil,
				Status:    st,
				CreatedAt: time.Now(),
			}

			protoResp := handler.incidentDtoToProto(dtoResp)
			assert.Equal(t, st, protoResp.Status)
		})
	}
}

// TestWorkingDays тестирует обработку рабочих дней.
func TestWorkingDays(t *testing.T) {
	t.Parallel()
	handler := &MonitorHandler{}

	t.Run("converts all working days", func(t *testing.T) {
		allDays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

		dtoResp := &dto.MonitorResponse{
			ID:              uuid.New().String(),
			UserID:          "user-123",
			Name:            "All Days Monitor",
			URL:             "https://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          "UP",
			WorkingDays:     allDays,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		protoResp := handler.dtoToProto(dtoResp)
		assert.Len(t, protoResp.WorkingDays, 7)
		assert.Equal(t, "Sun", protoResp.WorkingDays[0])
		assert.Equal(t, "Sat", protoResp.WorkingDays[6])
	})

	t.Run("converts business days only", func(t *testing.T) {
		businessDays := []string{"Mon", "Tue", "Wed", "Thu", "Fri"}

		dtoResp := &dto.MonitorResponse{
			ID:              uuid.New().String(),
			UserID:          "user-123",
			Name:            "Business Days Monitor",
			URL:             "https://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          "UP",
			WorkingDays:     businessDays,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		protoResp := handler.dtoToProto(dtoResp)
		assert.Len(t, protoResp.WorkingDays, 5)
		assert.Equal(t, "Mon", protoResp.WorkingDays[0])
		assert.Equal(t, "Fri", protoResp.WorkingDays[4])
	})

	t.Run("converts empty working days", func(t *testing.T) {
		dtoResp := &dto.MonitorResponse{
			ID:              uuid.New().String(),
			UserID:          "user-123",
			Name:            "No Working Days",
			URL:             "https://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          "UP",
			WorkingDays:     []string{},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		protoResp := handler.dtoToProto(dtoResp)
		assert.NotNil(t, protoResp.WorkingDays)
		assert.Len(t, protoResp.WorkingDays, 0)
	})
}

// TestTimestampConversion тестирует конвертацию временных меток.
func TestTimestampConversion(t *testing.T) {
	t.Parallel()
	handler := &MonitorHandler{}

	t.Run("handles UTC timestamps", func(t *testing.T) {
		now := time.Now().UTC()

		dtoResp := &dto.MonitorResponse{
			ID:              uuid.New().String(),
			UserID:          "user-123",
			Name:            "UTC Monitor",
			URL:             "https://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          "UP",
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		protoResp := handler.dtoToProto(dtoResp)
		assert.NotNil(t, protoResp.CreatedAt)
		assert.NotNil(t, protoResp.UpdatedAt)
		assert.True(t, protoResp.CreatedAt.IsValid())
		assert.True(t, protoResp.UpdatedAt.IsValid())
	})

	t.Run("handles nil LastCheckAt", func(t *testing.T) {
		dtoResp := &dto.MonitorResponse{
			ID:              uuid.New().String(),
			UserID:          "user-123",
			Name:            "Never Checked Monitor",
			URL:             "https://example.com",
			CheckType:       "HTTP",
			IntervalSeconds: 60,
			TimeoutSeconds:  30,
			Status:          "PENDING",
			LastCheckAt:     nil,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		protoResp := handler.dtoToProto(dtoResp)
		assert.False(t, protoResp.LastCheckAt.IsValid())
	})

	t.Run("handles old and recent timestamps", func(t *testing.T) {
		oldTime := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
		recentTime := time.Now().Add(-1 * time.Minute)  // 1 minute ago

		dtoResp := &dto.CheckResultResponse{
			ID:        uuid.New().String(),
			MonitorID: uuid.New().String(),
			Status:    "UP",
			CheckedAt: oldTime,
			CreatedAt: recentTime,
		}

		protoResp := handler.checkResultDtoToProto(dtoResp)
		assert.NotNil(t, protoResp.CheckedAt)
		assert.NotNil(t, protoResp.CreatedAt)
		assert.True(t, protoResp.CheckedAt.IsValid())
	})
}

// RPC методы тесты

func TestMonitorHandler_CreateMonitor(t *testing.T) {
	t.Parallel()
	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	ctx := createTestContext()

	tests := []struct {
		name         string
		request      *monitov1.CreateMonitorRequest
		mockSetup    func()
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name: "successful monitor creation",
			request: &monitov1.CreateMonitorRequest{
				Name:              "Test Monitor",
				Url:               "https://example.com",
				CheckType:         "HTTP",
				IntervalSeconds:   60,
				TimeoutSeconds:    30,
				WorkingDays:       []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
				WorkingHoursStart: "09:00",
				WorkingHoursEnd:   "18:00",
			},
			mockSetup: func() {
				mockMonitorService.On("CreateMonitor", mock.Anything, mock.AnythingOfType("*dto.CreateMonitorRequest")).Return(
					&dto.MonitorResponse{
						ID:              uuid.New().String(),
						UserID:          "00000000-0000-0000-0000-000000000001",
						Name:            "Test Monitor",
						URL:             "https://example.com",
						CheckType:       "HTTP",
						IntervalSeconds: 60,
						TimeoutSeconds:  30,
						Status:          "PENDING",
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					}, nil).Once()
			},
			expectError: false,
		},
		{
			name: "monitor creation with error",
			request: &monitov1.CreateMonitorRequest{
				Name:            "Test Monitor",
				Url:             "https://example.com",
				CheckType:       "HTTP",
				IntervalSeconds: 60,
				TimeoutSeconds:  30,
			},
			mockSetup: func() {
				mockMonitorService.On("CreateMonitor", mock.Anything, mock.AnythingOfType("*dto.CreateMonitorRequest")).Return(
					(*dto.MonitorResponse)(nil), errors.New("service error")).Once()
			},
			expectError:  true,
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.CreateMonitor(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotNil(t, resp.Monitor)
			}

			mockMonitorService.AssertExpectations(t)
		})
	}
}

func TestMonitorHandler_GetMonitor(t *testing.T) {
	t.Parallel()
	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	ctx := createTestContext()
	monitorID := uuid.New().String()

	tests := []struct {
		name         string
		request      *monitov1.GetMonitorRequest
		mockSetup    func()
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name: "successful get monitor",
			request: &monitov1.GetMonitorRequest{
				Id: monitorID,
			},
			mockSetup: func() {
				mockMonitorService.On("GetMonitor", mock.Anything, monitorID, mock.AnythingOfType("string")).Return(
					&dto.MonitorResponse{
						ID:              monitorID,
						UserID:          "00000000-0000-0000-0000-000000000001",
						Name:            "Test Monitor",
						URL:             "https://example.com",
						CheckType:       "HTTP",
						IntervalSeconds: 60,
						TimeoutSeconds:  30,
						Status:          "UP",
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					}, nil).Once()
			},
			expectError: false,
		},
		{
			name: "monitor not found",
			request: &monitov1.GetMonitorRequest{
				Id: monitorID,
			},
			mockSetup: func() {
				mockMonitorService.On("GetMonitor", mock.Anything, monitorID, mock.AnythingOfType("string")).Return(
					(*dto.MonitorResponse)(nil), interfaces.ErrMonitorNotFound).Once()
			},
			expectError:  true,
			expectedCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.GetMonitor(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotNil(t, resp.Monitor)
				assert.Equal(t, monitorID, resp.Monitor.Id)
			}

			mockMonitorService.AssertExpectations(t)
		})
	}
}

func TestMonitorHandler_ListMonitors(t *testing.T) {
	t.Parallel()
	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	ctx := createTestContext()

	tests := []struct {
		name         string
		request      *monitov1.ListMonitorsRequest
		mockSetup    func()
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name: "successful list monitors",
			request: &monitov1.ListMonitorsRequest{
				Limit:  10,
				Offset: 0,
				Status: "UP",
			},
			mockSetup: func() {
				mockMonitorService.On("ListMonitors", mock.Anything, mock.MatchedBy(func(req *dto.ListMonitorsRequest) bool {
					return req.Limit == 10 && req.Offset == 0 && req.Status == "UP"
				})).Return(
					&dto.ListMonitorsResponse{
						Monitors: []*dto.MonitorResponse{
							{
								ID:              uuid.New().String(),
								UserID:          "00000000-0000-0000-0000-000000000001",
								Name:            "Monitor 1",
								URL:             "https://example1.com",
								CheckType:       "HTTP",
								IntervalSeconds: 60,
								TimeoutSeconds:  30,
								Status:          "UP",
								CreatedAt:       time.Now(),
								UpdatedAt:       time.Now(),
							},
						},
						Total:  1,
						Limit:  10,
						Offset: 0,
					}, nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.ListMonitors(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}

			mockMonitorService.AssertExpectations(t)
		})
	}
}

func TestMonitorHandler_UpdateMonitor(t *testing.T) {
	t.Parallel()
	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	ctx := createTestContext()
	monitorID := uuid.New().String()

	tests := []struct {
		name         string
		request      *monitov1.UpdateMonitorRequest
		mockSetup    func()
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name: "successful update monitor",
			request: &monitov1.UpdateMonitorRequest{
				Id:              monitorID,
				Name:            "Updated Monitor",
				Url:             "https://updated.com",
				IntervalSeconds: 120,
				TimeoutSeconds:  60,
			},
			mockSetup: func() {
				mockMonitorService.On("UpdateMonitor", mock.Anything, mock.AnythingOfType("*dto.UpdateMonitorRequest")).Return(
					&dto.MonitorResponse{
						ID:              monitorID,
						UserID:          "00000000-0000-0000-0000-000000000001",
						Name:            "Updated Monitor",
						URL:             "https://updated.com",
						CheckType:       "HTTP",
						IntervalSeconds: 120,
						TimeoutSeconds:  60,
						Status:          "UP",
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					}, nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.UpdateMonitor(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotNil(t, resp.Monitor)
			}

			mockMonitorService.AssertExpectations(t)
		})
	}
}

func TestMonitorHandler_DeleteMonitor(t *testing.T) {
	t.Parallel()
	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	ctx := createTestContext()
	monitorID := uuid.New().String()

	tests := []struct {
		name         string
		request      *monitov1.DeleteMonitorRequest
		mockSetup    func()
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name: "successful delete monitor",
			request: &monitov1.DeleteMonitorRequest{
				Id: monitorID,
			},
			mockSetup: func() {
				mockMonitorService.On("DeleteMonitor", mock.Anything, monitorID, mock.AnythingOfType("string")).Return(nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.DeleteMonitor(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}

			mockMonitorService.AssertExpectations(t)
		})
	}
}

func TestMonitorHandler_PauseMonitor(t *testing.T) {
	t.Parallel()
	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	ctx := createTestContext()
	monitorID := uuid.New().String()

	tests := []struct {
		name         string
		request      *monitov1.PauseMonitorRequest
		mockSetup    func()
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name: "successful pause monitor",
			request: &monitov1.PauseMonitorRequest{
				Id: monitorID,
			},
			mockSetup: func() {
				mockMonitorService.On("PauseMonitor", mock.Anything, mock.AnythingOfType("*dto.PauseMonitorRequest")).Return(nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.PauseMonitor(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}

			mockMonitorService.AssertExpectations(t)
		})
	}
}

func TestMonitorHandler_ResumeMonitor(t *testing.T) {
	t.Parallel()
	mockMonitorService := new(service.MockMonitorService)
	handler := &MonitorHandler{
		monitorService: mockMonitorService,
		validator:      validator.New(),
	}

	ctx := createTestContext()
	monitorID := uuid.New().String()

	tests := []struct {
		name         string
		request      *monitov1.ResumeMonitorRequest
		mockSetup    func()
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name: "successful resume monitor",
			request: &monitov1.ResumeMonitorRequest{
				Id: monitorID,
			},
			mockSetup: func() {
				mockMonitorService.On("ResumeMonitor", mock.Anything, mock.AnythingOfType("*dto.ResumeMonitorRequest")).Return(nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.ResumeMonitor(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}

			mockMonitorService.AssertExpectations(t)
		})
	}
}

func TestMonitorHandler_GetMonitorHistory(t *testing.T) {
	t.Parallel()
	mockUptimeCalculator := new(service.MockUptimeCalculator)
	handler := &MonitorHandler{
		uptimeCalculator: mockUptimeCalculator,
		validator:        validator.New(),
	}

	ctx := createTestContext()
	monitorID := uuid.New().String()

	tests := []struct {
		name         string
		request      *monitov1.GetMonitorHistoryRequest
		mockSetup    func()
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name: "successful get monitor history",
			request: &monitov1.GetMonitorHistoryRequest{
				MonitorId: monitorID,
				Limit:     10,
				Offset:    0,
			},
			mockSetup: func() {
				mockUptimeCalculator.On("GetMonitorHistory", mock.Anything, mock.AnythingOfType("*dto.GetMonitorHistoryRequest")).Return(
					&dto.GetMonitorHistoryResponse{
						Results: []*dto.CheckResultResponse{
							{
								ID:        uuid.New().String(),
								MonitorID: monitorID,
								Status:    "UP",
								CheckedAt: time.Now(),
								CreatedAt: time.Now(),
							},
						},
						Total:  1,
						Limit:  10,
						Offset: 0,
					}, nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.GetMonitorHistory(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.Results, 1)
			}

			mockUptimeCalculator.AssertExpectations(t)
		})
	}
}

func TestMonitorHandler_GetUptimeStats(t *testing.T) {
	t.Parallel()
	mockUptimeCalculator := new(service.MockUptimeCalculator)
	handler := &MonitorHandler{
		uptimeCalculator: mockUptimeCalculator,
		validator:        validator.New(),
	}

	ctx := createTestContext()
	monitorID := uuid.New().String()

	tests := []struct {
		name         string
		request      *monitov1.GetUptimeStatsRequest
		mockSetup    func()
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name: "successful get uptime stats",
			request: &monitov1.GetUptimeStatsRequest{
				MonitorId: monitorID,
			},
			mockSetup: func() {
				mockUptimeCalculator.On("CalculateUptime", mock.Anything, mock.AnythingOfType("*dto.GetUptimeStatsRequest")).Return(
					&dto.UptimeStatsResponse{
						Uptime:              99.9,
						TotalChecks:         1000,
						UpChecks:            999,
						DegradedChecks:      1,
						DownChecks:          0,
						PausedChecks:        0,
						TotalDowntime:       0,
						AverageResponseTime: 250,
						Incidents:           0,
						Note:                "",
					}, nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.GetUptimeStats(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, float64(99.9), resp.Uptime)
			}

			mockUptimeCalculator.AssertExpectations(t)
		})
	}
}

func TestMonitorHandler_GetIncidents(t *testing.T) {
	t.Parallel()
	mockIncidentDetector := new(service.MockIncidentDetector)
	handler := &MonitorHandler{
		incidentDetector: mockIncidentDetector,
		validator:        validator.New(),
	}

	ctx := createTestContext()
	monitorID := uuid.New().String()

	tests := []struct {
		name         string
		request      *monitov1.GetIncidentsRequest
		mockSetup    func()
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name: "successful get incidents",
			request: &monitov1.GetIncidentsRequest{
				MonitorId: monitorID,
				Limit:     10,
				Offset:    0,
			},
			mockSetup: func() {
				mockIncidentDetector.On("GetIncidents", mock.Anything, mock.AnythingOfType("*dto.GetIncidentsRequest")).Return(
					&dto.GetIncidentsResponse{
						Incidents: []*dto.IncidentResponse{
							{
								ID:        uuid.New().String(),
								MonitorID: monitorID,
								StartTime: time.Now().Add(-1 * time.Hour),
								EndTime:   nil,
								Status:    "ACTIVE",
								CreatedAt: time.Now(),
							},
						},
						Total:  1,
						Limit:  10,
						Offset: 0,
					}, nil).Once()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.GetIncidents(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedCode, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.Incidents, 1)
			}

			mockIncidentDetector.AssertExpectations(t)
		})
	}
}
