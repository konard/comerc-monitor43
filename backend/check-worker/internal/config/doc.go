// Package config содержит конфигурацию check-worker.
//
// Конфигурация загружается из переменных окружения через caarlos0/env.
// Поддерживается .env файл через godotenv.
//
// Основные параметры:
//   - CHECK_WORKER_SCHEDULER_ADDRESS: адрес scheduler-service для регистрации и heartbeat
//   - CHECK_WORKER_MONITOR_ADDRESS: адрес monitor-service для отправки результатов проверок
//   - CHECK_WORKER_NAME: уникальное имя воркера
//   - CHECK_WORKER_ZONE: зона доступности для привязки проверок
//   - CHECK_WORKER_MAX_CONCURRENT_CHECKS: максимальное число параллельных проверок
//   - CHECK_WORKER_HEARTBEAT_INTERVAL: интервал отправки heartbeat
package config
