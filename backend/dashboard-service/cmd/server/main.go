package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"

	"github.com/raul/monitor/backend/dashboard-service/internal/config"
	"github.com/raul/monitor/backend/dashboard-service/internal/repository/postgres"
	applogger "github.com/raul/monitor/backend/dashboard-service/pkg/logger"
	"github.com/raul/monitor/backend/dashboard-service/pkg/telemetry"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := applogger.New(cfg.OTELLogLevel)

	tracer, err := telemetry.NewTracer(cfg.OTELServiceName, cfg.OTELExporterOTLPEndpoint)
	if err != nil {
		logger.Error("failed to initialize tracer", "error", err)
		os.Exit(1)
	}

	metrics, err := telemetry.NewMetrics(cfg.OTELServiceName)
	if err != nil {
		logger.Error("failed to initialize metrics", "error", err)
		os.Exit(1)
	}

	db := postgres.NewDB(cfg.DSN())

	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	db.SetObservability(tracer.GetTracer(), metrics)

	// Автоматический запуск миграций при старте; поддержка `migrate down/reset` через CLI-аргументы.
	if err := runMigrations(db.DB.DB); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	deps := &Dependencies{
		Cfg:     cfg,
		Logger:  logger,
		Tracer:  tracer,
		Metrics: metrics,
		DB:      db,
	}

	server := NewServer(deps)

	if err := server.Start(ctx); err != nil {
		logger.Error("failed to start server", "error", err)
		os.Exit(1)
	}

	logger.Info("dashboard-service started successfully")
	if err := server.WaitForSignals(); err != nil {
		logger.Error("server error", "error", err)
	}
}

// runMigrations применяет миграции goose автоматически при старте.
// Поддерживает CLI: `migrate down`, `migrate reset` для ручного управления.
func runMigrations(db *sql.DB) error {
	goose.SetTableName("goose_db_version")
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if len(os.Args) > 2 && os.Args[2] == "down" {
			return goose.Down(db, "./migrations")
		}
		if len(os.Args) > 2 && os.Args[2] == "reset" {
			return goose.Reset(db, "./migrations")
		}
	}
	return goose.Up(db, "./migrations")
}
