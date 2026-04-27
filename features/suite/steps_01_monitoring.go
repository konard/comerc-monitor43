//go:build bdd

package suite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	monitorv1 "github.com/raul/monitor/api/proto/monitor/v1"
)

// RegisterMonitoringSteps регистрирует шаги для эпика 01_monitoring.
// Покрывает 3 legacy-сценария: uc_01_01_01 (создание), uc_01_01_02 (получение),
// uc_01_01_05 (удаление). Остальные сценарии в .feature остаются undefined
// и отображаются godog как pending (runner работает в non-strict режиме).
func RegisterMonitoringSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	ensureMonitoringMaps(state)

	ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
		state.MonitorTier = "Free"
		state.LastMonitor = nil
		state.Monitors = make(map[uuid.UUID]*MonitorInfo)
		state.MonitorsByName = make(map[string]*MonitorInfo)
		state.MonitorList = nil
		state.MultipleCreateErrs = nil
		return c, nil
	})

	// === Контекст ===

	ctx.Step(`^пользователь аутентифицирован с тиром "([^"]*)"$`, func(tier string) error {
		if state.UserID == uuid.Nil {
			state.UserID = uuid.New()
		}
		state.MonitorTier = normalizeTier(tier)
		return nil
	})

	// === Create ===

	ctx.Step(`^пользователь создаёт монитор с параметрами:$`, func(table *godog.Table) error {
		return createMonitorFromTable(stack, state, table)
	})

	ctx.Step(`^монитор создан успешно$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected successful creation, got error: %w", state.LastErr)
		}
		if state.LastMonitor == nil {
			return fmt.Errorf("last monitor is nil")
		}
		if state.LastMonitor.ID == uuid.Nil {
			return fmt.Errorf("monitor ID is empty")
		}
		return nil
	})

	ctx.Step(`^монитор имеет статус "([^"]*)"$`, func(expected string) error {
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor in state")
		}
		if !strings.EqualFold(state.LastMonitor.Status, expected) {
			return fmt.Errorf("expected status %q, got %q", expected, state.LastMonitor.Status)
		}
		return nil
	})

	ctx.Step(`^монитор имеет имя "([^"]*)"$`, func(expected string) error {
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor in state")
		}
		if state.LastMonitor.Name != expected {
			return fmt.Errorf("expected name %q, got %q", expected, state.LastMonitor.Name)
		}
		return nil
	})

	ctx.Step(`^монитор имеет URL "([^"]*)"$`, func(expected string) error {
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor in state")
		}
		if state.LastMonitor.URL != expected {
			return fmt.Errorf("expected url %q, got %q", expected, state.LastMonitor.URL)
		}
		return nil
	})

	// === Get ===

	ctx.Step(`^монитор существует с параметрами:$`, func(table *godog.Table) error {
		if err := createMonitorFromTable(stack, state, table); err != nil {
			return err
		}
		if state.LastErr != nil {
			return fmt.Errorf("create existing monitor: %w", state.LastErr)
		}
		if state.LastMonitor == nil {
			return fmt.Errorf("create existing monitor: monitor is nil")
		}
		return nil
	})

	ctx.Step(`^пользователь запрашивает информацию о мониторе$`, func() error {
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor ID available")
		}
		c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), state.UserID, state.MonitorTier), 5*time.Second)
		defer cancel()

		resp, err := stack.MonitorClient.GetMonitor(c, &monitorv1.GetMonitorRequest{
			Id: state.LastMonitor.ID.String(),
		})
		if err != nil {
			state.LastErr = err
			return nil
		}
		state.LastErr = nil
		state.LastMonitor = monitorFromResponse(resp.Monitor)
		state.Monitors[state.LastMonitor.ID] = state.LastMonitor
		return nil
	})

	ctx.Step(`^информация о мониторе получена успешно$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected successful retrieval, got error: %w", state.LastErr)
		}
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor in state")
		}
		return nil
	})

	// === Delete ===

	ctx.Step(`^пользователь удаляет монитор$`, func() error {
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor ID available")
		}
		c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), state.UserID, state.MonitorTier), 5*time.Second)
		defer cancel()

		_, err := stack.MonitorClient.DeleteMonitor(c, &monitorv1.DeleteMonitorRequest{
			Id: state.LastMonitor.ID.String(),
		})
		if err != nil {
			state.LastErr = err
			return nil
		}
		state.LastErr = nil
		return nil
	})

	ctx.Step(`^монитор удалён успешно$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected successful deletion, got error: %w", state.LastErr)
		}
		return nil
	})

	ctx.Step(`^получена ошибка "([^"]*)"$`, func(expected string) error {
		if state.LastErr == nil {
			return fmt.Errorf("expected error %q, got nil", expected)
		}
		if !strings.Contains(strings.ToLower(state.LastErr.Error()), strings.ToLower(expected)) {
			return fmt.Errorf("expected error containing %q, got: %v", expected, state.LastErr)
		}
		return nil
	})

	// === Update ===

	ctx.Step(`^пользователь обновляет монитор с параметрами:$`, func(table *godog.Table) error {
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor ID available for update")
		}
		req := &monitorv1.UpdateMonitorRequest{Id: state.LastMonitor.ID.String()}
		// Pre-fill from existing monitor to avoid empty fields if .feature не передаёт всё.
		req.Name = state.LastMonitor.Name
		req.Url = state.LastMonitor.URL
		req.IntervalSeconds = state.LastMonitor.IntervalSeconds
		req.TimeoutSeconds = state.LastMonitor.TimeoutSeconds
		for _, row := range table.Rows {
			if len(row.Cells) < 2 {
				continue
			}
			key := row.Cells[0].Value
			val := row.Cells[1].Value
			switch key {
			case "name":
				req.Name = val
			case "url":
				req.Url = val
			case "interval":
				var n int32
				if _, err := fmt.Sscanf(val, "%d", &n); err != nil {
					return fmt.Errorf("invalid interval %q: %w", val, err)
				}
				req.IntervalSeconds = n
			case "timeout":
				var n int32
				if _, err := fmt.Sscanf(val, "%d", &n); err != nil {
					return fmt.Errorf("invalid timeout %q: %w", val, err)
				}
				req.TimeoutSeconds = n
			}
		}

		c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), state.UserID, state.MonitorTier), 5*time.Second)
		defer cancel()

		resp, err := stack.MonitorClient.UpdateMonitor(c, req)
		if err != nil {
			state.LastErr = err
			return nil
		}
		state.LastErr = nil
		state.LastMonitor = monitorFromResponse(resp.Monitor)
		state.Monitors[state.LastMonitor.ID] = state.LastMonitor
		return nil
	})

	ctx.Step(`^монитор обновлён успешно$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected successful update, got error: %w", state.LastErr)
		}
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor in state")
		}
		return nil
	})

	// === Get by explicit ID ===

	ctx.Step(`^пользователь запрашивает монитор с ID "([^"]*)"$`, func(id string) error {
		c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), ensureUserID(state), state.MonitorTier), 5*time.Second)
		defer cancel()

		resp, err := stack.MonitorClient.GetMonitor(c, &monitorv1.GetMonitorRequest{Id: id})
		if err != nil {
			state.LastErr = err
			state.LastMonitor = nil
			return nil
		}
		state.LastErr = nil
		state.LastMonitor = monitorFromResponse(resp.Monitor)
		return nil
	})

	// === List ===

	ctx.Step(`^пользователь запрашивает список мониторов$`, func() error {
		c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), ensureUserID(state), state.MonitorTier), 5*time.Second)
		defer cancel()

		resp, err := stack.MonitorClient.ListMonitors(c, &monitorv1.ListMonitorsRequest{Limit: 100})
		if err != nil {
			state.LastErr = err
			state.MonitorList = nil
			return nil
		}
		state.LastErr = nil
		state.MonitorList = resp.Monitors
		return nil
	})

	ctx.Step(`^список мониторов пуст$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected successful list, got error: %w", state.LastErr)
		}
		if len(state.MonitorList) != 0 {
			return fmt.Errorf("expected empty list, got %d monitors", len(state.MonitorList))
		}
		return nil
	})

	ctx.Step(`^список мониторов содержит "(\d+)" монитора$`, func(expected int) error {
		if state.LastErr != nil {
			return fmt.Errorf("expected successful list, got error: %w", state.LastErr)
		}
		if len(state.MonitorList) != expected {
			return fmt.Errorf("expected %d monitors in list, got %d", expected, len(state.MonitorList))
		}
		return nil
	})

	ctx.Step(`^список мониторов содержит все созданные мониторы$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected successful list, got error: %w", state.LastErr)
		}
		found := make(map[string]bool, len(state.MonitorList))
		for _, m := range state.MonitorList {
			found[m.Id] = true
		}
		for id := range state.Monitors {
			if !found[id.String()] {
				return fmt.Errorf("monitor %s missing from list", id)
			}
		}
		return nil
	})

	// === Pause / Resume ===

	ctx.Step(`^пользователь приостанавливает монитор$`, func() error {
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor ID available")
		}
		c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), state.UserID, state.MonitorTier), 5*time.Second)
		defer cancel()

		_, err := stack.MonitorClient.PauseMonitor(c, &monitorv1.PauseMonitorRequest{Id: state.LastMonitor.ID.String()})
		if err != nil {
			state.LastErr = err
			return nil
		}
		state.LastErr = nil
		return nil
	})

	ctx.Step(`^монитор приостановлен успешно$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected successful pause, got error: %w", state.LastErr)
		}
		return nil
	})

	ctx.Step(`^пользователь возобновляет монитор$`, func() error {
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor ID available")
		}
		c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), state.UserID, state.MonitorTier), 5*time.Second)
		defer cancel()

		_, err := stack.MonitorClient.ResumeMonitor(c, &monitorv1.ResumeMonitorRequest{Id: state.LastMonitor.ID.String()})
		if err != nil {
			state.LastErr = err
			return nil
		}
		state.LastErr = nil
		return nil
	})

	ctx.Step(`^монитор возобновлён успешно$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected successful resume, got error: %w", state.LastErr)
		}
		return nil
	})

	ctx.Step(`^статус монитора отличается от "([^"]*)"$`, func(unexpected string) error {
		if state.LastMonitor == nil {
			return fmt.Errorf("no monitor in state")
		}
		if strings.EqualFold(state.LastMonitor.Status, unexpected) {
			return fmt.Errorf("expected status to differ from %q, got %q", unexpected, state.LastMonitor.Status)
		}
		return nil
	})

	// === Multiple create ===

	ctx.Step(`^пользователь создаёт мониторы с именами:$`, func(table *godog.Table) error {
		if state.UserID == uuid.Nil {
			state.UserID = uuid.New()
		}
		if state.MonitorTier == "" {
			state.MonitorTier = "Free"
		}
		state.MultipleCreateErrs = state.MultipleCreateErrs[:0]
		for _, row := range table.Rows {
			if len(row.Cells) < 1 {
				continue
			}
			name := row.Cells[0].Value
			if name == "" {
				continue
			}
			req := &monitorv1.CreateMonitorRequest{
				Name:            name,
				Url:             "https://" + shortRandHex() + ".example.com",
				CheckType:       "HTTP",
				IntervalSeconds: 60,
				TimeoutSeconds:  30,
			}
			c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), state.UserID, state.MonitorTier), 5*time.Second)
			resp, err := stack.MonitorClient.CreateMonitor(c, req)
			cancel()
			if err != nil {
				state.MultipleCreateErrs = append(state.MultipleCreateErrs, err)
				continue
			}
			info := monitorFromResponse(resp.Monitor)
			state.LastMonitor = info
			state.Monitors[info.ID] = info
			state.MonitorsByName[info.Name] = info
		}
		return nil
	})

	ctx.Step(`^все мониторы созданы успешно$`, func() error {
		if len(state.MultipleCreateErrs) > 0 {
			return fmt.Errorf("got %d errors during bulk create, first: %v",
				len(state.MultipleCreateErrs), state.MultipleCreateErrs[0])
		}
		return nil
	})
}

