package handler

import (
	grpcstd "github.com/pure-golang/adapters/grpc/std"
	reportingv1 "github.com/raul/monitor/api/proto/reporting"
	"google.golang.org/grpc"

	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/auth"
)

type Server struct {
	grpcServer *grpcstd.Server
}

func NewServer(
	port int,
	reportingHandler *ReportingHandler,
	authMiddleware *auth.AuthMiddleware,
) *Server {
	grpcServer := grpcstd.New(
		grpcstd.Config{
			Host:          "",
			Port:          port,
			EnableReflect: true,
		},
		func(s *grpc.Server) {
			reportingv1.RegisterReportingServiceServer(s, reportingHandler)
		},
		grpcstd.WithUnaryInterceptor(authMiddleware.UnaryInterceptor()),
	)

	return &Server{
		grpcServer: grpcServer,
	}
}

func (s *Server) Start() error {
	return s.grpcServer.Start()
}

func (s *Server) Close() error {
	return s.grpcServer.Close()
}
