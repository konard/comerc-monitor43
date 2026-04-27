//go:build bdd

package suite

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	monitorv1 "github.com/raul/monitor/api/proto/monitor/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// pipelineSteps реализует шаги Gherkin для cross-service интеграции
// monitor-service ↔ dashboard и monitor-service ↔ scheduler.
type pipelineSteps struct {
	stack *Stack
	state *ScenarioState

	// Состояние сценария удаления через событие monitor.deleted.
	deleteMonitorID uuid.UUID
	deleteUserID    uuid.UUID

	// Список созданных через CreateMonitor мониторов для scheduler-сценария.
	schedulerMonitorIDs []string

	// Результаты последних scheduler-вызовов ListMonitors/GetMonitor.
	lastListResp *monitorv1.ListMonitorsResponse
	lastGetResp  *monitorv1.GetMonitorResponse
	lastGetErr   error
}

// RegisterMonitoringPipelineSteps регистрирует шаги для cross-service интеграции
// в эпике 01_monitoring (uc_01_06_*).
func RegisterMonitoringPipelineSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	s := &pipelineSteps{stack: stack, state: state}

	ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
		s.deleteMonitorID = uuid.Nil
		s.deleteUserID = uuid.Nil
		s.schedulerMonitorIDs = nil
		s.lastListResp = nil
		s.lastGetResp = nil
		s.lastGetErr = nil
		return c, nil
	})

	// === uc_01_06_01: Create → publish → dashboard ===

	ctx.Step(`^публикуется событие monitor\.created для созданного монитора$`, s.stepPublishCreatedForLastMonitor)
	ctx.Step(`^монитор появляется в dashboard с тем же URL$`, s.stepMonitorAppearsInDashboard)

	// === uc_01_06_02: Publish create → publish delete → dashboard removes ===

	ctx.Step(`^в dashboard опубликовано событие monitor\.created для нового монитора$`, s.stepPublishCreatedForNewMonitor)
	ctx.Step(`^запись о мониторе присутствует в dashboard$`, s.stepDeletedMonitorPresentInDashboard)
	ctx.Step(`^публикуется событие monitor\.deleted для этого монитора$`, s.stepPublishDeletedForMonitor)
	ctx.Step(`^запись о мониторе удалена из dashboard$`, s.stepDeletedMonitorRemovedFromDashboard)

	// === uc_01_06_03: Scheduler discovery ===

	ctx.Step(`^пользователь создаёт мониторы для scheduler:$`, s.stepCreateSchedulerMonitors)
	ctx.Step(`^список мониторов в статусе "([^"]*)" содержит "([^"]*)" записи$`, s.stepListMonitorsByStatusCount)
	ctx.Step(`^пользователь приостанавливает первый созданный монитор$`, s.stepPauseFirstSchedulerMonitor)
	ctx.Step(`^приостановленный монитор имеет статус "([^"]*)"$`, s.stepPausedMonitorStatus)

	// === uc_01_06_04..07: Scheduler ↔ monitor-service ===

	ctx.Step(`^scheduler запрашивает список мониторов через ListMonitors$`, s.stepSchedulerListMonitors)
	ctx.Step(`^получен список из "([^"]*)" мониторов$`, s.stepSchedulerListCount)
	ctx.Step(`^каждый монитор в списке имеет непустой идентификатор и положительный интервал$`, s.stepSchedulerListEachMonitorValid)
	ctx.Step(`^scheduler запрашивает детали созданного монитора через GetMonitor$`, s.stepSchedulerGetMonitor)
	ctx.Step(`^детали монитора совпадают с параметрами:$`, s.stepSchedulerGetMonitorMatches)
	ctx.Step(`^статус полученного монитора равен "([^"]*)"$`, s.stepSchedulerGetMonitorStatus)
	ctx.Step(`^получена ошибка NotFound$`, s.stepSchedulerGetMonitorNotFound)
}

// stepSchedulerListMonitors вызывает ListMonitors без фильтра — как scheduler опрашивает планируемые мониторы.
func (s *pipelineSteps) stepSchedulerListMonitors() error {
	ensureMonitoringMaps(s.state)
	if s.state.UserID == uuid.Nil {
		s.state.UserID = uuid.New()
	}
	c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), s.state.UserID, s.state.MonitorTier), 5*time.Second)
	defer cancel()

	resp, err := s.stack.MonitorClient.ListMonitors(c, &monitorv1.ListMonitorsRequest{Limit: 100})
	if err != nil {
		return fmt.Errorf("list monitors: %w", err)
	}
	s.lastListResp = resp
	return nil
}

