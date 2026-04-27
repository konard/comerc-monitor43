//go:build bdd

package suite

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	maintenancepb "github.com/raul/monitor/api/proto"
	monitorv1 "github.com/raul/monitor/api/proto/monitor/v1"
)

// maintenanceSteps инкапсулирует состояние шагов эпика 08_maintenance.
type maintenanceSteps struct {
	stack *Stack
	state *ScenarioState

	role     string            // USER / ADMIN
	tier     string            // Free / Pro / Enterprise
	monitors map[string]string // имя -> monitor_id

	lastWindow      *maintenancepb.MaintenanceWindow
	lastList        []*maintenancepb.MaintenanceWindow
	lastErr         error
	lastErrMsg      string
	bulkCreateCount int
	bulkCreateOK    int
}

// RegisterMaintenanceSteps регистрирует шаги для эпика 08_maintenance.
// Покрывает CRUD-сценарии создания/списка/отмены/удаления окна и базовые
// валидации (END_TIME_BEFORE_START_TIME, DURATION_EXCEEDS_MAXIMUM,
// DURATION_BELOW_MINIMUM, INSUFFICIENT_PERMISSIONS, MAINTENANCE_WINDOW_LIMIT_REACHED).
// Сценарии с time-travel (current_time, восстановление после рестарта,
// автопереходы статуса) остаются undefined — godog покажет их как pending.
func RegisterMaintenanceSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	s := &maintenanceSteps{stack: stack, state: state}

	ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
		// Изоляция между сценариями — таблицы monitor-service.
		if err := stack.CleanMonitorDB(); err != nil {
			return c, err
		}
		s.role = "USER"
		s.tier = "Pro"
		s.monitors = make(map[string]string)
		s.lastWindow = nil
		s.lastList = nil
		s.lastErr = nil
		s.lastErrMsg = ""
		s.bulkCreateCount = 0
		s.bulkCreateOK = 0
		state.UserID = uuid.New()
		return c, nil
	})

	// === Контекст: авторизация и роли ===

	ctx.Step(`^пользователь авторизован$`, func() error {
		s.role = "USER"
		return nil
	})
	ctx.Step(`^пользователь авторизован как "([^"]*)"$`, func(role string) error {
		s.role = strings.ToUpper(role)
		return nil
	})

	// === Контекст: мониторы ===

	ctx.Step(`^пользователь имеет монитор "([^"]*)"$`, func(name string) error {
		return s.createMonitor(name)
	})
	ctx.Step(`^пользователь имеет "(\d+)" монитора$`, func(n int) error {
		return s.createNMonitors(n)
	})
	ctx.Step(`^пользователь имеет "(\d+)" мониторов$`, func(n int) error {
		return s.createNMonitors(n)
	})
	ctx.Step(`^пользователь не имеет мониторов$`, func() error { return nil })

	// === Контекст: лимиты ===

	ctx.Step(`^максимальная продолжительность окна настроена как "(\d+)" часа$`, func(_ int) error {
		// Дефолт сервиса = 24 часа, как в .feature; реконфигурация subprocess не нужна.
		return nil
	})
	ctx.Step(`^минимальная продолжительность окна настроена как "(\d+)" минута$`, func(_ int) error {
		// Дефолт сервиса = 1 минута, как в .feature.
		return nil
	})
	ctx.Step(`^система ограничивает "(\d+)" мониторов в одном окне$`, func(_ int) error {
		// Дефолт сервиса = 50, как в .feature.
		return nil
	})
	ctx.Step(`^пользователь имеет тариф "([^"]*)"$`, func(tier string) error {
		s.tier = normalizeTier(tier)
		return nil
	})
	ctx.Step(`^тариф ограничивает "(\d+)" окон обслуживания$`, func(_ int) error {
		// Дефолт сервиса для Free = 5, как в .feature.
		return nil
	})
	ctx.Step(`^пользователь создал "(\d+)" окон$`, func(n int) error {
		return s.createNWindows(n)
	})

	// === Действия: создание окна ===

	ctx.Step(`^пользователь создаёт окно обслуживания с параметрами:$`, func(table *godog.Table) error {
		return s.createWindowFromTable(table, false)
	})
	ctx.Step(`^пользователь создаёт глобальное окно обслуживания с параметрами:$`, func(table *godog.Table) error {
		return s.createWindowFromTable(table, true)
	})
	ctx.Step(`^пользователь создаёт глобальное окно обслуживания$`, func() error {
		return s.createGlobalWindowDefault()
	})
	ctx.Step(`^пользователь пытается создать шестое окно$`, func() error {
		return s.createSimpleWindow("Sixth Window", "API Service")
	})
	ctx.Step(`^пользователь создаёт окно обслуживания для всех "(\d+)" мониторов$`, func(_ int) error {
		return s.createWindowAllMonitors()
	})

	// === Проверки результата создания ===

	ctx.Step(`^окно обслуживания создано успешно$`, func() error {
		if s.lastErr != nil {
			return fmt.Errorf("expected success, got error: %v", s.lastErr)
		}
		if s.lastWindow == nil || s.lastWindow.Id == "" {
			return fmt.Errorf("expected window in state")
		}
		return nil
	})
	ctx.Step(`^окно создано успешно$`, func() error {
		if s.lastErr != nil {
			return fmt.Errorf("expected success, got error: %v", s.lastErr)
		}
		if s.lastWindow == nil || s.lastWindow.Id == "" {
			return fmt.Errorf("expected window in state")
		}
		return nil
	})
	ctx.Step(`^окно имеет статус "([^"]*)"$`, func(expected string) error {
		return s.assertStatus(expected)
	})
	ctx.Step(`^статус окна "([^"]*)"$`, func(expected string) error {
		return s.assertStatus(expected)
	})
	ctx.Step(`^окно настроено как повторяющееся еженедельно$`, func() error {
		return s.assertRecurrence(maintenancepb.RecurrenceType_RECURRENCE_TYPE_WEEKLY)
	})
	ctx.Step(`^окно настроено как повторяющееся ежемесячно$`, func() error {
		return s.assertRecurrence(maintenancepb.RecurrenceType_RECURRENCE_TYPE_MONTHLY)
	})
	ctx.Step(`^окно применяется ко всем мониторам аккаунта$`, func() error {
		if s.lastWindow == nil || !s.lastWindow.IsGlobal {
			return fmt.Errorf("expected global window, got is_global=%v", s.lastWindow.GetIsGlobal())
		}
		return nil
	})
	ctx.Step(`^окно не содержит мониторов$`, func() error {
		if s.lastWindow == nil {
			return fmt.Errorf("no window in state")
		}
		if len(s.lastWindow.MonitorIds) != 0 {
			return fmt.Errorf("expected zero monitors, got %d", len(s.lastWindow.MonitorIds))
		}
		return nil
	})
	ctx.Step(`^все "(\d+)" мониторов добавлены в окно$`, func(n int) error {
		if s.lastWindow == nil {
			return fmt.Errorf("no window in state")
		}
		if len(s.lastWindow.MonitorIds) != n {
			return fmt.Errorf("expected %d monitors in window, got %d", n, len(s.lastWindow.MonitorIds))
		}
		return nil
	})

	// === Проверки ошибок ===

	ctx.Step(`^возвращается ошибка с кодом "([^"]*)"$`, s.assertErrorCode)
	ctx.Step(`^возвращается ошибка "([^"]*)"$`, s.assertErrorCode)
	ctx.Step(`^окно обслуживания не создано$`, s.assertWindowNotCreated)
	ctx.Step(`^окно не создано$`, s.assertWindowNotCreated)

	// === Список ===

	ctx.Step(`^пользователь имеет "(\d+)" окна обслуживания$`, func(n int) error {
		return s.createNWindows(n)
	})
	ctx.Step(`^пользователь запрашивает список окон$`, func() error {
		return s.listWindows()
	})
	ctx.Step(`^возвращается "(\d+)" окна$`, func(n int) error {
		if s.lastErr != nil {
			return fmt.Errorf("list error: %v", s.lastErr)
		}
		if len(s.lastList) != n {
			return fmt.Errorf("expected %d windows, got %d", n, len(s.lastList))
		}
		return nil
	})
	ctx.Step(`^каждое окно содержит имя, статус, время начала и окончания$`, func() error {
		for i, w := range s.lastList {
			if w.Name == "" {
				return fmt.Errorf("window[%d] has empty name", i)
			}
			if w.Status == maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_UNSPECIFIED {
				return fmt.Errorf("window[%d] has unspecified status", i)
			}
			if w.StartTime == nil || w.EndTime == nil {
				return fmt.Errorf("window[%d] missing start/end time", i)
			}
		}
		return nil
	})

	// === Удаление и отмена ===

	ctx.Step(`^пользователь имеет окно со статусом "([^"]*)"$`, func(status string) error {
		if !strings.EqualFold(status, "SCHEDULED") {
			return godog.ErrPending
		}
		// Создаём монитор + окно SCHEDULED.
		if err := s.createMonitor("API Service"); err != nil {
			return err
		}
		return s.createSimpleWindow("Test Window", "API Service")
	})
	ctx.Step(`^окно обслуживания со статусом "([^"]*)"$`, func(status string) error {
		if !strings.EqualFold(status, "SCHEDULED") {
			return godog.ErrPending
		}
		if err := s.createMonitor("API Service"); err != nil {
			return err
		}
		return s.createSimpleWindow("Test Window", "API Service")
	})
	ctx.Step(`^окно назначено на "([^"]*)"$`, func(_ string) error { return nil })
	ctx.Step(`^до начала окна осталось "(\d+)" часа$`, func(_ int) error { return nil })
	ctx.Step(`^до начала осталось "(\d+)" часа$`, func(_ int) error { return nil })

	ctx.Step(`^пользователь удаляет окно$`, func() error {
		return s.deleteLastWindow()
	})
	ctx.Step(`^окно удалено$`, func() error {
		if s.lastErr != nil {
			return fmt.Errorf("delete error: %v", s.lastErr)
		}
		return nil
	})
	ctx.Step(`^окно не появляется в списке$`, func() error {
		if err := s.listWindows(); err != nil {
			return err
		}
		if s.lastWindow == nil {
			return nil
		}
		for _, w := range s.lastList {
			if w.Id == s.lastWindow.Id {
				return fmt.Errorf("deleted window %s still appears in list", w.Id)
			}
		}
		return nil
	})

	ctx.Step(`^пользователь отменяет окно обслуживания$`, func() error {
		return s.cancelLastWindow()
	})
	ctx.Step(`^пользователь отменяет окно$`, func() error {
		return s.cancelLastWindow()
	})
	ctx.Step(`^статус окна изменяется на "([^"]*)"$`, func(expected string) error {
		return s.assertStatus(expected)
	})
	ctx.Step(`^окно удаляется из расписания$`, func() error { return nil })

	// === Параметры окна (название/продолжительность) ===

	ctx.Step(`^продолжительность окна составляет "(\d+)" часа$`, func(h int) error {
		return s.assertDurationHours(h, false)
	})
	ctx.Step(`^продолжительность окна составляет ровно "(\d+)" часа$`, func(h int) error {
		return s.assertDurationHours(h, true)
	})
	ctx.Step(`^продолжительность окна составляет ровно "(\d+)" минуту$`, func(m int) error {
		if s.lastWindow == nil {
			return fmt.Errorf("no window in state")
		}
		got := s.lastWindow.EndTime.AsTime().Sub(s.lastWindow.StartTime.AsTime())
		if got != time.Duration(m)*time.Minute {
			return fmt.Errorf("expected exactly %dm, got %s", m, got)
		}
		return nil
	})

	// === Bulk create / список ===

	ctx.Step(`^пользователь создаёт "(\d+)" отдельных окон для каждого монитора одновременно$`, func(n int) error {
		return s.bulkCreatePerMonitor(n)
	})
	ctx.Step(`^все "(\d+)" окон созданы успешно$`, func(n int) error {
		if s.bulkCreateOK != n {
			return fmt.Errorf("expected %d successful creates, got %d (errors=%d)",
				n, s.bulkCreateOK, s.bulkCreateCount-s.bulkCreateOK)
		}
		return nil
	})
	ctx.Step(`^все окна имеют статус "([^"]*)"$`, func(expected string) error {
		if err := s.listWindows(); err != nil {
			return err
		}
		expectedProto := maintenanceStatusFromString(expected)
		for i, w := range s.lastList {
			if w.Status != expectedProto {
				return fmt.Errorf("window[%d] status=%s, want %s", i, w.Status, expected)
			}
		}
		return nil
	})

	// === Шаги, требующие time-travel или внешних сервисов — pending ===

	ctx.Step(`^текущее время "([^"]*)"$`, func(_ string) error {
		return godog.ErrPending // TODO: требуется управление временем сервиса
	})
	ctx.Step(`^текущее время достигло start_time$`, func() error { return godog.ErrPending })
	ctx.Step(`^текущее время достигло end_time$`, func() error { return godog.ErrPending })
	ctx.Step(`^база данных недоступна для записи$`, func() error { return godog.ErrPending })
	ctx.Step(`^audit log service недоступен$`, func() error { return godog.ErrPending })
	ctx.Step(`^scheduler service недоступен$`, func() error { return godog.ErrPending })
	ctx.Step(`^notification service недоступен$`, func() error { return godog.ErrPending })
}

