package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

type Config struct {
	FailureThreshold int           // ошибок до Open (default: 5)
	Timeout          time.Duration // время в Open до HalfOpen (default: 30s)
	SuccessThreshold int           // успехов в HalfOpen до Closed (default: 1)
}

func (c Config) withDefaults() Config {
	if c.FailureThreshold <= 0 {
		c.FailureThreshold = 5
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	if c.SuccessThreshold <= 0 {
		c.SuccessThreshold = 1
	}
	return c
}

type CircuitBreaker struct {
	cfg Config

	mu          sync.Mutex
	state       State
	failures    int
	successes   int
	lastFailure time.Time
}

func New(cfg Config) *CircuitBreaker {
	return &CircuitBreaker{
		cfg:   cfg.withDefaults(),
		state: StateClosed,
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.refreshState()
	return cb.state
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	err := fn(ctx)
	cb.recordResult(err)
	return err
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.refreshState()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		return false
	case StateHalfOpen:
		return true
	}
	return false
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		if err != nil {
			cb.failures++
			if cb.failures >= cb.cfg.FailureThreshold {
				cb.transition(StateOpen)
			}
		} else {
			cb.failures = 0
		}

	case StateOpen:
		if err != nil {
			cb.lastFailure = time.Now()
		}

	case StateHalfOpen:
		if err != nil {
			cb.transition(StateOpen)
		} else {
			cb.successes++
			if cb.successes >= cb.cfg.SuccessThreshold {
				cb.transition(StateClosed)
			}
		}
	}
}

func (cb *CircuitBreaker) transition(to State) {
	cb.state = to

	switch to {
	case StateOpen:
		cb.lastFailure = time.Now()
	case StateClosed:
		cb.failures = 0
		cb.successes = 0
	case StateHalfOpen:
		cb.successes = 0
	}
}

func (cb *CircuitBreaker) refreshState() {
	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) >= cb.cfg.Timeout {
			cb.transition(StateHalfOpen)
		}
	}
}

var ErrCircuitOpen = errors.New("circuit breaker is open")

func IsErrCircuitOpen(err error) bool {
	return errors.Is(err, ErrCircuitOpen)
}

func FormatErrCircuitOpen(service string) error {
	return fmt.Errorf("%s: %w", service, ErrCircuitOpen)
}
