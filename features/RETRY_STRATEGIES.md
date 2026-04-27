# RETRY_STRATEGIES.md — Стратегии повторных попыток и восстановления

Канонический источник retry-логики для всех сервисов системы мониторинга.
При реализации retry в feature сценариях — ссылайтесь на этот файл.

---

## Стратегии

### 1. Exponential Backoff with Jitter (по умолчанию для webhook/API)

```
delay_ms = base_delay_ms × (2 ^ attempt_number) + random_jitter(−25%, +25%)
```

| Параметр | Значение |
|----------|----------|
| `base_delay` | 30 секунд |
| `max_delay` | 60 секунд |
| `max_attempts` | 3 |
| `jitter` | ±25% от рассчитанной задержки |

**Применяется:** webhook доставка, внешние API вызовы
**Сценарии:** `uc_02_03_09`, `uc_02_03_17`, `uc_02_03_18`

---

### 2. Linear Backoff (для платёжных операций)

```
delay_ms = base_delay_ms × attempt_number
```

| Параметр | Значение |
|----------|----------|
| `base_delay` | 1 час |
| `max_attempts` | 3 |

**Применяется:** payment processing, идемпотентные операции с внешними системами

---

### 3. Fixed Delay (для БД и кэша)

| Параметр | Значение |
|----------|----------|
| `delay` | 1 секунда |
| `max_attempts` | 5 |

**Применяется:** database reconnect, cache miss fallback

---

## Retryable error codes

Следующие коды ошибок допускают повторную попытку:

| Код | Стратегия | Описание |
|-----|-----------|----------|
| `DELIVERY_TIMEOUT` | Exponential Backoff | Таймаут webhook/email |
| `DELIVERY_EMAIL_SERVICE_UNAVAILABLE` | Exponential Backoff | Email сервис временно недоступен |
| `ALERT_MESSAGE_QUEUE_UNAVAILABLE` | Fixed Delay | Очередь сообщений недоступна |
| `ALERT_DATABASE_TIMEOUT` | Fixed Delay | Таймаут БД |
| `SERVICE_UNAVAILABLE` | Exponential Backoff | Сервис временно недоступен (503) |
| `TIMEOUT` | Exponential Backoff | Общий таймаут (504) |

---

## Non-retryable error codes

Следующие коды ошибок НЕ допускают повторную попытку:

| Код | Описание |
|-----|----------|
| `DELIVERY_PERMANENT_ERROR` | Постоянная ошибка (напр. неверный адрес) |
| `DELIVERY_SSL_CERTIFICATE_INVALID` | Невалидный SSL сертификат |
| `INVALID_URL` | Невалидный URL webhook |
| `INVALID_EMAIL` | Невалидный email адрес |
| `AUTH_REQUIRED` | Требуется аутентификация |
| `FORBIDDEN` | Доступ запрещён |
| `ALERT_ALREADY_ACKNOWLEDGED` | Алерт уже подтверждён |
| `ALERT_NOT_ACKNOWLEDGEABLE` | Недопустимый статус для подтверждения |
| `CONFLICT` | Конфликт состояния (409) |

---

## Dead Letter Queue (DLQ)

При исчерпании всех попыток:

1. Событие помещается в DLQ
2. Записывается audit log с `delivery_retry_exhausted`
3. Канал переводится в `DISABLED` если ошибка постоянная
4. Оператор уведомляется (если настроен fallback канал)

**Сценарии:** `uc_02_03_09` (retry exhausted)

---

## Cooldown между алертами

Cooldown — отдельный механизм от retry:

| Параметр | Значение |
|----------|----------|
| Период cooldown | 15 минут |
| Область применения | Per-monitor, per-status (DOWN и DEGRADED независимы) |
| Сброс | При успешном восстановлении монитора |

**Сценарии:** `uc_02_02_13`, `uc_02_02_14`, `uc_02_02_54`, `uc_02_02_56`

---

*Версия: 1.0 | Обновлено: 2026-04-03*
