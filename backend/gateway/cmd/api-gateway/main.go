package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcadapter "github.com/raul/monitor/gateway/internal/adapter/grpc"
	httppool "github.com/raul/monitor/gateway/internal/adapter/http"
	"github.com/raul/monitor/gateway/internal/handler"
	"github.com/raul/monitor/gateway/internal/infrastructure/config"
	"github.com/raul/monitor/gateway/internal/infrastructure/health"
	"github.com/raul/monitor/gateway/internal/infrastructure/logger"
	grpcrepository "github.com/raul/monitor/gateway/internal/repository/grpc"
	"github.com/raul/monitor/gateway/internal/service"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}

func run(ctx context.Context) error {
	// 1. Load configuration
	cfg, err := config.NewConfig()
	if err != nil {
		return err
	}

	// 2. Initialize structured logger
	structuredLogger := logger.NewDefaultLogger(cfg.Observability.LogLevel)

	// Log startup
	structuredLogger.Info("Starting API Gateway",
		logger.StringField("version", "1.0.0"),
		logger.StringField("port", cfg.Server.Port),
		logger.StringField("auth_service", cfg.Auth.Address),
	)

	// 3. Initialize gRPC connection pool
	grpcPool, err := grpcadapter.New(&grpcadapter.Config{
		Address:            cfg.Auth.Address,
		MaxConns:           10,
		MaxIdleConns:       2,
		MaxConnAge:         300,
		MaxConnIdleTime:    120,
		ConnectTimeout:     10,
		Timeout:            30,
		MaxRecvMsgSize:     4194304,
		MaxSendMsgSize:     4194304,
		EnableTLS:          false,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := grpcPool.Close(ctx); closeErr != nil {
			structuredLogger.Error("Failed to close gRPC pool", logger.ErrorField(closeErr))
		}
	}()

	// 4. Initialize repositories using connection pool
	authRepo, err := grpcrepository.NewAuthRepositoryFromPool(grpcPool)
	if err != nil {
		structuredLogger.Error("Failed to initialize auth repository",
			logger.ErrorField(err),
		)
		return err
	}

	// Initialize HTTP pool for service repository
	httpPool := httppool.New(&httppool.Config{
		Timeout:               int(cfg.Downstream.ServiceTimeout.Seconds()),
		ResponseHeaderTimeout: int(cfg.Downstream.ServiceTimeout.Seconds()),
	})
	serviceRepo := grpcrepository.NewServiceRepositoryFromPool(
		httpPool,
		cfg.Downstream.Services,
	)

	// 5. Initialize health checker
	healthChecker := health.NewChecker(5 * time.Second)

	// Register health checks
	healthChecker.Register("auth-service", "auth", func(ctx context.Context) error {
		return grpcPool.HealthCheck(ctx)
	})

	services := make([]string, 0, len(cfg.Downstream.Services))
	for name := range cfg.Downstream.Services {
		healthChecker.Register(name, "service", func(ctx context.Context) error {
			_, err := serviceRepo.CheckHealth(ctx, name)
			if err != nil {
				structuredLogger.Error(
					"Downstream service health check failed",
					logger.StringField("service", name),
					logger.ErrorField(err),
				)
			}
			// For now, don't fail check if service is unreachable
			// The actual health check will happen during request
			return nil
		})
		services = append(services, name)
	}

	// 6. Initialize service health checker
	serviceHealthChecker := grpcrepository.NewServiceHealthChecker(serviceRepo, services)

	// 7. Initialize services
	authService := service.NewAuthService(authRepo)
	gatewayService := service.NewGatewayService(authRepo, serviceRepo, serviceHealthChecker, services)

	// 8. Initialize server
	stdLogger := log.New(os.Stdout, "", 0)
	server, err := handler.NewServer(
		":"+cfg.Server.Port,
		authService,
		gatewayService,
		cfg.Downstream.GrpcAddresses,
		cfg.Downstream.DashboardWSAddr,
		cfg.Server.AllowedOrigins,
		cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout,
		cfg.Server.IdleTimeout,
		stdLogger,
	)
	if err != nil {
		return err
	}

	// 9. Start server in goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			structuredLogger.Error("Server failed to start",
				logger.ErrorField(err),
			)
			serverErrChan <- err
		}
	}()

	structuredLogger.Info("API Gateway started successfully")

	// 10. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-quit:
		structuredLogger.Info("Shutting down API Gateway...")
	case err := <-serverErrChan:
		return err
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(stopCtx); err != nil {
		structuredLogger.Error("Shutdown error",
			logger.ErrorField(err),
		)
		return err
	}

	structuredLogger.Info("API Gateway stopped gracefully")
	return nil
}
