package health

import (
	"context"
	"fmt"
	"time"
)

// CheckFunc represents a health check function
type CheckFunc func(ctx context.Context) error

// Check represents a health check for a dependency (stateless)
type Check struct {
	Name      string
	Component string
	CheckFunc CheckFunc
	Timeout   time.Duration
}

// Result represents the result of a health check
type Result struct {
	Healthy   bool      `json:"healthy"`
	Message   string    `json:"message,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	LatencyMS int64     `json:"latency_ms"`
}

// Checker manages health checks for all dependencies (stateless)
type Checker struct {
	checks    map[string]Check
	timeout   time.Duration
	startTime time.Time
}

// NewChecker creates a new stateless health checker
func NewChecker(timeout time.Duration) *Checker {
	return &Checker{
		checks:    make(map[string]Check),
		timeout:   timeout,
		startTime: time.Now(),
	}
}

// Register registers a new health check
func (c *Checker) Register(name, component string, checkFunc CheckFunc) {
	c.checks[name] = Check{
		Name:      name,
		Component: component,
		CheckFunc: checkFunc,
		Timeout:   c.timeout,
	}
}

// Unregister removes a health check
func (c *Checker) Unregister(name string) {
	delete(c.checks, name)
}

// Check performs a health check for a specific dependency
func (c *Checker) Check(ctx context.Context, name string) (*Result, error) {
	check, ok := c.checks[name]
	if !ok {
		return nil, fmt.Errorf("health check not found: %s", name)
	}

	start := time.Now()

	// Create timeout context if needed
	checkCtx := ctx
	if check.Timeout > 0 {
		var cancel context.CancelFunc
		checkCtx, cancel = context.WithTimeout(ctx, check.Timeout)
		defer cancel()
	}

	err := check.CheckFunc(checkCtx)
	latency := time.Since(start)

	result := Result{
		Healthy:   err == nil,
		Timestamp: start,
		LatencyMS: latency.Milliseconds(),
	}

	if err != nil {
		result.Error = err.Error()
		result.Message = fmt.Sprintf("%s check failed", check.Name)
	} else {
		result.Message = fmt.Sprintf("%s is healthy", check.Name)
	}

	return &result, nil
}

// CheckAll performs health checks for all registered dependencies
func (c *Checker) CheckAll(ctx context.Context) map[string]*Result {
	results := make(map[string]*Result)

	for name := range c.checks {
		result, err := c.Check(ctx, name)
		if err != nil {
			results[name] = &Result{
				Healthy:   false,
				Message:   "Check failed",
				Error:     err.Error(),
				Timestamp: time.Now(),
				LatencyMS: 0,
			}
		} else {
			results[name] = result
		}
	}

	return results
}

// GetStatus returns the overall health status (performs real checks)
func (c *Checker) GetStatus() string {
	results := c.CheckAll(context.Background())

	for _, result := range results {
		if !result.Healthy {
			return "unhealthy"
		}
	}

	return "healthy"
}

// IsHealthy checks if all dependencies are healthy
func (c *Checker) IsHealthy(ctx context.Context) bool {
	results := c.CheckAll(ctx)

	for _, result := range results {
		if !result.Healthy {
			return false
		}
	}

	return true
}

// GetCheck returns a registered check by name
func (c *Checker) GetCheck(name string) (Check, bool) {
	check, ok := c.checks[name]
	return check, ok
}

// ListChecks returns all registered check names
func (c *Checker) ListChecks() []string {
	names := make([]string, 0, len(c.checks))
	for name := range c.checks {
		names = append(names, name)
	}
	return names
}

// GetUptime returns the uptime of the checker
func (c *Checker) GetUptime() time.Duration {
	return time.Since(c.startTime)
}
