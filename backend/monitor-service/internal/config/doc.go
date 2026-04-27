// Package config предоставляет конфигурацию monitor-service.
//
// Пакет использует библиотеку envconfig для загрузки конфигурации из
// переменных окружения с валидацией и значениями по умолчанию.
//
// Основные компоненты:
//   - Config: Главная структура конфигурации
//   - LoadConfig(): Загрузка и валидация конфигурации
//
// Использование:
//
//	cfg, err := config.LoadConfig()
//	if err != nil {
//		log.Fatalf("failed to load config: %v", err)
//	}
//
// Переменные окружения:
//
//	DB_HOST=localhost           - Хост PostgreSQL
//	DB_PORT=5432                - Порт PostgreSQL
//	DB_NAME=monitor             - Имя базы данных
//	DB_USER=postgres            - Пользователь БД
//	DB_PASSWORD=password        - Пароль БД
//
//	RABBITMQ_URL=amqp://...     - URL RabbitMQ
//
//	OTEL_SERVICE_NAME=monitor   - Имя сервиса для трассировки
//
// Полный список переменных см. в .env.example
package config