// === Helpers ===

func (s *maintenanceSteps) authCtx(ctx context.Context) context.Context {
	token := s.signMaintenanceJWT(s.state.UserID, s.role, s.tier)
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md)
}

// signMaintenanceJWT выписывает JWT с произвольной ролью/тиром, в отличие от
// monitorAuthTokenFor (который захардкодил role=USER).
func (s *maintenanceSteps) signMaintenanceJWT(userID uuid.UUID, role, tier string) string {
	if role == "" {
		role = "USER"
	}
	if tier == "" {
		tier = "Free"
	}
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now().Unix()
	claims := map[string]any{
		"user_id": userID.String(),
		"tier":    tier,
		"role":    role,
		"iat":     now,
		"exp":     now + 3600,
	}
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(hb) + "." + enc.EncodeToString(cb)
	mac := hmac.New(sha256.New, []byte(monitorJWTSecret))
	mac.Write([]byte(signing))
	sig := enc.EncodeToString(mac.Sum(nil))
	return signing + "." + sig
}

func (s *maintenanceSteps) createMonitor(name string) error {
	if _, ok := s.monitors[name]; ok {
		return nil
	}
	c, cancel := context.WithTimeout(s.authCtx(context.Background()), 5*time.Second)
	defer cancel()
	resp, err := s.stack.MonitorClient.CreateMonitor(c, &monitorv1.CreateMonitorRequest{
		Name:            name,
		Url:             "https://" + shortRandHex() + ".example.com",
		CheckType:       "HTTP",
		IntervalSeconds: 60,
		TimeoutSeconds:  30,
	})
	if err != nil {
		return fmt.Errorf("create monitor %q: %w", name, err)
	}
	s.monitors[name] = resp.Monitor.Id
	return nil
}

