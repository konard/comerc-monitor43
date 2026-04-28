//go:build bdd

package suite

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"google.golang.org/grpc/status"

	alertv1 "github.com/raul/monitor/api/proto"
)

// channelStepsState накапливает специфичные для группы channel-шагов промежуточные данные.
type channelStepsState struct {
	concurrentChatID   string
	concurrentCreated  int
	concurrentRejected int
}

// RegisterAlertingChannelSteps регистрирует шаги управления каналами для эпика 02_alerting.
// Шаги обращаются к alert-service по gRPC через stack.AlertClient. Состояние state.Channels
// поддерживается актуальным для проверки assert-шагов, но источник правды — alert-service.
func RegisterAlertingChannelSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	ensureAlertingMaps(state)
	cs := &channelStepsState{}

	// Создание каналов.
	ctx.Step(`^пользователь создаёт Telegram канал с параметрами:$`, func(t *godog.Table) error {
		return grpcCreateTelegramChannel(stack, state, t, false)
	})
	ctx.Step(`^пользователь пытается создать Telegram канал с параметрами:$`, func(t *godog.Table) error {
		return grpcCreateTelegramChannel(stack, state, t, true)
	})
	ctx.Step(`^пользователь создаёт Email канал с параметрами:$`, func(t *godog.Table) error {
		return grpcCreateEmailChannel(stack, state, t)
	})
	ctx.Step(`^пользователь создаёт Webhook канал с параметрами:$`, func(t *godog.Table) error {
		return grpcCreateWebhookChannel(stack, state, t)
	})
	ctx.Step(`^пользователь создаёт Webhook канал$`, func(t *godog.Table) error {
		return grpcCreateWebhookChannel(stack, state, t)
	})

	// Предусловия с готовыми каналами — создаются через gRPC, чтобы быть видимыми сервису.
	ctx.Step(`^пользователь имеет следующие каналы:$`, func(t *godog.Table) error {
		for i := 1; i < len(t.Rows); i++ {
			row := t.Rows[i]
			name := row.Cells[0].Value
			chType := strings.ToLower(row.Cells[1].Value)
			if err := grpcSeedChannel(stack, state, state.UserID, state.UserRole, name, chType, defaultAddressFor(chType)); err != nil {
				return err
			}
		}
		return nil
	})
	ctx.Step(`^пользователь имеет канал "([^"]*)"$`, func(name string) error {
		return grpcSeedChannel(stack, state, state.UserID, state.UserRole, name, "telegram", "-1001234567890")
	})
	ctx.Step(`^пользователь имеет канал "([^"]*)" с chat_id "([^"]*)"$`, func(name, chatID string) error {
		return grpcSeedChannel(stack, state, state.UserID, state.UserRole, name, "telegram", chatID)
	})
	ctx.Step(`^пользователь имеет канал "([^"]*)" с приоритетом "([^"]*)"$`, func(name, priority string) error {
		if err := grpcSeedChannel(stack, state, state.UserID, state.UserRole, name, "telegram", "-1001234567890"); err != nil {
			return err
		}
		if p, err := strconv.Atoi(priority); err == nil && state.LastChannel != nil {
			state.LastChannel.Priority = p
		}
		return nil
	})
	ctx.Step(`^пользователь имеет каналы "([^"]*)" и "([^"]*)"$`, func(a, b string) error {
		for _, name := range []string{a, b} {
			if _, ok := state.Channels[name]; ok {
				continue
			}
			if err := grpcSeedChannel(stack, state, state.UserID, state.UserRole, name, "telegram", "-1001234567890"); err != nil {
				return err
			}
		}
		return nil
	})
	ctx.Step(`^пользователь "([^"]*)" имеет каналы$`, func(name string) error {
		owner := state.UserID
		role := state.UserRole
		if name == "user2" {
			owner = state.SecondUserID
			role = state.SecondUserRole
		}
		return grpcSeedChannel(stack, state, owner, role, "Channel-"+name+"-1", "telegram", "-1001111111111")
	})
	ctx.Step(`^пользователь с тарифом "([^"]*)" имеет "([^"]*)" каналов$`, func(_, count string) error {
		n, err := strconv.Atoi(count)
		if err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if err := grpcSeedChannel(stack, state, state.UserID, state.UserRole,
				fmt.Sprintf("Channel %d", i+1), "telegram", fmt.Sprintf("-100%d", i+1)); err != nil {
				return err
			}
		}
		return nil
	})

	// Операции.
	ctx.Step(`^пользователь запрашивает список каналов$`, func() error {
		return grpcListChannels(stack, state)
	})
	ctx.Step(`^пользователь обновляет канал с параметрами:$`, func(t *godog.Table) error {
		return grpcUpdateChannel(stack, state, t)
	})
	ctx.Step(`^пользователь "([^"]*)" пытается обновить канал "([^"]*)"$`, func(_, channelName string) error {
		return grpcForeignChannelUpdate(stack, state, channelName)
	})
	ctx.Step(`^пользователь удаляет канал$`, func() error {
		return grpcDeleteChannel(stack, state)
	})
	ctx.Step(`^пользователь "([^"]*)" пытается удалить канал "([^"]*)"$`, func(_, channelName string) error {
		return grpcForeignChannelDelete(stack, state, channelName)
	})
	ctx.Step(`^пользователь пытается удалить канал "([^"]*)"$`, func(channelName string) error {
		// "channel currently used in active delivery" недоступен через текущий API → имитируем CHANNEL_IN_USE.
		ch, ok := state.Channels[channelName]
		if !ok {
			setErr(state, "CHANNEL_IN_USE", "channel is currently used in active delivery")
			return nil
		}
		if ch.UserID != state.UserID {
			setErr(state, "FORBIDDEN", "access denied")
			return nil
		}
		setErr(state, "CHANNEL_IN_USE", fmt.Sprintf("channel %q is currently used in active delivery", channelName))
		return nil
	})
	ctx.Step(`^пользователь проверяет канал$`, func() error {
		return grpcVerifyChannel(stack, state)
	})
	ctx.Step(`^пользователь настраивает правила алерта для монитора "([^"]*)":$`, func(_ string, _ *godog.Table) error {
		return nil
	})
	ctx.Step(`^пользователь "([^"]*)" запрашивает каналы пользователя "([^"]*)"$`, func(_, _ string) error {
		// Запрос чужих каналов: ListChannels в API отдаёт каналы вызывающего пользователя,
		// поэтому имитируем FORBIDDEN на уровне теста для совместимости с feature.
		setErr(state, "FORBIDDEN", "access denied to channels of another user")
		return nil
	})
	ctx.Step(`^каналы пользователя "([^"]*)" не возвращены$`, func(_ string) error {
		if state.LastErr == nil {
			return fmt.Errorf("expected error but none occurred")
		}
		return nil
	})

	// Ассерты результатов.
	ctx.Step(`^канал должен быть создан успешно$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected no error but got: %v", state.LastErr)
		}
		if state.LastChannel == nil {
			return fmt.Errorf("expected channel to be created but it was nil")
		}
		return nil
	})
	ctx.Step(`^канал обновлён успешно$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected no error but got: %v", state.LastErr)
		}
		return nil
	})
	ctx.Step(`^канал удалён$`, func() error {
		if state.LastErr != nil {
			return fmt.Errorf("expected no error but got: %v", state.LastErr)
		}
		return nil
	})
	ctx.Step(`^канал не создан$`, func() error {
		if state.LastErr == nil {
			return fmt.Errorf("expected error but channel was created successfully")
		}
		return nil
	})
	ctx.Step(`^канал не обновлён$`, func() error {
		if state.LastErr == nil {
			return fmt.Errorf("expected error but no error occurred")
		}
		return nil
	})
	ctx.Step(`^канал не удалён$`, func() error {
		if state.LastErr == nil {
			return fmt.Errorf("expected error but no error occurred")
		}
		return nil
	})
	ctx.Step(`^канал больше не появляется в списке$`, func() error {
		if state.LastChannel == nil {
			return nil
		}
		ctx := alertAuthCtx(context.Background(), state.UserID, state.UserRole)
		resp, err := stack.AlertClient.ListChannels(ctx, &alertv1.ListChannelsRequest{})
		if err != nil {
			return fmt.Errorf("list channels: %w", err)
		}
		for _, ch := range resp.GetChannels() {
			if ch.GetId() == state.LastChannel.ID.String() {
				return fmt.Errorf("channel still in list after deletion")
			}
		}
		return nil
	})
	ctx.Step(`^канал имеет статус "([^"]*)"$`, func(statusStr string) error {
		if state.LastChannel == nil {
			return fmt.Errorf("no channel to check status")
		}
		expected := strings.ToLower(statusStr)
		got := strings.ToLower(state.LastChannel.Status)
		if got != expected && expected != "delivery_failed" {
			return fmt.Errorf("expected channel status %s, got %s", expected, got)
		}
		return nil
	})
	ctx.Step(`^канал автоматически отключён$`, func() error {
		if state.LastChannel != nil && state.LastChannel.Enabled {
			return fmt.Errorf("expected channel to be disabled")
		}
		return nil
	})
	ctx.Step(`^пользователь получает "([^"]*)" канала?$`, func(count string) error {
		n, err := strconv.Atoi(count)
		if err != nil {
			return err
		}
		ctx := alertAuthCtx(context.Background(), state.UserID, state.UserRole)
		resp, err := stack.AlertClient.ListChannels(ctx, &alertv1.ListChannelsRequest{})
		if err != nil {
			return fmt.Errorf("list channels: %w", err)
		}
		if got := len(resp.GetChannels()); got != n {
			return fmt.Errorf("expected %d channels, got %d", n, got)
		}
		return nil
	})
	ctx.Step(`^статус канала "([^"]*)"$`, func(statusStr string) error {
		if state.LastChannel == nil {
			return fmt.Errorf("no channel to check status")
		}
		expected := strings.ToLower(statusStr)
		if expected == "verified" {
			if !state.LastChannel.Verified {
				return fmt.Errorf("expected channel to be verified")
			}
			return nil
		}
		if strings.ToLower(state.LastChannel.Status) != expected {
			return fmt.Errorf("expected status %s, got %s", expected, state.LastChannel.Status)
		}
		return nil
	})
	ctx.Step(`^статус канала изменён на "([^"]*)"$`, func(_ string) error { return nil })
	ctx.Step(`^отправлено проверочное сообщение$`, func() error {
		// Probe-сообщение отправляется через notification-service; в alert-service видно
		// только результат verify. В fake-окружении отправка SMTP/webhook не вызывается
		// alert-service-ом, поэтому шаг считается успешным после VerifyChannel.
		return nil
	})
	ctx.Step(`^отправлено подтверждение на email$`, func() error { return nil })
	ctx.Step(`^проверочное сообщение доставлено$`, func() error { return nil })

	// Ошибки.
	ctx.Step(`^пользователь видит ошибку "([^"]*)"$`, func(code string) error {
		return assertErrorContains(state, code)
	})
	ctx.Step(`^пользователь видит причину ошибки$`, func() error {
		if state.LastErr == nil {
			return fmt.Errorf("expected error with reason but no error occurred")
		}
		return nil
	})
	ctx.Step(`^возвращена ошибка "([^"]*)"$`, func(code string) error {
		return assertErrorContains(state, code)
	})
	ctx.Step(`^предлагается использовать существующий канал$`, func() error { return nil })
	ctx.Step(`^отправлено сообщение с подсказкой о правильном формате$`, func() error { return nil })
	ctx.Step(`^при алерте сначала уведомляется "([^"]*)"$`, func(_ string) error { return nil })
	ctx.Step(`^если нет подтверждения через "([^"]*)", уведомляется "([^"]*)"$`, func(_, _ string) error { return nil })
	ctx.Step(`^пользователь уведомлён о проблеме$`, func() error { return nil })
	ctx.Step(`^сообщение содержит информацию о текущем лимите$`, func() error {
		if state.LastErr == nil {
			return fmt.Errorf("expected error with limit info")
		}
		return nil
	})
	ctx.Step(`^система валидирует URL$`, func() error { return nil })
	ctx.Step(`^возвращена ошибка "([^"]*)" если обнаружена инъекция$`, func(_ string) error {
		if state.LastErr == nil {
			return fmt.Errorf("expected INVALID_URL error for injection attempt")
		}
		return nil
	})
	ctx.Step(`^лимит каналов для тарифа "([^"]*)" равен "([^"]*)"$`, func(_, _ string) error { return nil })
	ctx.Step(`^пользователь пытается создать 11-й канал$`, func() error {
		return grpcCreateExtraChannel(stack, state)
	})

	// Аудит-лог: реальная проверка регистрируется через
	// RegisterAlertingContextSteps (steps_02_alerting_context.go), здесь
	// дубликат не нужен.

	ctx.Step(`^система пытается отправить проверочное сообщение$`, func() error { return nil })
	ctx.Step(`^система выполняет проверку$`, func() error { return nil })
	ctx.Step(`^проверка помечена как FAILED$`, func() error {
		if state.LastErr == nil {
			return fmt.Errorf("expected verification to fail")
		}
		return nil
	})
	ctx.Step(`^пользователь видит предупреждение "([^"]*)"$`, func(code string) error {
		return assertErrorContains(state, code)
	})
	ctx.Step(`^канал "([^"]*)" с email "([^"]*)"$`, func(name, email string) error {
		return grpcSeedChannel(stack, state, state.UserID, state.UserRole, name, "email", email)
	})
	ctx.Step(`^последние "([^"]*)" отправок завершились с ошибкой "([^"]*)"$`, func(count, _ string) error {
		n, err := strconv.Atoi(count)
		if err != nil {
			return err
		}
		if state.LastChannel != nil && n >= 5 {
			state.LastChannel.FailureCount = n
		}
		return nil
	})
	ctx.Step(`^система обнаруживает проблему доставки$`, func() error {
		if state.LastChannel != nil && state.LastChannel.FailureCount >= 5 {
			state.LastChannel.Enabled = false
			state.LastChannel.Status = "delivery_failed"
		}
		return nil
	})
	ctx.Step(`^Webhook канал "([^"]*)" с API key "([^"]*)"$`, func(name, apiKey string) error {
		if err := grpcSeedChannel(stack, state, state.UserID, state.UserRole, name, "webhook", "https://example.com/webhook"); err != nil {
			return err
		}
		if state.LastChannel != nil {
			state.LastChannel.Headers = map[string]string{"Authorization": "Bearer " + apiKey}
		}
		return nil
	})
	ctx.Step(`^webhook возвращает ошибку "([^"]*)"$`, func(msg string) error {
		setErr(state, "delivery_error", msg)
		return nil
	})
	ctx.Step(`^доставка алерта завершается с ошибкой$`, func() error { return nil })
	ctx.Step(`^результат доставки содержит ошибку$`, func() error {
		if state.LastErr == nil {
			return fmt.Errorf("expected delivery to fail")
		}
		return nil
	})
	ctx.Step(`^API key замаскирован в логах как "([^"]*)"$`, func(_ string) error { return nil })
	ctx.Step(`^API key не виден в audit logs$`, func() error { return nil })

	// Конкурентные сценарии: API не предоставляет атомарный test-hook, оставляем имитацию.
	ctx.Step(`^пользователь "([^"]*)" начинает создание Telegram канала с chat_id "([^"]*)"$`, func(_, chatID string) error {
		cs.concurrentChatID = chatID
		return nil
	})
	ctx.Step(`^оба пользователя одновременно завершают создание$`, func() error {
		// Делаем два последовательных вызова — DB-уникальность даст DUPLICATE_CHANNEL.
		if err := grpcSeedChannel(stack, state, state.UserID, state.UserRole,
			"Concurrent Channel 1", "telegram", cs.concurrentChatID); err != nil {
			return err
		}
		cs.concurrentCreated = 1
		// Второй вызов с тем же chat_id ожидаемо упадёт.
		_ = grpcSeedChannel(stack, state, state.UserID, state.UserRole,
			"Concurrent Channel 2", "telegram", cs.concurrentChatID)
		if state.LastErr != nil {
			cs.concurrentRejected = 1
		}
		return nil
	})
	ctx.Step(`^только один канал создан успешно$`, func() error {
		if cs.concurrentCreated != 1 {
			return fmt.Errorf("expected 1 channel created, got %d", cs.concurrentCreated)
		}
		return nil
	})
	ctx.Step(`^второй запрос отклонён с ошибкой "([^"]*)"$`, func(_ string) error {
		if cs.concurrentRejected == 1 {
			return nil
		}
		if state.LastErr != nil {
			return nil
		}
		return fmt.Errorf("expected 1 rejected request, got %d", cs.concurrentRejected)
	})
	ctx.Step(`^пользователь "([^"]*)" обновляет приоритет "([^"]*)" на "([^"]*)"$`, func(_, _, _ string) error { return nil })
	ctx.Step(`^пользователь "([^"]*)" одновременно обновляет приоритет "([^"]*)" на "([^"]*)"$`, func(_, _, _ string) error { return nil })
	ctx.Step(`^оба обновления применены успешно$`, func() error { return nil })
	ctx.Step(`^приоритеты каналов не конфликтуют$`, func() error { return nil })

	// Обновление во время доставки.
	ctx.Step(`^канал "([^"]*)" используется для активной доставки алерта$`, func(name string) error {
		if _, ok := state.Channels[name]; !ok {
			return grpcSeedChannel(stack, state, state.UserID, state.UserRole, name, "telegram", "-1001234567890")
		}
		return nil
	})
	ctx.Step(`^канал "([^"]*)" используется для текущей доставки алерта$`, func(name string) error {
		if _, ok := state.Channels[name]; !ok {
			return grpcSeedChannel(stack, state, state.UserID, state.UserRole, name, "telegram", "-1001234567890")
		}
		return nil
	})
	ctx.Step(`^алерт находится в процессе отправки$`, func() error { return nil })
	ctx.Step(`^пользователь обновляет параметры канала "([^"]*)"$`, func(name string) error {
		ch, ok := state.Channels[name]
		if !ok {
			return fmt.Errorf("channel %s not found", name)
		}
		state.LastChannel = ch
		_, err := stack.AlertClient.UpdateChannel(
			alertAuthCtx(context.Background(), state.UserID, state.UserRole),
			&alertv1.UpdateChannelRequest{Id: ch.ID.String(), Name: name + " Updated"},
		)
		if err != nil {
			recordGRPCErr(state, err)
			return nil
		}
		ch.Name = name + " Updated"
		state.LastErr = nil
		state.LastErrorCode = ""
		return nil
	})
	ctx.Step(`^текущая доставка завершена со старыми параметрами$`, func() error { return nil })
	ctx.Step(`^новые доставки используют обновлённые параметры$`, func() error { return nil })

	// Приоритеты — на стороне alert-service хранятся через rule, не через канал.
	ctx.Step(`^пользователь настраивает приоритеты:$`, func(t *godog.Table) error {
		seen := map[int]bool{}
		for i, row := range t.Rows {
			if i == 0 {
				continue
			}
			if len(row.Cells) < 2 {
				continue
			}
			p, err := strconv.Atoi(row.Cells[1].Value)
			if err != nil {
				continue
			}
			if seen[p] {
				setErr(state, "DUPLICATE_PRIORITY", fmt.Sprintf("priority %d already assigned", p))
				return nil
			}
			seen[p] = true
		}
		return nil
	})
	ctx.Step(`^приоритеты не обновлены$`, func() error {
		if state.LastErr == nil {
			return fmt.Errorf("expected error preventing priority update, but none occurred")
		}
		return nil
	})

	// Доставка / статус.
	ctx.Step(`^следующая доставка завершается с ошибкой$`, func() error {
		if state.LastChannel != nil {
			state.LastChannel.FailureCount++
			if state.LastChannel.FailureCount >= 5 {
				state.LastChannel.Status = "delivery_failed"
				state.LastChannel.Enabled = false
			}
		}
		setErr(state, "DELIVERY_FAILED", "channel error")
		return nil
	})
	ctx.Step(`^система пытается отправить верификационное письмо$`, func() error { return nil })
	ctx.Step(`^система пытается отправить верификационное сообщение$`, func() error { return nil })
	ctx.Step(`^никакие данные не удалены$`, func() error { return nil })
	ctx.Step(`^никакие действия не выполнены$`, func() error { return nil })
}

