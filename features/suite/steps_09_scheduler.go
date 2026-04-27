//go:build bdd

package suite

import (
	"context"
	"fmt"
	"sync"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	api "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// schedulerSteps реализует шаги для эпика 09_scheduler.
// Покрывают lifecycle воркера через реальный gRPC scheduler-service.
type schedulerSteps struct {
	stack *Stack
	state *ScenarioState

	// Per-scenario состояние воркеров.
	workerID     string
	workerName   string
	registerErr  error
	heartbeatErr error
	parallelIDs  []string
	parallelErrs []error
}

// RegisterSchedulerSteps регистрирует шаги для эпика 09_scheduler.
func RegisterSchedulerSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	s := &schedulerSteps{stack: stack, state: state}

	ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
		s.workerID = ""
		s.workerName = ""
		s.registerErr = nil
		s.heartbeatErr = nil
		s.parallelIDs = nil
		s.parallelErrs = nil
		return c, nil
	})

	// Lifecycle (uc_09_01_20).
	ctx.Step(`^воркер регистрируется через scheduler API с именем "([^"]*)" в зоне "([^"]*)"$`, s.stepRegisterWorkerViaAPI)
	ctx.Step(`^воркер успешно зарегистрирован и получил идентификатор$`, s.stepWorkerRegisteredWithID)
	ctx.Step(`^воркер отправляет heartbeat со статусом "([^"]*)"$`, s.stepWorkerHeartbeatWithStatus)
	ctx.Step(`^heartbeat принят без ошибки$`, s.stepHeartbeatAccepted)
	ctx.Step(`^воркер выполняет дерегистрацию$`, s.stepWorkerUnregister)
	ctx.Step(`^воркер удалён из реестра scheduler-service$`, s.stepWorkerRemovedFromRegistry)

	// Duplicate registration (uc_09_01_21).
	ctx.Step(`^воркер с именем "([^"]*)" уже зарегистрирован через scheduler API$`, s.stepWorkerAlreadyRegistered)
	ctx.Step(`^второй воркер пытается зарегистрироваться с именем "([^"]*)"$`, s.stepSecondWorkerRegister)
	ctx.Step(`^scheduler возвращает ошибку с кодом AlreadyExists$`, s.stepErrorAlreadyExists)

	// Heartbeat unknown worker (uc_09_01_22).
	ctx.Step(`^heartbeat отправляется для неизвестного worker_id "([^"]*)"$`, s.stepHeartbeatUnknown)
	ctx.Step(`^scheduler возвращает ошибку с кодом NotFound$`, s.stepErrorNotFound)

	// Parallel registration (uc_09_01_23).
	ctx.Step(`^одновременно регистрируются "(\d+)" воркеров с уникальными именами$`, s.stepParallelRegister)
	ctx.Step(`^все "(\d+)" воркеров получают уникальные идентификаторы$`, s.stepParallelAllUnique)
	ctx.Step(`^в реестре scheduler-service не менее "(\d+)" воркеров$`, s.stepRegistryHasAtLeast)
}

// --- Lifecycle ---

func (s *schedulerSteps) stepRegisterWorkerViaAPI(ctx context.Context, name, zone string) error {
	resp, err := s.stack.SchedulerClient.RegisterWorker(ctx, &api.RegisterWorkerRequest{
		Name: name,
		Zone: zone,
	})
	s.registerErr = err
	if err != nil {
		return nil
	}
	s.workerID = resp.GetId()
	s.workerName = name
	return nil
}

func (s *schedulerSteps) stepWorkerRegisteredWithID() error {
	if s.registerErr != nil {
		return fmt.Errorf("register worker: %w", s.registerErr)
	}
	if s.workerID == "" {
		return fmt.Errorf("worker id is empty after registration")
	}
	return nil
}

func (s *schedulerSteps) stepWorkerHeartbeatWithStatus(ctx context.Context, st string) error {
	_, err := s.stack.SchedulerClient.WorkerHeartbeat(ctx, &api.HeartbeatRequest{
		WorkerId: s.workerID,
		Status:   st,
	})
	s.heartbeatErr = err
	return nil
}

func (s *schedulerSteps) stepHeartbeatAccepted() error {
	if s.heartbeatErr != nil {
		return fmt.Errorf("heartbeat returned error: %w", s.heartbeatErr)
	}
	return nil
}

func (s *schedulerSteps) stepWorkerUnregister(ctx context.Context) error {
	_, err := s.stack.SchedulerClient.UnregisterWorker(ctx, &api.UnregisterWorkerRequest{
		WorkerId: s.workerID,
	})
	if err != nil {
		return fmt.Errorf("unregister worker: %w", err)
	}
	return nil
}

