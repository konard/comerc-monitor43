//go:build bdd

package suite

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
)

// dashboardSteps реализует шаги Gherkin для эпика 03_dashboard.
type dashboardSteps struct {
	stack *Stack
	state *ScenarioState

	monitorID  string
	incidentID string
	userID     string

	// namedMonitors хранит monitor_id по метке (например "A", "B") для сценариев
	// с несколькими мониторами.
	namedMonitors map[string]string
	// namedIncidents хранит incident_id по метке для сценариев с несколькими инцидентами.
	namedIncidents map[string]string
}

// RegisterDashboardSteps регистрирует шаги для эпика 03_dashboard.
func RegisterDashboardSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	s := &dashboardSteps{stack: stack, state: state}

	ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
		s.monitorID = uuid.New().String()
		s.incidentID = uuid.New().String()
		s.userID = uuid.New().String()
		s.namedMonitors = map[string]string{}
		s.namedIncidents = map[string]string{}
		return c, nil
	})

	// Полностью реализованные шаги для migrated сценариев паузы/возобновления,
	// проверок и инцидентов.
	ctx.Step(`^пользователь имеет монитор "([^"]*)"$`, s.stepUserHasMonitor)
	ctx.Step(`^монитор создан событием monitor\.created$`, s.stepMonitorCreatedViaEvent)
	ctx.Step(`^публикуется событие monitor\.paused$`, s.stepPublishMonitorPaused)
	ctx.Step(`^публикуется событие monitor\.resumed$`, s.stepPublishMonitorResumed)
	ctx.Step(`^публикуется событие check\.completed со статусом "([^"]*)"$`, s.stepPublishCheckCompleted)
	ctx.Step(`^публикуется событие check\.failed с ошибкой "([^"]*)"$`, s.stepPublishCheckFailed)
	ctx.Step(`^публикуется событие incident\.detected$`, s.stepPublishIncidentDetected)
	ctx.Step(`^публикуется событие incident\.resolved$`, s.stepPublishIncidentResolved)
	ctx.Step(`^статус монитора в dashboard равен "([^"]*)"$`, s.stepMonitorStatusEquals)
	ctx.Step(`^в check_history появляется запись со статусом "([^"]*)"$`, s.stepCheckHistoryContainsStatus)
	ctx.Step(`^в check_history появляется запись с ошибкой "([^"]*)"$`, s.stepCheckHistoryContainsError)
	ctx.Step(`^в incidents появляется активный инцидент$`, s.stepIncidentActiveExists)
	ctx.Step(`^инцидент в incidents отмечен как разрешённый$`, s.stepIncidentResolved)

	// Шаги для миграции flow-тестов: check_result_flow_test.go и incident_flow_test.go.
	ctx.Step(`^в системе есть монитор$`, s.stepSystemHasMonitor)
	ctx.Step(`^в системе есть монитор "([^"]*)"$`, s.stepSystemHasNamedMonitor)
	ctx.Step(`^в системе есть инцидент "([^"]*)"$`, s.stepSystemHasNamedIncident)
	ctx.Step(`^публикуется событие check\.completed с кодом "([^"]*)" и временем "([^"]*)"$`, s.stepPublishCheckCompletedWithCodeAndTime)
	ctx.Step(`^запись содержит status_code "([^"]*)" и response_time_ms "([^"]*)"$`, s.stepCheckHistoryFieldsMatch)
	ctx.Step(`^поле error_message пустое$`, s.stepCheckHistoryErrorMessageEmpty)
	ctx.Step(`^запись содержит error_message "([^"]*)"$`, s.stepCheckHistoryErrorMessageEquals)
	ctx.Step(`^поле status_code пустое$`, s.stepCheckHistoryStatusCodeEmpty)
	ctx.Step(`^публикуется последовательность check событий:$`, s.stepPublishCheckSequence)
	ctx.Step(`^в check_history появляется "([^"]*)" записи для монитора$`, s.stepCheckHistoryCountForCurrentMonitor)
	ctx.Step(`^статусы в порядке checked_at равны "([^"]*)"$`, s.stepCheckHistoryStatusesOrderedEqual)
	ctx.Step(`^публикуется событие check\.completed для монитора "([^"]*)" со статусом "([^"]*)"$`, s.stepPublishCheckCompletedForNamedMonitor)
	ctx.Step(`^публикуется событие check\.failed для монитора "([^"]*)" со статусом "([^"]*)"$`, s.stepPublishCheckFailedForNamedMonitor)
	ctx.Step(`^в check_history для монитора "([^"]*)" ровно "([^"]*)" запись$`, s.stepCheckHistoryCountForNamedMonitor)

	ctx.Step(`^поле started_at инцидента заполнено$`, s.stepIncidentStartedAtSet)
	ctx.Step(`^поле ended_at инцидента пустое$`, s.stepIncidentEndedAtNull)
	ctx.Step(`^поле ended_at инцидента заполнено$`, s.stepIncidentEndedAtSet)
	ctx.Step(`^поле duration_seconds инцидента не отрицательное$`, s.stepIncidentDurationNonNegative)
	ctx.Step(`^опубликовано событие incident\.detected$`, s.stepPublishIncidentDetected)
	ctx.Step(`^публикуется событие incident\.resolved через "([^"]*)" секунду$`, s.stepPublishIncidentResolvedAfterDelay)
	ctx.Step(`^публикуется событие incident\.detected для инцидента "([^"]*)"$`, s.stepPublishIncidentDetectedForNamed)
	ctx.Step(`^публикуется событие incident\.resolved для инцидента "([^"]*)"$`, s.stepPublishIncidentResolvedForNamed)
	ctx.Step(`^в incidents оба инцидента "([^"]*)" и "([^"]*)" имеют статус "([^"]*)"$`, s.stepIncidentsBothHaveStatus)
	ctx.Step(`^инцидент "([^"]*)" имеет статус "([^"]*)"$`, s.stepNamedIncidentStatusEquals)
	ctx.Step(`^инцидент "([^"]*)" остаётся в статусе "([^"]*)"$`, s.stepNamedIncidentStatusEquals)
	ctx.Step(`^фантомная запись для инцидента "([^"]*)" отсутствует$`, s.stepNamedIncidentAbsent)

	// Шаги для миграции monitor_dashboard_test.go (uc_03_01_18a-d).
	ctx.Step(`^в системе создан монитор через событие "monitor\.created"$`, s.stepMonitorCreatedViaEvent)
	ctx.Step(`^публикуется событие "([^"]*)" со статусом "([^"]*)"$`, s.stepPublishStatusEvent)
	ctx.Step(`^в dashboard статус монитора становится "([^"]*)"$`, s.stepMonitorStatusEquals)
	ctx.Step(`^публикуется событие "monitor\.created" с именем "([^"]*)"$`, s.stepPublishMonitorCreatedWithName)
	ctx.Step(`^в dashboard появляется запись монитора с именем "([^"]*)"$`, s.stepDashboardHasMonitorWithName)
	ctx.Step(`^публикуется невалидное JSON сообщение с routing key "([^"]*)"$`, s.stepPublishMalformed)

	// Шаги-заглушки для остальных сценариев feature-файлов.
	// Они возвращают ErrPending, чтобы сценарии отмечались как pending, а не failed.
	// При Strict=false в runner.go это не роняет сьюту.
	s.registerPendingSteps(ctx)
}

