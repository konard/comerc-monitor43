package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/raul/monitor/backend/notification-template-service/internal/config"
	"github.com/raul/monitor/backend/notification-template-service/internal/handler"
	"github.com/raul/monitor/backend/notification-template-service/internal/infrastructure/health"
	"github.com/raul/monitor/backend/notification-template-service/internal/infrastructure/logging"
	"github.com/raul/monitor/backend/notification-template-service/internal/infrastructure/tracing"
	"github.com/raul/monitor/backend/notification-template-service/internal/repository"
	"github.com/raul/monitor/backend/notification-template-service/internal/service"
)

func main() {
	// Загрузка конфигурации
	var cfg config.Config
	if err := loadConfig(&cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Инициализация логирования
	logger := logging.New(cfg.LogLevel, cfg.Environment)

	// Инициализация tracing
	if cfg.JaegerEndpoint != "" {
		if err := tracing.Init(cfg.JaegerEndpoint, cfg.Environment); err != nil {
			logger.Warnf("failed to init tracing: %v", err)
		}
	}

	// Подключение к базе данных
	db, err := connectDB(&cfg)
	if err != nil {
		logger.Fatal().Msgf("failed to connect to database: %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Errorf("failed to close database: %v", closeErr)
		}
	}()

	// Автоматический запуск миграций при старте
	if err := runMigrations(db.DB); err != nil {
		logger.Fatal().Msgf("failed to run migrations: %v", err)
	}

	// Инициализация репозиториев
	templateRepo := repository.NewPostgresTemplateRepository(db)

	// Инициализация сервисов
	templateSvc := service.NewTemplateService(templateRepo, logger.ZLogger(), &cfg)
	rendererSvc := service.NewRendererService(templateRepo, logger.ZLogger(), &cfg)
	validatorSvc := service.NewValidatorService(logger.ZLogger())

	// Инициализация health checker
	healthChecker := health.NewChecker(db, logger.ZLogger())

	// Создание gRPC сервера
	srv := newServer(&cfg, templateSvc, rendererSvc, validatorSvc, logger)

	// Запуск сервера
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logger.Infof("starting notification-template-service on %s", cfg.ServerAddress)
		listener, err := net.Listen("tcp", cfg.ServerAddress)
		if err != nil {
			logger.Fatal().Msgf("failed to listen: %v", err)
		}

		if err := srv.Serve(listener); err != nil {
			logger.Fatal().Msgf("server failed: %v", err)
		}
	}()

	// Запуск HTTP сервера для health checks
	go func() {
		healthAddr := ":8080"
		logger.Infof("starting health check server on %s", healthAddr)

		// Регистрируем handlers
		mux := http.NewServeMux()
		mux.HandleFunc("/health", healthChecker.Handler())
		mux.Handle("/metrics", promhttp.Handler())

		if err := http.ListenAndServe(healthAddr, mux); err != nil { //nolint:gosec // G114: внутренний health endpoint без таймаутов допустим
			logger.Errorf("health check server failed: %v", err)
		}
	}()

	// Graceful shutdown
	shutdown(ctx, cancel, srv, logger, &cfg, db)
}

func loadConfig(cfg *config.Config) error {
	if err := envconfig.Process("notification_template", cfg); err != nil {
		return fmt.Errorf("failed to load config from env: %w", err)
	}
	return cfg.Validate()
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

func connectDB(cfg *config.Config) (*sqlx.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sqlx.ConnectContext(ctx, "postgres", cfg.DatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настройка connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func newServer(
	cfg *config.Config,
	templateSvc service.TemplateService,
	rendererSvc service.RendererService,
	validatorSvc service.ValidatorService,
	logger *logging.Logger,
) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(logger),
			tracing.UnaryServerInterceptor(),
		),
	)

	h := handler.New(templateSvc, rendererSvc, validatorSvc)
	v1.RegisterNotificationTemplateServiceServer(srv, h)

	// Регистрируем health service
	healthSrv := grpchealth.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Включаем reflection для разработки
	reflection.Register(srv)

	return srv
}

func shutdown(
	ctx context.Context,
	cancel context.CancelFunc,
	srv *grpc.Server,
	logger *logging.Logger,
	cfg *config.Config,
	db *sqlx.DB,
) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Infof("received signal %v, initiating graceful shutdown", sig)

	cancel()

	// Остановка gRPC сервера с timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	stopped := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		logger.Infof("server stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Warnf("shutdown timeout, forcing server stop")
		srv.Stop()
	}

	// Закрытие соединения с БД
	if err := db.Close(); err != nil {
		logger.Errorf("failed to close database: %v", err)
	}

	logger.Infof("shutdown complete")
}
