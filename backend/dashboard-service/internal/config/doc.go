// Package config предоставляет конфигурацию dashboard-service.
//
// Использование:
//
//	cfg, err := config.Load()
//	addr := cfg.ServerAddress()
//
// Переменные окружения:
//
//	SERVER_PORT   — порт gRPC сервера (default: 9095)
//	HTTP_PORT     — порт HTTP/WebSocket сервера (default: 8095)
//	DB_HOST       — хост PostgreSQL (default: localhost)
//	DB_PORT       — порт PostgreSQL (default: 5432)
//	DB_NAME       — имя базы данных (default: dashboard)
//	DB_USER       — пользователь БД (default: postgres)
//	DB_PASSWORD   — пароль БД (default: postgres)
//	RABBITMQ_URL  — URL RabbitMQ (default: amqp://guest:guest@localhost:5672/)
//	JWT_SECRET    — секретный ключ JWT
//	OTEL_SERVICE_NAME — имя сервиса для трейсинга (default: dashboard-service)
package config
