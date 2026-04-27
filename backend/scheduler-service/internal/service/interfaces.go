package service

import (
	"github.com/raul/monitor/backend/scheduler-service/internal/repository/interfaces"
)

// Aliases для репозиториев из package interfaces
type WorkerRepository = interfaces.WorkerRepository
type CheckRepository = interfaces.ScheduledCheckRepository
type AuditRepository = interfaces.AuditRepository
type DistributedLocker = interfaces.DistributedLocker
