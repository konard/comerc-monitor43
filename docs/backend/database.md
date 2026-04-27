# Database Architecture

## Технологический стек

**Primary Database:** PostgreSQL 15+
**Provider:** Timeweb Cloud (соответствие 152-ФЗ)
**Migration Tool:** Goose
**Backup Strategy:** Regular backups + Point-in-time recovery

---

## Ключевые решения

### Data Storage Strategy

**Только изменения, не каждый пинг:**
- Храним только **last state** монитора
- Фиксируем в БД только **изменения** (events)
- Игнорируем промежуточные проверки без смены статуса

**Retention (по тарифам):**
- Free: 7 дней
- Starter: 30 дней
- Professional: 90 дней
- Business: 1 год

### Cache Strategy

**Minimal cache approach:**
- Redis не для кэширования (изначально)
- Stateless JWT (session storage не нужен)
- Возможность добавить Redis позже для hot data

---

## Database Schema

### Core Tables

#### 1. `users`
Пользователи системы.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    tier VARCHAR(50) DEFAULT 'free', -- free, starter, professional, business

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- 152-ФЗ: encrypted fields
    phone_encrypted TEXT, -- field-level encryption
    -- indexes
    INDEX idx_users_email (email),
    INDEX idx_users_tier (tier)
);

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

#### 2. `monitors`
Текущее состояние мониторов (last state).

```sql
CREATE TABLE monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Configuration
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    method VARCHAR(10) DEFAULT 'GET', -- GET, HEAD
    interval_seconds INTEGER NOT NULL, -- 30, 60, 120, 300, 600, 1800
    timeout_seconds INTEGER DEFAULT 10,

    -- Auth & Headers (JSONB)
    basic_auth JSONB, -- {"username": "...", "password": "..."}
    headers JSONB, -- {"Authorization": "Bearer ..."}
    body_regex TEXT, -- keyword monitoring (v1.5+)

    -- Current State (updated on changes only)
    status VARCHAR(20) DEFAULT 'pending', -- pending, up, down, degraded
    last_check_time TIMESTAMP,
    last_error TEXT,
    last_response_time_ms INTEGER,

    -- Check config
    expected_status_codes INTEGER[], -- [200, 201, ...]
    follow_redirects BOOLEAN DEFAULT true,
    max_redirects INTEGER DEFAULT 5,

    -- Metadata
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Indexes
    INDEX idx_monitors_user_id (user_id),
    INDEX idx_monitors_status (status),
    INDEX idx_monitors_next_check (next_check_time),
    INDEX idx_monitors_enabled (enabled)
);

CREATE TRIGGER update_monitors_updated_at
    BEFORE UPDATE ON monitors
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

#### 3. `monitor_events`
История изменений (только события, не каждый пинг).

```sql
CREATE TABLE monitor_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,

    -- Event type
    event_type VARCHAR(50) NOT NULL, -- status_change, config_change, paused, resumed
    old_status VARCHAR(20), -- up, down, degraded
    new_status VARCHAR(20),

    -- Event details
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    response_time_ms INTEGER,
    error_message TEXT,

    -- Additional metadata (JSONB)
    metadata JSONB, -- {"check_interval": 60, "reason": "..."}

    -- Partitioning (будет добавлено позже)
    -- PARTITION BY RANGE (timestamp)

    -- Indexes
    INDEX idx_monitor_events_monitor_id (monitor_id),
    INDEX idx_monitor_events_timestamp (timestamp),
    INDEX idx_monitor_events_type (event_type)
);

-- Partitioning по месяцам (будет добавлено при масштабировании)
-- CREATE TABLE monitor_events_2026_03 PARTITION OF monitor_events
-- FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
```

#### 4. `alert_channels`
Каналы уведомлений пользователя.

```sql
CREATE TABLE alert_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Channel type
    type VARCHAR(20) NOT NULL, -- email, telegram, webhook
    enabled BOOLEAN DEFAULT true,

    -- Channel-specific config (JSONB)
    config JSONB NOT NULL,
    -- Email: {"address": "user@example.com"}
    -- Telegram: {"chat_id": "123456789", "bot_token": "..."}
    -- Webhook: {"url": "https://...", "method": "POST", "headers": {...}}

    -- Verification
    verified BOOLEAN DEFAULT false,
    verification_token TEXT,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Indexes
    INDEX idx_alert_channels_user_id (user_id),
    INDEX idx_alert_channels_type (type),
    INDEX idx_alert_channels_enabled (enabled)
);

CREATE TRIGGER update_alert_channels_updated_at
    BEFORE UPDATE ON alert_channels
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

#### 5. `alerts`
История отправленных алертов.