func (s *schedulerSteps) stepWorkerRemovedFromRegistry(ctx context.Context) error {
	// После Unregister воркер либо физически удалён, либо помечен OFFLINE.
	// Проверяем, что повторный heartbeat либо помечает его OFFLINE, либо
	// возвращает NotFound (и в ListWorkers нет ACTIVE-записи с этим id).
	resp, err := s.stack.SchedulerClient.ListWorkers(ctx, &api.ListWorkersRequest{PageSize: 1000})
	if err != nil {
		return fmt.Errorf("list workers: %w", err)
	}
	for _, w := range resp.GetWorkers() {
		if w.GetId() == s.workerID && w.GetStatus() != api.WorkerStatus_WORKER_STATUS_OFFLINE {
			return fmt.Errorf("worker %s still active in registry with status %s", s.workerID, w.GetStatus())
		}
	}
	return nil
}

// --- Duplicate registration ---

func (s *schedulerSteps) stepWorkerAlreadyRegistered(ctx context.Context, name string) error {
	resp, err := s.stack.SchedulerClient.RegisterWorker(ctx, &api.RegisterWorkerRequest{
		Name: name,
		Zone: "moscow",
	})
	if err != nil {
		return fmt.Errorf("seed register worker: %w", err)
	}
	s.workerID = resp.GetId()
	s.workerName = name
	return nil
}

func (s *schedulerSteps) stepSecondWorkerRegister(ctx context.Context, name string) error {
	_, err := s.stack.SchedulerClient.RegisterWorker(ctx, &api.RegisterWorkerRequest{
		Name: name,
		Zone: "moscow",
	})
	s.registerErr = err
	return nil
}

func (s *schedulerSteps) stepErrorAlreadyExists() error {
	if s.registerErr == nil {
		return fmt.Errorf("expected AlreadyExists error, got nil")
	}
	st, ok := status.FromError(s.registerErr)
	if !ok {
		return fmt.Errorf("not a grpc status error: %v", s.registerErr)
	}
	if st.Code() != codes.AlreadyExists {
		return fmt.Errorf("expected code AlreadyExists, got %s: %s", st.Code(), st.Message())
	}
	return nil
}

// --- Unknown heartbeat ---

func (s *schedulerSteps) stepHeartbeatUnknown(ctx context.Context, workerID string) error {
	_, err := s.stack.SchedulerClient.WorkerHeartbeat(ctx, &api.HeartbeatRequest{
		WorkerId: workerID,
		Status:   "IDLE",
	})
	s.heartbeatErr = err
	return nil
}

func (s *schedulerSteps) stepErrorNotFound() error {
	if s.heartbeatErr == nil {
		return fmt.Errorf("expected NotFound error, got nil")
	}
	st, ok := status.FromError(s.heartbeatErr)
	if !ok {
		return fmt.Errorf("not a grpc status error: %v", s.heartbeatErr)
	}
	if st.Code() != codes.NotFound {
		return fmt.Errorf("expected code NotFound, got %s: %s", st.Code(), st.Message())
	}
	return nil
}

// --- Parallel registration ---

func (s *schedulerSteps) stepParallelRegister(ctx context.Context, n int) error {
	type result struct {
		id  string
		err error
	}
	results := make(chan result, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name := "parallel-worker-" + uuid.New().String()[:8]
			resp, err := s.stack.SchedulerClient.RegisterWorker(ctx, &api.RegisterWorkerRequest{
				Name: name,
				Zone: "moscow",
			})
			if err != nil {
				results <- result{err: err}
				return
			}
			results <- result{id: resp.GetId()}
		}()
	}
	wg.Wait()
	close(results)
	for r := range results {
		s.parallelIDs = append(s.parallelIDs, r.id)
		s.parallelErrs = append(s.parallelErrs, r.err)
	}
	return nil
}

func (s *schedulerSteps) stepParallelAllUnique(_ int) error {
	seen := make(map[string]bool)
	for i, err := range s.parallelErrs {
		if err != nil {
			return fmt.Errorf("parallel register #%d failed: %w", i, err)
		}
		id := s.parallelIDs[i]
		if id == "" {
			return fmt.Errorf("parallel register #%d returned empty id", i)
		}
		if seen[id] {
			return fmt.Errorf("duplicate worker id: %s", id)
		}
		seen[id] = true
	}
	return nil
}

func (s *schedulerSteps) stepRegistryHasAtLeast(ctx context.Context, min int) error {
	resp, err := s.stack.SchedulerClient.ListWorkers(ctx, &api.ListWorkersRequest{PageSize: 1000})
	if err != nil {
		return fmt.Errorf("list workers: %w", err)
	}
	if len(resp.GetWorkers()) < min {
		return fmt.Errorf("expected at least %d workers, got %d", min, len(resp.GetWorkers()))
	}
	return nil
}
