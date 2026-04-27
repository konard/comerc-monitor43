package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	apiv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/raul/monitor/backend/integration-service/internal/model"
)

// MonitorClient gRPC клиент для Monitor Service.
type MonitorClient struct {
	client apiv1.MonitorServiceClient
}

// NewMonitorClient создаёт новый MonitorClient.
func NewMonitorClient(conn *grpc.ClientConn) *MonitorClient {
	return &MonitorClient{
		client: apiv1.NewMonitorServiceClient(conn),
	}
}

// CreateMonitor создаёт новый монитор в Monitor Service.
func (c *MonitorClient) CreateMonitor(ctx context.Context, userID uuid.UUID, data *model.MonitorImportData) (*uuid.UUID, error) {
	// Конвертируем ExpectedStatus из *int в string
	expectedStatusCode := "200" // default
	if data.ExpectedStatus != nil {
		expectedStatusCode = fmt.Sprintf("%d", *data.ExpectedStatus)
	}

	// Конвертируем MonitorImportData в CreateMonitorRequest
	req := &apiv1.CreateMonitorRequest{
		Name:                data.Name,
		Url:                 data.URL,
		CheckType:           apiv1.CheckType_CHECK_TYPE_HTTP,             // Импорт поддерживает только HTTP-проверки
		IntervalSeconds:     int32(getOrDefault(data.CheckInterval, 60)), //nolint:gosec // G115: значение ограничено валидными интервалами
		Timeout:             durationpb.New(getDurationOrDefault(data.Timeout, 30)),
		ExpectedStatusCode:  expectedStatusCode,
		ExpectedBodyPattern: getOrDefaultString(data.ExpectedPattern, ""),
		SslVerify:           true,
		Headers:             data.Headers,
	}

	// Вызываем gRPC
	resp, err := c.client.CreateMonitor(ctx, req)
	if err != nil {
		return nil, err
	}

	// Парсим ID созданного монитора
	monitorID, err := uuid.Parse(resp.Id)
	if err != nil {
		return nil, err
	}

	return &monitorID, nil
}

// GetMonitorByName получает монитор по имени для пользователя.
// Примечание: Загружает все мониторы пользователя. При большом количестве (>1000)
// рекомендуется добавить циклическую загрузку с пагинацией.
func (c *MonitorClient) GetMonitorByName(ctx context.Context, userID uuid.UUID, name string) (*uuid.UUID, error) {
	req := &apiv1.ListMonitorsRequest{
		UserId:   userID.String(),
		PageSize: 1000, // Загружаем до 1000 мониторов за раз
	}

	resp, err := c.client.ListMonitors(ctx, req)
	if err != nil {
		return nil, err
	}

	// Ищем монитор с таким же именем
	for _, monitor := range resp.Monitors {
		if monitor.Name == name {
			monitorID, err := uuid.Parse(monitor.Id)
			if err != nil {
				return nil, err
			}
			return &monitorID, nil
		}
	}

	// Монитор не найден
	return nil, nil
}

// getOrDefault возвращает значение или дефолтное.
func getOrDefault(ptr *int, defaultVal int) int {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

// getDurationOrDefault возвращает duration или дефолтное.
func getDurationOrDefault(ptr *int, defaultSeconds int) time.Duration {
	seconds := getOrDefault(ptr, defaultSeconds)
	return time.Duration(seconds) * time.Second
}

// getOrDefaultString возвращает строку или дефолтное значение.
func getOrDefaultString(ptr *string, defaultVal string) string {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}