```sql
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    alert_channel_id UUID NOT NULL REFERENCES alert_channels(id) ON DELETE CASCADE,

    -- Alert details
    status VARCHAR(20) NOT NULL, -- triggered, acknowledged, resolved
    event_id UUID NOT NULL REFERENCES monitor_events(id),

    -- Delivery tracking
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    failed_at TIMESTAMP,
    error_message TEXT,

    -- Retry tracking
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),

    -- Indexes
    INDEX idx_alerts_monitor_id (monitor_id),
    INDEX idx_alerts_alert_channel_id (alert_channel_id),
    INDEX idx_alerts_status (status),
    INDEX idx_alerts_created_at (created_at)
);
```

### Billing Tables

#### 6. `subscriptions`
Подписки пользователей.

```sql
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Plan details
    tier VARCHAR(50) NOT NULL, -- free, starter, professional, business
    monitors_limit INTEGER NOT NULL,
    check_interval_min INTEGER NOT NULL, -- min seconds

    -- Payment details
    price_rub INTEGER NOT NULL, -- price in kopeks (500₽ = 50000)
    currency VARCHAR(3) DEFAULT 'RUB',

    -- Status
    status VARCHAR(20) NOT NULL, -- active, cancelled, expired, trial
    start_date DATE NOT NULL,
    end_date DATE,

    -- Yandex Kassa
    kassa_subscription_id TEXT,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Indexes
    INDEX idx_subscriptions_user_id (user_id),
    INDEX idx_subscriptions_status (status),
    INDEX idx_subscriptions_end_date (end_date)
);

CREATE TRIGGER update_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

#### 7. `payments`
История платежей.

```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id),

    -- Payment details
    amount_rub INTEGER NOT NULL, -- in kopeks
    currency VARCHAR(3) DEFAULT 'RUB',
    method VARCHAR(20) NOT NULL, -- sbp, card, invoice

    -- Yandex Kassa
    kassa_payment_id TEXT,
    kassa_confirmation_url TEXT,

    -- Status
    status VARCHAR(20) NOT NULL, -- pending, succeeded, failed, cancelled
    paid_at TIMESTAMP,

    -- Metadata
    metadata JSONB, -- additional payment details

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),

    -- Indexes
    INDEX idx_payments_user_id (user_id),
    INDEX idx_payments_status (status),
    INDEX idx_payments_created_at (created_at)
);
```

---

## Database Relationships

```
users (1) ──< (N) monitors ──< (N) monitor_events
   │
   ├──< (N) alert_channels ──< (N) alerts
   │
   └──< (N) subscriptions ──< (N) payments
```

---

## Data Management

### Automatic Data Cleanup

**Job: Удаление старых событий**

```sql
-- Free tier: 7 days
DELETE FROM monitor_events
WHERE timestamp < NOW() - INTERVAL '7 days'
  AND monitor_id IN (SELECT id FROM monitors WHERE user_id IN (SELECT id FROM users WHERE tier = 'free'));

-- Starter: 30 days
DELETE FROM monitor_events
WHERE timestamp < NOW() - INTERVAL '30 days'
  AND monitor_id IN (SELECT id FROM monitors WHERE user_id IN (SELECT id FROM users WHERE tier = 'starter'));

-- Professional: 90 days
DELETE FROM monitor_events
WHERE timestamp < NOW() - INTERVAL '90 days'
  AND monitor_id IN (SELECT id FROM monitors WHERE user_id IN (SELECT id FROM users WHERE tier = 'professional'));

-- Business: 1 year (не удаляем)
```

**Implementation:**
- Scheduled job в Check Workers
- Запускается каждый день в 2:00 AM
- Partition dropping вместо DELETE (при масштабировании)

---

## Миграции

### Migration Tool: Goose

**Установка:**
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

**Создание миграции:**
```bash
goose -dir migrations create create_users_table sql
```

**Применение миграций:**
```bash
goose -dir migrations postgres "user=postgres dbname=milan sslmode=require" up
```

**Структура миграций:**
```
migrations/
├── 00001_create_users_table.sql
├── 00002_create_monitors_table.sql
├── 00003_create_monitor_events_table.sql
├── 00004_create_alert_channels_table.sql
├── 00005_create_alerts_table.sql
├── 00006_create_subscriptions_table.sql
├── 00007_create_payments_table.sql
└── 00008_create_indexes.sql
```

**Zero-downtime:**
- Не требуется для MVP
- Will stop services, apply migrations, restart

---

## Optimization Strategies

### Indexes

**Critical indexes:**
- `monitors(user_id, enabled)` - для получения мониторов пользователя
- `monitors(next_check_time)` - для scheduler
- `monitor_events(monitor_id, timestamp)` - для истории

**Future:**
- Partitioning `monitor_events` по timestamp
- Materialized views для агрегатов (SLA, uptime %)

### Connection Pooling

**PgBouncer configuration:**
```
[databases]
milan = host=localhost port=5432 dbname=milan

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 50
```

---

**Дата:** 2026-03-08
**Версия:** 1.0
