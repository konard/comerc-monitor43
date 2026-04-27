package suite

import (
	"time"

	"github.com/google/uuid"

	authv1 "github.com/raul/monitor/api/proto"
	monitorv1 "github.com/raul/monitor/api/proto/monitor/v1"
)

// MonitorInfo описывает монитор в состоянии BDD-сценария.
type MonitorInfo struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Name            string
	URL             string
	CheckType       string
	IntervalSeconds int32
	TimeoutSeconds  int32
	Status          string
	CreatedAt       time.Time
}

// AlertChannelInfo описывает канал оповещений в состоянии BDD-сценария.
// Поля приближены к legacy-модели alert-service, но независимы от её типов,
// чтобы шаги собирались без зависимостей от внутренних пакетов сервиса.
type AlertChannelInfo struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Name         string
	Type         string // "telegram" | "email" | "webhook"
	Address      string // chat_id для telegram, email для email, url для webhook
	Method       string // для webhook
	Headers      map[string]string
	Status       string // "unverified" | "active" | "delivery_failed" | ...
	Enabled      bool
	Verified     bool
	Priority     int
	FailureCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AlertAlertInfo описывает alert в состоянии BDD-сценария.
type AlertAlertInfo struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	MonitorID  uuid.UUID
	Type       string
	Status     string
	Severity   string
	Message    string
	CreatedAt  time.Time
	ResolvedAt *time.Time
}

// AuditLogRow повторяет столбцы auth_audit_log, нужные BDD-шагам для
// проверки записей о team-операциях (uc_05_02_*).
type AuditLogRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	EventType string
	Success   bool
	IPAddress string
	UserAgent string
	CreatedAt time.Time
}

// AlertRuleInfo описывает правило алерта в состоянии BDD-сценария.
type AlertRuleInfo struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	MonitorID           uuid.UUID
	Enabled             bool
	ConsecutiveFailures int32
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ScenarioState хранит состояние одного BDD-сценария.
type ScenarioState struct {
	Provider     string
	AuthResp     *authv1.AuthResponse
	LastErr      error
	UserEmail    string
	ValidateResp *authv1.ValidateTokenResponse
	RefreshResp  *authv1.RefreshTokenResponse

	// Состояние эпика 02_alerting.
	UserID         uuid.UUID
	UserRole       string // "USER" / "ADMIN" / "GUEST" / ""
	SecondUserID   uuid.UUID
	SecondUserRole string
	MonitorID      uuid.UUID
	Channels       map[string]*AlertChannelInfo // key: name
	LastChannel    *AlertChannelInfo
	LastAlert      *AlertAlertInfo
	Alerts         map[uuid.UUID]*AlertAlertInfo
	AlertRules     map[uuid.UUID]*AlertRuleInfo // key: monitor_id
	LastErrorCode  string                       // gRPC status code или "" если нет ошибки

	// Состояние эпика 01_monitoring.
	MonitorTier        string
	LastMonitor        *MonitorInfo
	Monitors           map[uuid.UUID]*MonitorInfo
	MonitorsByName     map[string]*MonitorInfo
	MonitorList        []*monitorv1.Monitor
	MultipleCreateErrs []error

	// Состояние эпика 05_security / team management (us=05_02).
	OrgID            uuid.UUID
	OrgName          string
	OwnerAuth        *authv1.AuthResponse // OWNER, выпущенный отдельно от тестового юзера
	OwnerUserID      uuid.UUID
	OwnerEmail       string
	SecondAuth       *authv1.AuthResponse // вторая регистрация (приглашённый, target и т.п.)
	SecondUserEmail  string
	SecondUserOrgID  uuid.UUID // OrgID, в которой второй юзер MEMBER (= OrgID тестового OWNER'а)
	LastInvite       *authv1.Invite
	LastInviteToken  string
	LastInviteEmail  string
	LastMembers      []*authv1.Membership
	TargetUserID     uuid.UUID // user_id для change_role/remove_member
	TargetEmail      string
	OrgB_ID          uuid.UUID // для сценария изоляции uc_05_02_30
	ActorAccessToken string    // bearer-токен, который шаги используют для team-RPC
	LastAuditLog     *AuditLogRow
	RemovedUserToken string // JWT удалённого участника для проверки SESSION_REVOKED (uc_05_02_28)

	// Состояние эпика 04_billing.
	BillingUserID      uuid.UUID
	LastPlans          *authv1.SubscriptionPlans
	LastSubscription   *authv1.Subscription
	LastCheckout       *authv1.CheckoutResponse
	LastPayments       *authv1.PaymentHistoryResponse
	BillingLastErr     error
	BillingLastErrCode string
}
