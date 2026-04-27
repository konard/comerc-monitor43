package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_closed_success(t *testing.T) {
	t.Parallel()

	cb := New(Config{FailureThreshold: 3, Timeout: 1 * time.Second, SuccessThreshold: 1})

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 0, cb.failures)
}

func TestCircuitBreaker_closed_failures_then_open(t *testing.T) {
	t.Parallel()

	cb := New(Config{FailureThreshold: 3, Timeout: 100 * time.Millisecond, SuccessThreshold: 1})

	testErr := errors.New("connection refused")

	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
		assert.Error(t, err)
	}

	assert.Equal(t, StateOpen, cb.State())

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	assert.True(t, IsErrCircuitOpen(err))
}

func TestCircuitBreaker_closed_failure_then_success_resets(t *testing.T) {
	t.Parallel()

	cb := New(Config{FailureThreshold: 3, Timeout: 100 * time.Millisecond, SuccessThreshold: 1})

	testErr := errors.New("timeout")

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})
	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 1, cb.failures)

	err = cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 0, cb.failures)
}

func TestCircuitBreaker_open_to_halfopen_to_closed(t *testing.T) {
	t.Parallel()

	cb := New(Config{FailureThreshold: 2, Timeout: 50 * time.Millisecond, SuccessThreshold: 1})

	testErr := errors.New("refused")

	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
		assert.Error(t, err)
	}
	assert.Equal(t, StateOpen, cb.State())

	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, StateHalfOpen, cb.State())

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_halfopen_failure_back_to_open(t *testing.T) {
	t.Parallel()

	cb := New(Config{FailureThreshold: 2, Timeout: 50 * time.Millisecond, SuccessThreshold: 1})

	testErr := errors.New("refused")

	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
		assert.Error(t, err)
	}

	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, StateHalfOpen, cb.State())

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return testErr
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreaker_halfopen_multiple_successes(t *testing.T) {
	t.Parallel()

	cb := New(Config{FailureThreshold: 2, Timeout: 50 * time.Millisecond, SuccessThreshold: 3})

	testErr := errors.New("refused")

	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
		assert.Error(t, err)
	}

	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, StateHalfOpen, cb.State())

	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, StateHalfOpen, cb.State())
	}

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_default_config(t *testing.T) {
	t.Parallel()

	cb := New(Config{})

	assert.Equal(t, 5, cb.cfg.FailureThreshold)
	assert.Equal(t, 30*time.Second, cb.cfg.Timeout)
	assert.Equal(t, 1, cb.cfg.SuccessThreshold)
	assert.Equal(t, StateClosed, cb.State())
}

func TestIsErrCircuitOpen(t *testing.T) {
	t.Parallel()

	assert.True(t, IsErrCircuitOpen(ErrCircuitOpen))
	assert.True(t, IsErrCircuitOpen(FormatErrCircuitOpen("monitor-service")))
	assert.False(t, IsErrCircuitOpen(errors.New("other error")))
}

func TestFormatErrCircuitOpen(t *testing.T) {
	t.Parallel()

	err := FormatErrCircuitOpen("monitor-service")
	assert.Contains(t, err.Error(), "monitor-service")
	assert.True(t, IsErrCircuitOpen(err))
}

func TestState_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "CLOSED"},
		{StateOpen, "OPEN"},
		{StateHalfOpen, "HALF_OPEN"},
		{State(99), "UNKNOWN"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.state.String())
		})
	}
}

func TestCircuitBreaker_open_record_failure_updates_lastFailure(t *testing.T) {
	t.Parallel()

	cb := New(Config{FailureThreshold: 2, Timeout: 100 * time.Millisecond, SuccessThreshold: 1})
	testErr := errors.New("refused")

	// переходим в Open
	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func(_ context.Context) error { return testErr })
		assert.Error(t, err)
	}
	assert.Equal(t, StateOpen, cb.State())

	// recordResult в состоянии Open с ошибкой обновляет lastFailure
	cb.mu.Lock()
	cb.state = StateOpen
	cb.mu.Unlock()

	err := cb.Execute(context.Background(), func(_ context.Context) error {
		// не дойдёт до этого — CB открыт
		return testErr
	})
	assert.Error(t, err)

	// всё ещё Open
	assert.Equal(t, StateOpen, cb.State())
}
