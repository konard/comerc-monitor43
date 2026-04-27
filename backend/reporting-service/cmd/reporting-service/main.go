package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
	httpstd "github.com/pure-golang/adapters/httpserver/std"
	adapterlogger "github.com/pure-golang/adapters/logger"
	"github.com/pure-golang/adapters/metrics"
	tracingjaeger "github.com/pure-golang/adapters/tracing"
	"github.com/pure-golang/adapters/tracing/jaeger"

	"github.com/raul/monitor/backend/reporting-service/internal/config"
	"github.com/raul/monitor/backend/reporting-service/internal/handler"
	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/export"
	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/monitor_client"
	"github.com/raul/monitor/backend/reporting-service/internal/repository/postgres"
	"github.com/raul/monitor/backend/reporting-service/internal/service"
)

func main() {
	ctx := context.Background()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(ctx, log); err != nil {
		log.ErrorContext(ctx, "reporting-service failed to start", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	//nolint:errcheck // .env file is optional, failure is acceptable
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	adapterlogger.InitDefault(adapterlogger.Config{
		Provider: adapterlogger.ProviderStdJson,
		Level:    adapterlogger.INFO,
	})

	log.InfoContext(ctx, "starting reporting-service",
		"grpc_port", cfg.Server.GRPCPort,
		"http_port", cfg.Server.Port,
	)

	var traceProvider tracingjaeger.Provider
	if cfg.Tracing.Enabled {
		traceProvider, err = tracingjaeger.Init(jaeger.NewProviderBuilder(jaeger.Config{
			EndPoint:    cfg.Tracing.Endpoint,
			ServiceName: cfg.Tracing.ServiceName,
			AppVersion:  os.Getenv("VERSION"),
		}))
		if err != nil {
			log.WarnContext(ctx, "failed to initialize tracing, using noop", "error", err)
		}
		defer func() {
			if traceProvider != nil {
				if closeErr := traceProvider.Close(); closeErr != nil {
					log.ErrorContext(ctx, "failed to close tracing provider", "error", closeErr)
				}
			}
		}()
	}

	var metricsCloser io.Closer
	if cfg.Metrics.Enabled {
		metricsCloser, err = metrics.InitDefault(metrics.Config{
			Host:                  "0.0.0.0",
			Port:                  cfg.Metrics.Port,
			HttpServerReadTimeout: 30,
		})
		if err != nil {
			log.WarnContext(ctx, "failed to start metrics server", "error", err)
		} else {
			log.InfoContext(ctx, "metrics server started", "port", cfg.Metrics.Port)
			defer func() {
				if closeErr := metricsCloser.Close(); closeErr != nil {
					log.ErrorContext(ctx, "failed to close metrics server", "error", closeErr)
				}
			}()
		}
	}

	db := postgres.NewDB(cfg.Database)
	if err := db.Connect(ctx); err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.ErrorContext(ctx, "failed to close database", "error", err)
		}
	}()

	if err := runMigrations(db); err != nil {
		return errors.Wrap(err, "failed to run migrations")
	}

	monitorClient, err := monitor_client.NewMonitorClient(cfg.MonitorService.GRPCAddress)
	if err != nil {
		return errors.Wrap(err, "failed to create monitor client")
	}
	defer func() {
		if err := monitorClient.Close(); err != nil {
			log.ErrorContext(ctx, "failed to close monitor client", "error", err)
		}
	}()

	csvExporter := export.NewCSVExporter()
	pdfExporter := export.NewPDFExporter()

	reportRepo := postgres.NewReportRepository(db)

	slaService := service.NewSLAService(monitorClient, reportRepo, log, cfg.Reporting.RetentionDays)
	analyticsService := service.NewAnalyticsService(monitorClient, log, cfg.Reporting.MaxExportRows)
	exportService := service.NewExportService(monitorClient, reportRepo, csvExporter, pdfExporter, log, cfg.Reporting.MaxExportRows)

	reportingHandler := handler.NewReportingHandler(
		slaService,
		analyticsService,
		exportService,
		log,
	)

	authMiddleware := auth.NewAuthMiddleware(cfg.Auth.SecretKey)
	log.InfoContext(ctx, "auth middleware initialized")

	grpcServer := handler.NewServer(cfg.Server.GRPCPort, reportingHandler, authMiddleware)

	log.InfoContext(ctx, "grpc server registered",
		"service", "reporting.ReportingService",
		"auth_enabled", true,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := db.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, writeErr := w.Write([]byte(`{"status":"unhealthy"}`)); writeErr != nil {
				log.WarnContext(r.Context(), "failed to write unhealthy health response", "error", writeErr)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, writeErr := w.Write([]byte(`{"status":"healthy"}`)); writeErr != nil {
			log.WarnContext(r.Context(), "failed to write healthy health response", "error", writeErr)
		}
	})

	httpServer := httpstd.NewDefault(
		httpstd.Config{
			Host: "",
			Port: cfg.Server.Port,
		},
		mux,
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		if err := grpcServer.Start(); err != nil {
			errCh <- errors.Wrap(err, "grpc server error")
		}
	}()

	go func() {
		if err := httpServer.Start(); err != nil {
			errCh <- errors.Wrap(err, "http server error")
		}
	}()

	log.InfoContext(ctx, "reporting-service started successfully")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.InfoContext(ctx, "shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		return err
	}

	log.InfoContext(ctx, "shutting down...")

	if err := grpcServer.Close(); err != nil {
		log.ErrorContext(ctx, "failed to stop grpc server", "error", err)
	}
	if err := httpServer.Close(); err != nil {
		log.ErrorContext(ctx, "failed to stop http server", "error", err)
	}

	log.InfoContext(ctx, "shutdown complete")
	return nil
}

func runMigrations(db *postgres.DB) error {
	goose.SetTableName("goose_reporting_version")
	if err := goose.SetDialect("postgres"); err != nil {
		return errors.Wrap(err, "failed to set dialect")
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