// stepSchedulerListCount проверяет количество мониторов в последнем ответе ListMonitors.
func (s *pipelineSteps) stepSchedulerListCount(expected string) error {
	if s.lastListResp == nil {
		return fmt.Errorf("no list response captured")
	}
	var expectedCount int
	if _, err := fmt.Sscanf(expected, "%d", &expectedCount); err != nil {
		return fmt.Errorf("invalid expected count %q: %w", expected, err)
	}
	if len(s.lastListResp.Monitors) != expectedCount {
		return fmt.Errorf("expected %d monitors, got %d", expectedCount, len(s.lastListResp.Monitors))
	}
	return nil
}

// stepSchedulerListEachMonitorValid проверяет базовые инварианты записей списка.
func (s *pipelineSteps) stepSchedulerListEachMonitorValid() error {
	if s.lastListResp == nil {
		return fmt.Errorf("no list response captured")
	}
	for i, m := range s.lastListResp.Monitors {
		if m.Id == "" {
			return fmt.Errorf("monitor[%d] has empty id", i)
		}
		if m.Name == "" {
			return fmt.Errorf("monitor[%d] has empty name", i)
		}
		if m.IntervalSeconds <= 0 {
			return fmt.Errorf("monitor[%d] has non-positive interval %d", i, m.IntervalSeconds)
		}
	}
	return nil
}

// stepSchedulerGetMonitor вызывает GetMonitor для последнего созданного монитора и сохраняет результат/ошибку.
func (s *pipelineSteps) stepSchedulerGetMonitor() error {
	if s.state.LastMonitor == nil {
		return fmt.Errorf("no last monitor in state")
	}
	c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), s.state.UserID, s.state.MonitorTier), 5*time.Second)
	defer cancel()

	resp, err := s.stack.MonitorClient.GetMonitor(c, &monitorv1.GetMonitorRequest{Id: s.state.LastMonitor.ID.String()})
	s.lastGetResp = resp
	s.lastGetErr = err
	return nil
}

// stepSchedulerGetMonitorMatches сверяет поля name/interval/timeout последнего GetMonitor-ответа с таблицей.
func (s *pipelineSteps) stepSchedulerGetMonitorMatches(table *godog.Table) error {
	if s.lastGetErr != nil {
		return fmt.Errorf("get monitor failed: %w", s.lastGetErr)
	}
	if s.lastGetResp == nil || s.lastGetResp.Monitor == nil {
		return fmt.Errorf("no get monitor response captured")
	}
	m := s.lastGetResp.Monitor
	for _, row := range table.Rows {
		if len(row.Cells) < 2 {
			continue
		}
		key := row.Cells[0].Value
		val := row.Cells[1].Value
		switch key {
		case "name":
			if m.Name != val {
				return fmt.Errorf("expected name %q, got %q", val, m.Name)
			}
		case "interval":
			var n int32
			if _, err := fmt.Sscanf(val, "%d", &n); err != nil {
				return fmt.Errorf("invalid interval %q: %w", val, err)
			}
			if m.IntervalSeconds != n {
				return fmt.Errorf("expected interval %d, got %d", n, m.IntervalSeconds)
			}
		case "timeout":
			var n int32
			if _, err := fmt.Sscanf(val, "%d", &n); err != nil {
				return fmt.Errorf("invalid timeout %q: %w", val, err)
			}
			if m.TimeoutSeconds != n {
				return fmt.Errorf("expected timeout %d, got %d", n, m.TimeoutSeconds)
			}
		}
	}
	return nil
}

// stepSchedulerGetMonitorStatus проверяет статус последнего GetMonitor-ответа.
func (s *pipelineSteps) stepSchedulerGetMonitorStatus(expected string) error {
	if s.lastGetErr != nil {
		return fmt.Errorf("get monitor failed: %w", s.lastGetErr)
	}
	if s.lastGetResp == nil || s.lastGetResp.Monitor == nil {
		return fmt.Errorf("no get monitor response captured")
	}
	if s.lastGetResp.Monitor.Status != expected {
		return fmt.Errorf("expected status %q, got %q", expected, s.lastGetResp.Monitor.Status)
	}
	return nil
}

// stepSchedulerGetMonitorNotFound проверяет, что последний GetMonitor вернул NotFound.
func (s *pipelineSteps) stepSchedulerGetMonitorNotFound() error {
	if s.lastGetErr == nil {
		return fmt.Errorf("expected NotFound error, got nil")
	}
	st, ok := status.FromError(s.lastGetErr)
	if !ok {
		return fmt.Errorf("expected gRPC status error, got %T: %v", s.lastGetErr, s.lastGetErr)
	}
	if st.Code() != codes.NotFound {
		return fmt.Errorf("expected NotFound, got %s", st.Code())
	}
	return nil
}