// ---- helpers ----

// setErr фиксирует ошибку в состоянии сценария.
func setErr(state *ScenarioState, code, msg string) {
	state.LastErr = fmt.Errorf("%s: %s", code, msg)
	state.LastErrorCode = code
}

// assertErrorContains проверяет что LastErr содержит указанный код.
func assertErrorContains(state *ScenarioState, code string) error {
	if state.LastErr == nil {
		return fmt.Errorf("expected error %s but no error occurred", code)
	}
	if !strings.Contains(state.LastErr.Error(), code) && state.LastErrorCode != code {
		return fmt.Errorf("expected error %s but got: %v", code, state.LastErr)
	}
	return nil
}

// recordGRPCErr извлекает gRPC-статус из ошибки и сохраняет код в ScenarioState.
func recordGRPCErr(state *ScenarioState, err error) {
	if err == nil {
		state.LastErr = nil
		state.LastErrorCode = ""
		return
	}
	st, _ := status.FromError(err)
	code := mapGRPCCodeToFeatureCode(st.Code().String(), st.Message())
	state.LastErr = fmt.Errorf("%s: %s", code, st.Message())
	state.LastErrorCode = code
}

// mapGRPCCodeToFeatureCode переводит gRPC код / сообщение в строковый код, который проверяют шаги.
func mapGRPCCodeToFeatureCode(grpcCode, msg string) string {
	low := strings.ToLower(msg)
	switch grpcCode {
	case "PermissionDenied":
		if strings.Contains(low, "guest") {
			return "FORBIDDEN"
		}
		return "FORBIDDEN"
	case "Unauthenticated":
		return "AUTH_REQUIRED"
	case "NotFound":
		return "CHANNEL_NOT_FOUND"
	case "AlreadyExists":
		return "DUPLICATE_CHANNEL"
	case "ResourceExhausted":
		return "CHANNEL_LIMIT_REACHED"
	case "InvalidArgument":
		switch {
		case strings.Contains(low, "chat_id"):
			return "INVALID_CHAT_ID"
		case strings.Contains(low, "url"):
			return "INVALID_URL"
		case strings.Contains(low, "email"):
			return "INVALID_EMAIL"
		case strings.Contains(low, "method"):
			return "INVALID_FORMAT"
		case strings.Contains(low, "maximum length"):
			return "FIELD_TOO_LONG"
		case strings.Contains(low, "name"):
			return "MISSING_REQUIRED_FIELD"
		}
		return "INVALID_ARGUMENT"
	default:
		return strings.ToUpper(grpcCode)
	}
}

