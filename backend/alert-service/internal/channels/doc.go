// Package channels предоставляет клиенты для отправки уведомлений.
//
// Основные компоненты:
//   - TelegramClient: Клиент для отправки сообщений через Telegram API
//   - EmailClient: Клиент для отправки email через SMTP
//   - WebhookClient: Клиент для HTTP webhook вызовов
//
// Использование:
//
//	telegramClient := channels.NewTelegramClient(botToken, "")
//	emailClient := channels.NewEmailClient(host, port, from)
//	webhookClient := channels.NewWebhookClient(timeout)
//
// Ключевые особенности:
//   - Поддержка таймаутов через context
//   - Обработка ошибок с контекстом
//   - Асинхронная отправка с goroutines для блокирующих операций
//
// Контекст DOD:
//   - Соответствует требованию DOD 7.3 Secrets Management
//   - Секреты только через переменные окружения
//   - Поддержка внешних зависимостей с proper error handling
package channels
