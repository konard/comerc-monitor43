// Package delivery предоставляет бизнес-логику доставки уведомлений.
//
// Основные компоненты:
//   - AlertDeliveryService: Сервис для асинхронной доставки уведомлений
//   - ServiceOption: Функциональные опции для конфигурации сервиса
//   - Background worker: Фоновый процессор pending попыток доставки
//
// Использование:
//
//	alertDeliveryService := NewAlertDeliveryService(
//	    deliveryAttemptRepo,
//	    alertChannelRepo,
//	    alertRepo,
//	    telegramClient,
//	    emailClient,
//	    webhookClient,
//	    tracer,
//	    metrics,
//	    WithMaxRetries(3),
//	    WithRetryInterval(5*time.Minute),
//	)
//	go alertDeliveryService.ProcessPendingAttempts(context.Background())
//
// Ключевые особенности:
//   - Асинхронная доставка уведомлений через background worker
//   - Retry логика с фиксированным интервалом
//   - Поддержка нескольких каналов (Telegram, Email, Webhook)
//   - Обработка ошибок и логирование результатов
//
// Контекст DOD:
//   - Соответствует требованию DOD 5.4 Error Handling
//   - Различие между retryable и non-retryable ошибками
//   - Dead letter queue для неудачных доставок
package delivery
