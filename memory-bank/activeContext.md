# Active Context

**Last Updated:** 2026-04-03

## Current Work Focus

**Фаза 1 (Build Blockers) — ЗАВЕРШЕНА.** Все 4 сервиса с BUILD FAILURE починены. Следующий этап — Фаза 2: Coverage >80%.

---

## Backend DOD Status (2026-04-03, после Фазы 1)

Источник: `backend/DOD.md` v2.2, `DOD.md` (root), `features/DOD.md` v1.3.

| Сервис | Build | Coverage | Infra | DOD |
|---|---|---|---|---|
| **alert-service** | ✅ | **85.3%** ✅ | All ✅ | ✅ COMPLIANT |
| auth-service | ✅ | **17.1%** ❌ | Dockerfile ❌, dc ❌ | ❌ |
| billing-service | ✅ | — | Dockerfile ❌, dc ❌ | ❌ |
| check-worker | ✅ | ⚠️ | migrations ❌ | ❌ |
| dashboard-service | ✅ | **33.9%** ❌ | All ✅ | ❌ |
| gateway | ✅ | — | Dockerfile ✅, .env ✅, dc ✅ | ❌ |
| integration-service | ✅ | **20.7%** ❌ | All ✅ | ❌ |
| monitor-service | ✅ | **54.7%** ❌ | All ✅ | ❌ |
| notification-template | ✅ | — | .env ✅ | ❌ |
| reporting-service | ✅ | **66.4%** ❌ | All ✅ | ❌ |
| scheduler-service | ✅ | — | All ✅ | ❌ |

### Что сделано в Фазе 1

- **scheduler-service**: удалены дубликаты методов, добавлен `createAuditLog`, исправлен type mismatch
- **billing-service**: `go mod tidy`, исправлены proto импорты (`api` → `api/proto`), adapter APIs, TODO убраны
- **gateway**: `go mod tidy`, созданы Dockerfile, docker-compose.yml, .env.example
- **notification-template-service**: `go mod tidy`, созданы .env.example, исправлены proto imports, sqlx.ErrNotFound → sql.ErrNoRows, добавлен `ZLogger()` в logging wrapper, починены тесты

### Ключевые изменения в api/proto

- Сгенерирован `notification_template.pb.go` с корректными путями (`proto/common.proto` вместо `common.proto`)
- Импорт в обоих сервисах изменён с `github.com/raul/monitor/api` → `github.com/raul/monitor/api/proto`

---

## Features DOD Status (2026-04-03)

24 feature-файла, 718 сценариев.

| Метрика | Факт | Цель | |
|---|---|---|---|
| `@use_case` coverage | 100% | 100% | ✅ |
| Структура (@epic, @user_story) | 100% | 100% | ✅ |
| Error code формат | 100% | 100% | ✅ |
| Alternative coverage (общее) | **12.4%** | >80% | ❌ |
| ERROR_CODES.md | отсутствует | обязателен | ❌ |
| INTEGRATION_MATRIX.md | отсутствует | обязателен | ❌ |
| RETRY_STRATEGIES.md | отсутствует | обязателен | ❌ |

Alert-service features: ✅ 80–81% alternative coverage (единственный compliant epic).

---

## Next Steps

### Приоритет 1 — Coverage >80% (Фаза 2)
1. auth-service (17% → 80%) — handler + repository + инфраструктура
2. integration-service (21% → 80%) — handler + repository + webhook service
3. dashboard-service (34% → 80%) — repository + consumer + WebSocket
4. monitor-service (55% → 80%) — middleware + repository + service
5. reporting-service (66% → 80%) — model + monitor_client + repository

### Приоритет 2 — Infra (Фаза 3, параллельно)
- auth-service: Dockerfile + docker-compose
- billing-service: Dockerfile + docker-compose
- check-worker: migrations/ + тестовый timeout

### Приоритет 3 — Features DOD (Фаза 4)
- Создать ERROR_CODES.md, INTEGRATION_MATRIX.md, RETRY_STRATEGIES.md
- Добавить alternative сценарии до >80% в epics 01, 03, 05, 07, 08

---

## Important Patterns

1. **Constructor:** `New()` — одно значение (не ошибку), Config по значению
2. **Errors:** `errors.Wrap(err, "failed to ...")`, никогда не игнорировать
3. **Комментарии:** русский; log/error messages — английский, строчная первая буква
4. **Graceful Shutdown:** SIGINT/SIGTERM, timeout 30s
5. **Health:** gRPC health probe (grpc_health_probe -addr=:PORT)
6. **Auth:** JWT interceptor → проверка user_id ресурса в handlers
