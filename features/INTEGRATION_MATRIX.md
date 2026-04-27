# INTEGRATION_MATRIX.md — Матрица интеграций между эпиками

Документирует точки интеграции между эпиками системы мониторинга.
Каждая запись: источник события → потребитель → механизм → сценарии.

---

## Матрица зависимостей

| Источник | Потребитель | Событие | Механизм | Сценарии |
|----------|-------------|---------|----------|---------|
| Epic 01 (Мониторы) | Epic 02 (Алерты) | Изменение статуса монитора (DOWN/UP/DEGRADED) | gRPC вызов / внутренняя очередь | uc_02_02_20, uc_02_02_22 |
| Epic 02 (Алерты) | Epic 02 (Доставка) | Создание алерта | Внутренний вызов сервиса | uc_02_03_01, uc_02_03_02, uc_02_03_03 |
| Epic 08 (Maintenance) | Epic 02 (Алерты) | Начало/конец maintenance window | gRPC вызов | uc_02_03_20 |
| Epic 01 (Мониторы) | Epic 08 (Maintenance) | Плановые работы на мониторе | gRPC вызов | — |
| Epic 04 (Биллинг) | Epic 01 (Мониторы) | Лимиты подписки на количество мониторов | gRPC вызов | — |
| Epic 06 (Отчёты) | Epic 01 (Мониторы) | SLA расчёты запрашивают историю монитора | gRPC вызов | — |
| Epic 08 (Maintenance) | Epic 06 (Отчёты) | Время maintenance исключается из SLA | gRPC вызов | — |

---

## Epic 02 — Алертинг (внутренние интеграции)

### Alert Channels → Alert Triggering
- Канал создаётся/изменяется/удаляется пользователем
- Alert rule ссылается на channel по ID
- При удалении канала: alert rules с этим каналом блокируются
- Сценарии: `uc_02_01_05`, `uc_02_02_42`

### Alert Triggering → Alert Delivery
- Срабатывание правила (N consecutive failures) создаёт Alert
- Alert передаётся в delivery pipeline
- Recovery создаёт recovery-уведомление
- Сценарии: `uc_02_02_20`, `uc_02_03_01`, `uc_02_03_04`

### Alert Delivery → Alert Channels
- Delivery читает конфигурацию канала (тип, credentials, приоритет)
- При постоянной ошибке доставки канал переводится в DISABLED
- Сценарии: `uc_02_01_12`, `uc_02_03_15`

### Maintenance Windows → Alert Suppression
- Активное maintenance window подавляет доставку алертов
- Алерт создаётся, но не доставляется (suppressed=true)
- После окончания maintenance: алерты доставляются если монитор ещё DOWN
- Сценарии: `uc_02_03_20`

---

## Epic 01 → Epic 02: Триггеры алертов

**Контракт:**
```
Событие: MonitorStatusChanged
Поля: monitor_id, old_status, new_status, consecutive_failures, timestamp
Источник: monitor-service
Потребитель: alert-service
```

**Логика:**
1. `new_status = DOWN` и `consecutive_failures >= threshold` → создать Alert (FIRING)
2. `new_status = UP` и существует активный Alert → создать Alert (RESOLVED)
3. `new_status = DEGRADED` → создать Alert с severity=WARNING

---

## Epic 08 → Epic 02: Подавление алертов

**Контракт:**
```
Запрос: MaintenanceActive(monitor_id, timestamp)
Ответ: bool (активно ли maintenance)
Источник: maintenance-service
Потребитель: alert-service (при каждой доставке)
```

---

*Версия: 1.0 | Обновлено: 2026-04-03*