// stepPublishCreatedForLastMonitor публикует monitor.created для state.LastMonitor.
func (s *pipelineSteps) stepPublishCreatedForLastMonitor(ctx context.Context) error {
	if s.state.LastMonitor == nil {
		return fmt.Errorf("no last monitor to publish event for")
	}
	m := s.state.LastMonitor
	userID := m.UserID
	if userID == uuid.Nil {
		userID = uuid.New()
	}
	payload := map[string]any{
		"event_type": "monitor.created",
		"monitor_id": m.ID.String(),
		"user_id":    userID.String(),
		"name":       m.Name,
		"url":        m.URL,
		"timestamp":  time.Now().Unix(),
	}
	if err := s.stack.PublishEvent(ctx, "monitor.created", payload); err != nil {
		return fmt.Errorf("publish monitor.created: %w", err)
	}
	return nil
}

// stepMonitorAppearsInDashboard ждёт появления записи в dashboard и сверяет URL.
func (s *pipelineSteps) stepMonitorAppearsInDashboard(ctx context.Context) error {
	if s.state.LastMonitor == nil {
		return fmt.Errorf("no last monitor in state")
	}
	monitorID := s.state.LastMonitor.ID.String()
	expectedURL := s.state.LastMonitor.URL

	if err := waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var count int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM monitor_statuses WHERE id = $1", monitorID,
		).Scan(&count)
		return err == nil && count > 0
	}); err != nil {
		return fmt.Errorf("monitor %s did not appear in dashboard: %w", monitorID, err)
	}

	var storedURL string
	if err := s.stack.DashboardDB.QueryRowContext(ctx,
		"SELECT url FROM monitor_statuses WHERE id = $1", monitorID,
	).Scan(&storedURL); err != nil {
		return fmt.Errorf("query stored url: %w", err)
	}
	if storedURL != expectedURL {
		return fmt.Errorf("expected dashboard url %q, got %q", expectedURL, storedURL)
	}
	return nil
}

// stepPublishCreatedForNewMonitor публикует monitor.created для нового UUID.
func (s *pipelineSteps) stepPublishCreatedForNewMonitor(ctx context.Context) error {
	s.deleteMonitorID = uuid.New()
	s.deleteUserID = uuid.New()
	payload := map[string]any{
		"event_type": "monitor.created",
		"monitor_id": s.deleteMonitorID.String(),
		"user_id":    s.deleteUserID.String(),
		"name":       "Delete Test Monitor",
		"url":        "https://delete-test.example.com",
		"timestamp":  time.Now().Unix(),
	}
	if err := s.stack.PublishEvent(ctx, "monitor.created", payload); err != nil {
		return fmt.Errorf("publish monitor.created: %w", err)
	}
	return nil
}

// stepDeletedMonitorPresentInDashboard ждёт появления записи о мониторе для удаления.
func (s *pipelineSteps) stepDeletedMonitorPresentInDashboard(ctx context.Context) error {
	if s.deleteMonitorID == uuid.Nil {
		return fmt.Errorf("delete monitor id is empty")
	}
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var count int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM monitor_statuses WHERE id = $1", s.deleteMonitorID.String(),
		).Scan(&count)
		return err == nil && count > 0
	})
}

// stepPublishDeletedForMonitor публикует monitor.deleted для ранее созданного монитора.
func (s *pipelineSteps) stepPublishDeletedForMonitor(ctx context.Context) error {
	if s.deleteMonitorID == uuid.Nil {
		return fmt.Errorf("delete monitor id is empty")
	}
	payload := map[string]any{
		"event_type": "monitor.deleted",
		"monitor_id": s.deleteMonitorID.String(),
		"user_id":    s.deleteUserID.String(),
		"name":       "Delete Test Monitor",
		"timestamp":  time.Now().Unix(),
	}
	return s.stack.PublishEvent(ctx, "monitor.deleted", payload)
}

// stepDeletedMonitorRemovedFromDashboard ждёт исчезновения записи о мониторе.
func (s *pipelineSteps) stepDeletedMonitorRemovedFromDashboard(ctx context.Context) error {
	if s.deleteMonitorID == uuid.Nil {
		return fmt.Errorf("delete monitor id is empty")
	}
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var count int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM monitor_statuses WHERE id = $1", s.deleteMonitorID.String(),
		).Scan(&count)
		return err == nil && count == 0
	})
}

