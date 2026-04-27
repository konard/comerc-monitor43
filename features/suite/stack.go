//go:build bdd

package suite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	amqp "github.com/rabbitmq/amqp091-go"
	authv1 "github.com/raul/monitor/api/proto"
	monitorv1 "github.com/raul/monitor/api/proto/monitor/v1"
	reportingv1 "github.com/raul/monitor/api/proto/reporting"
	dashboardtc "github.com/raul/monitor/backend/dashboard-service/testconsumer"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcrabbitmq "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Stack управляет контейнерами и subprocess-сервисом для BDD тестов.
type Stack struct {
	pgContainer    *tcpostgres.PostgresContainer
	redisContainer *tcredis.RedisContainer
	rmqContainer   *tcrabbitmq.RabbitMQContainer
	rmqHost        string
	rmqPort        int
	fakeOAuth      *FakeOAuthServer
	process        *os.Process
	Client         authv1.AuthServiceClient
	conn           *grpc.ClientConn
	authDB         *sqlx.DB
	AuthJWTSecret  string
	DashboardDB    *sqlx.DB
	cancelConsumer context.CancelFunc

	// Alert-service ресурсы.
	alertProcess        *os.Process
	alertConn           *grpc.ClientConn
	alertDBCleanup      func()
	AlertDB             *sqlx.DB
	AlertClient         authv1.AlertServiceClient
	AlertFakeSMTP       *FakeSMTPServer
	AlertFakeWebhook    *FakeWebhookServer
	AlertFakeTLSWebhook *FakeWebhookTLSServer

	// Billing-service ресурсы.
	billingProcess   *os.Process
	billingConn      *grpc.ClientConn
	billingDBCleanup func()
	BillingDB        *sqlx.DB
	BillingClient    authv1.BillingServiceClient

	// Monitor-service ресурсы.
	monitorProcess   *os.Process
	monitorConn      *grpc.ClientConn
	monitorDBCleanup func()
	MonitorDB        *sqlx.DB
	MonitorClient    monitorv1.MonitorServiceClient

	// Scheduler-service ресурсы.
	schedulerProcess   *os.Process
	schedulerConn      *grpc.ClientConn
	schedulerDBCleanup func()
	SchedulerDB        *sqlx.DB
	SchedulerClient    authv1.SchedulerServiceClient

	// Check-worker subprocess ресурсы.
	checkWorkerProcess *os.Process
	CheckWorkerID      string
	CheckWorkerName    string
	CheckWorkerZone    string

	// Reporting-service ресурсы.
	reportingProcess   *os.Process
	reportingConn      *grpc.ClientConn
	reportingDBCleanup func()
	ReportingDB        *sqlx.DB
	ReportingClient    reportingv1.ReportingServiceClient

	// Integration-service ресурсы.
	integrationProcess   *os.Process
	integrationConn      *grpc.ClientConn
	integrationDBCleanup func()
	IntegrationDB        *sqlx.DB
	IntegrationWebhook   authv1.WebhookIntegrationServiceClient
	IntegrationAPIKey    authv1.APIKeyServiceClient
	IntegrationImport    authv1.ImportServiceClient
}

