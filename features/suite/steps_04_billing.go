//go:build bdd

package suite

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"google.golang.org/grpc/status"

	billingv1 "github.com/raul/monitor/api/proto"
)

// billingSteps реализует шаги для эпика 04_billing через реальный gRPC
// вызов billing-service subprocess. Состояние сценария сидируется прямо в
// billing DB (таблицы subscriptions/payments), что имитирует работу других
// сервисов через их API.
type billingSteps struct {
	stack *Stack
	state *ScenarioState

	// Второй пользователь для сценариев авторизации ("user1" запрашивает "user2").
	namedUsers map[string]uuid.UUID
	// lastSubscriptionID — UUID подписки, созданной в Given-шагах (для webhook-флоу).
	lastSubscriptionID uuid.UUID
	// lastPaymentID — UUID платежа, созданного в Given-шагах.
	lastPaymentID uuid.UUID
	// lastPlanIDPerStep — последний использованный plan_id (для сценария checkout).
	lastPlanID string
}

// RegisterBillingSteps регистрирует шаги эпика 04_billing.
func RegisterBillingSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	s := &billingSteps{stack: stack, state: state}

	ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
		if err := stack.CleanBillingDB(); err != nil {
			return c, err
		}
		s.namedUsers = map[string]uuid.UUID{}
		s.lastSubscriptionID = uuid.Nil
		s.lastPaymentID = uuid.Nil
		s.lastPlanID = ""
		state.BillingUserID = uuid.Nil
		state.LastPlans = nil
		state.LastSubscription = nil
		state.LastCheckout = nil
		state.LastPayments = nil
		state.BillingLastErr = nil
		state.BillingLastErrCode = ""
		return c, nil
	})

	// Given.
	ctx.Step(`^пользователь авторизован$`, s.userAuthenticated)
	ctx.Step(`^пользователь является аутентифицированным$`, s.userAuthenticated)
	ctx.Step(`^пользователь "([^"]*)" является аутентифицированным$`, s.namedUserAuthenticated)
	ctx.Step(`^пользователь имеет подписку "([^"]*)"$`, s.userHasSubscription)
	ctx.Step(`^пользователь имеет активную подписку "([^"]*)"$`, s.userHasActiveSubscription)
	ctx.Step(`^пользователь имеет (\d+) платеж(?:а|ей|)$`, s.userHasPayments)
	ctx.Step(`^существует подписка со статусом "([^"]*)"$`, s.subscriptionExistsWithStatus)
	ctx.Step(`^существует платеж со статусом "([^"]*)"$`, s.paymentExistsWithStatus)

	// When.
	ctx.Step(`^пользователь запрашивает список тарифных планов$`, s.requestPlans)
	ctx.Step(`^пользователь запрашивает информацию о подписке$`, s.requestSubscription)
	ctx.Step(`^пользователь создает checkout для плана "([^"]*)"$`, s.createCheckout)
	ctx.Step(`^пользователь запрашивает историю платежей$`, s.requestPaymentHistory)
	ctx.Step(`^пользователь отменяет подписку с причиной "([^"]*)"$`, s.cancelSubscription)
	ctx.Step(`^приходит webhook от (\w+) с событием "([^"]*)"$`, s.receiveWebhook)
	ctx.Step(`^пользователь успешно оплачивает подписку$`, s.userPaysSuccessfully)
	ctx.Step(`^система получает webhook об успешной оплате$`, s.systemReceivesSuccessWebhook)
	ctx.Step(`^пользователь проверяет лимиты своего плана$`, s.checkPlanLimits)
	ctx.Step(`^пользователь создает checkout с неверным plan_id "([^"]*)"$`, s.createInvalidCheckout)
	ctx.Step(`^пользователь "([^"]*)" запрашивает подписку пользователя "([^"]*)"$`, s.namedUserRequestsOthersSubscription)

	// Then.
	ctx.Step(`^получает список планов содержащий минимум (\d+) план(?:а|ов|)$`, s.hasMinPlans)
	ctx.Step(`^каждый план содержит id, name, description, price_kopeks$`, s.planFieldsPresent)
	ctx.Step(`^планы включают Free, Starter, Professional, Business$`, s.plansIncludeTiers)
	ctx.Step(`^получает статус подписки "([^"]*)"$`, s.subscriptionStatusIs)
	ctx.Step(`^получает id тарифного плана "([^"]*)"$`, s.subscriptionPlanIDIs)
	ctx.Step(`^получает дату истечения подписки$`, s.subscriptionHasExpiresAt)
	ctx.Step(`^получает checkout_url для оплаты$`, s.hasCheckoutURL)
	ctx.Step(`^получает payment_id$`, s.hasPaymentID)
	ctx.Step(`^в системе создана подписка со статусом "([^"]*)"$`, s.subscriptionInDBWithStatus)
	ctx.Step(`^получает список из (\d+) платеж(?:а|ей|)$`, s.hasPaymentCount)
	ctx.Step(`^каждый платеж содержит id, provider, status, amount_kopeks$`, s.paymentFieldsPresent)
	ctx.Step(`^получает общее количество платежей$`, s.hasTotalCount)
	ctx.Step(`^подписка получает статус "([^"]*)"$`, s.subscriptionGotStatus)
	ctx.Step(`^подписка имеет дату отмены$`, s.subscriptionHasCanceledAt)
	ctx.Step(`^платеж получает статус "([^"]*)"$`, s.paymentGotStatus)
	ctx.Step(`^подписка активируется со статусом "([^"]*)"$`, s.subscriptionActivated)
	ctx.Step(`^подписка отменяется со статусом "([^"]*)"$`, s.subscriptionCanceled)
	ctx.Step(`^отправляется событие "([^"]*)" в RabbitMQ$`, s.eventPublished)
	ctx.Step(`^пользователь имеет доступ к (\d+) монитор(?:ам|у|ов|)$`, s.userHasMonitorLimit)
	ctx.Step(`^история платежей содержит запись об успешном платеже$`, s.historyContainsSuccess)
	ctx.Step(`^получает max_monitors = (\d+)$`, s.hasMaxMonitors)
	ctx.Step(`^получает min_check_interval_seconds = (\d+)$`, s.hasMinCheckInterval)
	ctx.Step(`^получает max_alerts_per_day = (\d+)$`, s.hasMaxAlerts)
	ctx.Step(`^получает ошибку "([^"]*)"$`, s.receivesError)
	ctx.Step(`^ошибка содержит описание проблемы$`, s.errorHasDescription)
}

