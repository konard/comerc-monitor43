// Package channels предоставляет бизнес-логику работы с каналами уведомлений.
//
// Основные компоненты:
//   - ChannelService: Сервис для управления каналами уведомлений пользователя
//   - Верификация каналов: Проверка доступности email, Telegram, webhook
//   - Получение списка: Список настроенных каналов пользователя
//
// Использование:
//
//	channelService := channels.NewChannelService(
//	    alertChannelRepo,
//	    telegramClient,
//	    emailClient,
//	    webhookClient,
//	    tracer,
//	)
//	channels, err := channelService.GetUserChannels(ctx, userID)
//
// Ключевые особенности:
//   - Верификация каналов перед сохранением
//   - Поддержка разных типов каналов (Email, Telegram, Webhook, SMS)
//   - Обработка ошибок верификации с контекстом
//   - Асинхронная проверка доступности каналов
//
// Контекст DOD:
//   - Соответствует требованию DOD 7.1 Input Validation
//   - Валидация адресов email, webhook URL
//   - Проверка доступности каналов перед использованием
package channels
