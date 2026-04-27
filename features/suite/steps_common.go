package suite

import (
	"strings"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
)

// providerKey нормализует имя OAuth-провайдера к внутреннему ключу.
func providerKey(name string) string {
	switch strings.ToLower(name) {
	case "google":
		return "google"
	case "github":
		return "github"
	case "yandex":
		return "yandex"
	case "vk":
		return "vk"
	case "mail.ru":
		return "mailru"
	case "telegram":
		return "telegram"
	default:
		return strings.ToLower(name)
	}
}

// RegisterCommonAuthSteps регистрирует общие Given-шаги авторизации.
func RegisterCommonAuthSteps(ctx *godog.ScenarioContext, state *ScenarioState) {
	ctx.Step(`^пользователь авторизован$`, func() error {
		authorizePrimaryUser(state, "USER")
		return nil
	})

	ctx.Step(`^пользователь с ролью "([^"]*)"$`, func(role string) error {
		authorizePrimaryUser(state, role)
		return nil
	})

	ctx.Step(`^пользователь "([^"]*)" с ролью "([^"]*)"$`, func(name, role string) error {
		authorizeNamedUser(state, name, role)
		return nil
	})

	ctx.Step(`^пользователь "([^"]*)" авторизован$`, func(name string) error {
		authorizeNamedUser(state, name, "USER")
		return nil
	})

	ctx.Step(`^пользователь не авторизован$`, func() error {
		state.UserID = uuid.Nil
		state.UserRole = ""
		return nil
	})
}

func authorizePrimaryUser(state *ScenarioState, role string) {
	if state.UserID == uuid.Nil {
		state.UserID = uuid.New()
	}
	state.UserRole = role
	if state.MonitorID == uuid.Nil {
		state.MonitorID = uuid.New()
	}
}

func authorizeNamedUser(state *ScenarioState, name, role string) {
	if isPrimaryBDDUser(name) {
		authorizePrimaryUser(state, role)
		return
	}
	if state.SecondUserID == uuid.Nil {
		state.SecondUserID = uuid.New()
	}
	state.SecondUserRole = role
}

func isPrimaryBDDUser(name string) bool {
	switch strings.ToLower(name) {
	case "user1", "usera":
		return true
	default:
		return false
	}
}