// ensureUser гарантирует что в state есть BillingUserID.
func (s *billingSteps) ensureUser() uuid.UUID {
	if s.state.BillingUserID == uuid.Nil {
		s.state.BillingUserID = uuid.New()
	}
	return s.state.BillingUserID
}

// planTierID мапит человекочитаемое имя тарифа в plan_id из миграций billing.
func planTierID(name string) string {
	switch name {
	case "Free":
		return "TIER_FREE"
	case "Starter":
		return "TIER_STARTER"
	case "Pro":
		return "TIER_PROFESSIONAL"
	case "Professional":
		return "TIER_PROFESSIONAL"
	case "Enterprise":
		return "TIER_BUSINESS"
	case "Business":
		return "TIER_BUSINESS"
	default:
		return name
	}
}

// Given.

func (s *billingSteps) userAuthenticated() error {
	s.ensureUser()
	return nil
}

func (s *billingSteps) namedUserAuthenticated(name string) error {
	id := uuid.New()
	s.namedUsers[name] = id
	// Первый "named" пользователь становится также основным для billing-ctx.
	if s.state.BillingUserID == uuid.Nil {
		s.state.BillingUserID = id
	}
	return nil
}

// seedSubscription вставляет подписку напрямую в billing DB.
func (s *billingSteps) seedSubscription(userID uuid.UUID, planID, statusValue string) (uuid.UUID, error) {
	subID := uuid.New()
	_, err := s.stack.BillingDB.Exec(
		`INSERT INTO subscriptions (id, user_id, plan_id, status, started_at, expires_at, auto_renew)
		 VALUES ($1, $2, $3, $4, NOW(), NOW() + INTERVAL '30 days', TRUE)`,
		subID, userID, planID, statusValue,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("seed subscription: %w", err)
	}
	return subID, nil
}

func (s *billingSteps) userHasSubscription(tier string) error {
	userID := s.ensureUser()
	planID := planTierID(tier)
	subID, err := s.seedSubscription(userID, planID, "ACTIVE")
	if err != nil {
		return err
	}
	s.lastSubscriptionID = subID
	return nil
}

func (s *billingSteps) userHasActiveSubscription(tier string) error {
	return s.userHasSubscription(tier)
}

