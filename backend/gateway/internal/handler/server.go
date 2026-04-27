package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	proto "github.com/raul/monitor/api/proto"
	monitorv1 "github.com/raul/monitor/api/proto/monitor/v1"
	reporting "github.com/raul/monitor/api/proto/reporting"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Server represents the HTTP server.
type Server struct {
	httpServer        *http.Server
	authHandler       *AuthHandler
	gatewayHandler    *GatewayHandler
	authMiddleware    *AuthMiddleware
	loggingMiddleware *LoggingMiddleware
}

// NewServer создаёт HTTP-сервер с grpc-gateway мультиплексором для всех backend сервисов.
func NewServer(
	addr string,
	authService AuthService,
	gatewayService GatewayService,
	grpcAddresses map[string]string,
	dashboardWSAddr string,
	allowedOrigins []string,
	readTimeout, writeTimeout, idleTimeout time.Duration,
	logger *log.Logger,
) (*Server, error) {
	authHandler := NewAuthHandler(authService)
	gatewayHandler := NewGatewayHandler(gatewayService)
	authMiddleware := NewAuthMiddleware(authService)
	loggingMiddleware := NewLoggingMiddleware(logger)

	gwMux, err := newGrpcGatewayMux(grpcAddresses)
	if err != nil {
		return nil, fmt.Errorf("failed to setup grpc-gateway: %w", err)
	}

	mux := http.NewServeMux()

	// Технические эндпоинты (без аутентификации).
	mux.HandleFunc("/health", gatewayHandler.Health)
	mux.HandleFunc("/ready", gatewayHandler.Readiness)
	mux.HandleFunc("/live", gatewayHandler.Liveness)

	// Валидация токена (внутренний эндпоинт для nginx auth_request).
	mux.HandleFunc("/api/v1/gateway/validate", authHandler.ValidateToken)

	// WebSocket прокси к dashboard-service (отдельный HTTP-сервер).
	if dashboardWSAddr != "" {
		mux.Handle("/ws", newWSReverseProxy(dashboardWSAddr))
	}

	// Все REST маршруты проксируются через grpc-gateway.
	mux.Handle("/api/v1/", gwMux)

	// Порядок middleware: CORS → Logging → Auth → mux.
	httpHandler := corsMiddleware(allowedOrigins)(loggingMiddleware.Middleware(authMiddleware.Middleware(mux)))

	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      httpHandler,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		authHandler:       authHandler,
		gatewayHandler:    gatewayHandler,
		authMiddleware:    authMiddleware,
		loggingMiddleware: loggingMiddleware,
	}, nil
}

// newGrpcGatewayMux создаёт и регистрирует все backend сервисы в grpc-gateway мультиплексоре.
func newGrpcGatewayMux(addrs map[string]string) (*runtime.ServeMux, error) {
	// Пробрасываем Authorization хедер как gRPC metadata во все downstream сервисы.
	mux := runtime.NewServeMux(
		runtime.WithMetadata(func(_ context.Context, r *http.Request) metadata.MD {
			md := metadata.MD{}
			if auth := r.Header.Get("Authorization"); auth != "" {
				md.Set("authorization", auth)
			}
			if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
				md.Set("x-trace-id", traceID)
			}
			if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
				md.Set("x-request-id", requestID)
			}
			return md
		}),
	)

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	ctx := context.Background()

	type registration struct {
		name     string
		register func() error
	}

	regs := []registration{
		{
			name: "auth",
			register: func() error {
				addr := addrs["auth"]
				if addr == "" {
					return nil
				}
				return proto.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "monitor",
			register: func() error {
				addr := addrs["monitor"]
				if addr == "" {
					return nil
				}
				return monitorv1.RegisterMonitorServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "alert",
			register: func() error {
				addr := addrs["alert"]
				if addr == "" {
					return nil
				}
				return proto.RegisterAlertServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "billing",
			register: func() error {
				addr := addrs["billing"]
				if addr == "" {
					return nil
				}
				return proto.RegisterBillingServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "dashboard",
			register: func() error {
				addr := addrs["dashboard"]
				if addr == "" {
					return nil
				}
				return proto.RegisterDashboardServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "scheduler",
			register: func() error {
				addr := addrs["scheduler"]
				if addr == "" {
					return nil
				}
				return proto.RegisterSchedulerServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "reporting",
			register: func() error {
				addr := addrs["reporting"]
				if addr == "" {
					return nil
				}
				return reporting.RegisterReportingServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "maintenance",
			register: func() error {
				addr := addrs["maintenance"]
				if addr == "" {
					return nil
				}
				return proto.RegisterMaintenanceWindowServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "templates",
			register: func() error {
				addr := addrs["templates"]
				if addr == "" {
					return nil
				}
				return proto.RegisterNotificationTemplateServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "integrations-webhooks",
			register: func() error {
				addr := addrs["integrations"]
				if addr == "" {
					return nil
				}
				return proto.RegisterWebhookIntegrationServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "integrations-apikeys",
			register: func() error {
				addr := addrs["integrations"]
				if addr == "" {
					return nil
				}
				return proto.RegisterAPIKeyServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
		{
			name: "integrations-imports",
			register: func() error {
				addr := addrs["integrations"]
				if addr == "" {
					return nil
				}
				return proto.RegisterImportServiceHandlerFromEndpoint(ctx, mux, addr, opts)
			},
		},
	}

	for _, r := range regs {
		if err := r.register(); err != nil {
			return nil, fmt.Errorf("failed to register %s service handler: %w", r.name, err)
		}
	}

	return mux, nil
}

// newWSReverseProxy создаёт reverse proxy для проксирования WebSocket соединений к dashboard-service.
func newWSReverseProxy(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "invalid websocket upstream", http.StatusBadGateway)
		})
	}
	proxy := httputil.NewSingleHostReverseProxy(u)

	// Стандартный Director переписывает Host — для WS это нужно сохранить.
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = "/ws"
		// Передаём оригинальный Host чтобы upstream мог идентифицировать запрос.
		req.Host = u.Host
	}

	return proxy
}

// Start запускает HTTP-сервер.
func (s *Server) Start() error {
	log.Printf("Starting API Gateway on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown выполняет graceful shutdown сервера.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API Gateway...")
	return s.httpServer.Shutdown(ctx)
}

// GetHTTPServer возвращает underlying http.Server.
func (s *Server) GetHTTPServer() *http.Server {
	return s.httpServer
}
