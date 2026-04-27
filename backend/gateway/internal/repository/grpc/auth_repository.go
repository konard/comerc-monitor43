package grpc

import (
	"context"
	"fmt"
	"time"

	authv1 "github.com/raul/monitor/api/proto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grpcadapter "github.com/raul/monitor/gateway/internal/adapter/grpc"
	"github.com/raul/monitor/gateway/internal/model"
	"github.com/raul/monitor/gateway/internal/repository/interfaces"
)

// tracer используется для OpenTelemetry трассировки операций с auth сервисом.
var tracer = otel.Tracer("gateway/repository/grpc")

// authRepository implements the AuthRepository interface using gRPC
type authRepository struct {
	pool   *grpcadapter.Pool
	client authv1.AuthServiceClient
}

// NewAuthRepository creates a new gRPC-based auth repository with direct connection
func NewAuthRepository(addr string) (interfaces.AuthRepository, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	conn.Connect()

	return NewAuthRepositoryFromClient(authv1.NewAuthServiceClient(conn)), nil
}

// NewAuthRepositoryFromClient creates a repository with an existing auth client
func NewAuthRepositoryFromClient(client authv1.AuthServiceClient) interfaces.AuthRepository {
	return &authRepository{
		pool:   nil,
		client: client,
	}
}

// NewAuthRepositoryFromPool creates a repository using a connection pool
func NewAuthRepositoryFromPool(pool *grpcadapter.Pool) (interfaces.AuthRepository, error) {
	return &authRepository{
		pool:   pool,
		client: nil,
	}, nil
}

// ValidateToken validates an access token and returns user context
func (r *authRepository) ValidateToken(ctx context.Context, token string) (*model.TokenValidationResult, error) {
	ctx, span := tracer.Start(ctx, "authRepository.ValidateToken")
	defer span.End()

	span.SetAttributes(attribute.String("rpc.system", "grpc"))

	start := time.Now()
	defer func() {
		// замер задержки для логирования в будущем
		latency := time.Since(start).Milliseconds()
		_ = latency
	}()

	var client authv1.AuthServiceClient
	var err error

	if r.pool != nil {
		conn, err := r.pool.Get(ctx)
		if err != nil {
			return model.NewTokenValidationError(model.ErrInvalidToken), nil
		}
		defer r.pool.Put(conn)
		client = authv1.NewAuthServiceClient(conn)
	} else {
		client = r.client
	}

	req := &authv1.ValidateTokenRequest{AccessToken: token}
	resp, err := client.ValidateToken(ctx, req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return model.NewTokenValidationError(model.ErrInvalidToken), nil
	}

	if !resp.Valid {
		return model.NewTokenValidationResult(false, "", "", ""), nil
	}

	return model.NewTokenValidationResult(true, resp.UserId, resp.Email, resp.Tier), nil
}

// CheckHealth checks if the auth service is healthy
func (r *authRepository) CheckHealth(ctx context.Context) (bool, error) {
	ctx, span := tracer.Start(ctx, "authRepository.CheckHealth")
	defer span.End()

	span.SetAttributes(attribute.String("rpc.system", "grpc"))

	start := time.Now()
	defer func() {
		// замер задержки для логирования в будущем
		latency := time.Since(start).Milliseconds()
		_ = latency
	}()

	var client authv1.AuthServiceClient
	var err error

	if r.pool != nil {
		conn, err := r.pool.Get(ctx)
		if err != nil {
			return false, err
		}
		defer r.pool.Put(conn)
		client = authv1.NewAuthServiceClient(conn)
	} else {
		client = r.client
	}

	// Simple health check via a lightweight call
	// In production, use a dedicated health check endpoint
	_, err = client.ValidateToken(ctx, &authv1.ValidateTokenRequest{AccessToken: ""})
	if err != nil {
		return false, err
	}
	return true, nil
}