// stepUserHasMonitor создаёт монитор с именем name через событие monitor.created.
func (s *dashboardSteps) stepUserHasMonitor(ctx context.Context, name string) error {
	payload := map[string]any{
		"event_type": "monitor.created",
		"monitor_id": s.monitorID,
		"user_id":    s.userID,
		"name":       name,
		"url":        fmt.Sprintf("https://%s.example.com", s.monitorID[:8]),
		"timestamp":  time.Now().Unix(),
	}
	if err := s.stack.PublishEvent(ctx, "monitor.created", payload); err != nil {
		return fmt.Errorf("publish monitor.created: %w", err)
	}
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var count int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM monitor_statuses WHERE id = $1", s.monitorID,
		).Scan(&count)
		return err == nil && count > 0
	})
}

func (s *dashboardSteps) stepMonitorCreatedViaEvent(ctx context.Context) error {
	return s.stepUserHasMonitor(ctx, "BDD Monitor")
}

func (s *dashboardSteps) stepPublishMonitorPaused(ctx context.Context) error {
	payload := map[string]any{
		"event_type": "monitor.paused",
		"monitor_id": s.monitorID,
		"status":     "PAUSED",
	}
	return s.stack.PublishEvent(ctx, "monitor.paused", payload)
}

func (s *dashboardSteps) stepPublishMonitorResumed(ctx context.Context) error {
	payload := map[string]any{
		"event_type": "monitor.resumed",
		"monitor_id": s.monitorID,
		"status":     "UP",
	}
	return s.stack.PublishEvent(ctx, "monitor.resumed", payload)
}

func (s *dashboardSteps) stepPublishCheckCompleted(ctx context.Context, status string) error {
	statusCode := 200
	rt := 142.5
	payload := map[string]any{
		"event_type":       "check.completed",
		"monitor_id":       s.monitorID,
		"status":           status,
		"status_code":      statusCode,
		"response_time_ms": rt,
	}
	return s.stack.PublishEvent(ctx, "check.completed", payload)
}

