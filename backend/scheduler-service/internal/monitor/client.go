package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	monitov1 "github.com/raul/monitor/api/proto/monitor/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// serviceJWTTTL определяет срок жизни service-to-service JWT
const serviceJWTTTL = 24 * time.Hour

// serviceJWTRefreshBefore задаёт окно до истечения, в котором токен пере-выпускается
const serviceJWTRefreshBefore = 1 * time.Hour

// jwtClaims совместим с токенами, которые принимает monitor-service.
type jwtClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Tier   string `json:"tier"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type Client struct {
	address   string
	conn      *grpc.ClientConn
	client    monitov1.MonitorServiceClient
	logger    *slog.Logger
	connected bool

	jwtSecret []byte

	tokenMu     sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

func NewClient(address string, logger *slog.Logger, jwtSecret string) *Client {
	return &Client{
		address:   address,
		logger:    logger,
		jwtSecret: []byte(jwtSecret),
	}
}

func (c *Client) Connect(ctx context.Context) error {
	conn, err := grpc.NewClient(c.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(c.authUnaryInterceptor),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to monitor service: %w", err)
	}
	c.conn = conn
	c.client = monitov1.NewMonitorServiceClient(conn)
	c.connected = true

	c.logger.InfoContext(ctx, "connected to monitor service", "address", c.address)
	return nil
}

func (c *Client) Close() error {
	c.connected = false
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) IsConnected() bool {
	return c.connected
}

// authUnaryInterceptor добавляет service-to-service JWT в метаданные каждого gRPC-запроса.
func (c *Client) authUnaryInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	token, err := c.serviceToken()
	if err != nil {
		return fmt.Errorf("failed to obtain service token: %w", err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	return invoker(ctx, method, req, reply, cc, opts...)
}

// serviceToken возвращает закэшированный JWT или выпускает новый, если истёк или скоро истечёт.
func (c *Client) serviceToken() (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	now := time.Now()
	if c.cachedToken != "" && now.Before(c.tokenExpiry.Add(-serviceJWTRefreshBefore)) {
		return c.cachedToken, nil
	}

	expiry := now.Add(serviceJWTTTL)
	claims := jwtClaims{
		UserID: "scheduler-service",
		Email:  "",
		Tier:   "Enterprise",
		Role:   "SERVICE",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiry),
			Issuer:    "scheduler-service",
			Subject:   "scheduler-service",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(c.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign service token: %w", err)
	}

	c.cachedToken = signed
	c.tokenExpiry = expiry
	return signed, nil
}

func (c *Client) GetMonitor(ctx context.Context, monitorID string) (*monitov1.Monitor, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.client.GetMonitor(ctx, &monitov1.GetMonitorRequest{
		Id: monitorID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get monitor: %w", err)
	}

	return resp.Monitor, nil
}

func (c *Client) ListActiveMonitors(ctx context.Context, pageSize int) ([]*monitov1.Monitor, error) {
	var allMonitors []*monitov1.Monitor
	offset := int32(0)

	for {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		resp, err := c.client.ListMonitors(ctx, &monitov1.ListMonitorsRequest{
			Status: "",
			Limit:  int32(pageSize), //nolint:gosec // G115: значения в допустимом диапазоне
			Offset: offset,
		})
		cancel()

		if err != nil {
			return nil, fmt.Errorf("failed to list monitors (offset %d): %w", offset, err)
		}

		allMonitors = append(allMonitors, resp.Monitors...)

		if int32(len(allMonitors)) >= resp.Total || len(resp.Monitors) < pageSize { //nolint:gosec // G115: значения в допустимом диапазоне
			break
		}
		offset += int32(pageSize) //nolint:gosec // G115: значения в допустимом диапазоне
	}

	return allMonitors, nil
}

func (c *Client) IsMonitorPaused(ctx context.Context, monitorID string) (bool, error) {
	mon, err := c.GetMonitor(ctx, monitorID)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return false, fmt.Errorf("monitor not found: %s", monitorID)
		}
		return false, err
	}
	return mon.Status == "PAUSED", nil
}

func (c *Client) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return true
	}

	switch st.Code() {
	case codes.OK:
		return false
	case codes.Canceled, codes.DeadlineExceeded, codes.Unavailable, codes.ResourceExhausted:
		return true
	case codes.NotFound, codes.InvalidArgument, codes.FailedPrecondition:
		return false
	default:
		return true
	}
}
