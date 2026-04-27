package dto

import (
	"time"

	"github.com/google/uuid"
)

// RegisterWorkerRequest - DTO для регистрации воркера.
type RegisterWorkerRequest struct {
	Name     string
	Zone     string
	Metadata map[string]string
}

// WorkerHeartbeatRequest - DTO для heartbeat воркера.
type WorkerHeartbeatRequest struct {
	WorkerID           uuid.UUID
	Status             string
	ChecksCompleted    int
	ChecksFailed       int
	AvgCheckDurationMs float64
}

// UnregisterWorkerRequest - DTO для отключения воркера.
type UnregisterWorkerRequest struct {
	WorkerID uuid.UUID
}

// GetWorkerStatusRequest - DTO для получения статуса воркера.
type GetWorkerStatusRequest struct {
	WorkerID uuid.UUID
}

// ListWorkersRequest - DTO для списка воркеров.
type ListWorkersRequest struct {
	Status   string
	Zone     string
	Page     int
	PageSize int
}

// WorkerResponse - DTO ответа с информацией о воркере.
type WorkerResponse struct {
	ID                 uuid.UUID
	Name               string
	Status             string
	Zone               string
	LastHeartbeat      time.Time
	ChecksCompleted    int
	ChecksFailed       int
	AvgCheckDurationMs float64
	Metadata           map[string]string
}

// WorkersListResponse - DTO ответа со списком воркеров.
type WorkersListResponse struct {
	Workers  []*WorkerResponse
	Total    int
	Page     int
	PageSize int
}
