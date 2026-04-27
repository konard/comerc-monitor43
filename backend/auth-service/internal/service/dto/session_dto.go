package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/raul/monitor/backend/auth-service/internal/model"
)

// ListSessionsRequest представляет запрос на получение списка сессий.
type ListSessionsRequest struct {
	UserID           uuid.UUID
	Limit            int
	Offset           int
	CurrentSessionID string // ID текущей сессии для пометки в ответе
}

// ListSessionsResponse представляет ответ со списком сессий.
type ListSessionsResponse struct {
	Sessions []*SessionDTO
	Total    int
}

// RevokeSessionRequest представляет запрос на отзыв сессии.
type RevokeSessionRequest struct {
	SessionID string
	UserID    uuid.UUID // ID пользователя для проверки владения сессией
}

// SessionDTO represents a session data transfer object
type SessionDTO struct {
	ID           string
	UserID       uuid.UUID
	DeviceInfo   string
	IPAddress    string
	CreatedAt    time.Time
	LastActiveAt time.Time
	IsCurrent    bool
}

// ToSessionDTO converts a Session model to SessionDTO
func ToSessionDTO(session *model.Session, isCurrent bool) *SessionDTO {
	return &SessionDTO{
		ID:           session.ID,
		UserID:       session.UserID,
		DeviceInfo:   session.DeviceInfo,
		IPAddress:    session.IPAddress,
		CreatedAt:    session.CreatedAt,
		LastActiveAt: session.LastActiveAt,
		IsCurrent:    isCurrent,
	}
}

// ToSessionDTOs converts a slice of Session models to SessionDTOs
func ToSessionDTOs(sessions []*model.Session, currentSessionID string) []*SessionDTO {
	result := make([]*SessionDTO, len(sessions))
	for i, s := range sessions {
		isCurrent := s.ID == currentSessionID
		result[i] = ToSessionDTO(s, isCurrent)
	}
	return result
}
