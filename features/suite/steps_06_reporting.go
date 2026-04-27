//go:build bdd

package suite

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	reportingv1 "github.com/raul/monitor/api/proto/reporting"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// reportingSteps реализует шаги для эпика 06_reporting.
// Покрывают валидацию параметров и контроль доступа к SLA-отчётам через
// реальный gRPC reporting-service.
type reportingSteps struct {
	stack *Stack
	state *ScenarioState

	// Per-scenario состояние.
	userID       uuid.UUID
	otherUserID  uuid.UUID
	monitorID    uuid.UUID
	reportID     uuid.UUID
	listResp     *reportingv1.ListSLAReportsResponse
	listErr      error
	generateErr  error
	getErr       error
	seededCount  int
}

// RegisterReportingSteps регистрирует шаги для эпика 06_reporting.
func RegisterReportingSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	s := &reportingSteps{stack: stack, state: state}

	ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
		s.userID = uuid.New()
		s.otherUserID = uuid.New()
		s.monitorID = uuid.New()
		s.reportID = uuid.Nil
		s.listResp = nil
		s.listErr = nil
		s.generateErr = nil
		s.getErr = nil
		s.seededCount = 0
		return c, nil
	})

	// Validation: date range и future period.
	ctx.Step(`^монитор имеет данные$`, s.stepMonitorHasData)
	ctx.Step(`^пользователь генерирует SLA отчёт с "([^"]*)" по "([^"]*)"$`, s.stepGenerateSLAWithDates)
	ctx.Step(`^возвращается ошибка "([^"]*)"$`, s.stepErrorWithCode)

	// List SLA отчётов.
	ctx.Step(`^существует "(\d+)" SLA отчётов$`, s.stepSeedSLAReports)
	ctx.Step(`^пользователь запрашивает список отчётов$`, s.stepListSLAReports)
	ctx.Step(`^возвращается "(\d+)" отчётов$`, s.stepListReturnsCount)
	ctx.Step(`^отчёты отсортированы по дате создания \(сначала новые\)$`, s.stepListSortedDesc)

	// Cross-user access (uc_06_01_04a).
	ctx.Step(`^пользователь "([^"]*)" создал SLA отчёт$`, s.stepUser1CreatedReport)
	ctx.Step(`^пользователь "([^"]*)" авторизован$`, s.stepUser2Authorized)
	ctx.Step(`^пользователь "([^"]*)" пытается просмотреть отчёт$`, s.stepUser2GetsReport)
	// Reuse stepErrorWithCode для FORBIDDEN.
}

// --- Validation steps ---

func (s *reportingSteps) stepMonitorHasData(_ context.Context) error {
	// Нет фактической сидинг-операции: handler reporting-service выполняет
	// validatePeriod до обращения к monitor-service.
	return nil
}

// stepGenerateSLAWithDates вызывает GenerateSLAReport с датами в формате YYYY-MM-DD.
func (s *reportingSteps) stepGenerateSLAWithDates(ctx context.Context, fromStr, toStr string) error {
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return fmt.Errorf("parse from date: %w", err)
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return fmt.Errorf("parse to date: %w", err)
	}

	authCtx := reportingAuthCtx(ctx, s.userID)
	_, s.generateErr = s.stack.ReportingClient.GenerateSLAReport(authCtx, &reportingv1.GenerateSLAReportRequest{
		MonitorId: s.monitorID.String(),
		From:      timestamppb.New(from),
		To:        timestamppb.New(to),
	})
	return nil
}

// stepErrorWithCode проверяет, что любой из последних вызовов вернул gRPC статус с
// кодом, соответствующим ожидаемому бизнес-коду ошибки.
func (s *reportingSteps) stepErrorWithCode(expected string) error {
	err := s.lastError()
	if err == nil {
		return fmt.Errorf("expected error %q, got nil", expected)
	}
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("not a grpc status error: %v", err)
	}

	expectedCode := mapBusinessCodeToGRPC(expected)
	if st.Code() != expectedCode {
		return fmt.Errorf("expected grpc code %s for %q, got %s: %s", expectedCode, expected, st.Code(), st.Message())
	}
	return nil
}

func (s *reportingSteps) lastError() error {
	if s.getErr != nil {
		return s.getErr
	}
	if s.generateErr != nil {
		return s.generateErr
	}
	if s.listErr != nil {
		return s.listErr
	}
	return nil
}