func (s *dashboardSteps) stepPublishCheckFailed(ctx context.Context, errMsg string) error {
	payload := map[string]any{
		"event_type":    "check.failed",
		"monitor_id":    s.monitorID,
		"status":        "DOWN",
		"error_message": errMsg,
	}
	return s.stack.PublishEvent(ctx, "check.failed", payload)
}

func (s *dashboardSteps) stepPublishIncidentDetected(ctx context.Context) error {
	payload := map[string]any{
		"event_type":  "incident.detected",
		"incident_id": s.incidentID,
		"monitor_id":  s.monitorID,
	}
	return s.stack.PublishEvent(ctx, "incident.detected", payload)
}

func (s *dashboardSteps) stepPublishIncidentResolved(ctx context.Context) error {
	payload := map[string]any{
		"event_type":  "incident.resolved",
		"incident_id": s.incidentID,
		"monitor_id":  s.monitorID,
	}
	return s.stack.PublishEvent(ctx, "incident.resolved", payload)
}

func (s *dashboardSteps) stepMonitorStatusEquals(ctx context.Context, expected string) error {
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var status string
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT status FROM monitor_statuses WHERE id = $1", s.monitorID,
		).Scan(&status)
		return err == nil && status == expected
	})
}

func (s *dashboardSteps) stepCheckHistoryContainsStatus(ctx context.Context, expected string) error {
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var count int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM check_history WHERE monitor_id = $1 AND status = $2",
			s.monitorID, expected,
		).Scan(&count)
		return err == nil && count > 0
	})
}

func (s *dashboardSteps) stepCheckHistoryContainsError(ctx context.Context, expected string) error {
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var count int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM check_history WHERE monitor_id = $1 AND error_message = $2",
			s.monitorID, expected,
		).Scan(&count)
		return err == nil && count > 0
	})
}

func (s *dashboardSteps) stepIncidentActiveExists(ctx context.Context) error {
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var count int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM incidents WHERE id = $1 AND status = 'ACTIVE'", s.incidentID,
		).Scan(&count)
		return err == nil && count > 0
	})
}

func (s *dashboardSteps) stepIncidentResolved(ctx context.Context) error {
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var count int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM incidents WHERE id = $1 AND ended_at IS NOT NULL", s.incidentID,
		).Scan(&count)
		return err == nil && count > 0
	})
}

// stepPublishStatusEvent публикует событие event с указанным статусом для текущего монитора.
func (s *dashboardSteps) stepPublishStatusEvent(ctx context.Context, event, status string) error {
	payload := map[string]any{
		"event_type": event,
		"monitor_id": s.monitorID,
		"user_id":    s.userID,
		"status":     status,
	}
	// Для upsert-событий (monitor.created/updated) consumer требует user_id + name + url.
	if event == "monitor.created" || event == "monitor.updated" {
		payload["name"] = "BDD Monitor"
		payload["url"] = fmt.Sprintf("https://%s.example.com", s.monitorID[:8])
		payload["timestamp"] = time.Now().Unix()
	}
	return s.stack.PublishEvent(ctx, event, payload)
}

// stepPublishMonitorCreatedWithName публикует monitor.created с заданным именем.
func (s *dashboardSteps) stepPublishMonitorCreatedWithName(ctx context.Context, name string) error {
	payload := map[string]any{
		"event_type": "monitor.created",
		"monitor_id": s.monitorID,
		"user_id":    s.userID,
		"name":       name,
		"url":        fmt.Sprintf("https://%s.example.com", s.monitorID[:8]),
		"timestamp":  time.Now().Unix(),
	}
	return s.stack.PublishEvent(ctx, "monitor.created", payload)
}

// stepDashboardHasMonitorWithName проверяет, что в dashboard есть монитор с указанным именем.
func (s *dashboardSteps) stepDashboardHasMonitorWithName(ctx context.Context, name string) error {
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var dbName string
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT name FROM monitor_statuses WHERE id = $1", s.monitorID,
		).Scan(&dbName)
		return err == nil && dbName == name
	})
}

// stepPublishMalformed публикует невалидный JSON с указанным routing key.
// Consumer должен Nack-нуть сообщение и продолжить работу.
func (s *dashboardSteps) stepPublishMalformed(ctx context.Context, routingKey string) error {
	return s.stack.PublishRaw(ctx, routingKey, []byte(`{not valid json`))
}

// stepSystemHasMonitor allocates a fresh monitor_id для текущего сценария.
// Не создаёт строку в monitor_statuses (для проверок check_history/incidents этого
// достаточно — там нет FK на monitor_statuses).
func (s *dashboardSteps) stepSystemHasMonitor(_ context.Context) error {
	if s.monitorID == "" {
		s.monitorID = uuid.New().String()
	}
	return nil
}