// stepCreateSchedulerMonitors создаёт серию мониторов через gRPC monitor-service.
func (s *pipelineSteps) stepCreateSchedulerMonitors(table *godog.Table) error {
	ensureMonitoringMaps(s.state)
	if s.state.UserID == uuid.Nil {
		s.state.UserID = uuid.New()
	}
	if s.state.MonitorTier == "" {
		s.state.MonitorTier = "Pro"
	}

	if len(table.Rows) < 2 {
		return fmt.Errorf("expected header row + at least one monitor row")
	}

	// Первая строка — заголовок (name, url).
	header := table.Rows[0]
	nameIdx, urlIdx := -1, -1
	for i, cell := range header.Cells {
		switch cell.Value {
		case "name":
			nameIdx = i
		case "url":
			urlIdx = i
		}
	}
	if nameIdx == -1 || urlIdx == -1 {
		return fmt.Errorf("table must have 'name' and 'url' columns")
	}

	s.schedulerMonitorIDs = make([]string, 0, len(table.Rows)-1)
	for _, row := range table.Rows[1:] {
		if len(row.Cells) <= nameIdx || len(row.Cells) <= urlIdx {
			continue
		}
		req := &monitorv1.CreateMonitorRequest{
			Name:            row.Cells[nameIdx].Value,
			Url:             row.Cells[urlIdx].Value,
			CheckType:       "HTTP",
			IntervalSeconds: 30,
			TimeoutSeconds:  10,
		}
		c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), s.state.UserID, s.state.MonitorTier), 5*time.Second)
		resp, err := s.stack.MonitorClient.CreateMonitor(c, req)
		cancel()
		if err != nil {
			return fmt.Errorf("create monitor %q: %w", req.Name, err)
		}
		s.schedulerMonitorIDs = append(s.schedulerMonitorIDs, resp.Monitor.Id)
		info := monitorFromResponse(resp.Monitor)
		s.state.LastMonitor = info
		s.state.Monitors[info.ID] = info
		s.state.MonitorsByName[info.Name] = info
	}
	return nil
}

// stepListMonitorsByStatusCount проверяет количество мониторов в указанном статусе.
func (s *pipelineSteps) stepListMonitorsByStatusCount(status, expected string) error {
	var expectedCount int
	if _, err := fmt.Sscanf(expected, "%d", &expectedCount); err != nil {
		return fmt.Errorf("invalid expected count %q: %w", expected, err)
	}

	c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), s.state.UserID, s.state.MonitorTier), 5*time.Second)
	defer cancel()

	resp, err := s.stack.MonitorClient.ListMonitors(c, &monitorv1.ListMonitorsRequest{
		Status: status,
		Limit:  100,
	})
	if err != nil {
		return fmt.Errorf("list monitors: %w", err)
	}
	if len(resp.Monitors) != expectedCount {
		return fmt.Errorf("expected %d monitors with status %q, got %d", expectedCount, status, len(resp.Monitors))
	}
	return nil
}

// stepPauseFirstSchedulerMonitor приостанавливает первый созданный в сценарии монитор.
func (s *pipelineSteps) stepPauseFirstSchedulerMonitor() error {
	if len(s.schedulerMonitorIDs) == 0 {
		return fmt.Errorf("no scheduler monitors created")
	}
	pauseID := s.schedulerMonitorIDs[0]

	c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), s.state.UserID, s.state.MonitorTier), 5*time.Second)
	defer cancel()

	if _, err := s.stack.MonitorClient.PauseMonitor(c, &monitorv1.PauseMonitorRequest{Id: pauseID}); err != nil {
		return fmt.Errorf("pause monitor: %w", err)
	}
	return nil
}

// stepPausedMonitorStatus проверяет, что приостановленный монитор имеет ожидаемый статус.
func (s *pipelineSteps) stepPausedMonitorStatus(expected string) error {
	if len(s.schedulerMonitorIDs) == 0 {
		return fmt.Errorf("no scheduler monitors created")
	}
	pauseID := s.schedulerMonitorIDs[0]

	c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), s.state.UserID, s.state.MonitorTier), 5*time.Second)
	defer cancel()

	resp, err := s.stack.MonitorClient.GetMonitor(c, &monitorv1.GetMonitorRequest{Id: pauseID})
	if err != nil {
		return fmt.Errorf("get monitor: %w", err)
	}
	if resp.Monitor.Status != expected {
		return fmt.Errorf("expected status %q, got %q", expected, resp.Monitor.Status)
	}
	return nil
}
