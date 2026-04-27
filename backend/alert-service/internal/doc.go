// Package internal содержит всю приватную логику alert-service.
//
// Структура субпакетов:
//   - model:          доменные сущности и ошибки
//   - repository:     интерфейсы репозиториев и реализация (postgres)
//   - service:        бизнес-логика (channels, delivery, triggering, mute, flapping, escalation, maintenance, throttling, audit, dto)
//   - handler:        gRPC-обработчики и HTTP-gateway аннотации
//   - infrastructure: внешние адаптеры (auth JWT, gRPC interceptors)
//   - config:         конфигурация из переменных окружения
//   - health:         gRPC health check
//   - channels:       клиенты внешних каналов (Telegram, Email, Webhook)
//   - adapters:       инициализация и компоновка адаптеров
//   - testutil:       вспомогательные утилиты для интеграционных тестов
package internal