// mapBusinessCodeToGRPC отображает бизнес-коды ошибок reporting-service на
// соответствующие gRPC-коды по логике handler.handleError.
func mapBusinessCodeToGRPC(code string) codes.Code {
	switch code {
	case "INVALID_PERIOD_FORMAT", "INVALID_DATE_RANGE", "PERIOD_EXCEEDS_RETENTION":
		return codes.InvalidArgument
	case "REPORT_NOT_FOUND", "MONITOR_NOT_FOUND":
		return codes.NotFound
	case "FORBIDDEN", "UNAUTHORIZED_ACCESS":
		return codes.PermissionDenied
	case "SERVICE_UNAVAILABLE":
		return codes.Unavailable
	case "TIMEOUT":
		return codes.DeadlineExceeded
	case "CONFLICT":
		return codes.Aborted
	default:
		return codes.Unknown
	}
}

// --- List steps ---

// stepSeedSLAReports создаёт N отчётов в БД reporting-service для текущего пользователя.
func (s *reportingSteps) stepSeedSLAReports(ctx context.Context, n int) error {
	now := time.Now().UTC()
	for i := 0; i < n; i++ {
		id := uuid.New()
		// Каждому отчёту даём уникальный (monitor_id, period_start, period_end), чтобы
		// удовлетворить unique-индекс idx_sla_reports_monitor_period.
		monitorID := uuid.New()
		periodStart := now.Add(-time.Duration(i+1) * 24 * time.Hour)
		periodEnd := now.Add(-time.Duration(i) * 24 * time.Hour)
		// created_at смещаем по индексу, чтобы порядок был стабилен.
		createdAt := now.Add(-time.Duration(i) * time.Minute)

		_, err := s.stack.ReportingDB.ExecContext(ctx, `
			INSERT INTO sla_reports (
				id, monitor_id, user_id, monitor_name, period_start, period_end,
				availability, total_checks, up_checks, down_checks, degraded_checks,
				paused_checks, total_downtime_seconds, incidents_count, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		`,
			id, monitorID, s.userID, fmt.Sprintf("monitor-%d", i),
			periodStart, periodEnd,
			99.5, 100, 99, 1, 0, 0, int64(60), 1, createdAt,
		)
		if err != nil {
			return fmt.Errorf("seed sla report #%d: %w", i, err)
		}
	}
	s.seededCount = n
	return nil
}

func (s *reportingSteps) stepListSLAReports(ctx context.Context) error {
	authCtx := reportingAuthCtx(ctx, s.userID)
	resp, err := s.stack.ReportingClient.ListSLAReports(authCtx, &reportingv1.ListSLAReportsRequest{
		Limit: 100,
	})
	s.listResp = resp
	s.listErr = err
	return nil
}

func (s *reportingSteps) stepListReturnsCount(expected int) error {
	if s.listErr != nil {
		return fmt.Errorf("list returned error: %w", s.listErr)
	}
	if s.listResp == nil {
		return fmt.Errorf("list response is nil")
	}
	if len(s.listResp.Reports) != expected {
		return fmt.Errorf("expected %d reports, got %d", expected, len(s.listResp.Reports))
	}
	return nil
}

func (s *reportingSteps) stepListSortedDesc() error {
	if s.listResp == nil || len(s.listResp.Reports) < 2 {
		return nil
	}
	for i := 1; i < len(s.listResp.Reports); i++ {
		prev := s.listResp.Reports[i-1].CreatedAt.AsTime()
		cur := s.listResp.Reports[i].CreatedAt.AsTime()
		if cur.After(prev) {
			return fmt.Errorf("reports not sorted desc by created_at: report[%d]=%s after report[%d]=%s",
				i, cur, i-1, prev)
		}
	}
	return nil
}

// --- Cross-user access ---

func (s *reportingSteps) stepUser1CreatedReport(ctx context.Context, _ string) error {
	// Создаём отчёт, принадлежащий другому пользователю.
	s.reportID = uuid.New()
	monitorID := uuid.New()
	now := time.Now().UTC()
	_, err := s.stack.ReportingDB.ExecContext(ctx, `
		INSERT INTO sla_reports (
			id, monitor_id, user_id, monitor_name, period_start, period_end,
			availability, total_checks, up_checks, down_checks, degraded_checks,
			paused_checks, total_downtime_seconds, incidents_count, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`,
		s.reportID, monitorID, s.otherUserID, "user1-monitor",
		now.Add(-24*time.Hour), now,
		99.0, 100, 99, 1, 0, 0, int64(60), 1, now,
	)
	if err != nil {
		return fmt.Errorf("seed user1 report: %w", err)
	}
	return nil
}

func (s *reportingSteps) stepUser2Authorized(_ context.Context, _ string) error {
	// userID уже сгенерирован в Before; именно он играет роль user2.
	return nil
}

func (s *reportingSteps) stepUser2GetsReport(ctx context.Context, _ string) error {
	authCtx := reportingAuthCtx(ctx, s.userID)
	_, s.getErr = s.stack.ReportingClient.GetSLAReport(authCtx, &reportingv1.GetSLAReportRequest{
		ReportId: s.reportID.String(),
	})
	return nil
}
