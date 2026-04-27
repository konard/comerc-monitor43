package testconsumer

import (
	"github.com/jmoiron/sqlx"

	"github.com/raul/monitor/backend/dashboard-service/internal/consumer"
	dashpostgres "github.com/raul/monitor/backend/dashboard-service/internal/repository/postgres"
	"github.com/raul/monitor/backend/dashboard-service/internal/service/realtime"
	applogger "github.com/raul/monitor/backend/dashboard-service/pkg/logger"
)

// Build создаёт реальный EventConsumer с реальными репозиториями.
// db должна указывать на тестовую базу данных с применёнными миграциями.
// amqpURL — адрес RabbitMQ контейнера.
func Build(db *sqlx.DB, amqpURL string) *consumer.EventConsumer {
	logger := applogger.New("warn")

	// Оборачиваем sqlx.DB в dashboard-service DB тип
	dashDB := dashpostgres.WrapDB(db)

	statusRepo := dashpostgres.NewMonitorStatusRepository(dashDB)
	checkRepo := dashpostgres.NewCheckHistoryRepository(dashDB)
	incidentRepo := dashpostgres.NewIncidentRepository(dashDB)

	// Hub для WebSocket — в тестах просто создаём без запуска
	hub := realtime.NewHub()

	return consumer.NewEventConsumer(amqpURL, statusRepo, checkRepo, incidentRepo, hub, logger)
}