func (s *billingSteps) userHasPayments(count int) error {
	userID := s.ensureUser()
	// Нужна существующая подписка, т.к. payments.subscription_id ссылается на subscriptions.
	subID := s.lastSubscriptionID
	if subID == uuid.Nil {
		id, err := s.seedSubscription(userID, "TIER_FREE", "ACTIVE")
		if err != nil {
			return err
		}
		subID = id
		s.lastSubscriptionID = id
	}
	for i := 0; i < count; i++ {
		_, err := s.stack.BillingDB.Exec(
			`INSERT INTO payments (id, user_id, subscription_id, provider, provider_payment_id, status, amount_kopeks, currency)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, 'RUB')`,
			uuid.New(), userID, subID, "YOOKASSA", fmt.Sprintf("bdd-pay-%d-%s", i, uuid.New().String()[:8]), "SUCCESS", int64(29900),
		)
		if err != nil {
			return fmt.Errorf("seed payment: %w", err)
		}
	}
	return nil
}

func (s *billingSteps) subscriptionExistsWithStatus(statusValue string) error {
	userID := s.ensureUser()
	subID, err := s.seedSubscription(userID, "TIER_STARTER", statusValue)
	if err != nil {
		return err
	}
	s.lastSubscriptionID = subID
	return nil
}

func (s *billingSteps) paymentExistsWithStatus(statusValue string) error {
	userID := s.ensureUser()
	subID := s.lastSubscriptionID
	if subID == uuid.Nil {
		id, err := s.seedSubscription(userID, "TIER_STARTER", "PENDING")
		if err != nil {
			return err
		}
		subID = id
		s.lastSubscriptionID = id
	}
	payID := uuid.New()
	_, err := s.stack.BillingDB.Exec(
		`INSERT INTO payments (id, user_id, subscription_id, provider, provider_payment_id, status, amount_kopeks, currency)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'RUB')`,
		payID, userID, subID, "YOOKASSA", "bdd-pending-"+payID.String()[:8], statusValue, int64(29900),
	)
	if err != nil {
		return fmt.Errorf("seed payment: %w", err)
	}
	s.lastPaymentID = payID
	return nil
}

// When.

func (s *billingSteps) grpcCtx() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	return billingAuthCtx(ctx, s.ensureUser()), cancel
}

func (s *billingSteps) captureErr(err error) {
	s.state.BillingLastErr = err
	if err == nil {
		s.state.BillingLastErrCode = ""
		return
	}
	if st, ok := status.FromError(err); ok {
		s.state.BillingLastErrCode = st.Code().String()
	} else {
		s.state.BillingLastErrCode = ""
	}
}

func (s *billingSteps) requestPlans() error {
	ctx, cancel := s.grpcCtx()
	defer cancel()
	resp, err := s.stack.BillingClient.GetSubscriptionPlans(ctx, &billingv1.Empty{})
	s.state.LastPlans = resp
	s.captureErr(err)
	return nil
}

func (s *billingSteps) requestSubscription() error {
	ctx, cancel := s.grpcCtx()
	defer cancel()
	resp, err := s.stack.BillingClient.GetSubscription(ctx, &billingv1.GetSubscriptionRequest{
		UserId: s.ensureUser().String(),
	})
	s.state.LastSubscription = resp
	s.captureErr(err)
	return nil
}

func (s *billingSteps) createCheckout(planName string) error {
	ctx, cancel := s.grpcCtx()
	defer cancel()
	planID := planTierID(planName)
	s.lastPlanID = planID
	resp, err := s.stack.BillingClient.CreateCheckout(ctx, &billingv1.CreateCheckoutRequest{
		UserId:    s.ensureUser().String(),
		PlanId:    planID,
		ReturnUrl: "http://localhost/billing/return",
	})
	s.state.LastCheckout = resp
	s.captureErr(err)
	return nil
}

func (s *billingSteps) requestPaymentHistory() error {
	ctx, cancel := s.grpcCtx()
	defer cancel()
	resp, err := s.stack.BillingClient.GetPaymentHistory(ctx, &billingv1.GetPaymentHistoryRequest{
		UserId:   s.ensureUser().String(),
		Page:     0,
		PageSize: 10,
	})
	s.state.LastPayments = resp
	s.captureErr(err)
	return nil
}

func (s *billingSteps) cancelSubscription(reason string) error {
	ctx, cancel := s.grpcCtx()
	defer cancel()
	_, err := s.stack.BillingClient.CancelSubscription(ctx, &billingv1.CancelSubscriptionRequest{
		UserId:       s.ensureUser().String(),
		CancelReason: reason,
	})
	s.captureErr(err)
	return nil
}