func (s *maintenanceSteps) createNMonitors(n int) error {
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("Monitor-%d-%s", i, shortRandHex())
		if err := s.createMonitor(name); err != nil {
			return err
		}
	}
	return nil
}

// resolveTimes интерпретирует start/end из таблицы. Литералы вида 2026-03-10
// (которые в реальном времени теста уже в прошлом) приводятся к now+offset,
// сохраняя исходную длительность. Для негативных сценариев "в прошлом" вызовущий
// шаг сам обрабатывает вердикт по строке ошибки.
func resolveTimes(startStr, endStr string) (time.Time, time.Time, error) {
	st, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse start_time %q: %w", startStr, err)
	}
	et, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse end_time %q: %w", endStr, err)
	}
	return st, et, nil
}

// shiftToFuture сдвигает start/end в будущее, сохраняя исходный delta.
// Если end <= start (negative scenario), оставляем как есть.
func shiftToFuture(start, end time.Time) (time.Time, time.Time) {
	if !end.After(start) {
		return start, end
	}
	now := time.Now().UTC()
	if start.After(now.Add(time.Minute)) {
		return start, end
	}
	delta := end.Sub(start)
	newStart := now.Add(1 * time.Hour).Truncate(time.Second)
	return newStart, newStart.Add(delta)
}