// ensureUserID гарантирует наличие UserID в состоянии для запросов без явной аутентификации.
func ensureUserID(state *ScenarioState) uuid.UUID {
	if state.UserID == uuid.Nil {
		state.UserID = uuid.New()
	}
	return state.UserID
}

// shortRandHex возвращает короткий случайный hex-идентификатор для уникальных URL.
func shortRandHex() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
}

// ensureMonitoringMaps лениво инициализирует map-поля состояния для эпика monitoring.
func ensureMonitoringMaps(state *ScenarioState) {
	if state.Monitors == nil {
		state.Monitors = make(map[uuid.UUID]*MonitorInfo)
	}
	if state.MonitorsByName == nil {
		state.MonitorsByName = make(map[string]*MonitorInfo)
	}
}

// normalizeTier приводит тир из .feature к формату, который ожидает monitor-service.
func normalizeTier(tier string) string {
	switch strings.ToLower(tier) {
	case "free", "":
		return "Free"
	case "starter":
		return "Starter"
	case "pro":
		return "Pro"
	case "enterprise":
		return "Enterprise"
	default:
		// Возвращаем ввод с заглавной первой буквой.
		if tier == "" {
			return "Free"
		}
		return strings.ToUpper(tier[:1]) + strings.ToLower(tier[1:])
	}
}