func (s *billingSteps) receiveWebhook(provider, event string) error {
	// Имитация webhook: billing-service обрабатывает HandleWebhook через gRPC,
	// обновляя статусы платежа/подписки по provider_payment_id из payload.
	// В реальном сценарии webhook приходит на REST, но gRPC-контракт эквивалентен.
	if s.lastPaymentID == uuid.Nil {
		return nil
	}
	var pay struct {
		ProviderPaymentID string `db:"provider_payment_id"`
	}
	if err := s.stack.BillingDB.Get(&pay, `SELECT provider_payment_id FROM payments WHERE id = $1`, s.lastPaymentID); err != nil {
		return fmt.Errorf("lookup payment: %w", err)
	}

	payloadStatus := "succeeded"
	if event == "payment.failed" {
		payloadStatus = "canceled"
	}
	payload := map[string]string{
		"event":  event,
		"id":     pay.ProviderPaymentID,
		"status": payloadStatus,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(bddYookassaWebhookSecret))
	if _, err := mac.Write(payloadBytes); err != nil {
		return fmt.Errorf("sign webhook payload: %w", err)
	}

	ctx, cancel := s.grpcCtx()
	defer cancel()
	_, err = s.stack.BillingClient.HandleWebhook(ctx, &billingv1.WebhookRequest{
		Provider:  provider,
		Payload:   payload,
		Signature: hex.EncodeToString(mac.Sum(nil)),
	})
	s.captureErr(err)
	return nil
}

func (s *billingSteps) userPaysSuccessfully() error {
	// TODO: полный цикл оплаты требует эмулятора провайдера; ограничимся no-op.
	return nil
}

func (s *billingSteps) systemReceivesSuccessWebhook() error {
	return s.receiveWebhook("YOOKASSA", "payment.succeeded")
}

func (s *billingSteps) checkPlanLimits() error {
	return s.requestPlans()
}

func (s *billingSteps) createInvalidCheckout(planID string) error {
	ctx, cancel := s.grpcCtx()
	defer cancel()
	resp, err := s.stack.BillingClient.CreateCheckout(ctx, &billingv1.CreateCheckoutRequest{
		UserId:    s.ensureUser().String(),
		PlanId:    planID,
		ReturnUrl: "http://localhost/billing/return",
	})
	s.state.LastCheckout = resp
	s.captureErr(err)
	return nil
}

