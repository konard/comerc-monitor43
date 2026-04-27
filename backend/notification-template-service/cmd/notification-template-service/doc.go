// Package main — точка входа notification-template-service.
//
// Сервис управляет шаблонами уведомлений для алертов.
//
// Функциональность:
//
//   - CRUD операций с шаблонами
//   - Рендеринг шаблонов с переменными
//   - Валидация синтаксиса шаблонов
//   - Управление шаблонами по умолчанию
//   - Поддержка различных каналов (Email, Telegram, Webhook, Slack, Discord, SMS)
//   - Поддержка типов уведомлений (Monitor Up/Down, Degraded, Certificate expiry, etc.)
//
// Использование:
//
//	# Запуск сервиса
//	./notification-template-service --config config/config.yaml
//
//	# Сгенерировать gRPC код
//	make generate
package main