// stepSystemHasNamedMonitor сохраняет monitor_id под меткой label.
func (s *dashboardSteps) stepSystemHasNamedMonitor(_ context.Context, label string) error {
	if _, ok := s.namedMonitors[label]; !ok {
		s.namedMonitors[label] = uuid.New().String()
	}
	return nil
}

// stepSystemHasNamedIncident сохраняет incident_id под меткой label.
// Также создаёт связанный monitor_id для удобства публикации событий.
func (s *dashboardSteps) stepSystemHasNamedIncident(_ context.Context, label string) error {
	if _, ok := s.namedIncidents[label]; !ok {
		s.namedIncidents[label] = uuid.New().String()
	}
	if _, ok := s.namedMonitors[label]; !ok {
		s.namedMonitors[label] = uuid.New().String()
	}
	return nil
}

// stepPublishCheckCompletedWithCodeAndTime публикует check.completed с заданным
// status_code (число) и response_time_ms (float).
func (s *dashboardSteps) stepPublishCheckCompletedWithCodeAndTime(ctx context.Context, code, rt string) error {
	var statusCode int
	if _, err := fmt.Sscanf(code, "%d", &statusCode); err != nil {
		return fmt.Errorf("parse status_code %q: %w", code, err)
	}
	var responseTime float64
	if _, err := fmt.Sscanf(rt, "%f", &responseTime); err != nil {
		return fmt.Errorf("parse response_time_ms %q: %w", rt, err)
	}
	payload := map[string]any{
		"event_type":       "check.completed",
		"monitor_id":       s.monitorID,
		"status":           "UP",
		"status_code":      statusCode,
		"response_time_ms": responseTime,
	}
	return s.stack.PublishEvent(ctx, "check.completed", payload)
}

// stepCheckHistoryFieldsMatch проверяет что в check_history последняя запись
// имеет ожидаемые status_code и response_time_ms.
func (s *dashboardSteps) stepCheckHistoryFieldsMatch(ctx context.Context, expectedCode, expectedRT string) error {
	var wantCode int
	if _, err := fmt.Sscanf(expectedCode, "%d", &wantCode); err != nil {
		return fmt.Errorf("parse expected status_code %q: %w", expectedCode, err)
	}
	var wantRT float64
	if _, err := fmt.Sscanf(expectedRT, "%f", &wantRT); err != nil {
		return fmt.Errorf("parse expected response_time_ms %q: %w", expectedRT, err)
	}
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var code int
		var rt float64
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT status_code, response_time_ms FROM check_history WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT 1",
			s.monitorID,
		).Scan(&code, &rt)
		if err != nil {
			return false
		}
		// Сравнение float через эпсилон
		const eps = 0.01
		diff := rt - wantRT
		if diff < 0 {
			diff = -diff
		}
		return code == wantCode && diff <= eps
	})
}

// stepCheckHistoryErrorMessageEmpty проверяет что error_message в последней
// записи check_history равен NULL.
func (s *dashboardSteps) stepCheckHistoryErrorMessageEmpty(ctx context.Context) error {
	var errMsg *string
	err := s.stack.DashboardDB.QueryRowContext(ctx,
		"SELECT error_message FROM check_history WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT 1",
		s.monitorID,
	).Scan(&errMsg)
	if err != nil {
		return fmt.Errorf("query error_message: %w", err)
	}
	if errMsg != nil {
		return fmt.Errorf("expected error_message to be NULL, got %q", *errMsg)
	}
	return nil
}

// stepCheckHistoryErrorMessageEquals проверяет что error_message в последней
// записи check_history равен ожидаемому значению.
func (s *dashboardSteps) stepCheckHistoryErrorMessageEquals(ctx context.Context, expected string) error {
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var errMsg *string
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT error_message FROM check_history WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT 1",
			s.monitorID,
		).Scan(&errMsg)
		return err == nil && errMsg != nil && *errMsg == expected
	})
}

// stepCheckHistoryStatusCodeEmpty проверяет что status_code в последней записи NULL.
func (s *dashboardSteps) stepCheckHistoryStatusCodeEmpty(ctx context.Context) error {
	var code *int
	err := s.stack.DashboardDB.QueryRowContext(ctx,
		"SELECT status_code FROM check_history WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT 1",
		s.monitorID,
	).Scan(&code)
	if err != nil {
		return fmt.Errorf("query status_code: %w", err)
	}
	if code != nil {
		return fmt.Errorf("expected status_code to be NULL, got %d", *code)
	}
	return nil
}

