//go:build bdd

package suite

import (
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
)

// RegisterAlertingAuthSteps регистрирует alerting-specific auth context.
func RegisterAlertingAuthSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	ensureAlertingMaps(state)

	ctx.Step(`^пользователь "([^"]*)" имеет канал "([^"]*)"$`, func(name, channelName string) error {
		var ownerID uuid.UUID
		role := "USER"
		switch name {
		case "user1":
			if state.UserID == uuid.Nil {
				state.UserID = uuid.New()
			}
			if state.UserRole == "" {
				state.UserRole = role
			}
			ownerID = state.UserID
			role = state.UserRole
		case "user2":
			if state.SecondUserID == uuid.Nil {
				state.SecondUserID = uuid.New()
			}
			if state.SecondUserRole == "" {
				state.SecondUserRole = role
			}
			ownerID = state.SecondUserID
			role = state.SecondUserRole
		default:
			ownerID = state.UserID
			role = state.UserRole
		}
		return grpcSeedChannel(stack, state, ownerID, role, channelName, "telegram", "-1001234567890")
	})

	// Затухание неиспользуемых параметров на этапе линтинга.
	_ = stack
	_ = time.Now
}

// ensureAlertingMaps лениво инициализирует map-поля состояния для эпика alerting.
func ensureAlertingMaps(state *ScenarioState) {
	if state.Channels == nil {
		state.Channels = make(map[string]*AlertChannelInfo)
	}
	if state.Alerts == nil {
		state.Alerts = make(map[uuid.UUID]*AlertAlertInfo)
	}
	if state.AlertRules == nil {
		state.AlertRules = make(map[uuid.UUID]*AlertRuleInfo)
	}
}

// newAlertChannel создаёт новый канал с дефолтами.
func newAlertChannel(userID uuid.UUID, name, chType, address string) *AlertChannelInfo {
	now := time.Now()
	ch := &AlertChannelInfo{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		Type:      chType,
		Address:   address,
		Status:    "unverified",
		Enabled:   true,
		Verified:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if chType == "webhook" {
		ch.Method = "POST"
	}
	return ch
}
