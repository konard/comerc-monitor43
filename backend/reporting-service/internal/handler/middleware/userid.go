package middleware

import (
	"context"

	"github.com/raul/monitor/backend/reporting-service/internal/infrastructure/auth"
)

// ExtractUserID извлекает идентификатор пользователя из контекста,
// установленного JWT интерцептором аутентификации.
func ExtractUserID(ctx context.Context) (string, error) {
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return "", err
	}
	return userID.String(), nil
}