// stepPublishCheckSequence публикует серию check событий из таблицы.
func (s *dashboardSteps) stepPublishCheckSequence(ctx context.Context, table *godog.Table) error {
	if len(table.Rows) < 2 {
		return fmt.Errorf("expected header row + at least one data row")
	}
	// Первая строка — заголовки.
	header := table.Rows[0]
	colEvent, colStatus := -1, -1
	for i, cell := range header.Cells {
		switch cell.Value {
		case "event_type":
			colEvent = i
		case "status":
			colStatus = i
		}
	}
	if colEvent == -1 || colStatus == -1 {
		return fmt.Errorf("expected columns event_type and status")
	}
	for _, row := range table.Rows[1:] {
		eventType := row.Cells[colEvent].Value
		status := row.Cells[colStatus].Value
		payload := map[string]any{
			"event_type": eventType,
			"monitor_id": s.monitorID,
			"status":     status,
		}
		if err := s.stack.PublishEvent(ctx, eventType, payload); err != nil {
			return fmt.Errorf("publish %s: %w", eventType, err)
		}
	}
	return nil
}

// stepCheckHistoryCountForCurrentMonitor проверяет что количество записей
// check_history для текущего monitorID равно expected.
func (s *dashboardSteps) stepCheckHistoryCountForCurrentMonitor(ctx context.Context, expected string) error {
	var want int
	if _, err := fmt.Sscanf(expected, "%d", &want); err != nil {
		return fmt.Errorf("parse expected count %q: %w", expected, err)
	}
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var got int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM check_history WHERE monitor_id = $1", s.monitorID,
		).Scan(&got)
		return err == nil && got == want
	})
}

// stepCheckHistoryStatusesOrderedEqual проверяет что статусы записей check_history
// в порядке checked_at ASC совпадают со списком expected (через запятую).
func (s *dashboardSteps) stepCheckHistoryStatusesOrderedEqual(ctx context.Context, expected string) error {
	want := splitCSV(expected)
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		rows, err := s.stack.DashboardDB.QueryContext(ctx,
			"SELECT status FROM check_history WHERE monitor_id = $1 ORDER BY checked_at ASC",
			s.monitorID,
		)
		if err != nil {
			return false
		}
		defer func() {
			//nolint:errcheck // close в defer, ошибка не влияет на результат
			_ = rows.Close()
		}()
		var got []string
		for rows.Next() {
			var st string
			if err := rows.Scan(&st); err != nil {
				return false
			}
			got = append(got, st)
		}
		if len(got) != len(want) {
			return false
		}
		for i := range got {
			if got[i] != want[i] {
				return false
			}
		}
		return true
	})
}

// stepPublishCheckCompletedForNamedMonitor публикует check.completed для монитора с меткой label.
func (s *dashboardSteps) stepPublishCheckCompletedForNamedMonitor(ctx context.Context, label, status string) error {
	id, ok := s.namedMonitors[label]
	if !ok {
		return fmt.Errorf("monitor %q not registered", label)
	}
	payload := map[string]any{
		"event_type": "check.completed",
		"monitor_id": id,
		"status":     status,
	}
	return s.stack.PublishEvent(ctx, "check.completed", payload)
}

// stepPublishCheckFailedForNamedMonitor публикует check.failed для монитора с меткой label.
func (s *dashboardSteps) stepPublishCheckFailedForNamedMonitor(ctx context.Context, label, status string) error {
	id, ok := s.namedMonitors[label]
	if !ok {
		return fmt.Errorf("monitor %q not registered", label)
	}
	payload := map[string]any{
		"event_type": "check.failed",
		"monitor_id": id,
		"status":     status,
	}
	return s.stack.PublishEvent(ctx, "check.failed", payload)
}

// stepCheckHistoryCountForNamedMonitor проверяет количество записей check_history
// для монитора с меткой label.
func (s *dashboardSteps) stepCheckHistoryCountForNamedMonitor(ctx context.Context, label, expected string) error {
	id, ok := s.namedMonitors[label]
	if !ok {
		return fmt.Errorf("monitor %q not registered", label)
	}
	var want int
	if _, err := fmt.Sscanf(expected, "%d", &want); err != nil {
		return fmt.Errorf("parse expected count %q: %w", expected, err)
	}
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var got int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM check_history WHERE monitor_id = $1", id,
		).Scan(&got)
		return err == nil && got == want
	})
}

// stepIncidentStartedAtSet проверяет что started_at заполнен для текущего incidentID.
func (s *dashboardSteps) stepIncidentStartedAtSet(ctx context.Context) error {
	var startedAt time.Time
	err := s.stack.DashboardDB.QueryRowContext(ctx,
		"SELECT started_at FROM incidents WHERE id = $1", s.incidentID,
	).Scan(&startedAt)
	if err != nil {
		return fmt.Errorf("query started_at: %w", err)
	}
	if startedAt.IsZero() {
		return fmt.Errorf("expected started_at to be non-zero")
	}
	return nil
}