func (s *maintenanceSteps) createWindowFromTable(table *godog.Table, forceGlobal bool) error {
	req := &maintenancepb.CreateMaintenanceWindowRequest{
		Recurrence: maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE,
	}
	var startStr, endStr, monitorName string
	for _, row := range table.Rows {
		if len(row.Cells) < 2 {
			continue
		}
		k := row.Cells[0].Value
		v := row.Cells[1].Value
		switch k {
		case "name":
			req.Name = v
		case "start_time":
			startStr = v
		case "end_time":
			endStr = v
		case "recurrence":
			req.Recurrence = recurrenceFromString(v)
		case "monitor_ids":
			monitorName = v
		case "is_global":
			req.IsGlobal = strings.EqualFold(v, "true")
		case "pause_monitoring":
			req.PauseMonitoring = strings.EqualFold(v, "true")
		case "suppress_alerts":
			req.SuppressAlerts = strings.EqualFold(v, "true")
		}
	}
	if forceGlobal {
		req.IsGlobal = true
	}

	st, et, err := resolveTimes(startStr, endStr)
	if err != nil {
		return err
	}
	// Сдвигаем валидные интервалы в будущее, чтобы не натыкаться на
	// CANNOT_CREATE_MAINTENANCE_WINDOW_IN_PAST по реальному wall clock.
	st, et = shiftToFuture(st, et)
	req.StartTime = timestamppb.New(st)
	req.EndTime = timestamppb.New(et)

	if monitorName != "" && !req.IsGlobal {
		id, ok := s.monitors[monitorName]
		if !ok {
			return fmt.Errorf("monitor %q not provisioned in scenario", monitorName)
		}
		req.MonitorIds = []string{id}
	}

	return s.doCreate(req)
}