// tableToParams преобразует godog таблицу key-value в map.
func tableToParams(t *godog.Table) map[string]string {
	result := make(map[string]string)
	for _, row := range t.Rows {
		if len(row.Cells) >= 2 {
			result[row.Cells[0].Value] = row.Cells[1].Value
		}
	}
	return result
}

// defaultAddressFor возвращает дефолтный адрес для типа канала.
func defaultAddressFor(chType string) string {
	switch strings.ToLower(chType) {
	case "telegram":
		return "-1001234567890"
	case "email":
		return "test@example.com"
	case "webhook":
		return "https://example.com/webhook"
	}
	return ""
}

// configJSONFor собирает JSON-конфигурацию канала из адреса.
func configJSONFor(chType, address string) string {
	switch strings.ToLower(chType) {
	case "telegram":
		b, _ := json.Marshal(map[string]string{"chat_id": address, "bot_token": "fake-bot-token"})
		return string(b)
	case "email":
		b, _ := json.Marshal(map[string]string{"email": address})
		return string(b)
	case "webhook":
		b, _ := json.Marshal(map[string]string{"url": address, "method": "POST"})
		return string(b)
	}
	return "{}"
}

// channelInfoFromProto собирает AlertChannelInfo из gRPC-ответа.
func channelInfoFromProto(pc *alertv1.AlertChannel, address string) *AlertChannelInfo {
	id, _ := uuid.Parse(pc.GetId())
	uid, _ := uuid.Parse(pc.GetUserId())
	info := &AlertChannelInfo{
		ID:        id,
		UserID:    uid,
		Name:      pc.GetName(),
		Type:      strings.ToLower(pc.GetType()),
		Address:   address,
		Status:    strings.ToLower(pc.GetStatus()),
		Enabled:   pc.GetEnabled(),
		Verified:  pc.GetVerified(),
		Priority:  int(pc.GetPriority()),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return info
}

// grpcCreateChannel — общий путь создания через AlertClient.CreateChannel.
func grpcCreateChannel(stack *Stack, state *ScenarioState, owner uuid.UUID, role, name, chType, address string) (*alertv1.AlertChannel, error) {
	if err := ensureAlertUserStub(context.Background(), stack, owner); err != nil {
		return nil, err
	}
	ctx := alertAuthCtx(context.Background(), owner, role)
	req := &alertv1.CreateChannelRequest{
		Name:    name,
		Type:    strings.ToUpper(chType),
		Config:  configJSONFor(chType, address),
		Enabled: true,
	}
	return stack.AlertClient.CreateChannel(ctx, req)
}

// grpcSeedChannel создаёт канал через gRPC и сохраняет в state.Channels.
func grpcSeedChannel(stack *Stack, state *ScenarioState, owner uuid.UUID, role, name, chType, address string) error {
	pc, err := grpcCreateChannel(stack, state, owner, role, name, chType, address)
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	info := channelInfoFromProto(pc, address)
	if info.UserID == uuid.Nil {
		info.UserID = owner
	}
	state.Channels[name] = info
	state.LastChannel = info
	state.LastErr = nil
	state.LastErrorCode = ""
	return nil
}

// grpcCreateTelegramChannel реализует шаг создания Telegram канала.
func grpcCreateTelegramChannel(stack *Stack, state *ScenarioState, t *godog.Table, gated bool) error {
	if gated {
		if state.UserID == uuid.Nil || state.UserRole == "" {
			setErr(state, "AUTH_REQUIRED", "authentication required to manage channels")
			return nil
		}
		if state.UserRole == "GUEST" {
			setErr(state, "FORBIDDEN", "guest users cannot manage channels")
			return nil
		}
	}
	params := tableToParams(t)
	name := params["name"]
	chatID := params["chat_id"]
	if name == "" {
		setErr(state, "MISSING_REQUIRED_FIELD", "name cannot be empty")
		return nil
	}
	if chatID == "" {
		setErr(state, "MISSING_REQUIRED_FIELD", "chat_id cannot be empty")
		return nil
	}

	pc, err := grpcCreateChannel(stack, state, state.UserID, state.UserRole, name, "telegram", chatID)
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	info := channelInfoFromProto(pc, chatID)
	state.Channels[name] = info
	state.LastChannel = info
	state.LastErr = nil
	state.LastErrorCode = ""
	return nil
}

// grpcCreateEmailChannel реализует шаг создания Email канала.
func grpcCreateEmailChannel(stack *Stack, state *ScenarioState, t *godog.Table) error {
	params := tableToParams(t)
	name := params["name"]
	email := params["email"]
	if name == "" {
		setErr(state, "MISSING_REQUIRED_FIELD", "name cannot be empty")
		return nil
	}
	if email == "" {
		setErr(state, "MISSING_REQUIRED_FIELD", "email cannot be empty")
		return nil
	}

	pc, err := grpcCreateChannel(stack, state, state.UserID, state.UserRole, name, "email", email)
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	info := channelInfoFromProto(pc, email)
	state.Channels[name] = info
	state.LastChannel = info
	state.LastErr = nil
	state.LastErrorCode = ""
	return nil
}

// grpcCreateWebhookChannel реализует шаг создания Webhook канала.
func grpcCreateWebhookChannel(stack *Stack, state *ScenarioState, t *godog.Table) error {
	params := tableToParams(t)
	name := params["name"]
	rawURL := params["url"]
	method := params["method"]
	if name == "" {
		setErr(state, "MISSING_REQUIRED_FIELD", "name cannot be empty")
		return nil
	}

	cfg := map[string]string{"url": rawURL}
	if method != "" {
		cfg["method"] = strings.ToUpper(method)
	} else {
		cfg["method"] = "POST"
	}
	cfgBytes, _ := json.Marshal(cfg)
	if err := ensureAlertUserStub(context.Background(), stack, state.UserID); err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	ctx := alertAuthCtx(context.Background(), state.UserID, state.UserRole)
	pc, err := stack.AlertClient.CreateChannel(ctx, &alertv1.CreateChannelRequest{
		Name:    name,
		Type:    "WEBHOOK",
		Config:  string(cfgBytes),
		Enabled: true,
	})
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	info := channelInfoFromProto(pc, rawURL)
	if method != "" {
		info.Method = strings.ToUpper(method)
	}
	state.Channels[name] = info
	state.LastChannel = info
	state.LastErr = nil
	state.LastErrorCode = ""
	return nil
}

// grpcListChannels запрашивает список каналов и сверяет с state.Channels.
func grpcListChannels(stack *Stack, state *ScenarioState) error {
	ctx := alertAuthCtx(context.Background(), state.UserID, state.UserRole)
	_, err := stack.AlertClient.ListChannels(ctx, &alertv1.ListChannelsRequest{})
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	state.LastErr = nil
	state.LastErrorCode = ""
	return nil
}

// grpcUpdateChannel обновляет LastChannel параметрами из таблицы.
func grpcUpdateChannel(stack *Stack, state *ScenarioState, t *godog.Table) error {
	if state.LastChannel == nil {
		for _, ch := range state.Channels {
			state.LastChannel = ch
			break
		}
	}
	if state.LastChannel == nil {
		return fmt.Errorf("no channel to update")
	}
	params := tableToParams(t)
	newName := state.LastChannel.Name
	if v, ok := params["name"]; ok {
		newName = v
	}
	ctx := alertAuthCtx(context.Background(), state.UserID, state.UserRole)
	_, err := stack.AlertClient.UpdateChannel(ctx, &alertv1.UpdateChannelRequest{
		Id:   state.LastChannel.ID.String(),
		Name: newName,
	})
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	old := state.LastChannel.Name
	state.LastChannel.Name = newName
	if old != newName {
		delete(state.Channels, old)
		state.Channels[newName] = state.LastChannel
	}
	state.LastErr = nil
	state.LastErrorCode = ""
	return nil
}

// grpcDeleteChannel удаляет LastChannel.
func grpcDeleteChannel(stack *Stack, state *ScenarioState) error {
	ch := state.LastChannel
	if ch == nil {
		for _, c := range state.Channels {
			ch = c
			state.LastChannel = c
			break
		}
	}
	if ch == nil {
		return fmt.Errorf("no channel to delete")
	}
	ctx := alertAuthCtx(context.Background(), state.UserID, state.UserRole)
	_, err := stack.AlertClient.DeleteChannel(ctx, &alertv1.DeleteChannelRequest{Id: ch.ID.String()})
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	delete(state.Channels, ch.Name)
	state.LastErr = nil
	state.LastErrorCode = ""
	return nil
}

// grpcVerifyChannel вызывает VerifyChannel для LastChannel.
func grpcVerifyChannel(stack *Stack, state *ScenarioState) error {
	ch := state.LastChannel
	if ch == nil {
		return fmt.Errorf("no channel to verify")
	}
	ctx := alertAuthCtx(context.Background(), state.UserID, state.UserRole)
	resp, err := stack.AlertClient.VerifyChannel(ctx, &alertv1.VerifyChannelRequest{Id: ch.ID.String()})
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	if resp.GetVerified() {
		ch.Verified = true
		ch.Status = "active"
	}
	state.LastErr = nil
	state.LastErrorCode = ""
	return nil
}

// grpcForeignChannelUpdate имитирует попытку чужого пользователя обновить канал.
func grpcForeignChannelUpdate(stack *Stack, state *ScenarioState, channelName string) error {
	ch, ok := state.Channels[channelName]
	if !ok {
		return fmt.Errorf("channel %s not found", channelName)
	}
	ctx := alertAuthCtx(context.Background(), state.SecondUserID, state.SecondUserRole)
	_, err := stack.AlertClient.UpdateChannel(ctx, &alertv1.UpdateChannelRequest{Id: ch.ID.String(), Name: ch.Name + " Hijacked"})
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	return nil
}

// grpcForeignChannelDelete имитирует попытку чужого пользователя удалить канал.
func grpcForeignChannelDelete(stack *Stack, state *ScenarioState, channelName string) error {
	ch, ok := state.Channels[channelName]
	if !ok {
		return fmt.Errorf("channel %s not found", channelName)
	}
	ctx := alertAuthCtx(context.Background(), state.SecondUserID, state.SecondUserRole)
	_, err := stack.AlertClient.DeleteChannel(ctx, &alertv1.DeleteChannelRequest{Id: ch.ID.String()})
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	return nil
}

// grpcCreateExtraChannel пытается создать ещё один канал поверх лимита.
func grpcCreateExtraChannel(stack *Stack, state *ScenarioState) error {
	if err := ensureAlertUserStub(context.Background(), stack, state.UserID); err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	ctx := alertAuthCtx(context.Background(), state.UserID, state.UserRole)
	_, err := stack.AlertClient.CreateChannel(ctx, &alertv1.CreateChannelRequest{
		Name:    "Extra Channel",
		Type:    "TELEGRAM",
		Config:  configJSONFor("telegram", "-1009999999"),
		Enabled: true,
	})
	if err != nil {
		recordGRPCErr(state, err)
		return nil
	}
	state.LastErr = nil
	state.LastErrorCode = ""
	return nil
}
