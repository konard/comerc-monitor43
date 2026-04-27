// Package auth предоставляет клиент для взаимодействия с auth-service.
//
// Пакет реализует gRPC клиент для валидации JWT токенов через auth-service:
//   - ValidateToken: валидация JWT токена и извлечение user_id
//
// Клиент использует gRPC соединение с auth-service для проверки токенов.
package auth

import (
	"context"

	"github.com/pkg/errors"
	authapi "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// AuthClient предоставляет методы для работы с auth-service.
type AuthClient struct {
	conn   *grpc.ClientConn
	client authapi.AuthServiceClient
}

// NewAuthClient создаёт новый AuthClient.
// Устанавливает gRPC соединение с auth-service.
func NewAuthClient(address string) (*AuthClient, error) {
	// Создаём gRPC соединение
	conn, err := grpc.NewClient(address, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials())) //nolint:staticcheck // SA1019: WithBlock — совместимость с текущей версией адаптера
	if err != nil {
		return nil, errors.Wrap(err, "failed to create auth service connection")
	}

	client := authapi.NewAuthServiceClient(conn)

	return &AuthClient{
		conn:   conn,
		client: client,
	}, nil
}

// NewAuthClientWithConn создаёт новый AuthClient с готовым соединением.
// Используется для тестирования или при переиспользовании соединений.
func NewAuthClientWithConn(conn *grpc.ClientConn) *AuthClient {
	client := authapi.NewAuthServiceClient(conn)
	return &AuthClient{
		conn:   conn,
		client: client,
	}
}

// ValidateToken валидирует JWT токен и возвращает user_id.
//
// Если токен невалиден или истёк, возвращается ошибка с кодом Unauthenticated.
func (c *AuthClient) ValidateToken(ctx context.Context, token string) (string, error) {
	userID, _, err := c.ValidateTokenWithTier(ctx, token)
	if err != nil {
		return "", errors.Wrap(err, "failed to validate token")
	}
	return userID, nil
}

// ValidateTokenWithTier валидирует JWT токен и возвращает user_id и subscription tier.
//
// Если токен невалиден или истёк, возвращается ошибка с кодом Unauthenticated.
func (c *AuthClient) ValidateTokenWithTier(ctx context.Context, token string) (string, string, error) {
	// Вызываем auth-service для валидации токена
	req := &authapi.ValidateTokenRequest{
		AccessToken: token,
	}

	resp, err := c.client.ValidateToken(ctx, req)
	if err != nil {
		// Проверяем статус ошибки
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.Unauthenticated {
				return "", "", errors.Wrap(st.Err(), "token validation failed")
			}
			return "", "", errors.Wrap(st.Err(), "auth service error")
		}
		return "", "", errors.Wrap(err, "failed to validate token")
	}

	// Проверяем валидность токена
	if !resp.Valid {
		return "", "", errors.New("invalid token")
	}

	// Возвращаем user_id
	if resp.UserId == "" {
		return "", "", errors.New("user_id is empty in token")
	}

	// Возвращаем tier (по умолчанию "Free" если пусто)
	tier := resp.Tier
	if tier == "" {
		tier = "Free"
	}

	return resp.UserId, tier, nil
}

// Close закрывает соединение с auth-service.
func (c *AuthClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
