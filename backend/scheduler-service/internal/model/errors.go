package model

import "errors"

type SchedulerError struct {
	Code    string
	Message string
}

func (e *SchedulerError) Error() string {
	return e.Code + ": " + e.Message
}

func NewSchedulerError(code, message string) *SchedulerError {
	return &SchedulerError{Code: code, Message: message}
}

var (
	ErrWorkerNotFound      = NewSchedulerError("WORKER_NOT_FOUND", "worker not found")
	ErrWorkerNameExists    = NewSchedulerError("WORKER_NAME_EXISTS", "worker name already exists")
	ErrWorkerNameRequired  = NewSchedulerError("WORKER_NAME_REQUIRED", "worker name is required")
	ErrWorkerNameTooLong   = NewSchedulerError("WORKER_NAME_TOO_LONG", "worker name exceeds 255 characters")
	ErrInvalidZone         = NewSchedulerError("INVALID_ZONE", "zone is invalid")
	ErrInvalidWorkerStatus = NewSchedulerError("INVALID_WORKER_STATUS", "invalid worker status")
	ErrNoWorkersAvailable  = NewSchedulerError("NO_WORKERS_AVAILABLE", "no idle workers available")

	ErrMonitorNotFound     = NewSchedulerError("MONITOR_NOT_FOUND", "monitor not found")
	ErrCheckAlreadyPending = NewSchedulerError("CHECK_ALREADY_PENDING", "check already pending for monitor")
	ErrInvalidTimeRange    = NewSchedulerError("INVALID_TIME_RANGE", "from_time must be before to_time")
	ErrCheckNotFound       = NewSchedulerError("CHECK_NOT_FOUND", "check not found")
)

func IsSchedulerError(err error) bool {
	var schedErr *SchedulerError
	return errors.As(err, &schedErr)
}