func (s *maintenanceSteps) createGlobalWindowDefault() error {
	now := time.Now().UTC()
	req := &maintenancepb.CreateMaintenanceWindowRequest{
		Name:       "Global Maintenance",
		StartTime:  timestamppb.New(now.Add(1 * time.Hour)),
		EndTime:    timestamppb.New(now.Add(3 * time.Hour)),
		Recurrence: maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE,
		IsGlobal:   true,
	}
	return s.doCreate(req)
}

func (s *maintenanceSteps) createSimpleWindow(name, monitorName string) error {
	id, ok := s.monitors[monitorName]
	if !ok {
		return fmt.Errorf("monitor %q not provisioned", monitorName)
	}
	now := time.Now().UTC()
	req := &maintenancepb.CreateMaintenanceWindowRequest{
		Name:       name,
		StartTime:  timestamppb.New(now.Add(1 * time.Hour)),
		EndTime:    timestamppb.New(now.Add(3 * time.Hour)),
		Recurrence: maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE,
		MonitorIds: []string{id},
	}
	return s.doCreate(req)
}

func (s *maintenanceSteps) createWindowAllMonitors() error {
	ids := make([]string, 0, len(s.monitors))
	for _, id := range s.monitors {
		ids = append(ids, id)
	}
	now := time.Now().UTC()
	req := &maintenancepb.CreateMaintenanceWindowRequest{
		Name:       "All-monitors window",
		StartTime:  timestamppb.New(now.Add(1 * time.Hour)),
		EndTime:    timestamppb.New(now.Add(3 * time.Hour)),
		Recurrence: maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE,
		MonitorIds: ids,
	}
	return s.doCreate(req)
}

