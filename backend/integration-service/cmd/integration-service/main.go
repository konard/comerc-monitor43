// Package main является точкой входа для integration-service.
//
// Service запускает gRPC сервер, webhook delivery service с worker pools,
// и обеспечивает интеграцию с PostgreSQL, RabbitMQ и Observability системами.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	grpcstd "github.com/pure-golang/adapters/grpc/std"
	httpstd "github.com/pure-golang/adapters/httpserver/std"
	adapterlogger "github.com/pure-golang/adapters/logger"
	"github.com/pure-golang/adapters/metrics"
	"github.com/pure-golang/adapters/queue/rabbitmq"
	tracingjaeger "github.com/pure-golang/adapters/tracing"
	"github.com/pure-golang/adapters/tracing/jaeger"
	amqp "github.com/rabbitmq/amqp091-go"
	integrationv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/raul/monitor/backend/integration-service/internal/client"
	grpchandler "github.com/raul/monitor/backend/integration-service/internal/handler"
	"github.com/raul/monitor/backend/integration-service/internal/health"
	"github.com/raul/monitor/backend/integration-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/integration-service/internal/infrastructure/config"
	"github.com/raul/monitor/backend/integration-service/internal/infrastructure/publisher"
	"github.com/raul/monitor/backend/integration-service/internal/repository/postgres"
	"github.com/raul/monitor/backend/integration-service/internal/service/apikey"
	importsvc "github.com/raul/monitor/backend/integration-service/internal/service/import"
	"github.com/raul/monitor/backend/integration-service/internal/service/security"
	"github.com/raul/monitor/backend/integration-service/internal/service/webhook"
	"github.com/raul/monitor/backend/integration-service/pkg/logger"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// Load .env file if exists
	//nolint:errcheck // .env file is optional, failure is acceptable
	_ = godotenv.Load()

	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Initialize logger
	adapterlogger.InitDefault(adapterlogger.Config{
		Provider: adapterlogger.ProviderStdJson,
		Level:    adapterlogger.INFO,
	})

	slogger := slog.Default()
	if err := logger.InitLogger("info"); err != nil {
		slogger.WarnContext(ctx, "Failed to initialize package logger", "error", err)
	}

	slogger.InfoContext(ctx, "Starting integration service",
		"http_port", cfg.ServerPort,
		"grpc_port", cfg.ServerGRPCPort,
	)

	// 3. Initialize tracing
	traceProvider, err := tracingjaeger.Init(jaeger.NewProviderBuilder(jaeger.Config{
		EndPoint:    cfg.OTELExporterOTLPEndpoint,
		ServiceName: "integration-service",
		AppVersion:  os.Getenv("VERSION"),
	}))
	if err != nil {
		slogger.WarnContext(ctx, "Failed to initialize tracing, using noop", "error", err)
	}
	defer func() {
		if traceProvider != nil {
			if closeErr := traceProvider.Close(); closeErr != nil {
				slogger.ErrorContext(ctx, "Failed to close tracing provider", "error", closeErr)
			}
		}
	}()

	// 4. Initialize metrics server
	var metricsCloser io.Closer
	if cfg.MetricsEnabled {
		metricsCloser, err = metrics.InitDefault(metrics.Config{
			Host:                  "0.0.0.0",
			Port:                  cfg.MetricsPort,
			HttpServerReadTimeout: 30,
		})
		if err != nil {
			slogger.WarnContext(ctx, "Failed to start metrics server", "error", err)
		} else {
			slogger.InfoContext(ctx, "Metrics server started", "port", cfg.MetricsPort)
			defer func() {
				if closeErr := metricsCloser.Close(); closeErr != nil {
					slogger.ErrorContext(ctx, "Failed to close metrics server", "error", closeErr)
				}
			}()
		}
	}

	// 5. Initialize PostgreSQL
	db, err := postgres.NewDB(cfg.DSN())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slogger.ErrorContext(ctx, "Failed to close database", "error", err)
		}
	}()

	// Run migrations
	if err := runMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slogger.InfoContext(ctx, "Database connected and migrations applied")

	// 6. Initialize RabbitMQ (optional)
	var _ *amqp.Connection
	var alertConsumer *publisher.AlertConsumer
	var rmqCloser func() error

	if cfg.RabbitMQEnabled {
		_, alertConsumer, rmqCloser, err = initRabbitMQ(ctx, cfg, slogger)
		if err != nil {
			return fmt.Errorf("failed to init rabbitmq: %w", err)
		}
		defer func() {
			if rmqCloser != nil {
				if err := rmqCloser(); err != nil {
					slogger.ErrorContext(ctx, "Failed to close RabbitMQ", "error", err)
				}
			}
		}()
	} else {
		slogger.InfoContext(ctx, "RabbitMQ disabled, webhook delivery disabled")
	}

	// 7. Initialize security services
	encryptor, err := security.NewEncryptionService(cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to create encryption service: %w", err)
	}

	keyGenerator := security.NewAPIKeyGenerator()

	// 8. Initialize repositories
	webhookRepo := postgres.NewWebhookRepository(db)
	webhookDeliveryRepo := postgres.NewWebhookDeliveryRepository(db)
	apiKeyRepo := postgres.NewAPIKeyRepository(db)
	apiKeyUsageRepo := postgres.NewAPIKeyUsageRepository(db)
	importHistoryRepo := postgres.NewImportHistoryRepository(db.DB)

	// 8.1. Initialize gRPC clients to external services
	// Monitor Service
	monitorServiceConn, err := grpc.NewClient(cfg.MonitorServiceGRPCAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to monitor service: %w", err)
	}
	defer func() {
		if closeErr := monitorServiceConn.Close(); closeErr != nil {
			slogger.ErrorContext(ctx, "Failed to close monitor service connection", "error", closeErr)
		}
	}()
	monitorClient := client.NewMonitorClient(monitorServiceConn)

	slogger.InfoContext(ctx, "Connected to Monitor Service", "address", cfg.MonitorServiceGRPCAddress)

	// Billing Service
	billingServiceConn, err := grpc.NewClient(cfg.BillingServiceGRPCAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to billing service: %w", err)
	}
	defer func() {
		if closeErr := billingServiceConn.Close(); closeErr != nil {
			slogger.ErrorContext(ctx, "Failed to close billing service connection", "error", closeErr)
		}
	}()
	billingClient := client.NewBillingClient(billingServiceConn)

	slogger.InfoContext(ctx, "Connected to Billing Service", "address", cfg.BillingServiceGRPCAddress)

	// 9. Initialize services
	rateLimiter := apikey.NewSlidingWindowRateLimiter()

	webhookService := webhook.NewWebhookService(webhookRepo, encryptor, keyGenerator, billingClient, nil)

	apiKeyService := apikey.NewAPIKeyService(
		apiKeyRepo,
		apiKeyUsageRepo,
		encryptor,
		keyGenerator,
		billingClient,
		rateLimiter,
	)

	var webhookDeliveryService *webhook.WebhookDeliveryService
	if cfg.RabbitMQEnabled {
		webhookDeliveryConfig := &webhook.WebhookDeliveryConfig{
			MaxWorkers:         10,
			WorkerQueueSize:    100,
			MaxRetries:         3,
			RetryIntervals:     []time.Duration{30 * time.Second, 60 * time.Second, 120 * time.Second},
			RetryCheckInterval: 30 * time.Second,
			HTTPTimeout:        10 * time.Second,
		}
		webhookDeliveryService = webhook.NewWebhookDeliveryService(webhookRepo, webhookDeliveryRepo, encryptor, webhookDeliveryConfig)
	}

	// Import Service
	importService := importsvc.NewImportService(importHistoryRepo, monitorClient)

	// 10. Initialize gRPC handlers
	webhookHandler := grpchandler.NewWebhookHandler(webhookService)
	apikeyHandler := grpchandler.NewAPIKeyHandler(apiKeyService)
	importHandler := grpchandler.NewImportHandler(importService)

	// 11. Set up HTTP handlers (health checks)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(db.DB))
	mux.HandleFunc("/ready", readyHandler(db.DB))

	// Initialize HTTP server
	httpServer := httpstd.New(
		httpstd.Config{
			Host: "",
			Port: cfg.ServerPort,
		},
		mux,
	)

	// 11.1. Initialize auth interceptor (JWT)
	jwtInterceptor := auth.NewJWTInterceptor(cfg.JWTSecret)
	slogger.InfoContext(ctx, "JWT interceptor initialized")

	// 12. Initialize gRPC server
	grpcServer := &grpcServerWrapper{server: grpcstd.New(
		grpcstd.Config{
			Host:          "",
			Port:          cfg.ServerGRPCPort,
			EnableReflect: true,
		},
		func(s *grpc.Server) {
			// Регистрируем gRPC сервисы
			integrationv1.RegisterWebhookIntegrationServiceServer(s, webhookHandler)
			integrationv1.RegisterAPIKeyServiceServer(s, apikeyHandler)
			integrationv1.RegisterImportServiceServer(s, importHandler)

			// Регистрируем gRPC Health Service
			if db != nil {
				healthChecker := health.NewHealthCheckerFromSQLxDB(db.DB)
				healthServer := health.NewGRPCHealthServer(healthChecker)
				grpc_health_v1.RegisterHealthServer(s, healthServer)
			}
		},
		grpcstd.WithUnaryInterceptor(grpchandler.TracingInterceptor()),
		grpcstd.WithUnaryInterceptor(jwtInterceptor.Unary()),
	)}

	// 13. Start webhook delivery service (if RabbitMQ enabled)
	if cfg.RabbitMQEnabled && webhookDeliveryService != nil {
		if err := webhookDeliveryService.Start(ctx); err != nil {
			return fmt.Errorf("failed to start webhook delivery service: %w", err)
		}
		defer func() {
			if stopErr := webhookDeliveryService.Stop(ctx); stopErr != nil {
				slogger.ErrorContext(ctx, "Failed to stop webhook delivery service", "error", stopErr)
			}
		}()

		// Start alert consumer
		if alertConsumer != nil {
			if err := alertConsumer.Start(ctx); err != nil {
				return fmt.Errorf("failed to start alert consumer: %w", err)
			}
			defer func() {
				if stopErr := alertConsumer.Stop(ctx); stopErr != nil {
					slogger.ErrorContext(ctx, "Failed to stop alert consumer", "error", stopErr)
				}
			}()
		}

		slogger.InfoContext(ctx, "Webhook delivery service started")
	}

	// 14. Start servers
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 2)

	go func() {
		if err := httpServer.Start(); err != nil {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	go func() {
		if err := grpcServer.Start(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	slogger.InfoContext(ctx, "Integration service started successfully")

	// 15. Graceful shutdown
	shutdown := setupShutdown()

	select {
	case <-shutdown:
		slogger.InfoContext(ctx, "Shutdown signal received")
	case err := <-errChan:
		return err
	}

	slogger.InfoContext(ctx, "Shutting down...")

	if err := httpServer.Close(); err != nil {
		slogger.ErrorContext(ctx, "Failed to stop HTTP server", "error", err)
	}
	if err := grpcServer.Close(); err != nil {
		slogger.ErrorContext(ctx, "Failed to stop gRPC server", "error", err)
	}

	slogger.InfoContext(ctx, "Shutdown complete")
	return nil
}

func initRabbitMQ(ctx context.Context, cfg *config.Config, slogger *slog.Logger) (*amqp.Connection, *publisher.AlertConsumer, func() error, error) {
	dialer := rabbitmq.NewDialer(cfg.RabbitMQURL(), &rabbitmq.DialerOptions{
		Logger: slogger.With(
			"component", "rabbitmq",
			"host", cfg.RabbitMQHost,
			"port", cfg.RabbitMQPort,
		),
	})

	rabbitmq.NewPublisher2(rabbitmq.PublisherConfig{
		Exchange: cfg.RabbitMQExchangeName(),
	}, dialer)

	// NOTE: Connection is managed by the dialer internally
	// We'll use nil for connection since it's handled internally by the publisher

	// Create alert consumer
	consumerConfig := &publisher.AlertConsumerConfig{
		ExchangeName: cfg.RabbitMQExchangeName(),
		RoutingKey:   "alert.triggered",
		QueueName:    "integration_service_webhooks",
	}

	// NOTE: Alert consumer will be initialized after webhookDeliveryService is created
	// The consumer is started later in the main flow after all dependencies are ready
	_ = consumerConfig

	closer := func() error {
		return dialer.Close()
	}

	slogger.InfoContext(ctx, "RabbitMQ initialized",
		"exchange", cfg.RabbitMQExchangeName(),
	)

	return nil, nil, closer, nil
}

func setupShutdown() <-chan os.Signal {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	return shutdown
}

// grpcServerWrapper wraps pure-golang Server to provide Start/Close methods
type grpcServerWrapper struct {
	server *grpcstd.Server
}

func (g *grpcServerWrapper) Start() error {
	return g.server.Start()
}

func (g *grpcServerWrapper) Close() error {
	return g.server.Close()
}

func healthHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result := checkHealth(ctx, db)

		w.Header().Set("Content-Type", "application/json")
		if result.Status == "healthy" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		fmt.Fprintf(w, `{"status":"%s","database":"%s"}`, result.Status, result.Database)
	}
}

func readyHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result := checkHealth(ctx, db)

		w.Header().Set("Content-Type", "application/json")
		if result.Status == "healthy" && result.Database == "healthy" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		fmt.Fprintf(w, `{"status":"%s","database":"%s"}`, result.Status, result.Database)
	}
}

type healthResult struct {
	Status   string
	Database string
}

func checkHealth(ctx context.Context, db *sqlx.DB) *healthResult {
	result := &healthResult{
		Status:   "healthy",
		Database: "healthy",
	}

	// Check database
	err := db.PingContext(ctx)
	if err != nil {
		result.Status = "unhealthy"
		result.Database = "unhealthy"
	}

	return result
}

func runMigrations(db *postgres.DB) error {
	goose.SetTableName("goose_db_version")
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if len(os.Args) > 2 && os.Args[2] == "down" {
			return goose.Down(db.DB.DB, "./migrations")
		}
		if len(os.Args) > 2 && os.Args[2] == "reset" {
			return goose.Reset(db.DB.DB, "./migrations")
		}
	}
	return goose.Up(db.DB.DB, "./migrations")
}