// createMonitorFromTable парсит godog-таблицу с параметрами и вызывает CreateMonitor.
func createMonitorFromTable(stack *Stack, state *ScenarioState, table *godog.Table) error {
	req := &monitorv1.CreateMonitorRequest{}
	for _, row := range table.Rows {
		if len(row.Cells) < 2 {
			continue
		}
		key := row.Cells[0].Value
		val := row.Cells[1].Value
		switch key {
		case "name":
			req.Name = val
		case "url":
			req.Url = val
		case "check_type":
			req.CheckType = strings.ToUpper(val)
		case "interval":
			var n int32
			if _, err := fmt.Sscanf(val, "%d", &n); err != nil {
				return fmt.Errorf("invalid interval %q: %w", val, err)
			}
			req.IntervalSeconds = n
		case "timeout":
			var n int32
			if _, err := fmt.Sscanf(val, "%d", &n); err != nil {
				return fmt.Errorf("invalid timeout %q: %w", val, err)
			}
			req.TimeoutSeconds = n
		}
	}

	if state.UserID == uuid.Nil {
		state.UserID = uuid.New()
	}
	if state.MonitorTier == "" {
		state.MonitorTier = "Free"
	}

	c, cancel := context.WithTimeout(monitorAuthCtx(context.Background(), state.UserID, state.MonitorTier), 5*time.Second)
	defer cancel()

	resp, err := stack.MonitorClient.CreateMonitor(c, req)
	if err != nil {
		state.LastErr = err
		return nil
	}
	state.LastErr = nil
	info := monitorFromResponse(resp.Monitor)
	state.LastMonitor = info
	state.Monitors[info.ID] = info
	state.MonitorsByName[info.Name] = info
	return nil
}

// monitorFromResponse конвертирует protobuf Monitor в локальный MonitorInfo.
func monitorFromResponse(m *monitorv1.Monitor) *MonitorInfo {
	if m == nil {
		return nil
	}
	info := &MonitorInfo{
		Name:            m.Name,
		URL:             m.Url,
		CheckType:       m.CheckType,
		IntervalSeconds: m.IntervalSeconds,
		TimeoutSeconds:  m.TimeoutSeconds,
		Status:          m.Status,
	}
	if id, err := uuid.Parse(m.Id); err == nil {
		info.ID = id
	}
	if uid, err := uuid.Parse(m.UserId); err == nil {
		info.UserID = uid
	}
	if m.CreatedAt != nil {
		info.CreatedAt = m.CreatedAt.AsTime()
	}
	return info
}
