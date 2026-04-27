# Billing Service

Сервис тарифов, подписок и платежей. Реализует gRPC API для управления подписками пользователей и обрабатывает webhook'и от платёжных провайдеров.

## Возможности

- Каталог тарифных планов
- Создание checkout-сессий и оплата подписок
- История платежей пользователя
- Отмена подписок
- Обработка webhook'ов от платёжных провайдеров (`payment.succeeded`, `payment.failed`)
- Активация/деактивация подписок
- Публикация событий в RabbitMQ
- gRPC API с JWT аутентификацией

## Use cases

| ID | Сценарий |
|---|---|
| `uc_04_01_01` | Получение списка тарифных планов |
| `uc_04_01_02` | Получение информации о подписке |
| `uc_04_01_03` | Создание checkout-сессии |
| `uc_04_01_04` | Получение истории платежей |
| `uc_04_01_05` | Отмена подписки |
| `uc_04_01_06` | Обработка webhook (`payment.succeeded` / `payment.failed`) |

## Архитектура

```
billing-service/
├── cmd/billing-service/   # Точка входа
├── internal/
│   ├── model/             # Доменные модели
│   ├── repository/        # PostgreSQL репозитории
│   ├── service/           # Бизнес-логика
│   ├── handler/           # gRPC обработчики
│   ├── infrastructure/    # Конфиг, JWT, RabbitMQ, провайдеры
│   └── adapters/          # Связка слоёв
├── migrations/            # Миграции БД
└── pkg/                   # logger, telemetry
```

## Конфигурация

Переменные окружения см. в `Dockerfile`/`docker-compose.yml` родительского `backend/`. Минимум: `DB_*`, `RABBITMQ_*`, `JWT_SECRET`, ключи провайдеров (`YOOKASSA_*`, `YANDEX_*`, `STRIPE_*`).

## Тестирование

- Unit-тесты — рядом с кодом, запускаются через `go test -short -race ./...`
- BDD-сценарии — в корневом `features/04_billing/` (запускаются через корневой Stack, см. `features/suite/steps_04_billing.go`)
