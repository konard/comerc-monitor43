// Package config загружает и валидирует конфигурацию сервиса аутентификации из переменных окружения.
//
// Основные группы переменных:
//   - SERVER_PORT, SERVER_GRPC_PORT — порты HTTP и gRPC серверов
//   - DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SSL_MODE — параметры PostgreSQL
//   - REDIS_HOST, REDIS_PORT, REDIS_PASSWORD, REDIS_DB — параметры Redis
//   - RABBITMQ_HOST, RABBITMQ_PORT, RABBITMQ_USER, RABBITMQ_PASSWORD, RABBITMQ_VHOST, RABBITMQ_ENABLED — параметры RabbitMQ
//   - JWT_SECRET, JWT_ACCESS_DURATION, JWT_REFRESH_DURATION — параметры JWT
//   - GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URI — параметры Google OAuth
//   - MAX_LOGIN_ATTEMPTS, ACCOUNT_LOCK_DURATION, SESSION_EXPIRY — настройки безопасности
//   - OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_SERVICE_NAME, OTEL_LOG_LEVEL — параметры observability
package config
