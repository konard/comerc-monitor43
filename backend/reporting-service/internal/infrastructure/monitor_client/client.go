package monitor_client

import (
	"context"
	"time"

	"github.com/pkg/errors"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MonitorClient определяет интерфейс для взаимодействия с monitor-service.
type MonitorClient interface {
	GetUptimeStats(ctx context.Context, monitorID string, from, to time.Time) (*monitov1.GetUptimeStatsResponse, error)
	GetCheckResults(ctx context.Context, monitorID string, from, to time.Time, limit, offset int) (*monitov1.GetCheckResultsResponse, error)
	GetIncidents(ctx context.Context, monitorID string, from, to time.Time, limit, offset int) (*monitov1.GetIncidentsResponse, error)
	GetMonitor(ctx context.Context, monitorID string) (*monitov1.GetMonitorResponse, error)
	Close() error
}

type grpcMonitorClient struct {
	conn   *grpc.ClientConn
	client monitov1.MonitorServiceClient
}

func NewMonitorClient(address string) (MonitorClient, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to monitor-service")
	}

	return &grpcMonitorClient{
		conn:   conn,
		client: monitov1.NewMonitorServiceClient(conn),
	}, nil
}

func (c *grpcMonitorClient) Close() error {
	return errors.Wrap(c.conn.Close(), "failed to close monitor-service connection")
}

func (c *grpcMonitorClient) GetUptimeStats(ctx context.Context, monitorID string, from, to time.Time) (*monitov1.GetUptimeStatsResponse, error) {
	resp, err := c.client.GetUptimeStats(ctx, &monitov1.GetUptimeStatsRequest{
		MonitorId: monitorID,
		From:      timestamppb.New(from),
		To:        timestamppb.New(to),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get uptime stats from monitor-service")
	}
	return resp, nil
}

func (c *grpcMonitorClient) GetCheckResults(ctx context.Context, monitorID string, from, to time.Time, limit, offset int) (*monitov1.GetCheckResultsResponse, error) {
	resp, err := c.client.GetCheckResults(ctx, &monitov1.GetCheckResultsRequest{
		MonitorId: monitorID,
		From:      timestamppb.New(from),
		To:        timestamppb.New(to),
		Limit:     int32(limit),  //nolint:gosec // G115: параметры пагинации в допустимом диапазоне
		Offset:    int32(offset), //nolint:gosec // G115: параметры пагинации в допустимом диапазоне
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get check results from monitor-service")
	}
	return resp, nil
}

func (c *grpcMonitorClient) GetIncidents(ctx context.Context, monitorID string, from, to time.Time, limit, offset int) (*monitov1.GetIncidentsResponse, error) {
	resp, err := c.client.GetIncidents(ctx, &monitov1.GetIncidentsRequest{
		MonitorId: monitorID,
		From:      timestamppb.New(from),
		To:        timestamppb.New(to),
		Limit:     int32(limit),  //nolint:gosec // G115: параметры пагинации в допустимом диапазоне
		Offset:    int32(offset), //nolint:gosec // G115: параметры пагинации в допустимом диапазоне
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get incidents from monitor-service")
	}
	return resp, nil
}

func (c *grpcMonitorClient) GetMonitor(ctx context.Context, monitorID string) (*monitov1.GetMonitorResponse, error) {
	resp, err := c.client.GetMonitor(ctx, &monitov1.GetMonitorRequest{
		Id: monitorID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get monitor from monitor-service")
	}
	return resp, nil
}