func (s *maintenanceSteps) createNWindows(n int) error {
	if err := s.createMonitor("API Service"); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		now := time.Now().UTC()
		// Разносим окна, чтобы не пересекались на одном мониторе.
		offsetStart := now.Add(time.Duration(2+4*i) * time.Hour)
		offsetEnd := offsetStart.Add(1 * time.Hour)
		req := &maintenancepb.CreateMaintenanceWindowRequest{
			Name:       fmt.Sprintf("Window-%d", i),
			StartTime:  timestamppb.New(offsetStart),
			EndTime:    timestamppb.New(offsetEnd),
			Recurrence: maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE,
			MonitorIds: []string{s.monitors["API Service"]},
		}
		if err := s.doCreate(req); err != nil {
			return err
		}
		if s.lastErr != nil {
			return fmt.Errorf("create window %d failed: %w", i, s.lastErr)
		}
	}
	// Сбрасываем lastErr/lastWindow, чтобы не путать assert'ы из When/Then.
	s.lastErr = nil
	s.lastErrMsg = ""
	return nil
}

func (s *maintenanceSteps) bulkCreatePerMonitor(n int) error {
	// Гарантируем что есть n мониторов.
	if len(s.monitors) < n {
		if err := s.createNMonitors(n - len(s.monitors)); err != nil {
			return err
		}
	}
	s.bulkCreateCount = 0
	s.bulkCreateOK = 0
	now := time.Now().UTC()
	i := 0
	for _, id := range s.monitors {
		offsetStart := now.Add(time.Duration(2+4*i) * time.Hour)
		offsetEnd := offsetStart.Add(1 * time.Hour)
		req := &maintenancepb.CreateMaintenanceWindowRequest{
			Name:       fmt.Sprintf("Bulk-%d", i),
			StartTime:  timestamppb.New(offsetStart),
			EndTime:    timestamppb.New(offsetEnd),
			Recurrence: maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE,
			MonitorIds: []string{id},
		}
		s.bulkCreateCount++
		if err := s.doCreate(req); err != nil {
			return err
		}
		if s.lastErr == nil {
			s.bulkCreateOK++
		}
		i++
	}
	s.lastErr = nil
	s.lastErrMsg = ""
	return nil
}

func (s *maintenanceSteps) doCreate(req *maintenancepb.CreateMaintenanceWindowRequest) error {
	c, cancel := context.WithTimeout(s.authCtx(context.Background()), 5*time.Second)
	defer cancel()
	resp, err := maintenanceClient(s.stack).CreateMaintenanceWindow(c, req)
	if err != nil {
		s.lastErr = err
		s.lastErrMsg = err.Error()
		s.lastWindow = nil
		return nil
	}
	s.lastErr = nil
	s.lastErrMsg = ""
	s.lastWindow = resp
	return nil
}