// stepIncidentEndedAtNull проверяет что ended_at равен NULL.
func (s *dashboardSteps) stepIncidentEndedAtNull(ctx context.Context) error {
	var endedAt *time.Time
	err := s.stack.DashboardDB.QueryRowContext(ctx,
		"SELECT ended_at FROM incidents WHERE id = $1", s.incidentID,
	).Scan(&endedAt)
	if err != nil {
		return fmt.Errorf("query ended_at: %w", err)
	}
	if endedAt != nil {
		return fmt.Errorf("expected ended_at to be NULL, got %v", *endedAt)
	}
	return nil
}

// stepIncidentEndedAtSet проверяет что ended_at заполнен.
func (s *dashboardSteps) stepIncidentEndedAtSet(ctx context.Context) error {
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var endedAt *time.Time
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT ended_at FROM incidents WHERE id = $1", s.incidentID,
		).Scan(&endedAt)
		return err == nil && endedAt != nil
	})
}

// stepIncidentDurationNonNegative проверяет что duration_seconds установлен и >= 0.
func (s *dashboardSteps) stepIncidentDurationNonNegative(ctx context.Context) error {
	var duration *int64
	err := s.stack.DashboardDB.QueryRowContext(ctx,
		"SELECT duration_seconds FROM incidents WHERE id = $1", s.incidentID,
	).Scan(&duration)
	if err != nil {
		return fmt.Errorf("query duration_seconds: %w", err)
	}
	if duration == nil {
		return fmt.Errorf("expected duration_seconds to be set")
	}
	if *duration < 0 {
		return fmt.Errorf("expected duration_seconds >= 0, got %d", *duration)
	}
	return nil
}

// stepPublishIncidentResolvedAfterDelay делает паузу delaySec секунд и публикует incident.resolved.
// Пауза необходима для того, чтобы duration_seconds при разрешении был положительным.
func (s *dashboardSteps) stepPublishIncidentResolvedAfterDelay(ctx context.Context, delay string) error {
	var delaySec int
	if _, err := fmt.Sscanf(delay, "%d", &delaySec); err != nil {
		return fmt.Errorf("parse delay %q: %w", delay, err)
	}
	time.Sleep(time.Duration(delaySec) * time.Second)
	return s.stepPublishIncidentResolved(ctx)
}

// stepPublishIncidentDetectedForNamed публикует incident.detected для инцидента с меткой label.
func (s *dashboardSteps) stepPublishIncidentDetectedForNamed(ctx context.Context, label string) error {
	incidentID, ok := s.namedIncidents[label]
	if !ok {
		return fmt.Errorf("incident %q not registered", label)
	}
	monitorID := s.namedMonitors[label]
	if monitorID == "" {
		monitorID = uuid.New().String()
		s.namedMonitors[label] = monitorID
	}
	payload := map[string]any{
		"event_type":  "incident.detected",
		"incident_id": incidentID,
		"monitor_id":  monitorID,
	}
	return s.stack.PublishEvent(ctx, "incident.detected", payload)
}

// stepPublishIncidentResolvedForNamed публикует incident.resolved для инцидента с меткой label.
func (s *dashboardSteps) stepPublishIncidentResolvedForNamed(ctx context.Context, label string) error {
	incidentID, ok := s.namedIncidents[label]
	if !ok {
		return fmt.Errorf("incident %q not registered", label)
	}
	monitorID := s.namedMonitors[label]
	if monitorID == "" {
		monitorID = uuid.New().String()
		s.namedMonitors[label] = monitorID
	}
	payload := map[string]any{
		"event_type":  "incident.resolved",
		"incident_id": incidentID,
		"monitor_id":  monitorID,
	}
	return s.stack.PublishEvent(ctx, "incident.resolved", payload)
}

// stepIncidentsBothHaveStatus проверяет что оба именованных инцидента имеют ожидаемый статус.
func (s *dashboardSteps) stepIncidentsBothHaveStatus(ctx context.Context, labelA, labelB, expected string) error {
	idA, okA := s.namedIncidents[labelA]
	idB, okB := s.namedIncidents[labelB]
	if !okA || !okB {
		return fmt.Errorf("incidents %q/%q not registered", labelA, labelB)
	}
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var count int
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM incidents WHERE id IN ($1, $2) AND status = $3",
			idA, idB, expected,
		).Scan(&count)
		return err == nil && count == 2
	})
}

// stepNamedIncidentStatusEquals проверяет статус именованного инцидента.
func (s *dashboardSteps) stepNamedIncidentStatusEquals(ctx context.Context, label, expected string) error {
	id, ok := s.namedIncidents[label]
	if !ok {
		return fmt.Errorf("incident %q not registered", label)
	}
	return waitForCondition(15*time.Second, 200*time.Millisecond, func() bool {
		var status string
		err := s.stack.DashboardDB.QueryRowContext(ctx,
			"SELECT status FROM incidents WHERE id = $1", id,
		).Scan(&status)
		return err == nil && status == expected
	})
}

