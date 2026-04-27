package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/integration-service/internal/infrastructure/auth"
)

// resolveUserID возвращает user_id из request либо из JWT context, если поле пустое.
// Это позволяет коротким REST-роутам (например, GET /api/v1/integrations/webhooks)
// работать с current user без обязательного query-параметра.
func resolveUserID(ctx context.Context, raw string) (uuid.UUID, error) {
	if raw == "" {
		jwtUserID, err := auth.RequireAuth(ctx)
		if err != nil {
			return uuid.Nil, err
		}
		raw = jwtUserID
	}

	userID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, invalidArgument("user_id", "invalid UUID format")
	}

	return userID, nil
}