// StartStack запускает postgres и redis контейнеры, fake OAuth сервер,
// собирает и запускает auth-service как subprocess.
func StartStack(ctx context.Context) (*Stack, error) {
	pgContainer, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithAdditionalWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(pgContainer)
		return nil, fmt.Errorf("start redis container: %w", err)
	}

	rmqContainer, err := tcrabbitmq.Run(ctx, "rabbitmq:3.13-management-alpine",
		tcrabbitmq.WithAdminUsername("guest"),
		tcrabbitmq.WithAdminPassword("guest"),
	)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(pgContainer)
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(redisContainer)
		return nil, fmt.Errorf("start rabbitmq container: %w", err)
	}

	rmqHost, err := rmqContainer.Host(ctx)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(pgContainer)
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(redisContainer)
		//nolint:errcheck // cleanup
		_ = rmqContainer.Terminate(ctx)
		return nil, fmt.Errorf("get rabbitmq host: %w", err)
	}

	rmqMappedPort, err := rmqContainer.MappedPort(ctx, tcrabbitmq.DefaultAMQPPort)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(pgContainer)
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(redisContainer)
		//nolint:errcheck // cleanup
		_ = rmqContainer.Terminate(ctx)
		return nil, fmt.Errorf("get rabbitmq port: %w", err)
	}
	rmqPort := int(rmqMappedPort.Num())

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(pgContainer)
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(redisContainer)
		return nil, fmt.Errorf("get postgres connection string: %w", err)
	}

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(pgContainer)
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(redisContainer)
		return nil, fmt.Errorf("get redis endpoint: %w", err)
	}

	if err := applyMigrations(dsn); err != nil {
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(pgContainer)
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(redisContainer)
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	authSqlxDB, err := sqlx.Open("postgres", dsn)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(pgContainer)
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(redisContainer)
		return nil, fmt.Errorf("open auth db: %w", err)
	}

	fakeOAuth, err := NewFakeOAuthServer()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(pgContainer)
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(redisContainer)
		return nil, fmt.Errorf("start fake oauth server: %w", err)
	}

	cleanup := func() {
		//nolint:errcheck // cleanup
		_ = fakeOAuth.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(pgContainer)
		//nolint:errcheck // cleanup
		_ = testcontainers.TerminateContainer(redisContainer)
		//nolint:errcheck // cleanup
		_ = rmqContainer.Terminate(ctx)
	}

	_, file, _, _ := runtime.Caller(0)
	// file: .../features/suite/stack.go → поднимаемся на два уровня
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")

	binPath, err := buildBinary(ctx, repoRoot)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("build auth-service: %w", err)
	}

	// Выделяем свободный порт для gRPC и отдаём его сервису.
	grpcPort, err := freePort()
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("find free grpc port: %w", err)
	}
	httpPort, err := freePort()
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("find free http port: %w", err)
	}

	// Разбираем хост/порт из redis endpoint (host:port).
	redisHost, redisPortStr, err := net.SplitHostPort(redisEndpoint)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("parse redis endpoint: %w", err)
	}
	redisPort, err := strconv.Atoi(redisPortStr)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("parse redis port: %w", err)
	}

	// Создаём dashboard БД и запускаем consumer.
	rmqURL := fmt.Sprintf("amqp://guest:guest@%s:%d/", rmqHost, rmqPort)
	dashboardDSN, dashboardDSNCleanup, err := createDashboardDB(ctx, pgContainer)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("create dashboard db: %w", err)
	}

	dashDB, err := sqlx.Open("postgres", dashboardDSN)
	if err != nil {
		dashboardDSNCleanup()
		cleanup()
		return nil, fmt.Errorf("open dashboard db: %w", err)
	}

	if err := applyMigrationsTo(dashboardDSN, dashboardMigrationsPath()); err != nil {
		//nolint:errcheck // cleanup
		_ = dashDB.Close()
		dashboardDSNCleanup()
		cleanup()
		return nil, fmt.Errorf("apply dashboard migrations: %w", err)
	}

	dashConsumer := dashboardtc.Build(dashDB, rmqURL)
	consumerCtx, cancelConsumer := context.WithCancel(ctx)

	go func() {
		//nolint:errcheck // ошибки consumer логируются внутри
		_ = dashConsumer.Start(consumerCtx)
	}()

	// Ждём подключения consumer до продолжения.
	waitDeadline := time.Now().Add(15 * time.Second)
	for !dashConsumer.IsConnected() {
		if time.Now().After(waitDeadline) {
			cancelConsumer()
			//nolint:errcheck // cleanup
			_ = dashDB.Close()
			dashboardDSNCleanup()
			cleanup()
			return nil, fmt.Errorf("dashboard consumer did not connect within 15s")
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Разбираем postgres DSN для env-переменных.
	dbHost, dbPortStr, dbName, dbUser, dbPassword, err := parseDSN(dsn)
	if err != nil {
		cancelConsumer()
		//nolint:errcheck // cleanup
		_ = dashDB.Close()
		dashboardDSNCleanup()
		cleanup()
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}

	grpcAddr := fmt.Sprintf("127.0.0.1:%d", grpcPort)

	cmd := exec.CommandContext(ctx, binPath) //nolint:gosec // binPath собран из исходников
	cmd.Dir = filepath.Join(repoRoot, "backend", "auth-service")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SERVER_PORT=%d", httpPort),
		fmt.Sprintf("SERVER_GRPC_PORT=%d", grpcPort),
		fmt.Sprintf("DB_HOST=%s", dbHost),
		fmt.Sprintf("DB_PORT=%s", dbPortStr),
		fmt.Sprintf("DB_NAME=%s", dbName),
		fmt.Sprintf("DB_USER=%s", dbUser),
		fmt.Sprintf("DB_PASSWORD=%s", dbPassword),
		"DB_SSL_MODE=disable",
		fmt.Sprintf("REDIS_HOST=%s", redisHost),
		fmt.Sprintf("REDIS_PORT=%d", redisPort),
		"RABBITMQ_ENABLED=false",
		"METRICS_ENABLED=false",
		"JWT_SECRET=test-secret-for-bdd-tests-minimum32chars",
		"GOOGLE_CLIENT_ID=fake-client-id",
		"GOOGLE_CLIENT_SECRET=fake-client-secret",
		"GOOGLE_REDIRECT_URI=http://localhost/callback",
		fmt.Sprintf("GOOGLE_TOKEN_URL=%s", fakeOAuth.TokenURL()),
		fmt.Sprintf("GOOGLE_USERINFO_URL=%s", fakeOAuth.UserInfoURL()),
		"OTEL_EXPORTER_OTLP_ENDPOINT=",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cleanup()
		return nil, fmt.Errorf("start auth-service process: %w", err)
	}

	conn, err := waitForGRPC(ctx, grpcAddr, cmd, "auth-service")
	if err != nil {
		//nolint:errcheck // cleanup
		_ = cmd.Process.Kill()
		cleanup()
		return nil, fmt.Errorf("wait for grpc ready: %w", err)
	}

	// === Alert-service ===

	alertCleanup := func() {
		//nolint:errcheck // cleanup
		_ = cmd.Process.Kill()
		cancelConsumer()
		//nolint:errcheck // cleanup
		_ = dashDB.Close()
		dashboardDSNCleanup()
		//nolint:errcheck // cleanup
		_ = conn.Close()
		cleanup()
	}

	alertDSN, alertDBCleanup, err := createAlertDB(ctx, pgContainer)
	if err != nil {
		alertCleanup()
		return nil, fmt.Errorf("create alert db: %w", err)
	}

	alertDB, err := sqlx.Open("postgres", alertDSN)
	if err != nil {
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("open alert db: %w", err)
	}

	if err := applyAlertFKStubs(alertDSN); err != nil {
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("apply alert fk stubs: %w", err)
	}

	// Миграции alert-service применяются самим сервисом из main.go (goose.Up
	// с таблицей версий goose_alert_version). FK-стабы уже созданы выше — этого
	// достаточно, чтобы CREATE FOREIGN KEY на users(id)/monitors(id) прошёл.

	fakeSMTP, err := NewFakeSMTPServer()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("start fake smtp: %w", err)
	}

	fakeWebhook, err := NewFakeWebhookServer()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = fakeSMTP.Stop()
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("start fake webhook: %w", err)
	}

	fakeTLSWebhook, err := NewFakeWebhookTLSServer()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = fakeWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeSMTP.Stop()
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("start fake tls webhook: %w", err)
	}

	alertHost, alertPortStr, alertName, alertUser, alertPassword, err := parseDSN(alertDSN)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = fakeTLSWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeSMTP.Stop()
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("parse alert dsn: %w", err)
	}

	alertBin, err := buildAlertBinary(ctx, repoRoot)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = fakeTLSWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeSMTP.Stop()
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("build alert-service: %w", err)
	}

	alertGRPCPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = fakeTLSWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeSMTP.Stop()
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("find free alert grpc port: %w", err)
	}

	alertMetricsPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = fakeTLSWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeSMTP.Stop()
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("find free alert metrics port: %w", err)
	}

	alertAddr := fmt.Sprintf("127.0.0.1:%d", alertGRPCPort)

	alertCmd := exec.CommandContext(ctx, alertBin) //nolint:gosec // alertBin собран из исходников
	// alert-service main.go применяет goose.Up из ./migrations; мы уже их применили,
	// но goose.Up идемпотентен. Достаточно указать cwd на alert-service.
	alertCmd.Dir = alertServiceDir()
	alertCmd.Env = append(os.Environ(),
		fmt.Sprintf("SERVER_PORT=%d", alertGRPCPort),
		"SERVER_HOST=127.0.0.1",
		fmt.Sprintf("METRICS_PORT=%d", alertMetricsPort),
		fmt.Sprintf("DB_HOST=%s", alertHost),
		fmt.Sprintf("DB_PORT=%s", alertPortStr),
		fmt.Sprintf("DB_NAME=%s", alertName),
		fmt.Sprintf("DB_USER=%s", alertUser),
		fmt.Sprintf("DB_PASSWORD=%s", alertPassword),
		"DB_SSL_MODE=disable",
		"JWT_SECRET=test-secret-for-bdd-tests-minimum32chars",
		fmt.Sprintf("SMTP_HOST=%s", fakeSMTP.Host()),
		fmt.Sprintf("SMTP_PORT=%d", fakeSMTP.Port()),
		"SMTP_FROM=alerts@bdd.test",
		"TELEGRAM_BOT_TOKEN=fake-telegram-token",
		"OTEL_EXPORTER_OTLP_ENDPOINT=",
		"GRPC_REFLECTION=true",
		// Сжатые тайминги для BDD: алерт-сервис должен реагировать в миллисекундах,
		// а не в реальных 15/30 минутах.
		"ALERT_COOLDOWN_PERIOD=200ms",
		"ALERT_RATE_LIMIT_PERIOD=200ms",
		"ALERT_RATE_LIMIT_WINDOW=200ms",
		"ALERT_RATE_LIMIT_RETRY_DELAY=200ms",
		"ALERT_STORM_WINDOW=500ms",
		"ALERT_STORM_DURATION=500ms",
		"ESCALATION_TIMEOUT=500ms",
		"FLAPPING_WINDOW=500ms",
		"FLAPPING_EXIT_PERIOD=500ms",
		"ALERT_RETRY_BACKOFF_BASE=50ms",
		"ALERT_RETRY_MAX_DELAY=500ms",
		"DELIVERY_RETRY_INTERVAL=200ms",
	)
	alertCmd.Stdout = os.Stdout
	alertCmd.Stderr = os.Stderr

	if err := alertCmd.Start(); err != nil {
		//nolint:errcheck // cleanup
		_ = fakeTLSWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeSMTP.Stop()
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("start alert-service process: %w", err)
	}

	alertConn, err := waitForGRPC(ctx, alertAddr, alertCmd, "alert-service")
	if err != nil {
		//nolint:errcheck // cleanup
		_ = alertCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = fakeTLSWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeSMTP.Stop()
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
		return nil, fmt.Errorf("wait for alert grpc ready: %w", err)
	}

	// Параметр fakeWebhook сейчас используется только тестовыми шагами — сервис о нём
	// узнаёт только через alert_channels. Сохраняем в Stack для доступа из шагов.
	_ = fakeWebhook

	// === Billing-service ===

	fakeYookassa := NewFakeYookassaServer()

	billingCleanup := func() {
		fakeYookassa.Stop()
		//nolint:errcheck // cleanup
		_ = alertCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = alertConn.Close()
		//nolint:errcheck // cleanup
		_ = fakeTLSWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeWebhook.Stop(ctx)
		//nolint:errcheck // cleanup
		_ = fakeSMTP.Stop()
		//nolint:errcheck // cleanup
		_ = alertDB.Close()
		alertDBCleanup()
		alertCleanup()
	}

	billingDSN, billingDBCleanup, err := createBillingDB(ctx, pgContainer)
	if err != nil {
		billingCleanup()
		return nil, fmt.Errorf("create billing db: %w", err)
	}

	billingDB, err := sqlx.Open("postgres", billingDSN)
	if err != nil {
		billingDBCleanup()
		billingCleanup()
		return nil, fmt.Errorf("open billing db: %w", err)
	}

	if err := applyMigrationsTo(billingDSN, billingMigrationsPath()); err != nil {
		//nolint:errcheck // cleanup
		_ = billingDB.Close()
		billingDBCleanup()
		billingCleanup()
		return nil, fmt.Errorf("apply billing migrations: %w", err)
	}

	billingHost, billingPortStr, billingName, billingUser, billingPassword, err := parseDSN(billingDSN)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = billingDB.Close()
		billingDBCleanup()
		billingCleanup()
		return nil, fmt.Errorf("parse billing dsn: %w", err)
	}

	billingBin, err := buildBillingBinary(ctx, repoRoot)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = billingDB.Close()
		billingDBCleanup()
		billingCleanup()
		return nil, fmt.Errorf("build billing-service: %w", err)
	}

	billingGRPCPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = billingDB.Close()
		billingDBCleanup()
		billingCleanup()
		return nil, fmt.Errorf("find free billing grpc port: %w", err)
	}

	billingHTTPPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = billingDB.Close()
		billingDBCleanup()
		billingCleanup()
		return nil, fmt.Errorf("find free billing http port: %w", err)
	}

	billingMetricsPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = billingDB.Close()
		billingDBCleanup()
		billingCleanup()
		return nil, fmt.Errorf("find free billing metrics port: %w", err)
	}

	billingAddr := fmt.Sprintf("127.0.0.1:%d", billingGRPCPort)

	//nolint:gosec // billingBin собран из исходников репо
	billingCmd := exec.CommandContext(ctx, billingBin)
	// billing-service main.go применяет goose.Up из ./migrations; мы уже их применили,
	// но goose.Up идемпотентен. Достаточно указать cwd на billing-service.
	billingCmd.Dir = billingServiceDir()
	billingCmd.Env = append(os.Environ(),
		fmt.Sprintf("SERVER_PORT=%d", billingHTTPPort),
		fmt.Sprintf("SERVER_GRPC_PORT=%d", billingGRPCPort),
		fmt.Sprintf("METRICS_PORT=%d", billingMetricsPort),
		"METRICS_ENABLED=false",
		fmt.Sprintf("DB_HOST=%s", billingHost),
		fmt.Sprintf("DB_PORT=%s", billingPortStr),
		fmt.Sprintf("DB_NAME=%s", billingName),
		fmt.Sprintf("DB_USER=%s", billingUser),
		fmt.Sprintf("DB_PASSWORD=%s", billingPassword),
		"DB_SSL_MODE=disable",
		"JWT_SECRET="+billingJWTSecret,
		"AUTH_SERVICE_GRPC_ADDRESS="+grpcAddr,
		fmt.Sprintf("RABBITMQ_URL=%s", rmqURL),
		"RABBITMQ_EXCHANGE=billing.events",
		"OTEL_ENABLED=false",
		"OTEL_EXPORTER_OTLP_ENDPOINT=",
		// Сжатые тайминги для BDD, чтобы grace/trial-проверки не требовали долгих ожиданий.
		"DEFAULT_TRIAL_PERIOD_DAYS=7",
		"SUBSCRIPTION_GRACE_PERIOD_DAYS=3",
		// Минимальные стабы для billing config валидации (32+ символов и URL).
		"YOOKASSA_SHOP_ID=test-shop-bdd",
		"YOOKASSA_SECRET_KEY=test-secret-key-for-bdd-min-32-chars-long",
		"YOOKASSA_WEBHOOK_SECRET="+bddYookassaWebhookSecret,
		"YOOKASSA_RETURN_URL=https://bdd.test/return",
		"YOOKASSA_BASE_URL="+fakeYookassa.BaseURL(),
		"STRIPE_API_KEY=sk_test_bdd_minimum_32_chars_long_xxx",
	)
	billingCmd.Stdout = os.Stdout
	billingCmd.Stderr = os.Stderr

	if err := billingCmd.Start(); err != nil {
		//nolint:errcheck // cleanup
		_ = billingDB.Close()
		billingDBCleanup()
		billingCleanup()
		return nil, fmt.Errorf("start billing-service process: %w", err)
	}

	billingConn, err := waitForGRPC(ctx, billingAddr, billingCmd, "billing-service")
	if err != nil {
		//nolint:errcheck // cleanup
		_ = billingCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = billingDB.Close()
		billingDBCleanup()
		billingCleanup()
		return nil, fmt.Errorf("wait for billing grpc ready: %w", err)
	}

	// === Monitor-service ===

	monitorCleanup := func() {
		//nolint:errcheck // cleanup
		_ = billingCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = billingConn.Close()
		//nolint:errcheck // cleanup
		_ = billingDB.Close()
		billingDBCleanup()
		billingCleanup()
	}

	monitorDSN, monitorDBCleanup, err := createMonitorDB(ctx, pgContainer)
	if err != nil {
		monitorCleanup()
		return nil, fmt.Errorf("create monitor db: %w", err)
	}

	monitorDB, err := sqlx.Open("postgres", monitorDSN)
	if err != nil {
		monitorDBCleanup()
		monitorCleanup()
		return nil, fmt.Errorf("open monitor db: %w", err)
	}

	if err := applyMigrationsTo(monitorDSN, monitorMigrationsPath()); err != nil {
		//nolint:errcheck // cleanup
		_ = monitorDB.Close()
		monitorDBCleanup()
		monitorCleanup()
		return nil, fmt.Errorf("apply monitor migrations: %w", err)
	}

	monitorHost, monitorPortStr, monitorName, monitorUser, monitorPassword, err := parseDSN(monitorDSN)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = monitorDB.Close()
		monitorDBCleanup()
		monitorCleanup()
		return nil, fmt.Errorf("parse monitor dsn: %w", err)
	}

	monitorBin, err := buildMonitorBinary(ctx, repoRoot)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = monitorDB.Close()
		monitorDBCleanup()
		monitorCleanup()
		return nil, fmt.Errorf("build monitor-service: %w", err)
	}

	monitorGRPCPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = monitorDB.Close()
		monitorDBCleanup()
		monitorCleanup()
		return nil, fmt.Errorf("find free monitor grpc port: %w", err)
	}
	monitorHTTPPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = monitorDB.Close()
		monitorDBCleanup()
		monitorCleanup()
		return nil, fmt.Errorf("find free monitor http port: %w", err)
	}
	monitorMetricsPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = monitorDB.Close()
		monitorDBCleanup()
		monitorCleanup()
		return nil, fmt.Errorf("find free monitor metrics port: %w", err)
	}

	monitorAddr := fmt.Sprintf("127.0.0.1:%d", monitorGRPCPort)

	monitorCmd := exec.CommandContext(ctx, monitorBin) //nolint:gosec // monitorBin собран из исходников
	monitorCmd.Dir = filepath.Join(repoRoot, "backend", "monitor-service")
	monitorCmd.Env = append(os.Environ(),
		fmt.Sprintf("SERVER_PORT=%d", monitorHTTPPort),
		fmt.Sprintf("SERVER_GRPC_PORT=%d", monitorGRPCPort),
		fmt.Sprintf("METRICS_PORT=%d", monitorMetricsPort),
		"METRICS_ENABLED=false",
		fmt.Sprintf("DB_HOST=%s", monitorHost),
		fmt.Sprintf("DB_PORT=%s", monitorPortStr),
		fmt.Sprintf("DB_NAME=%s", monitorName),
		fmt.Sprintf("DB_USER=%s", monitorUser),
		fmt.Sprintf("DB_PASSWORD=%s", monitorPassword),
		"DB_SSL_MODE=disable",
		"RABBITMQ_ENABLED=false",
		"RATE_LIMIT_ENABLED=false",
		"JWT_SECRET="+monitorJWTSecret,
		"AUTH_SERVICE_GRPC_ADDRESS="+grpcAddr,
		"OTEL_EXPORTER_OTLP_ENDPOINT=",
	)
	monitorCmd.Stdout = os.Stdout
	monitorCmd.Stderr = os.Stderr

	if err := monitorCmd.Start(); err != nil {
		//nolint:errcheck // cleanup
		_ = monitorDB.Close()
		monitorDBCleanup()
		monitorCleanup()
		return nil, fmt.Errorf("start monitor-service process: %w", err)
	}

	monitorConn, err := waitForGRPC(ctx, monitorAddr, monitorCmd, "monitor-service")
	if err != nil {
		//nolint:errcheck // cleanup
		_ = monitorCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = monitorDB.Close()
		monitorDBCleanup()
		monitorCleanup()
		return nil, fmt.Errorf("wait for monitor grpc ready: %w", err)
	}

	// === Scheduler-service ===

	schedulerCleanup := func() {
		//nolint:errcheck // cleanup
		_ = monitorCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = monitorConn.Close()
		//nolint:errcheck // cleanup
		_ = monitorDB.Close()
		monitorDBCleanup()
		monitorCleanup()
	}

	schedulerDSN, schedulerDBCleanup, err := createSchedulerDB(ctx, pgContainer)
	if err != nil {
		schedulerCleanup()
		return nil, fmt.Errorf("create scheduler db: %w", err)
	}

	schedulerDB, err := sqlx.Open("postgres", schedulerDSN)
	if err != nil {
		schedulerDBCleanup()
		schedulerCleanup()
		return nil, fmt.Errorf("open scheduler db: %w", err)
	}

	schedulerHost, schedulerPortStr, schedulerName, schedulerUser, schedulerPassword, err := parseDSN(schedulerDSN)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = schedulerDB.Close()
		schedulerDBCleanup()
		schedulerCleanup()
		return nil, fmt.Errorf("parse scheduler dsn: %w", err)
	}

	schedulerBin, err := buildSchedulerBinary(ctx, repoRoot)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = schedulerDB.Close()
		schedulerDBCleanup()
		schedulerCleanup()
		return nil, fmt.Errorf("build scheduler-service: %w", err)
	}

	schedulerGRPCPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = schedulerDB.Close()
		schedulerDBCleanup()
		schedulerCleanup()
		return nil, fmt.Errorf("find free scheduler grpc port: %w", err)
	}
	schedulerHTTPPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = schedulerDB.Close()
		schedulerDBCleanup()
		schedulerCleanup()
		return nil, fmt.Errorf("find free scheduler http port: %w", err)
	}
	schedulerMetricsPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = schedulerDB.Close()
		schedulerDBCleanup()
		schedulerCleanup()
		return nil, fmt.Errorf("find free scheduler metrics port: %w", err)
	}

	schedulerAddr := fmt.Sprintf("127.0.0.1:%d", schedulerGRPCPort)

	schedulerCmd := exec.CommandContext(ctx, schedulerBin) //nolint:gosec // schedulerBin собран из исходников
	schedulerCmd.Dir = schedulerServiceDir()
	schedulerCmd.Env = append(os.Environ(),
		fmt.Sprintf("SERVER_PORT=%d", schedulerHTTPPort),
		fmt.Sprintf("SERVER_GRPC_PORT=%d", schedulerGRPCPort),
		fmt.Sprintf("METRICS_PORT=%d", schedulerMetricsPort),
		"METRICS_ENABLED=false",
		fmt.Sprintf("DB_HOST=%s", schedulerHost),
		fmt.Sprintf("DB_PORT=%s", schedulerPortStr),
		fmt.Sprintf("DB_NAME=%s", schedulerName),
		fmt.Sprintf("DB_USER=%s", schedulerUser),
		fmt.Sprintf("DB_PASSWORD=%s", schedulerPassword),
		"DB_SSL_MODE=disable",
		"RABBITMQ_ENABLED=false",
		"JWT_SECRET="+monitorJWTSecret,
		"MONITOR_SERVICE_GRPC_ADDRESS="+monitorAddr,
		"OTEL_EXPORTER_OTLP_ENDPOINT=",
	)
	schedulerCmd.Stdout = os.Stdout
	schedulerCmd.Stderr = os.Stderr

	if err := schedulerCmd.Start(); err != nil {
		//nolint:errcheck // cleanup
		_ = schedulerDB.Close()
		schedulerDBCleanup()
		schedulerCleanup()
		return nil, fmt.Errorf("start scheduler-service process: %w", err)
	}

	schedulerConn, err := waitForGRPC(ctx, schedulerAddr, schedulerCmd, "scheduler-service")
	if err != nil {
		//nolint:errcheck // cleanup
		_ = schedulerCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = schedulerDB.Close()
		schedulerDBCleanup()
		schedulerCleanup()
		return nil, fmt.Errorf("wait for scheduler grpc ready: %w", err)
	}

	// === Check-worker subprocess ===

	checkWorkerCleanup := func() {
		//nolint:errcheck // cleanup
		_ = schedulerCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = schedulerConn.Close()
		//nolint:errcheck // cleanup
		_ = schedulerDB.Close()
		schedulerDBCleanup()
		schedulerCleanup()
	}

	checkWorkerBin, err := buildCheckWorkerBinary(ctx, repoRoot)
	if err != nil {
		checkWorkerCleanup()
		return nil, fmt.Errorf("build check-worker: %w", err)
	}

	checkWorkerMetricsPort, err := freePort()
	if err != nil {
		checkWorkerCleanup()
		return nil, fmt.Errorf("find free check-worker metrics port: %w", err)
	}

	checkWorkerName := fmt.Sprintf("worker-bdd-%d", randSuffix())
	checkWorkerZone := "bdd"

	//nolint:gosec // checkWorkerBin собран из исходников
	checkWorkerCmd := exec.CommandContext(ctx, checkWorkerBin)
	checkWorkerCmd.Dir = filepath.Join(repoRoot, "backend", "check-worker")
	checkWorkerCmd.Env = append(os.Environ(),
		fmt.Sprintf("CHECK_WORKER_SCHEDULER_ADDRESS=%s", schedulerAddr),
		fmt.Sprintf("CHECK_WORKER_MONITOR_ADDRESS=%s", monitorAddr),
		fmt.Sprintf("CHECK_WORKER_NAME=%s", checkWorkerName),
		fmt.Sprintf("CHECK_WORKER_ZONE=%s", checkWorkerZone),
		"CHECK_WORKER_MAX_CONCURRENT_CHECKS=10",
		"CHECK_WORKER_CHECK_TIMEOUT=10s",
		"CHECK_WORKER_POLL_INTERVAL=500ms",
		"CHECK_WORKER_HEARTBEAT_INTERVAL=1s",
		"CHECK_WORKER_METRICS_ENABLED=false",
		fmt.Sprintf("CHECK_WORKER_METRICS_PORT=%d", checkWorkerMetricsPort),
		"CHECK_WORKER_RETRY_MAX_ATTEMPTS=3",
		"CHECK_WORKER_RETRY_BASE_DELAY=200ms",
		"CHECK_WORKER_RETRY_MAX_DELAY=2s",
		"CHECK_WORKER_RESULT_QUEUE_MAX_SIZE=1000",
		"CHECK_WORKER_RESULT_QUEUE_TTL=10m",
		"CHECK_WORKER_QUEUE_FLUSH_INTERVAL=2s",
		"OTEL_EXPORTER_OTLP_ENDPOINT=",
	)
	checkWorkerCmd.Stdout = os.Stdout
	checkWorkerCmd.Stderr = os.Stderr

	if err := checkWorkerCmd.Start(); err != nil {
		checkWorkerCleanup()
		return nil, fmt.Errorf("start check-worker process: %w", err)
	}

	// Ждём, пока worker зарегистрируется в scheduler_workers таблице.
	checkWorkerID, err := waitForWorkerRegistration(ctx, schedulerDB, checkWorkerName, 30*time.Second)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = checkWorkerCmd.Process.Kill()
		checkWorkerCleanup()
		return nil, fmt.Errorf("wait for check-worker registration: %w", err)
	}

	// === Reporting-service ===

	reportingCleanup := func() {
		//nolint:errcheck // cleanup
		_ = checkWorkerCmd.Process.Kill()
		checkWorkerCleanup()
	}

	reportingDSN, reportingDBCleanup, err := createReportingDB(ctx, pgContainer)
	if err != nil {
		reportingCleanup()
		return nil, fmt.Errorf("create reporting db: %w", err)
	}

	reportingDB, err := sqlx.Open("postgres", reportingDSN)
	if err != nil {
		reportingDBCleanup()
		reportingCleanup()
		return nil, fmt.Errorf("open reporting db: %w", err)
	}

	reportingHost, reportingPortStr, reportingName, reportingUser, reportingPassword, err := parseDSN(reportingDSN)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = reportingDB.Close()
		reportingDBCleanup()
		reportingCleanup()
		return nil, fmt.Errorf("parse reporting dsn: %w", err)
	}

	reportingBin, err := buildReportingBinary(ctx, repoRoot)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = reportingDB.Close()
		reportingDBCleanup()
		reportingCleanup()
		return nil, fmt.Errorf("build reporting-service: %w", err)
	}

	reportingGRPCPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = reportingDB.Close()
		reportingDBCleanup()
		reportingCleanup()
		return nil, fmt.Errorf("find free reporting grpc port: %w", err)
	}
	reportingHTTPPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = reportingDB.Close()
		reportingDBCleanup()
		reportingCleanup()
		return nil, fmt.Errorf("find free reporting http port: %w", err)
	}
	reportingMetricsPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = reportingDB.Close()
		reportingDBCleanup()
		reportingCleanup()
		return nil, fmt.Errorf("find free reporting metrics port: %w", err)
	}

	reportingAddr := fmt.Sprintf("127.0.0.1:%d", reportingGRPCPort)

	reportingCmd := exec.CommandContext(ctx, reportingBin) //nolint:gosec // reportingBin собран из исходников
	reportingCmd.Dir = reportingServiceDir()
	reportingCmd.Env = append(os.Environ(),
		fmt.Sprintf("SERVER_PORT=%d", reportingHTTPPort),
		fmt.Sprintf("SERVER_GRPC_PORT=%d", reportingGRPCPort),
		fmt.Sprintf("METRICS_PORT=%d", reportingMetricsPort),
		"METRICS_ENABLED=false",
		fmt.Sprintf("DB_HOST=%s", reportingHost),
		fmt.Sprintf("DB_PORT=%s", reportingPortStr),
		fmt.Sprintf("DB_NAME=%s", reportingName),
		fmt.Sprintf("DB_USER=%s", reportingUser),
		fmt.Sprintf("DB_PASSWORD=%s", reportingPassword),
		"DB_SSL_MODE=disable",
		"JWT_SECRET_KEY="+reportingJWTSecret,
		"MONITOR_SERVICE_GRPC_ADDRESS="+monitorAddr,
		"OTEL_ENABLED=false",
		"OTEL_EXPORTER_OTLP_ENDPOINT=",
	)
	reportingCmd.Stdout = os.Stdout
	reportingCmd.Stderr = os.Stderr

	if err := reportingCmd.Start(); err != nil {
		//nolint:errcheck // cleanup
		_ = reportingDB.Close()
		reportingDBCleanup()
		reportingCleanup()
		return nil, fmt.Errorf("start reporting-service process: %w", err)
	}

	reportingConn, err := waitForGRPC(ctx, reportingAddr, reportingCmd, "reporting-service")
	if err != nil {
		//nolint:errcheck // cleanup
		_ = reportingCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = reportingDB.Close()
		reportingDBCleanup()
		reportingCleanup()
		return nil, fmt.Errorf("wait for reporting grpc ready: %w", err)
	}

	// === Integration-service ===

	integrationCleanup := func() {
		//nolint:errcheck // cleanup
		_ = reportingCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = reportingConn.Close()
		//nolint:errcheck // cleanup
		_ = reportingDB.Close()
		reportingDBCleanup()
		reportingCleanup()
	}

	integrationDSN, integrationDBCleanup, err := createIntegrationDB(ctx, pgContainer)
	if err != nil {
		integrationCleanup()
		return nil, fmt.Errorf("create integration db: %w", err)
	}

	integrationDB, err := sqlx.Open("postgres", integrationDSN)
	if err != nil {
		integrationDBCleanup()
		integrationCleanup()
		return nil, fmt.Errorf("open integration db: %w", err)
	}

	if err := applyMigrationsTo(integrationDSN, integrationMigrationsPath()); err != nil {
		//nolint:errcheck // cleanup
		_ = integrationDB.Close()
		integrationDBCleanup()
		integrationCleanup()
		return nil, fmt.Errorf("apply integration migrations: %w", err)
	}

	integrationHost, integrationPortStr, integrationName, integrationUser, integrationPassword, err := parseDSN(integrationDSN)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = integrationDB.Close()
		integrationDBCleanup()
		integrationCleanup()
		return nil, fmt.Errorf("parse integration dsn: %w", err)
	}

	integrationBin, err := buildIntegrationBinary(ctx, repoRoot)
	if err != nil {
		//nolint:errcheck // cleanup
		_ = integrationDB.Close()
		integrationDBCleanup()
		integrationCleanup()
		return nil, fmt.Errorf("build integration-service: %w", err)
	}

	integrationGRPCPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = integrationDB.Close()
		integrationDBCleanup()
		integrationCleanup()
		return nil, fmt.Errorf("find free integration grpc port: %w", err)
	}
	integrationHTTPPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = integrationDB.Close()
		integrationDBCleanup()
		integrationCleanup()
		return nil, fmt.Errorf("find free integration http port: %w", err)
	}
	integrationMetricsPort, err := freePort()
	if err != nil {
		//nolint:errcheck // cleanup
		_ = integrationDB.Close()
		integrationDBCleanup()
		integrationCleanup()
		return nil, fmt.Errorf("find free integration metrics port: %w", err)
	}

	integrationAddr := fmt.Sprintf("127.0.0.1:%d", integrationGRPCPort)

	integrationCmd := exec.CommandContext(ctx, integrationBin) //nolint:gosec // integrationBin собран из исходников
	integrationCmd.Dir = integrationServiceDir()
	integrationCmd.Env = append(os.Environ(),
		fmt.Sprintf("SERVER_PORT=%d", integrationHTTPPort),
		fmt.Sprintf("SERVER_GRPC_PORT=%d", integrationGRPCPort),
		fmt.Sprintf("METRICS_PORT=%d", integrationMetricsPort),
		"METRICS_ENABLED=false",
		fmt.Sprintf("DB_HOST=%s", integrationHost),
		fmt.Sprintf("DB_PORT=%s", integrationPortStr),
		fmt.Sprintf("DB_NAME=%s", integrationName),
		fmt.Sprintf("DB_USER=%s", integrationUser),
		fmt.Sprintf("DB_PASSWORD=%s", integrationPassword),
		"DB_SSLMODE=disable",
		"RABBITMQ_ENABLED=false",
		"JWT_SECRET="+integrationJWTSecret,
		"ENCRYPTION_KEY="+integrationEncryptionKey,
		"AUTH_SERVICE_GRPC_ADDRESS="+grpcAddr,
		"BILLING_SERVICE_GRPC_ADDRESS="+billingAddr,
		"MONITOR_SERVICE_GRPC_ADDRESS="+monitorAddr,
		"OTEL_EXPORTER_OTLP_ENDPOINT=",
	)
	integrationCmd.Stdout = os.Stdout
	integrationCmd.Stderr = os.Stderr

	if err := integrationCmd.Start(); err != nil {
		//nolint:errcheck // cleanup
		_ = integrationDB.Close()
		integrationDBCleanup()
		integrationCleanup()
		return nil, fmt.Errorf("start integration-service process: %w", err)
	}

	integrationConn, err := waitForGRPC(ctx, integrationAddr, integrationCmd, "integration-service")
	if err != nil {
		//nolint:errcheck // cleanup
		_ = integrationCmd.Process.Kill()
		//nolint:errcheck // cleanup
		_ = integrationDB.Close()
		integrationDBCleanup()
		integrationCleanup()
		return nil, fmt.Errorf("wait for integration grpc ready: %w", err)
	}

	return &Stack{
		pgContainer:          pgContainer,
		redisContainer:       redisContainer,
		rmqContainer:         rmqContainer,
		rmqHost:              rmqHost,
		rmqPort:              rmqPort,
		fakeOAuth:            fakeOAuth,
		process:              cmd.Process,
		Client:               authv1.NewAuthServiceClient(conn),
		conn:                 conn,
		authDB:               authSqlxDB,
		AuthJWTSecret:        "test-secret-for-bdd-tests-minimum32chars",
		DashboardDB:          dashDB,
		cancelConsumer:       cancelConsumer,
		alertProcess:         alertCmd.Process,
		alertConn:            alertConn,
		alertDBCleanup:       alertDBCleanup,
		AlertDB:              alertDB,
		AlertClient:          authv1.NewAlertServiceClient(alertConn),
		AlertFakeSMTP:        fakeSMTP,
		AlertFakeWebhook:     fakeWebhook,
		AlertFakeTLSWebhook:  fakeTLSWebhook,
		billingProcess:       billingCmd.Process,
		billingConn:          billingConn,
		billingDBCleanup:     billingDBCleanup,
		BillingDB:            billingDB,
		BillingClient:        authv1.NewBillingServiceClient(billingConn),
		monitorProcess:       monitorCmd.Process,
		monitorConn:          monitorConn,
		monitorDBCleanup:     monitorDBCleanup,
		MonitorDB:            monitorDB,
		MonitorClient:        monitorv1.NewMonitorServiceClient(monitorConn),
		schedulerProcess:     schedulerCmd.Process,
		schedulerConn:        schedulerConn,
		schedulerDBCleanup:   schedulerDBCleanup,
		SchedulerDB:          schedulerDB,
		SchedulerClient:      authv1.NewSchedulerServiceClient(schedulerConn),
		checkWorkerProcess:   checkWorkerCmd.Process,
		CheckWorkerID:        checkWorkerID,
		CheckWorkerName:      checkWorkerName,
		CheckWorkerZone:      checkWorkerZone,
		reportingProcess:     reportingCmd.Process,
		reportingConn:        reportingConn,
		reportingDBCleanup:   reportingDBCleanup,
		ReportingDB:          reportingDB,
		ReportingClient:      reportingv1.NewReportingServiceClient(reportingConn),
		integrationProcess:   integrationCmd.Process,
		integrationConn:      integrationConn,
		integrationDBCleanup: integrationDBCleanup,
		IntegrationDB:        integrationDB,
		IntegrationWebhook:   authv1.NewWebhookIntegrationServiceClient(integrationConn),
		IntegrationAPIKey:    authv1.NewAPIKeyServiceClient(integrationConn),
		IntegrationImport:    authv1.NewImportServiceClient(integrationConn),
	}, nil
}

