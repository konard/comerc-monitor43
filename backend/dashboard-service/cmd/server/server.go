package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	dashboardproto "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/raul/monitor/backend/dashboard-service/internal/adapters"
	"github.com/raul/monitor/backend/dashboard-service/internal/config"
	"github.com/raul/monitor/backend/dashboard-service/internal/consumer"
	grpcserver "github.com/raul/monitor/backend/dashboard-service/internal/grpc"
	"github.com/raul/monitor/backend/dashboard-service/internal/infrastructure/auth"
	"github.com/raul/monitor/backend/dashboard-service/internal/repository/postgres"
	"github.com/raul/monitor/backend/dashboard-service/internal/service/realtime"
	"github.com/raul/monitor/backend/dashboard-service/internal/ws"
	applogger "github.com/raul/monitor/backend/dashboard-service/pkg/logger"
	apptelemetry "github.com/raul/monitor/backend/dashboard-service/pkg/telemetry"
)

type Server struct {
	cfg             *config.Config
	logger          *applogger.Logger
	tracer          *apptelemetry.Tracer
	metrics         *apptelemetry.Metrics
	db              *postgres.DB
	grpcServer      *grpc.Server
	httpServer      *http.Server
	hub             *realtime.Hub
	eventConsumer   *consumer.EventConsumer
	listener        net.Listener
	shutdownOnce    sync.Once
	shutdownTimeout time.Duration
}

type Dependencies struct {
	Cfg     *config.Config
	Logger  *applogger.Logger
	Tracer  *apptelemetry.Tracer
	Metrics *apptelemetry.Metrics
	DB      *postgres.DB
}

func NewServer(deps *Dependencies) *Server {
	s := &Server{
		cfg:             deps.Cfg,
		logger:          deps.Logger,
		tracer:          deps.Tracer,
		metrics:         deps.Metrics,
		shutdownTimeout: 30 * time.Second,
	}

	if deps.DB != nil {
		s.db = deps.DB
	}

	statusRepo := postgres.NewMonitorStatusRepository(deps.DB)
	checkRepo := postgres.NewCheckHistoryRepository(deps.DB)
	incidentRepo := postgres.NewIncidentRepository(deps.DB)

	hub := realtime.NewHub()
	s.hub = hub

	dashboardAdapter := adapters.NewDashboardAdapter(statusRepo, deps.Cfg.ExportDir)
	historyAdapter := adapters.NewHistoryAdapter(checkRepo, incidentRepo, deps.Cfg.ExportDir)

	authMiddleware := auth.NewAuthMiddleware(deps.Cfg.JWTSecret)

	dashboardService := grpcserver.NewDashboardServiceServer(dashboardAdapter, historyAdapter, authMiddleware)

	s.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcserver.TracingInterceptor(),
			grpcserver.MetricsInterceptor(deps.Metrics),
			authMiddleware.UnaryInterceptor(),
		),
	)

	dashboardproto.RegisterDashboardServiceServer(s.grpcServer, dashboardService)
	reflection.Register(s.grpcServer)

	wsHandler := ws.NewWSHandler(
		hub,
		auth.NewAuthenticator(deps.Cfg.JWTSecret),
		deps.Logger,
		deps.Cfg.WSWriteTimeout,
		deps.Cfg.WSPongTimeout,
		deps.Cfg.WSPingInterval,
		deps.Cfg.WSMaxMessageSize,
	)

	mux := http.NewServeMux()
	mux.Handle("/ws", wsHandler)
	mux.HandleFunc("/healthz", s.livenessHandler)
	mux.HandleFunc("/readyz", s.readinessHandler)

	s.httpServer = &http.Server{
		Addr:              deps.Cfg.HTTPAddress(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.eventConsumer = consumer.NewEventConsumer(
		deps.Cfg.RabbitMQURL,
		statusRepo, checkRepo, incidentRepo,
		hub, deps.Logger,
	)

	return s
}

func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.cfg.ServerAddress())
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = lis

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			s.logger.Error("gRPC server error", "error", err)
		}
	}()
	s.logger.Info("gRPC server starting", "address", s.cfg.ServerAddress())

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()
	s.logger.Info("HTTP/WebSocket server starting", "address", s.cfg.HTTPAddress())

	go s.hub.Run()
	s.logger.Info("WebSocket hub started")

	if s.eventConsumer != nil {
		if err := s.eventConsumer.Start(ctx); err != nil {
			s.logger.Error("failed to start event consumer", "error", err)
		}
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	var shutdownErr error
	s.shutdownOnce.Do(func() {
		s.logger.Info("shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
		defer cancel()

		if s.eventConsumer != nil {
			if err := s.eventConsumer.Close(); err != nil {
				s.logger.Error("failed to close event consumer", "error", err)
			}
		}

		if s.hub != nil {
			s.hub.Stop()
		}

		if s.httpServer != nil {
			if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
				s.logger.Error("failed to shutdown HTTP server", "error", err)
				if shutdownErr == nil {
					shutdownErr = fmt.Errorf("shutdown HTTP server: %w", err)
				}
			}
		}

		done := make(chan struct{})
		go func() {
			if s.grpcServer != nil {
				s.grpcServer.GracefulStop()
			}
			close(done)
		}()

		select {
		case <-done:
			s.logger.Info("server stopped gracefully")
		case <-shutdownCtx.Done():
			s.logger.Warn("server shutdown timeout, forcing stop")
			if s.grpcServer != nil {
				s.grpcServer.Stop()
			}
			shutdownErr = fmt.Errorf("shutdown timeout")
		}

		if s.db != nil {
			if err := s.db.Close(); err != nil {
				s.logger.Error("failed to close database", "error", err)
				if shutdownErr == nil {
					shutdownErr = fmt.Errorf("close database: %w", err)
				}
			}
		}

		if s.tracer != nil {
			if err := s.tracer.Shutdown(context.Background()); err != nil {
				s.logger.Error("failed to shutdown tracer", "error", err)
				if shutdownErr == nil {
					shutdownErr = fmt.Errorf("shutdown tracer: %w", err)
				}
			}
		}
	})
	return shutdownErr
}

func (s *Server) livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		s.logger.Warn("failed to write liveness response", "error", err)
	}
}

func (s *Server) readinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if s.db != nil {
		if err := s.db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, writeErr := w.Write([]byte("database not ready")); writeErr != nil {
				s.logger.Warn("failed to write readiness response", "error", writeErr)
			}
			return
		}
	}

	if s.eventConsumer != nil {
		if !s.eventConsumer.IsConnected() {
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, writeErr := w.Write([]byte("rabbitmq not ready")); writeErr != nil {
				s.logger.Warn("failed to write readiness response", "error", writeErr)
			}
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		s.logger.Warn("failed to write readiness response", "error", err)
	}
}

func (s *Server) WaitForSignals() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	s.logger.Info("received shutdown signal", "signal", sig.String())
	return s.Shutdown(context.Background())
}
