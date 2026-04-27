package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewChecker(t *testing.T) {
	checker := NewChecker(5 * time.Second)

	if checker.startTime.IsZero() {
		t.Error("expected startTime to be set")
	}

	if checker.timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", checker.timeout)
	}
}

func TestRegister(t *testing.T) {
	checker := NewChecker(5 * time.Second)

	checker.Register("test-check", "test-component", func(ctx context.Context) error {
		return nil
	})

	// Verify check was registered
	_, ok := checker.GetCheck("test-check")
	if !ok {
		t.Error("expected check to be registered")
	}
}

func TestCheck(t *testing.T) {
	t.Run("healthy check", func(t *testing.T) {
		checker := NewChecker(5 * time.Second)
		checker.Register("healthy-check", "test", func(ctx context.Context) error {
			return nil
		})

		result, err := checker.Check(context.Background(), "healthy-check")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !result.Healthy {
			t.Error("expected check to be healthy")
		}
	})

	t.Run("unhealthy check", func(t *testing.T) {
		checker := NewChecker(5 * time.Second)
		checker.Register("unhealthy-check", "test", func(ctx context.Context) error {
			return errors.New("check failed")
		})

		result, err := checker.Check(context.Background(), "unhealthy-check")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result.Healthy {
			t.Error("expected check to be unhealthy")
		}

		if result.Error == "" {
			t.Error("expected error message")
		}
	})

	t.Run("check not found", func(t *testing.T) {
		checker := NewChecker(5 * time.Second)

		_, err := checker.Check(context.Background(), "non-existent-check")

		if err == nil {
			t.Error("expected error for non-existent check")
		}
	})
}

func TestCheckAll(t *testing.T) {
	checker := NewChecker(5 * time.Second)

	// Register multiple checks
	checker.Register("check1", "component1", func(ctx context.Context) error {
		return nil
	})
	checker.Register("check2", "component2", func(ctx context.Context) error {
		return errors.New("check failed")
	})
	checker.Register("check3", "component3", func(ctx context.Context) error {
		return nil
	})

	results := checker.CheckAll(context.Background())

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Verify results
	if results["check1"].Healthy != true {
		t.Error("expected check1 to be healthy")
	}
	if results["check2"].Healthy != false {
		t.Error("expected check2 to be unhealthy")
	}
	if results["check3"].Healthy != true {
		t.Error("expected check3 to be healthy")
	}
}

func TestGetStatus(t *testing.T) {
	t.Run("all healthy", func(t *testing.T) {
		checker := NewChecker(5 * time.Second)
		checker.Register("check1", "test", func(ctx context.Context) error {
			return nil
		})
		checker.Register("check2", "test", func(ctx context.Context) error {
			return nil
		})

		status := checker.GetStatus()
		if status != "healthy" {
			t.Errorf("expected status 'healthy', got '%s'", status)
		}
	})

	t.Run("one unhealthy", func(t *testing.T) {
		checker := NewChecker(5 * time.Second)
		checker.Register("check1", "test", func(ctx context.Context) error {
			return nil
		})
		checker.Register("check2", "test", func(ctx context.Context) error {
			return errors.New("failed")
		})

		status := checker.GetStatus()
		if status != "unhealthy" {
			t.Errorf("expected status 'unhealthy', got '%s'", status)
		}
	})
}

func TestCheckTimeout(t *testing.T) {
	checker := NewChecker(100 * time.Millisecond)
	checker.Register("slow-check", "test", func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	result, err := checker.Check(context.Background(), "slow-check")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Healthy {
		t.Error("expected slow check to fail due to timeout")
	}
}

func TestGetUptime(t *testing.T) {
	checker := NewChecker(5 * time.Second)

	uptime := checker.GetUptime()
	if uptime == 0 {
		t.Error("expected non-zero uptime")
	}

	time.Sleep(10 * time.Millisecond)

	newUptime := checker.GetUptime()
	if newUptime <= uptime {
		t.Error("expected uptime to increase")
	}
}

func TestListChecks(t *testing.T) {
	checker := NewChecker(5 * time.Second)
	checker.Register("check1", "test", func(ctx context.Context) error {
		return nil
	})
	checker.Register("check2", "test", func(ctx context.Context) error {
		return nil
	})

	names := checker.ListChecks()
	if len(names) != 2 {
		t.Errorf("expected 2 checks, got %d", len(names))
	}
}

func TestUnregister(t *testing.T) {
	checker := NewChecker(5 * time.Second)
	checker.Register("check1", "test", func(ctx context.Context) error {
		return nil
	})

	checker.Unregister("check1")

	_, err := checker.Check(context.Background(), "check1")
	if err == nil {
		t.Error("expected error after unregistering check")
	}
}

func TestIsHealthy(t *testing.T) {
	t.Run("all healthy", func(t *testing.T) {
		checker := NewChecker(5 * time.Second)
		checker.Register("check1", "test", func(ctx context.Context) error {
			return nil
		})
		checker.Register("check2", "test", func(ctx context.Context) error {
			return nil
		})

		if !checker.IsHealthy(context.Background()) {
			t.Error("expected IsHealthy to return true")
		}
	})

	t.Run("one unhealthy", func(t *testing.T) {
		checker := NewChecker(5 * time.Second)
		checker.Register("check1", "test", func(ctx context.Context) error {
			return nil
		})
		checker.Register("check2", "test", func(ctx context.Context) error {
			return errors.New("failed")
		})

		if checker.IsHealthy(context.Background()) {
			t.Error("expected IsHealthy to return false")
		}
	})
}
