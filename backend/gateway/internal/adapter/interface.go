package adapter

import "context"

// Closer defines the interface for resources that need cleanup
type Closer interface {
	Close(ctx context.Context) error
}

// Starter defines the interface for resources that can be started
type Starter interface {
	Start(ctx context.Context) error
}

// Stoppable defines the interface for resources that can be stopped gracefully
type Stoppable interface {
	Stop(ctx context.Context) error
}

// StartStoppable combines Start and Stop for managed resources
type StartStoppable interface {
	Starter
	Stoppable
}

// HealthCheckable defines the interface for health-checkable resources
type HealthCheckable interface {
	HealthCheck(ctx context.Context) error
}
