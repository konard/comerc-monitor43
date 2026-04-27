//go:build bdd

package suite

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog"
)

// checkWorkerSteps реализует observation-style шаги для эпика 10_check_worker.
// Большинство сценариев — наблюдение состояния воркера через БД scheduler-service:
// регистрация, heartbeat, статус, last_heartbeat. Сценарии, требующие
// HTTP-мок-сервера для целевых проверок или контроля присвоения проверок
// (uc_10_02_*, uc_10_03_*), оставлены pending — их реализация требует
// расширения mock-инфраструктуры и контроля scheduler check assignment.
type checkWorkerSteps struct {
	stack *Stack
	state *ScenarioState
}

// RegisterCheckWorkerSteps регистрирует шаги для эпика 10_check_worker.
func RegisterCheckWorkerSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	s := &checkWorkerSteps{stack: stack, state: state}

	// Lifecycle: регистрация и идентификация воркера.
	// @implemented uc_10_01_01: запуск и регистрация воркера.
	ctx.Step(`^check-worker subprocess запущен$`, s.stepWorkerSubprocessRunning)
	ctx.Step(`^check-worker зарегистрирован в scheduler-service$`, s.stepWorkerRegisteredInScheduler)
	ctx.Step(`^worker_id присвоен и не пустой$`, s.stepWorkerIDAssigned)

	// @implemented uc_10_01_02: периодический heartbeat.
	ctx.Step(`^heartbeat обновляет last_heartbeat в течение "(\d+)" секунд$`, s.stepHeartbeatUpdates)

	// @implemented: Проверка статуса воркера.
	ctx.Step(`^статус воркера в scheduler_workers равен "([^"]*)"$`, s.stepWorkerStatusEquals)

	// @implemented: Проверка зоны воркера.
	ctx.Step(`^воркер находится в зоне "([^"]*)"$`, s.stepWorkerZoneEquals)
}

// stepWorkerSubprocessRunning подтверждает что check-worker subprocess стартанул.
func (s *checkWorkerSteps) stepWorkerSubprocessRunning() error {
	if s.stack.checkWorkerProcess == nil {
		return fmt.Errorf("check-worker subprocess is not running")
	}
	return nil
}

// stepWorkerRegisteredInScheduler проверяет, что worker есть в scheduler_workers.
func (s *checkWorkerSteps) stepWorkerRegisteredInScheduler(ctx context.Context) error {
	var count int
	err := s.stack.SchedulerDB.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM scheduler_workers WHERE name = $1`,
		s.stack.CheckWorkerName,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("query scheduler_workers: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("expected 1 worker registration for %q, got %d", s.stack.CheckWorkerName, count)
	}
	return nil
}

// stepWorkerIDAssigned проверяет, что у Stack есть присвоенный worker_id.
func (s *checkWorkerSteps) stepWorkerIDAssigned() error {
	if s.stack.CheckWorkerID == "" {
		return fmt.Errorf("CheckWorkerID is empty")
	}
	return nil
}

// stepHeartbeatUpdates ждёт обновления last_heartbeat за указанное число секунд.
func (s *checkWorkerSteps) stepHeartbeatUpdates(ctx context.Context, seconds int) error {
	var initial time.Time
	if err := s.stack.SchedulerDB.QueryRowxContext(ctx,
		`SELECT last_heartbeat FROM scheduler_workers WHERE name = $1`,
		s.stack.CheckWorkerName,
	).Scan(&initial); err != nil {
		return fmt.Errorf("query initial heartbeat: %w", err)
	}

	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	for time.Now().Before(deadline) {
		var current time.Time
		if err := s.stack.SchedulerDB.QueryRowxContext(ctx,
			`SELECT last_heartbeat FROM scheduler_workers WHERE name = $1`,
			s.stack.CheckWorkerName,
		).Scan(&current); err == nil && current.After(initial) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return fmt.Errorf("last_heartbeat did not advance within %ds", seconds)
}

// stepWorkerStatusEquals проверяет статус worker в scheduler_workers.
func (s *checkWorkerSteps) stepWorkerStatusEquals(ctx context.Context, expected string) error {
	var status string
	err := s.stack.SchedulerDB.QueryRowxContext(ctx,
		`SELECT status FROM scheduler_workers WHERE name = $1`,
		s.stack.CheckWorkerName,
	).Scan(&status)
	if err != nil {
		return fmt.Errorf("query worker status: %w", err)
	}
	if status != expected {
		return fmt.Errorf("expected worker status %q, got %q", expected, status)
	}
	return nil
}

// stepWorkerZoneEquals проверяет зону воркера в scheduler_workers.
func (s *checkWorkerSteps) stepWorkerZoneEquals(ctx context.Context, expected string) error {
	var zone string
	err := s.stack.SchedulerDB.QueryRowxContext(ctx,
		`SELECT zone FROM scheduler_workers WHERE name = $1`,
		s.stack.CheckWorkerName,
	).Scan(&zone)
	if err != nil {
		return fmt.Errorf("query worker zone: %w", err)
	}
	if zone != expected {
		return fmt.Errorf("expected worker zone %q, got %q", expected, zone)
	}
	return nil
}