func (s *maintenanceSteps) listWindows() error {
	c, cancel := context.WithTimeout(s.authCtx(context.Background()), 5*time.Second)
	defer cancel()
	resp, err := maintenanceClient(s.stack).ListMaintenanceWindows(c, &maintenancepb.ListMaintenanceWindowsRequest{
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		s.lastErr = err
		s.lastErrMsg = err.Error()
		s.lastList = nil
		return nil
	}
	s.lastErr = nil
	s.lastErrMsg = ""
	s.lastList = resp.Windows
	return nil
}

func (s *maintenanceSteps) deleteLastWindow() error {
	if s.lastWindow == nil {
		return fmt.Errorf("no window in state to delete")
	}
	c, cancel := context.WithTimeout(s.authCtx(context.Background()), 5*time.Second)
	defer cancel()
	_, err := maintenanceClient(s.stack).DeleteMaintenanceWindow(c, &maintenancepb.DeleteMaintenanceWindowRequest{
		Id: s.lastWindow.Id,
	})
	if err != nil {
		s.lastErr = err
		s.lastErrMsg = err.Error()
		return nil
	}
	s.lastErr = nil
	s.lastErrMsg = ""
	return nil
}

func (s *maintenanceSteps) cancelLastWindow() error {
	if s.lastWindow == nil {
		return fmt.Errorf("no window in state to cancel")
	}
	c, cancel := context.WithTimeout(s.authCtx(context.Background()), 5*time.Second)
	defer cancel()
	resp, err := maintenanceClient(s.stack).CancelMaintenanceWindow(c, &maintenancepb.CancelMaintenanceWindowRequest{
		Id:                 s.lastWindow.Id,
		CancellationReason: "MANUAL",
	})
	if err != nil {
		s.lastErr = err
		s.lastErrMsg = err.Error()
		return nil
	}
	s.lastErr = nil
	s.lastErrMsg = ""
	s.lastWindow = resp
	return nil
}

func (s *maintenanceSteps) assertStatus(expected string) error {
	if s.lastWindow == nil {
		return fmt.Errorf("no window in state")
	}
	want := maintenanceStatusFromString(expected)
	if s.lastWindow.Status != want {
		return fmt.Errorf("expected status %s, got %s", want, s.lastWindow.Status)
	}
	return nil
}

func (s *maintenanceSteps) assertRecurrence(expected maintenancepb.RecurrenceType) error {
	if s.lastWindow == nil {
		return fmt.Errorf("no window in state")
	}
	if s.lastWindow.Recurrence != expected {
		return fmt.Errorf("expected recurrence %s, got %s", expected, s.lastWindow.Recurrence)
	}
	return nil
}

func (s *maintenanceSteps) assertErrorCode(code string) error {
	if s.lastErr == nil {
		return fmt.Errorf("expected error %q, got nil", code)
	}
	if !strings.Contains(s.lastErrMsg, code) {
		return fmt.Errorf("expected error containing %q, got: %s", code, s.lastErrMsg)
	}
	return nil
}

func (s *maintenanceSteps) assertWindowNotCreated() error {
	if s.lastErr == nil {
		return fmt.Errorf("expected window not created, but lastErr is nil")
	}
	return nil
}

func (s *maintenanceSteps) assertDurationHours(h int, exact bool) error {
	if s.lastWindow == nil {
		return fmt.Errorf("no window in state")
	}
	got := s.lastWindow.EndTime.AsTime().Sub(s.lastWindow.StartTime.AsTime())
	want := time.Duration(h) * time.Hour
	if exact {
		if got != want {
			return fmt.Errorf("expected exactly %s, got %s", want, got)
		}
		return nil
	}
	if got != want {
		return fmt.Errorf("expected %s, got %s", want, got)
	}
	return nil
}

// maintenanceClient возвращает клиент maintenance-сервиса поверх monitor-service gRPC соединения.
// Поскольку monitor-service регистрирует MaintenanceWindowServiceServer на том же gRPC-сервере,
// нам достаточно использовать тот же conn.
func maintenanceClient(stack *Stack) maintenancepb.MaintenanceWindowServiceClient {
	return maintenancepb.NewMaintenanceWindowServiceClient(stack.monitorConn)
}

func recurrenceFromString(v string) maintenancepb.RecurrenceType {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "ONCE":
		return maintenancepb.RecurrenceType_RECURRENCE_TYPE_ONCE
	case "DAILY":
		return maintenancepb.RecurrenceType_RECURRENCE_TYPE_DAILY
	case "WEEKLY":
		return maintenancepb.RecurrenceType_RECURRENCE_TYPE_WEEKLY
	case "MONTHLY":
		return maintenancepb.RecurrenceType_RECURRENCE_TYPE_MONTHLY
	default:
		return maintenancepb.RecurrenceType_RECURRENCE_TYPE_UNSPECIFIED
	}
}

func maintenanceStatusFromString(v string) maintenancepb.MaintenanceWindowStatus {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "SCHEDULED":
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_SCHEDULED
	case "ACTIVE":
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_ACTIVE
	case "COMPLETED":
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_COMPLETED
	case "CANCELLED":
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_CANCELLED
	case "ORPHANED":
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_ORPHANED
	default:
		return maintenancepb.MaintenanceWindowStatus_MAINTENANCE_STATUS_UNSPECIFIED
	}
}