// stepNamedIncidentAbsent проверяет что для именованного инцидента в БД нет записи.
func (s *dashboardSteps) stepNamedIncidentAbsent(ctx context.Context, label string) error {
	id, ok := s.namedIncidents[label]
	if !ok {
		return fmt.Errorf("incident %q not registered", label)
	}
	var count int
	err := s.stack.DashboardDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM incidents WHERE id = $1", id,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("query incident count: %w", err)
	}
	if count != 0 {
		return fmt.Errorf("expected no phantom record for incident %q, got %d", label, count)
	}
	return nil
}

// splitCSV разбивает строку через запятую и обрезает пробелы у элементов.
func splitCSV(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			seg := s[start:i]
			// trim leading/trailing spaces
			j, k := 0, len(seg)
			for j < k && seg[j] == ' ' {
				j++
			}
			for k > j && seg[k-1] == ' ' {
				k--
			}
			out = append(out, seg[j:k])
			start = i + 1
		}
	}
	return out
}

// waitForCondition опрашивает check до timeout и возвращает ошибку если условие
// не выполнилось за отведённое время.
func waitForCondition(timeout, interval time.Duration, check func() bool) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("condition not met within %s", timeout)
}

// registerPendingSteps регистрирует заглушки для остальных шагов feature-файлов.
// Все возвращают godog.ErrPending.
func (s *dashboardSteps) registerPendingSteps(ctx *godog.ScenarioContext) {
	pending := func() error { return godog.ErrPending }
	pending1 := func(_ string) error { return godog.ErrPending }
	pending2 := func(_, _ string) error { return godog.ErrPending }
	pending3 := func(_, _, _ string) error { return godog.ErrPending }
	pendingTable := func(_ *godog.Table) error { return godog.ErrPending }

	// Auth/session/context.
	ctx.Step(`^пользователь авторизован в системе$`, pending)
	ctx.Step(`^пользователь не авторизован в системе$`, pending)
	ctx.Step(`^пользователь авторизован с ролью "([^"]*)"$`, pending1)
	ctx.Step(`^пользователь тенанта "([^"]*)" авторизован в системе$`, pending1)
	ctx.Step(`^тенант "([^"]*)" имеет мониторы$`, pending1)

	// Dashboard UI basics.
	ctx.Step(`^пользователь открывает дашборд$`, pending)
	ctx.Step(`^пользователь пытается открыть дашборд$`, pending)
	ctx.Step(`^пользователь просматривает дашборд$`, pending)
	ctx.Step(`^пользователь запрашивает дашборд$`, pending)
	ctx.Step(`^пользователь тенанта "([^"]*)" запрашивает дашборд$`, pending1)
	ctx.Step(`^отображается структура страницы:$`, pendingTable)
	ctx.Step(`^sidebar содержит разделы:$`, pendingTable)

	// Generic errors and audit.
	ctx.Step(`^возвращается ошибка с кодом "([^"]*)"$`, pending1)
	ctx.Step(`^ответ содержит структуру:$`, pendingTable)
	ctx.Step(`^действие логируется в audit log$`, pending)
	ctx.Step(`^действие логируется в audit log как "([^"]*)"$`, pending1)
	ctx.Step(`^действие логируется в audit log как failed$`, pending)
	ctx.Step(`^действие логируется в audit log:$`, pendingTable)
	ctx.Step(`^попытка доступа логируется в audit log как unauthorized$`, pending)
	ctx.Step(`^попытка логируется в audit log как access_denied$`, pending)
	ctx.Step(`^попытка доступа логируется в audit log как access_denied$`, pending)
	ctx.Step(`^попытка перехвата логируется в audit log как security_breach$`, pending)
	ctx.Step(`^попытка подписки логируется в audit log как access_denied$`, pending)
	ctx.Step(`^попытка подключения логируется в audit log как unauthorized$`, pending)
	ctx.Step(`^timeout логируется в audit log$`, pending)
	ctx.Step(`^ошибка логируется в audit log$`, pending)
	ctx.Step(`^ошибка логируется в audit log как cache_unavailable$`, pending)
	ctx.Step(`^ошибки логируются в audit log для каждого сервиса$`, pending)
	ctx.Step(`^синхронизация логируется в audit log$`, pending)
	ctx.Step(`^экспорт логируется в audit log$`, pending)
	ctx.Step(`^переход логируется в audit log$`, pending)
	ctx.Step(`^throttling логируется для мониторинга производительности$`, pending)
	ctx.Step(`^попытка экспорта логируется в audit log как failed$`, pending)
	ctx.Step(`^попытка восстановления логируется в audit log$`, pending)
	ctx.Step(`^ошибка логируется для анализа$`, pending)
	ctx.Step(`^файл экспорта логируется отдельно с метаданными$`, pending)
	ctx.Step(`^статистика разрывов логируется в audit log$`, pending)
	ctx.Step(`^процесс миграции логируется в audit log$`, pending)
	ctx.Step(`^попытка логируется как security_breach$`, pending)
	ctx.Step(`^log содержит timestamp и IP адрес$`, pending)
	ctx.Step(`^log содержит timestamp и ID монитора$`, pending)
	ctx.Step(`^log содержит timestamp и параметры запроса$`, pending)
	ctx.Step(`^log содержит timestamp$`, pending)
	ctx.Step(`^log содержит длительность сессии$`, pending)
	ctx.Step(`^log содержит предыдущие и новые значения фильтров$`, pending)
	ctx.Step(`^log содержит ID созданного файла$`, pending)

	// Monitors quantities / pagination / filters.
	ctx.Step(`^пользователь имеет мониторы$`, pending)
	ctx.Step(`^пользователь имеет мониторы:$`, pendingTable)
	ctx.Step(`^пользователь имеет мониторы с различными статусами:$`, pendingTable)
	ctx.Step(`^пользователь имеет мониторы с различными статусами и тегами:$`, pendingTable)
	ctx.Step(`^пользователь имеет мониторы с различными тегами$`, pending)
	ctx.Step(`^пользователь имеет "([^"]*)" мониторов$`, pending1)
	ctx.Step(`^пользователь создал ровно "([^"]*)" мониторов$`, pending1)
	ctx.Step(`^пользователь создал "([^"]*)" мониторов$`, pending1)
	ctx.Step(`^пользователь имеет подписку "([^"]*)" с лимитом "([^"]*)" мониторов$`, pending2)
	ctx.Step(`^пользователь имеет подписку "([^"]*)" с безлимитными мониторами$`, pending1)
	ctx.Step(`^пользователь видит "([^"]*)" монитора$`, pending1)
	ctx.Step(`^каждый монитор отображается с визуальными индикаторами:$`, pendingTable)
	ctx.Step(`^DOWN мониторы отображаются первыми в списке$`, pending)
	ctx.Step(`^рассчитывается общий uptime процент$`, pending)
	ctx.Step(`^отображается первая страница с "([^"]*)" мониторами$`, pending1)
	ctx.Step(`^доступна навигация пагинации:$`, pendingTable)
	ctx.Step(`^пользователь выбирает "([^"]*)"$`, pending1)
	ctx.Step(`^отображается "([^"]*)" мониторов на странице$`, pending1)
	ctx.Step(`^номер текущей страницы отображается в UI$`, pending)
	ctx.Step(`^пользователь запрашивает страницу с size="([^"]*)"$`, pending1)
	ctx.Step(`^автоматически применяется page_size="([^"]*)"$`, pending1)
	ctx.Step(`^пользователь применяет фильтр "([^"]*)"$`, pending1)
	ctx.Step(`^пользователь применяет фильтры:$`, pendingTable)
	ctx.Step(`^пользователь применяет фильтр с неверным параметром "([^"]*)"$`, pending1)
	ctx.Step(`^пользователь применяет фильтр с длиной "([^"]*)" символов$`, pending1)
	ctx.Step(`^отображаются только мониторы со статусом "([^"]*)" или "([^"]*)"$`, pending2)
	ctx.Step(`^мониторы со статусом "([^"]*)" исключаются из результатов$`, pending1)
	ctx.Step(`^выбранные фильтры сохраняются на сервере для пользователя$`, pending)
	ctx.Step(`^фильтр обрезается до "([^"]*)" символов$`, pending1)
	ctx.Step(`^возвращается предупреждение "([^"]*)"$`, pending1)
	ctx.Step(`^возвращается пустой список мониторов$`, pending)
	ctx.Step(`^возвращается пустой список проверок$`, pending)
	ctx.Step(`^предлагается изменить критерии фильтрации$`, pending)
	ctx.Step(`^результат кэшируется$`, pending)
	ctx.Step(`^мониторы упорядочены по умолчанию:$`, pendingTable)
	ctx.Step(`^"([^"]*)" \(([^)]*)\) отображается первым$`, pending2)
	ctx.Step(`^"([^"]*)" \(([^)]*)\) отображается вторым$`, pending2)

	// Otros placeholders — registrar como generic pending with capture patterns that
	// approximate remaining steps. Unmatched steps at runtime become "undefined",
	// which is non-fatal with Strict=false.
	_ = pending3
}