// Stop останавливает subprocess, закрывает соединения и контейнеры.
func (s *Stack) Stop(ctx context.Context) error {
	var errs []error

	if s.cancelConsumer != nil {
		s.cancelConsumer()
	}
	if s.integrationConn != nil {
		if err := s.integrationConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close integration grpc conn: %w", err))
		}
	}
	if s.integrationProcess != nil {
		//nolint:errcheck // игнорируем если процесс уже завершён
		_ = s.integrationProcess.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() {
			_, err := s.integrationProcess.Wait()
			done <- err
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			//nolint:errcheck // принудительное завершение
			_ = s.integrationProcess.Kill()
		}
	}
	if s.IntegrationDB != nil {
		if err := s.IntegrationDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close integration db: %w", err))
		}
	}
	if s.integrationDBCleanup != nil {
		s.integrationDBCleanup()
	}
	if s.reportingConn != nil {
		if err := s.reportingConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close reporting grpc conn: %w", err))
		}
	}
	if s.reportingProcess != nil {
		//nolint:errcheck // игнорируем если процесс уже завершён
		_ = s.reportingProcess.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() {
			_, err := s.reportingProcess.Wait()
			done <- err
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			//nolint:errcheck // принудительное завершение
			_ = s.reportingProcess.Kill()
		}
	}
	if s.ReportingDB != nil {
		if err := s.ReportingDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close reporting db: %w", err))
		}
	}
	if s.reportingDBCleanup != nil {
		s.reportingDBCleanup()
	}
	if s.checkWorkerProcess != nil {
		//nolint:errcheck // игнорируем если процесс уже завершён
		_ = s.checkWorkerProcess.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() {
			_, err := s.checkWorkerProcess.Wait()
			done <- err
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			//nolint:errcheck // принудительное завершение
			_ = s.checkWorkerProcess.Kill()
		}
	}
	if s.schedulerConn != nil {
		if err := s.schedulerConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close scheduler grpc conn: %w", err))
		}
	}
	if s.schedulerProcess != nil {
		//nolint:errcheck // игнорируем если процесс уже завершён
		_ = s.schedulerProcess.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() {
			_, err := s.schedulerProcess.Wait()
			done <- err
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			//nolint:errcheck // принудительное завершение
			_ = s.schedulerProcess.Kill()
		}
	}
	if s.SchedulerDB != nil {
		if err := s.SchedulerDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close scheduler db: %w", err))
		}
	}
	if s.schedulerDBCleanup != nil {
		s.schedulerDBCleanup()
	}
	if s.monitorConn != nil {
		if err := s.monitorConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close monitor grpc conn: %w", err))
		}
	}
	if s.monitorProcess != nil {
		//nolint:errcheck // игнорируем если процесс уже завершён
		_ = s.monitorProcess.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() {
			_, err := s.monitorProcess.Wait()
			done <- err
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			//nolint:errcheck // принудительное завершение
			_ = s.monitorProcess.Kill()
		}
	}
	if s.MonitorDB != nil {
		if err := s.MonitorDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close monitor db: %w", err))
		}
	}
	if s.monitorDBCleanup != nil {
		s.monitorDBCleanup()
	}
	if s.billingConn != nil {
		if err := s.billingConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close billing grpc conn: %w", err))
		}
	}
	if s.billingProcess != nil {
		//nolint:errcheck // игнорируем если процесс уже завершён
		_ = s.billingProcess.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() {
			_, err := s.billingProcess.Wait()
			done <- err
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			//nolint:errcheck // принудительное завершение
			_ = s.billingProcess.Kill()
		}
	}
	if s.BillingDB != nil {
		if err := s.BillingDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close billing db: %w", err))
		}
	}
	if s.billingDBCleanup != nil {
		s.billingDBCleanup()
	}
	if s.AlertFakeWebhook != nil {
		if err := s.AlertFakeWebhook.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop fake webhook: %w", err))
		}
	}
	if s.AlertFakeTLSWebhook != nil {
		if err := s.AlertFakeTLSWebhook.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop fake tls webhook: %w", err))
		}
	}
	if s.AlertFakeSMTP != nil {
		if err := s.AlertFakeSMTP.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop fake smtp: %w", err))
		}
	}
	if s.alertConn != nil {
		if err := s.alertConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close alert grpc conn: %w", err))
		}
	}
	if s.alertProcess != nil {
		//nolint:errcheck // игнорируем если процесс уже завершён
		_ = s.alertProcess.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() {
			_, err := s.alertProcess.Wait()
			done <- err
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			//nolint:errcheck // принудительное завершение
			_ = s.alertProcess.Kill()
		}
	}
	if s.AlertDB != nil {
		if err := s.AlertDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close alert db: %w", err))
		}
	}
	if s.alertDBCleanup != nil {
		s.alertDBCleanup()
	}
	if s.DashboardDB != nil {
		if err := s.DashboardDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close dashboard db: %w", err))
		}
	}
	if s.authDB != nil {
		if err := s.authDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close auth db: %w", err))
		}
	}
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close grpc conn: %w", err))
		}
	}
	if s.process != nil {
		//nolint:errcheck // игнорируем если процесс уже завершён
		_ = s.process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() {
			_, err := s.process.Wait()
			done <- err
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			//nolint:errcheck // принудительное завершение
			_ = s.process.Kill()
		}
	}
	if s.fakeOAuth != nil {
		if err := s.fakeOAuth.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop fake oauth: %w", err))
		}
	}
	if s.pgContainer != nil {
		if err := testcontainers.TerminateContainer(s.pgContainer); err != nil {
			errs = append(errs, fmt.Errorf("terminate postgres: %w", err))
		}
	}
	if s.redisContainer != nil {
		if err := testcontainers.TerminateContainer(s.redisContainer); err != nil {
			errs = append(errs, fmt.Errorf("terminate redis: %w", err))
		}
	}
	if s.rmqContainer != nil {
		if err := s.rmqContainer.Terminate(ctx); err != nil {
			errs = append(errs, fmt.Errorf("terminate rabbitmq: %w", err))
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// CleanAuthDB очищает таблицы auth-service между BDD-сценариями.
// Сохраняет goose-таблицы версий миграций.
func (s *Stack) CleanAuthDB() error {
	if s.authDB == nil {
		return nil
	}
	tables := []string{
		"invites", "memberships", "organizations",
		"refresh_tokens", "sessions", "auth_audit_log",
		"oauth_accounts", "users",
	}
	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", joinComma(tables))
	if _, err := s.authDB.Exec(stmt); err != nil {
		return fmt.Errorf("truncate auth tables: %w", err)
	}
	return nil
}

// AuthDB возвращает sqlx.DB подключение к auth-service БД.
// Используется BDD-шагами для прямых SQL-апдейтов (timing/seed сценариев).
func (s *Stack) AuthDB() *sqlx.DB {
	return s.authDB
}

// RabbitMQURL возвращает AMQP URL контейнера RabbitMQ.
func (s *Stack) RabbitMQURL() string {
	return fmt.Sprintf("amqp://guest:guest@%s:%d/", s.rmqHost, s.rmqPort)
}

// PublishEvent публикует JSON payload в exchange "monitor-events" с указанным routing key.
func (s *Stack) PublishEvent(ctx context.Context, routingKey string, payload any) error {
	conn, err := amqp.Dial(s.RabbitMQURL())
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}
	defer func() {
		//nolint:errcheck // закрываем после публикации
		_ = conn.Close()
	}()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	return ch.PublishWithContext(ctx, "monitor-events", routingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}

// PublishRaw публикует произвольный body в exchange "monitor-events" с указанным routing key.
// Используется для негативных сценариев (невалидный JSON, malformed payload).
func (s *Stack) PublishRaw(ctx context.Context, routingKey string, body []byte) error {
	conn, err := amqp.Dial(s.RabbitMQURL())
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}
	defer func() {
		//nolint:errcheck // закрываем после публикации
		_ = conn.Close()
	}()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}

	return ch.PublishWithContext(ctx, "monitor-events", routingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}

// CleanAlertDB трункует таблицы alert-service для изоляции сценариев.
func (s *Stack) CleanAlertDB() error {
	if s.AlertDB == nil {
		return nil
	}
	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", joinComma(alertDBTables))
	if _, err := s.AlertDB.Exec(stmt); err != nil {
		return fmt.Errorf("truncate alert tables: %w", err)
	}
	return nil
}

// ResetAlertFakes очищает накопленные сообщения в fake SMTP и webhook серверах.
func (s *Stack) ResetAlertFakes() {
	if s.AlertFakeSMTP != nil {
		s.AlertFakeSMTP.Reset()
	}
	if s.AlertFakeWebhook != nil {
		s.AlertFakeWebhook.Reset()
	}
	if s.AlertFakeTLSWebhook != nil {
		s.AlertFakeTLSWebhook.Reset()
	}
}

// joinComma склеивает строки через запятую.
func joinComma(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

// CleanMonitorDB трункует таблицы monitor-service для изоляции сценариев.
func (s *Stack) CleanMonitorDB() error {
	if s.MonitorDB == nil {
		return nil
	}
	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", joinComma(monitorDBTables))
	if _, err := s.MonitorDB.Exec(stmt); err != nil {
		return fmt.Errorf("truncate monitor tables: %w", err)
	}
	return nil
}

// CleanSchedulerDB трункует таблицы scheduler-service для изоляции сценариев.
func (s *Stack) CleanSchedulerDB() error {
	if s.SchedulerDB == nil {
		return nil
	}
	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", joinComma(schedulerDBTables))
	if _, err := s.SchedulerDB.Exec(stmt); err != nil {
		return fmt.Errorf("truncate scheduler tables: %w", err)
	}
	return nil
}

// CleanReportingDB трункует таблицы reporting-service для изоляции сценариев.
func (s *Stack) CleanReportingDB() error {
	if s.ReportingDB == nil {
		return nil
	}
	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", joinComma(reportingDBTables))
	if _, err := s.ReportingDB.Exec(stmt); err != nil {
		return fmt.Errorf("truncate reporting tables: %w", err)
	}
	return nil
}

// CleanIntegrationDB трункует таблицы integration-service для изоляции сценариев.
func (s *Stack) CleanIntegrationDB() error {
	if s.IntegrationDB == nil {
		return nil
	}
	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", joinComma(integrationDBTables))
	if _, err := s.IntegrationDB.Exec(stmt); err != nil {
		return fmt.Errorf("truncate integration tables: %w", err)
	}
	return nil
}

// buildIntegrationBinary собирает бинарник integration-service из исходников.
func buildIntegrationBinary(ctx context.Context, repoRoot string) (string, error) {
	binPath := filepath.Join(os.TempDir(), "integration-service-bdd")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
		"./backend/integration-service/cmd/integration-service")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %w\n%s", err, out)
	}
	return binPath, nil
}