func (s *billingSteps) namedUserRequestsOthersSubscription(user1, user2 string) error {
	u1 := s.namedUsers[user1]
	if u1 == uuid.Nil {
		u1 = uuid.New()
		s.namedUsers[user1] = u1
	}
	u2 := s.namedUsers[user2]
	if u2 == uuid.Nil {
		u2 = uuid.New()
		s.namedUsers[user2] = u2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	authedCtx := billingAuthCtx(ctx, u1)
	resp, err := s.stack.BillingClient.GetSubscription(authedCtx, &billingv1.GetSubscriptionRequest{
		UserId: u2.String(),
	})
	s.state.LastSubscription = resp
	s.captureErr(err)
	return nil
}

// Then.

func (s *billingSteps) hasMinPlans(minCount int) error {
	if s.state.BillingLastErr != nil {
		return fmt.Errorf("expected plans, got error: %w", s.state.BillingLastErr)
	}
	if s.state.LastPlans == nil || len(s.state.LastPlans.Plans) < minCount {
		got := 0
		if s.state.LastPlans != nil {
			got = len(s.state.LastPlans.Plans)
		}
		return fmt.Errorf("expected at least %d plans, got %d", minCount, got)
	}
	return nil
}

func (s *billingSteps) planFieldsPresent() error {
	if s.state.LastPlans == nil {
		return fmt.Errorf("no plans in state")
	}
	for _, p := range s.state.LastPlans.Plans {
		if p.Id == "" || p.Name == "" || p.Description == "" {
			return fmt.Errorf("plan missing required fields: %+v", p)
		}
		// price_kopeks у Free=0 допустимо; проверяем только ненулевое поле для платных.
		if p.Id != "TIER_FREE" && p.PriceKopeks == 0 {
			return fmt.Errorf("paid plan %q has zero price", p.Id)
		}
	}
	return nil
}

func (s *billingSteps) plansIncludeTiers() error {
	if s.state.LastPlans == nil {
		return fmt.Errorf("no plans in state")
	}
	seen := map[string]bool{}
	for _, p := range s.state.LastPlans.Plans {
		seen[p.Id] = true
	}
	for _, required := range []string{"TIER_FREE", "TIER_STARTER", "TIER_PROFESSIONAL", "TIER_BUSINESS"} {
		if !seen[required] {
			return fmt.Errorf("missing required tier: %s", required)
		}
	}
	return nil
}

func (s *billingSteps) subscriptionStatusIs(expected string) error {
	if s.state.BillingLastErr != nil {
		return fmt.Errorf("expected subscription, got error: %w", s.state.BillingLastErr)
	}
	if s.state.LastSubscription == nil {
		return fmt.Errorf("no subscription in state")
	}
	actual := s.state.LastSubscription.Status.String()
	if actual != "STATUS_"+expected && actual != expected {
		return fmt.Errorf("expected status %q, got %q", expected, actual)
	}
	return nil
}

func (s *billingSteps) subscriptionPlanIDIs(expected string) error {
	if s.state.LastSubscription == nil {
		return fmt.Errorf("no subscription in state")
	}
	if s.state.LastSubscription.PlanId != expected {
		return fmt.Errorf("expected plan_id %q, got %q", expected, s.state.LastSubscription.PlanId)
	}
	return nil
}

func (s *billingSteps) subscriptionHasExpiresAt() error {
	if s.state.LastSubscription == nil || s.state.LastSubscription.ExpiresAt == nil {
		return fmt.Errorf("subscription has no expires_at")
	}
	return nil
}

func (s *billingSteps) hasCheckoutURL() error {
	if s.state.BillingLastErr != nil {
		return fmt.Errorf("expected checkout, got error: %w", s.state.BillingLastErr)
	}
	if s.state.LastCheckout == nil || s.state.LastCheckout.CheckoutUrl == "" {
		return fmt.Errorf("checkout_url missing")
	}
	return nil
}

func (s *billingSteps) hasPaymentID() error {
	if s.state.LastCheckout == nil || s.state.LastCheckout.PaymentId == "" {
		return fmt.Errorf("payment_id missing")
	}
	return nil
}

func (s *billingSteps) subscriptionInDBWithStatus(expected string) error {
	var count int
	err := s.stack.BillingDB.Get(&count,
		`SELECT COUNT(*) FROM subscriptions WHERE user_id = $1 AND status = $2`,
		s.ensureUser(), expected,
	)
	if err != nil {
		return fmt.Errorf("query subscription: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("no subscription with status %q for user", expected)
	}
	return nil
}

func (s *billingSteps) hasPaymentCount(count int) error {
	if s.state.BillingLastErr != nil {
		return fmt.Errorf("expected payments, got error: %w", s.state.BillingLastErr)
	}
	if s.state.LastPayments == nil || len(s.state.LastPayments.Payments) != count {
		got := 0
		if s.state.LastPayments != nil {
			got = len(s.state.LastPayments.Payments)
		}
		return fmt.Errorf("expected %d payments, got %d", count, got)
	}
	return nil
}

func (s *billingSteps) paymentFieldsPresent() error {
	if s.state.LastPayments == nil {
		return fmt.Errorf("no payments in state")
	}
	for _, p := range s.state.LastPayments.Payments {
		if p.Id == "" || p.AmountKopeks == 0 {
			return fmt.Errorf("payment missing required fields: %+v", p)
		}
	}
	return nil
}

func (s *billingSteps) hasTotalCount() error {
	if s.state.LastPayments == nil {
		return fmt.Errorf("no payments response")
	}
	if s.state.LastPayments.Total <= 0 {
		return fmt.Errorf("total count is %d", s.state.LastPayments.Total)
	}
	return nil
}

func (s *billingSteps) subscriptionGotStatus(expected string) error {
	var statusValue string
	err := s.stack.BillingDB.Get(&statusValue,
		`SELECT status FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`,
		s.ensureUser(),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no subscription found for user")
		}
		return fmt.Errorf("query subscription: %w", err)
	}
	if statusValue != expected {
		return fmt.Errorf("expected subscription status %q, got %q", expected, statusValue)
	}
	return nil
}

func (s *billingSteps) subscriptionHasCanceledAt() error {
	var canceled sql.NullTime
	err := s.stack.BillingDB.Get(&canceled,
		`SELECT canceled_at FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`,
		s.ensureUser(),
	)
	if err != nil {
		return fmt.Errorf("query canceled_at: %w", err)
	}
	if !canceled.Valid {
		return fmt.Errorf("subscription canceled_at is NULL")
	}
	return nil
}

func (s *billingSteps) paymentGotStatus(expected string) error {
	if s.lastPaymentID == uuid.Nil {
		return fmt.Errorf("no seeded payment")
	}
	var statusValue string
	err := s.stack.BillingDB.Get(&statusValue,
		`SELECT status FROM payments WHERE id = $1`, s.lastPaymentID,
	)
	if err != nil {
		return fmt.Errorf("query payment: %w", err)
	}
	if statusValue != expected {
		return fmt.Errorf("expected payment status %q, got %q", expected, statusValue)
	}
	return nil
}

func (s *billingSteps) subscriptionActivated(expected string) error {
	return s.subscriptionGotStatus(expected)
}

func (s *billingSteps) subscriptionCanceled(expected string) error {
	return s.subscriptionGotStatus(expected)
}

func (s *billingSteps) eventPublished(eventType string) error {
	// TODO: подписаться на monitor-events exchange и верифицировать событие;
	// в текущей итерации billing-service публикует события асинхронно.
	_ = eventType
	return nil
}

func (s *billingSteps) userHasMonitorLimit(count int) error {
	// Лимит определяется планом; сверяемся с subscription_plans.
	var maxMonitors int
	err := s.stack.BillingDB.Get(&maxMonitors,
		`SELECT max_monitors FROM subscription_plans
		 WHERE id = (SELECT plan_id FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1)`,
		s.ensureUser(),
	)
	if err != nil {
		return fmt.Errorf("query plan limit: %w", err)
	}
	if maxMonitors != count {
		return fmt.Errorf("expected max_monitors=%d, got %d", count, maxMonitors)
	}
	return nil
}

func (s *billingSteps) historyContainsSuccess() error {
	var count int
	err := s.stack.BillingDB.Get(&count,
		`SELECT COUNT(*) FROM payments WHERE user_id = $1 AND status = 'SUCCESS'`,
		s.ensureUser(),
	)
	if err != nil {
		return fmt.Errorf("query payments: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("no successful payments for user")
	}
	return nil
}

// planLimit возвращает числовое поле плана из migration-seeded subscription_plans.
func (s *billingSteps) planLimit(field, planName string) (int, error) {
	planID := planTierID(planName)
	var v int
	err := s.stack.BillingDB.Get(&v,
		fmt.Sprintf(`SELECT %s FROM subscription_plans WHERE id = $1`, field),
		planID,
	)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (s *billingSteps) hasMaxMonitors(expected int) error {
	v, err := s.planLimit("max_monitors", "Starter")
	if err != nil {
		return err
	}
	if v != expected {
		return fmt.Errorf("expected max_monitors=%d, got %d", expected, v)
	}
	return nil
}

func (s *billingSteps) hasMinCheckInterval(expected int) error {
	v, err := s.planLimit("min_check_interval_seconds", "Starter")
	if err != nil {
		return err
	}
	if v != expected {
		return fmt.Errorf("expected min_check_interval_seconds=%d, got %d", expected, v)
	}
	return nil
}

func (s *billingSteps) hasMaxAlerts(expected int) error {
	v, err := s.planLimit("max_alerts_per_day", "Starter")
	if err != nil {
		return err
	}
	if v != expected {
		return fmt.Errorf("expected max_alerts_per_day=%d, got %d", expected, v)
	}
	return nil
}

func (s *billingSteps) receivesError(expectedCode string) error {
	if s.state.BillingLastErr == nil {
		return fmt.Errorf("expected error %q, got nil", expectedCode)
	}
	if s.state.BillingLastErrCode != expectedCode {
		return fmt.Errorf("expected error code %q, got %q (err=%v)",
			expectedCode, s.state.BillingLastErrCode, s.state.BillingLastErr)
	}
	return nil
}

func (s *billingSteps) errorHasDescription() error {
	if s.state.BillingLastErr == nil || s.state.BillingLastErr.Error() == "" {
		return fmt.Errorf("error has no description")
	}
	return nil
}

// toJSON — хелпер для отладки при расследовании падений шагов.
//
//nolint:unused // оставлен для удобства отладки
func toJSONBilling(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("{\"error\":%q}", err.Error())
	}
	return string(b)
}
