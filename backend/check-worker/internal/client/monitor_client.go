package client

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	api "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/check-worker/internal/config"
	"github.com/raul/monitor/backend/check-worker/internal/queue"
	"github.com/raul/monitor/backend/check-worker/internal/retry"
)

type MonitorClient struct {
	address   string
	conn      *grpc.ClientConn
	client    api.MonitorServiceClient
	logger    *slog.Logger
	connected bool

	// Queue для результатов при недоступности monitor-service
	resultQueue     *queue.ResultQueue
	queueFlushMutex sync.Mutex

	// Retry config
	retryConfig retry.Config

	// Background flush control
	flushCtx      context.Context
	flushCancel   context.CancelFunc
	flushWg       sync.WaitGroup
	flushInterval time.Duration
}

func NewMonitorClient(address string, logger *slog.Logger, cfg *config.Config) *MonitorClient {
	maxSize := 1000
	ttl := 24 * time.Hour
	flushInterval := 30 * time.Second

	if cfg != nil {
		maxSize = cfg.ResultQueueMaxSize
		ttl = cfg.ResultQueueTTL
		flushInterval = cfg.QueueFlushInterval
	}

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

	return &MonitorClient{
		address:       address,
		logger:        logger,
		retryConfig:   retryCfg,
		resultQueue:   queue.NewResultQueue(maxSize, ttl),
		flushInterval: flushInterval,
	}
}

func (c *MonitorClient) Connect(ctx context.Context) error {
	// Используем retry для соединения
	err := retry.DoWithRetry(ctx, c.retryConfig, func(ctx context.Context) error {
		conn, err := grpc.NewClient(c.address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return err
		}
		c.conn = conn
		c.client = api.NewMonitorServiceClient(conn)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to connect to monitor service after retries: %w", err)
	}

	c.connected = true
	c.logger.InfoContext(ctx, "connected to monitor service", "address", c.address)

	// Запускаем background flush для очереди
	c.startFlushLoop(ctx)

	return nil
}

func (c *MonitorClient) Close() error {
	c.connected = false

	// Останавливаем flush loop
	if c.flushCancel != nil {
		c.flushCancel()
		c.flushWg.Wait()
	}

	// Финальный flush перед закрытием
	c.flushQueue(context.Background())

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *MonitorClient) IsConnected() bool {
	return c.connected
}

func (c *MonitorClient) SubmitCheckResult(ctx context.Context, monitorID string, result *api.CheckResult) error {
	// Сначала пробуем отправить напрямую
	err := retry.DoWithRetryAndIsRetryable(ctx, c.retryConfig, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// TODO: Заменить на правильный RPC когда будет добавлен SubmitCheckResult в proto
		// Сейчас используем TriggerCheck как заглушку
		_, err := c.client.TriggerCheck(ctx, &api.TriggerCheckRequest{
			Id: monitorID,
		})
		return err
	}, c.isRetryableGRPCError)

	if err != nil {
		// Если не удалось отправить, добавляем в очередь
		c.logger.WarnContext(ctx, "failed to submit check result, queuing",
			"monitor_id", monitorID,
			"error", err,
			"queue_size", c.resultQueue.Size(),
		)

		queued := c.resultQueue.Enqueue(result, monitorID)
		if !queued {
			c.logger.ErrorContext(ctx, "queue overflow, result dropped",
				"monitor_id", monitorID,
				"dropped_count", c.resultQueue.DroppedCount(),
			)
		}
		return fmt.Errorf("failed to submit check result (queued): %w", err)
	}

	// Успешная отправка, пробуем flush очереди
	go c.flushQueue(ctx)

	return nil
}

func (c *MonitorClient) GetMonitor(ctx context.Context, monitorID string) (*api.Monitor, error) {
	var resp *api.Monitor

	err := retry.DoWithRetryAndIsRetryable(ctx, c.retryConfig, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var err error
		resp, err = c.client.GetMonitor(ctx, &api.GetMonitorRequest{
			Id: monitorID,
		})
		return err
	}, c.isRetryableGRPCError)

	if err != nil {
		return nil, fmt.Errorf("failed to get monitor after retries: %w", err)
	}

	return resp, nil
}

// flushQueue отправляет все результаты из очереди
func (c *MonitorClient) flushQueue(ctx context.Context) {
	c.queueFlushMutex.Lock()
	defer c.queueFlushMutex.Unlock()

	stats := c.resultQueue.Stats()
	if stats.Size == 0 {
		return
	}

	c.logger.DebugContext(ctx, "flushing result queue",
		"queue_size", stats.Size,
		"dropped", stats.Dropped,
	)

	// Забираем батчи по 10 результатов
	batchSize := 10
	flushed := 0

	for {
		batch := c.resultQueue.DequeueBatch(batchSize)
		if len(batch) == 0 {
			break
		}

		// Отправляем батч
		for _, item := range batch {
			err := retry.DoWithRetryAndIsRetryable(ctx, c.retryConfig, func(ctx context.Context) error {
				ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				_, err := c.client.TriggerCheck(ctx, &api.TriggerCheckRequest{
					Id: item.MonitorID,
				})
				return err
			}, c.isRetryableGRPCError)

			if err != nil {
				// Если не удалось отправить, возвращаем в очередь
				c.logger.WarnContext(ctx, "failed to flush queued result, re-queuing",
					"monitor_id", item.MonitorID,
					"error", err,
				)
				c.resultQueue.Enqueue(item.Result, item.MonitorID)
			} else {
				flushed++
			}
		}

		// Если отправили меньше batchSize, значит опустели
		if len(batch) < batchSize {
			break
		}
	}

	if flushed > 0 {
		c.logger.InfoContext(ctx, "flushed results from queue",
			"flushed_count", flushed,
			"remaining_size", c.resultQueue.Size(),
		)
	}
}

// startFlushLoop запускает background goroutine для периодического flush очереди
func (c *MonitorClient) startFlushLoop(ctx context.Context) {
	c.flushCtx, c.flushCancel = context.WithCancel(ctx)

	c.flushWg.Add(1)
	go func() {
		defer c.flushWg.Done()

		ticker := time.NewTicker(c.flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-c.flushCtx.Done():
				return
			case <-ticker.C:
				c.flushQueue(c.flushCtx)
			}
		}
	}()

	c.logger.InfoContext(ctx, "started queue flush loop", "interval", c.flushInterval)
}

// isRetryableGRPCError определяет, можно ли повторить gRPC ошибку
func (c *MonitorClient) isRetryableGRPCError(err error) bool {
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

func BuildCheckResultProto(success bool, statusCode int32, responseTimeMs float64, errorMessage string) *api.CheckResult {
	return &api.CheckResult{
		Success:        success,
		StatusCode:     statusCode,
		ResponseTimeMs: responseTimeMs,
		ErrorMessage:   errorMessage,
		CheckedAt:      timestamppb.Now(),
	}
}

// SubmitCheckResults отправляет пакет результатов (для пакетной отправки)
func (c *MonitorClient) SubmitCheckResults(ctx context.Context, results map[string]*api.CheckResult) error {
	if len(results) == 0 {
		return nil
	}

	// TODO: Реализовать batch API когда будет добавлен в proto
	// Пока отправляем по одному
	for checkID := range results {
		// Заглушка - просто логируем
		c.logger.DebugContext(ctx, "would submit result in batch",
			"check_id", checkID,
			"batch_size", len(results),
		)
	}

	return nil
}