// buildReportingBinary собирает бинарник reporting-service из исходников.
func buildReportingBinary(ctx context.Context, repoRoot string) (string, error) {
	binPath := filepath.Join(os.TempDir(), "reporting-service-bdd")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
		"./backend/reporting-service/cmd/reporting-service")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %w\n%s", err, out)
	}
	return binPath, nil
}

// buildSchedulerBinary собирает бинарник scheduler-service из исходников.
func buildSchedulerBinary(ctx context.Context, repoRoot string) (string, error) {
	binPath := filepath.Join(os.TempDir(), "scheduler-service-bdd")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
		"./backend/scheduler-service/cmd/scheduler-service")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %w\n%s", err, out)
	}
	return binPath, nil
}

// buildMonitorBinary собирает бинарник monitor-service из исходников.
func buildMonitorBinary(ctx context.Context, repoRoot string) (string, error) {
	binPath := filepath.Join(os.TempDir(), "monitor-service-bdd")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
		"./backend/monitor-service/cmd/monitor-service")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %w\n%s", err, out)
	}
	return binPath, nil
}

// CleanBillingDB трункует таблицы billing-service для изоляции сценариев.
// Справочник subscription_plans не затрагивается (данные ставятся миграциями).
func (s *Stack) CleanBillingDB() error {
	if s.BillingDB == nil {
		return nil
	}
	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", joinComma(billingDBTables))
	if _, err := s.BillingDB.Exec(stmt); err != nil {
		return fmt.Errorf("truncate billing tables: %w", err)
	}
	return nil
}

