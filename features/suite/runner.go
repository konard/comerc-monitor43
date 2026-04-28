//go:build bdd

package suite

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/google/uuid"
)

var bddOptions = godog.Options{
	Format: "pretty",
	Strict: false,
}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &bddOptions)
}

// RunEpic запускает godog-сюиту для эпика, лежащего рядом с вызывающим bdd_test.go.
// Определяет имя эпика и путь к .feature-файлам из пути вызывающего файла.
// Шаги выбираются по имени эпика автоматически.
func RunEpic(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("bdd: skipping in short mode")
	}

	_, callerFile, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("BDD: не удалось определить путь вызывающего файла")
	}
	epicDir := filepath.Dir(callerFile)
	epicName := filepath.Base(epicDir)

	ctx := context.Background()

	stack, err := StartStack(ctx)
	if err != nil {
		t.Fatalf("BDD stack: %v", err)
	}
	defer func() {
		//nolint:errcheck // cleanup в defer
		_ = stack.Stop(ctx)
	}()

	options := bddOptions
	if format := os.Getenv("BDD_GODOG_FORMAT"); format != "" {
		options.Format = format
	}
	options.Paths = []string{epicDir}
	options.Output = colors.Colored(os.Stdout)
	options.TestingT = t

	suite := godog.TestSuite{
		Name:    "bdd/" + epicName,
		Options: &options,
		ScenarioInitializer: func(gctx *godog.ScenarioContext) {
			state := &ScenarioState{}
			gctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
				state.UserEmail = fmt.Sprintf("user-%s@test.example.com", uuid.New().String()[:8])
				state.AuthResp = nil
				state.LastErr = nil
				if len(epicName) >= 2 && epicName[:2] == "02" {
					if err := stack.CleanAlertDB(); err != nil {
						return c, err
					}
					stack.ResetAlertFakes()
				}
				if len(epicName) >= 2 && epicName[:2] == "04" {
					if err := stack.CleanBillingDB(); err != nil {
						return c, err
					}
				}
				if len(epicName) >= 2 && epicName[:2] == "01" {
					if err := stack.CleanMonitorDB(); err != nil {
						return c, err
					}
				}
				if len(epicName) >= 2 && epicName[:2] == "05" {
					if err := stack.CleanAuthDB(); err != nil {
						return c, err
					}
				}
				if len(epicName) >= 2 && epicName[:2] == "09" {
					if err := stack.CleanSchedulerDB(); err != nil {
						return c, err
					}
				}
				if len(epicName) >= 2 && epicName[:2] == "07" {
					if err := stack.CleanIntegrationDB(); err != nil {
						return c, err
					}
				}
				if len(epicName) >= 2 && epicName[:2] == "06" {
					if err := stack.CleanReportingDB(); err != nil {
						return c, err
					}
				}
				return c, nil
			})
			registerEpicSteps(gctx, epicName, stack, state)
		},
	}
	if suite.Run() != 0 {
		t.Fatal("BDD suite failed")
	}
}

// registerEpicSteps регистрирует шаги для конкретного эпика.
//
// Stub-шаги (RegisterStub*Steps) регистрируются ПОСЛЕ доменных, чтобы
// доменные реализации имели приоритет при совпадении regex. См.
// experiments/gen_stubs.py для генерации stub-файлов и docs/BDD_STUB_NOTES.md
// для контракта по их постепенному вытеснению реальными реализациями.
func registerEpicSteps(ctx *godog.ScenarioContext, epicName string, stack *Stack, state *ScenarioState) {
	switch {
	case len(epicName) >= 2 && epicName[:2] == "01":
		RegisterCommonAuthSteps(ctx, state)
		RegisterMonitoringSteps(ctx, stack, state)
		RegisterMonitoringPipelineSteps(ctx, stack, state)
		RegisterStub01MonitoringSteps(ctx)
	case len(epicName) >= 2 && epicName[:2] == "02":
		RegisterCommonAuthSteps(ctx, state)
		RegisterAlertingAuthSteps(ctx, stack, state)
		RegisterAlertingChannelSteps(ctx, stack, state)
		RegisterAlertingAlertSteps(ctx, stack, state)
		RegisterAlertingDeliverySteps(ctx, stack, state)
		RegisterAlertingContextSteps(ctx, stack, state)
		RegisterStub02AlertingSteps(ctx)
	case len(epicName) >= 2 && epicName[:2] == "03":
		RegisterDashboardSteps(ctx, stack, state)
		RegisterStub03DashboardSteps(ctx)
	case len(epicName) >= 2 && epicName[:2] == "04":
		RegisterBillingSteps(ctx, stack, state)
		RegisterStub04BillingSteps(ctx)
	case len(epicName) >= 2 && epicName[:2] == "05":
		RegisterSecuritySteps(ctx, stack, state)
		RegisterTeamSteps(ctx, stack, state)
		RegisterStub05SecuritySteps(ctx)
	case len(epicName) >= 2 && epicName[:2] == "06":
		RegisterReportingSteps(ctx, stack, state)
		RegisterStub06ReportingSteps(ctx)
	case len(epicName) >= 2 && epicName[:2] == "07":
		RegisterIntegrationsSteps(ctx, stack, state)
		RegisterStub07IntegrationsSteps(ctx)
	case len(epicName) >= 2 && epicName[:2] == "08":
		RegisterMaintenanceSteps(ctx, stack, state)
		RegisterStub08MaintenanceSteps(ctx)
	case len(epicName) >= 2 && epicName[:2] == "09":
		RegisterSchedulerSteps(ctx, stack, state)
		RegisterStub09SchedulerSteps(ctx)
	case len(epicName) >= 2 && epicName[:2] == "10":
		RegisterCheckWorkerSteps(ctx, stack, state)
		RegisterStub10CheckWorkerSteps(ctx)
	default:
		// Для новых эпиков добавляй case выше.
	}
}
