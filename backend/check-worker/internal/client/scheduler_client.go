package client

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	api "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/check-worker/internal/config"
	"github.com/raul/monitor/backend/check-worker/internal/retry"
)

type SchedulerClient struct {
	address   string
	conn      *grpc.ClientConn
	client    api.SchedulerServiceClient
	workerID  string
	logger    *slog.Logger
	connected bool

	// Retry config
	retryConfig retry.Config
}

func safeInt32(v int, field string) (int32, error) {
	if v < math.MinInt32 || v > math.MaxInt32 {
		return 0, fmt.Errorf("%s out of int32 range: %d", field, v)
	}
	return int32(v), nil
}

func NewSchedulerClient(address string, logger *slog.Logger, cfg *config.Config) *SchedulerClient {
	retryCfg := retry.DefaultConfig()
	if cfg != nil {
		retryCfg = retry.Config{
			MaxAttempts: cfg.RetryMaxAttempts,
			BaseDelay:   cfg.RetryBaseDelay,
			MaxDelay:    cfg.RetryMaxDelay,
			Multiplier:  2.0,
			Jitter:      true,
		}
	}

	return &SchedulerClient{
		address:     address,
		logger:      logger,
		retryConfig: retryCfg,
	}
}

func (c *SchedulerClient) Connect(ctx context.Context) error {
	// Используем retry для соединения
	err := retry.DoWithRetry(ctx, c.retryConfig, func(ctx context.Context) error {
		conn, err := grpc.NewClient(c.address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return err
		}
		c.conn = conn
		c.client = api.NewSchedulerServiceClient(conn)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to connect to scheduler after retries: %w", err)
	}

	c.connected = true
	c.logger.InfoContext(ctx, "connected to scheduler service", "address", c.address)
	return nil
}

func (c *SchedulerClient) Close() error {
	c.connected = false
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *SchedulerClient) IsConnected() bool {
	return c.connected
}

func (c *SchedulerClient) RegisterWorker(ctx context.Context, name, zone string, metadata map[string]string) (string, error) {
	var resp *api.Worker

	err := retry.DoWithRetryAndIsRetryable(ctx, c.retryConfig, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var err error
		resp, err = c.client.RegisterWorker(ctx, &api.RegisterWorkerRequest{
			Name:     name,
			Zone:     zone,
			Metadata: metadata,
		})
		return err
	}, c.isRetryableGRPCError)

	if err != nil {
		return "", fmt.Errorf("failed to register worker after retries: %w", err)
	}

	c.workerID = resp.Id
	c.logger.InfoContext(ctx, "worker registered", "worker_id", resp.Id, "name", name)
	return resp.Id, nil
}

func (c *SchedulerClient) SendHeartbeat(ctx context.Context, heartbeatStatus string, checksCompleted, checksFailed int, avgDurationMs float64) error {
	completedCount, err := safeInt32(checksCompleted, "checks_completed")
	if err != nil {
		return err
	}
	failedCount, err := safeInt32(checksFailed, "checks_failed")
	if err != nil {
		return err
	}

	err = retry.DoWithRetryAndIsRetryable(ctx, c.retryConfig, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		_, err := c.client.WorkerHeartbeat(ctx, &api.HeartbeatRequest{
			WorkerId:           c.workerID,
			Status:             heartbeatStatus,
			ChecksCompleted:    completedCount,
			ChecksFailed:       failedCount,
			AvgCheckDurationMs: avgDurationMs,
		})
		return err
	}, c.isRetryableGRPCError)

	if err != nil {
		return fmt.Errorf("failed to send heartbeat after retries: %w", err)
	}
	return nil
}

func (c *SchedulerClient) UnregisterWorker(ctx context.Context) error {
	if c.workerID == "" {
		return nil
	}

	err := retry.DoWithRetryAndIsRetryable(ctx, c.retryConfig, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		_, err := c.client.UnregisterWorker(ctx, &api.UnregisterWorkerRequest{
			WorkerId: c.workerID,
		})
		return err
	}, c.isRetryableGRPCError)

	if err != nil {
		return fmt.Errorf("failed to unregister worker after retries: %w", err)
	}

	c.logger.InfoContext(ctx, "worker unregistered", "worker_id", c.workerID)
	return nil
}

func (c *SchedulerClient) GetScheduledChecks(ctx context.Context) ([]*api.ScheduledCheck, error) {
	var resp *api.ScheduledChecks

	err := retry.DoWithRetryAndIsRetryable(ctx, c.retryConfig, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var err error
		// Запрашиваем все pending-проверки, запланированные в прошлом вплоть до текущего момента
		resp, err = c.client.GetScheduledChecks(ctx, &api.GetScheduledChecksRequest{
			Status:   "pending",
			FromTime: timestamppb.New(time.Unix(0, 0)),
			ToTime:   timestamppb.New(time.Now()),
		})
		return err
	}, c.isRetryableGRPCError)

	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled checks after retries: %w", err)
	}

	return resp.Checks, nil
}

func (c *SchedulerClient) GetWorkerID() string {
	return c.workerID
}

// isRetryableGRPCError определяет, можно ли повторить gRPC ошибку
func (c *SchedulerClient) isRetryableGRPCError(err error) bool {
	if err == nil {
		return false
	}

	// Проверяем gRPC status code
	st, ok := status.FromError(err)
	if !ok {
		// Не gRPC ошибка, пробуем retry
		return true
	}

	switch st.Code() {
	case codes.OK:
		return false
	case codes.Canceled, codes.DeadlineExceeded:
		return true
	case codes.Unavailable:
		// Сервис временно недоступен - retry
		return true
	case codes.ResourceExhausted:
		// Too many requests - retry with backoff
		return true
	case codes.Aborted, codes.AlreadyExists:
		// Transient failures - retry
		return true
	case codes.NotFound, codes.FailedPrecondition, codes.InvalidArgument:
		// Эти ошибки не исправляются повторением
		return false
	default:
		// Для остальных ошибок пробуем retry
		return true
	}
}