// buildBillingBinary собирает бинарник billing-service из исходников.
func buildBillingBinary(ctx context.Context, repoRoot string) (string, error) {
	binPath := filepath.Join(os.TempDir(), "billing-service-bdd")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
		"./backend/billing-service/cmd/billing-service")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %w\n%s", err, out)
	}
	return binPath, nil
}

// buildAlertBinary собирает бинарник alert-service из исходников.
func buildAlertBinary(ctx context.Context, repoRoot string) (string, error) {
	binPath := filepath.Join(os.TempDir(), "alert-service-bdd")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
		"./backend/alert-service/cmd/alert-service")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %w\n%s", err, out)
	}
	return binPath, nil
}

// buildCheckWorkerBinary собирает бинарник check-worker из исходников.
func buildCheckWorkerBinary(ctx context.Context, repoRoot string) (string, error) {
	binPath := filepath.Join(os.TempDir(), "check-worker-bdd")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
		"./backend/check-worker/cmd/check-worker")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %w\n%s", err, out)
	}
	return binPath, nil
}

// waitForWorkerRegistration опрашивает scheduler_workers пока запись с указанным
// именем не появится. Возвращает worker_id из БД.
func waitForWorkerRegistration(ctx context.Context, db *sqlx.DB, name string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var id string
		err := db.QueryRowxContext(ctx,
			`SELECT id::text FROM scheduler_workers WHERE name = $1`, name).Scan(&id)
		if err == nil && id != "" {
			return id, nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return "", fmt.Errorf("worker %q not registered within %s", name, timeout)
}

// buildBinary собирает бинарник auth-service из исходников.
func buildBinary(ctx context.Context, repoRoot string) (string, error) {
	binPath := filepath.Join(os.TempDir(), "auth-service-bdd")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
		"./backend/auth-service/cmd/auth-service")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %w\n%s", err, out)
	}
	return binPath, nil
}

// freePort находит свободный TCP порт на localhost.
func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	//nolint:errcheck // закрываем временный listener
	_ = ln.Close()
	return port, nil
}

// waitForGRPC ждёт пока gRPC сервер не станет доступен (до 30 секунд).
// Сначала проверяет TCP-соединение, параллельно отслеживает падение процесса.
func waitForGRPC(ctx context.Context, addr string, cmd *exec.Cmd, serviceName string) (*grpc.ClientConn, error) {
	processDone := make(chan error, 1)
	go func() {
		processDone <- cmd.Wait()
	}()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-processDone:
			if err != nil {
				return nil, fmt.Errorf("%s exited before grpc ready: %w", serviceName, err)
			}
			return nil, fmt.Errorf("%s exited before grpc ready", serviceName)
		default:
		}

		// Проверяем TCP-доступность.
		tcpConn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			//nolint:errcheck // закрываем probe соединение
			_ = tcpConn.Close()
			// Порт открыт — создаём gRPC клиент.
			conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return nil, fmt.Errorf("create grpc client: %w", err)
			}
			return conn, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-processDone:
			if err != nil {
				return nil, fmt.Errorf("%s exited before grpc ready: %w", serviceName, err)
			}
			return nil, fmt.Errorf("%s exited before grpc ready", serviceName)
		case <-time.After(500 * time.Millisecond):
		}
	}
	return nil, fmt.Errorf("grpc server at %s not ready after 30s", addr)
}

// parseDSN разбирает postgres DSN в формате URL (postgres://user:pass@host:port/dbname?...).
func parseDSN(dsn string) (host, port, dbname, user, password string, err error) {
	u, parseErr := url.Parse(dsn)
	if parseErr != nil {
		err = fmt.Errorf("parse dsn url: %w", parseErr)
		return
	}
	host = u.Hostname()
	port = u.Port()
	user = u.User.Username()
	password, _ = u.User.Password()
	dbname = u.Path
	if len(dbname) > 0 && dbname[0] == '/' {
		dbname = dbname[1:]
	}
	if host == "" || port == "" || dbname == "" || user == "" {
		err = fmt.Errorf("incomplete dsn: host=%q port=%q dbname=%q user=%q", host, port, dbname, user)
	}
	return
}

// applyMigrations применяет SQL-миграции через goose к переданному DSN.
func applyMigrations(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer func() {
		//nolint:errcheck // закрываем временное соединение
		_ = db.Close()
	}()

	if err := waitForSQLReady(db, 30*time.Second); err != nil {
		return fmt.Errorf("wait for db ready: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	// Используем такое же имя таблицы версий, как auth-service main.go,
	// чтобы повторный goose.Up из subprocess стал no-op.
	goose.SetTableName("goose_auth_version")
	defer goose.SetTableName("goose_db_version")
	if err := goose.Up(db, migrationsPath()); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

// waitForSQLReady ждёт реального SQL-соединения после readiness контейнера.
func waitForSQLReady(db *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// migrationsPath возвращает путь к директории с SQL-миграциями auth-service.
func migrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename: .../features/suite/stack.go → ../../../backend/auth-service/migrations
	return filepath.Join(filepath.Dir(filename), "..", "..", "backend", "auth-service", "migrations")
}
