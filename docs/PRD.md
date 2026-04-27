# PRD: [Название] — Мониторинг доступности для РФ/СНГ

> **Версия:** 1.0
> **Дата:** Март 2026
> **Статус:** Черновик

---

## 1. Executive Summary

**[Название]** — SaaS-сервис мониторинга внешней доступности сайтов и API для рынка России и СНГ.

### Ключевая ценность

> *"Единственный локальный SaaS для мониторинга доступности с серверами проверки в РФ, российскими способами оплаты и простым UX в одну минуту"*

### Проблема

- E-commerce и fintech компании теряют **$5,600-9,000 за минуту** даунтайма
- Существующие решения (UptimeRobot, Pingdom) **не имеют серверов в РФ**
- **Не принимают российские платежи**
- "Звонок клиента раньше алерта" — MTTD 20-30 минут

### Решение

- Серверы проверки в РФ (Москва, СПб)
- Оплата через СБП
- Telegram-first подход
- Цены в рублях
- "Setup in 1 minute"

### Рыночная возможность

| Метрика | Значение |
|---------|----------|
| TAM (Total Addressable Market) | ~$3.1M |
| SOM (Serviceable Obtainable Market, Year 1) | ~$120K |
| Whitespace | Нет локального SaaS игрока |

---

## 2. Goals & Success Metrics

### North Star Metric

**"Monitored with confidence"** — количество мониторов, которые не пропустили критический инцидент за 30 дней.

### KPIs по этапам

#### Launch (Q2 2026)

| Метрика | Target |
|---------|--------|
| Active users | 100 |
| Paying users | 10 |
| Active monitors | 25 |
| MTTD | < 10 мин |
| False positive rate | <5% |
| Alert delivery time | <30 сек |

#### Growth (Q3 2026)

| Метрика | Target |
|---------|--------|
| Active users | 500 |
| Paying users | 50 |
| Active monitors | 500 |
| MTTD | < 5 мин |
| Free → Paid conversion | 10% |

#### Scale (Q4 2026)

| Метрика | Target |
|---------|--------|
| Active users | 2,000 |
| Paying users | 200 |
| Active monitors | 5,000 |
| MTTD | < 3 мин |
| MRR growth | >15% QoQ |

### Health Metrics

| Метрика | Q1 | Q2 | Q3 | Q4 |
|---------|----|----|----|----|
| Active monitors | — | 25 | 500 | 2,500 |
| Paying users | — | 10 | 50 | 200 |
| MRR (₽) | — | 15K | 75K | 300K (avg 1,500 ₽/user based on plan mix) |
| Churn | — | <10% | <6% | <5% |
| NPS | — | >40 | >50 | >60 |

---

## 3. User Personas

### 1. Александр — CTO E-commerce (Key Persona)

| Параметр | Значение |
|----------|----------|
| Возраст | 38 лет |
| Компания | 150 сотрудников, 30 в IT |
| Отрасль | E-commerce (маркетплейс) |
| Бюджет | $7K-10K/мес |
| Боли | "Звонок клиента раньше алерта", MTTR 2+ часа, 5+ инструментов |
| Цели | Снизить MTTR до 30 мин, единый стек, 99.9% uptime |

### 2. Дмитрий — SRE Lead в Fintech

| Параметр | Значение |
|----------|----------|
| Возраст | 33 года |
| Компания | 200 сотрудников, 50 в Engineering |
| Бюджет | $20K-25K/мес |
| Боли | PCI DSS compliance, shadow API, false positive алерты |
| Цели | 99.99% availability, MTTD < 5 мин, автоматизация |

### 3. Елена — Technical Lead

| Параметр | Значение |
|----------|----------|
| Бюджет | $3K-5K/мес |
| Боли | "Debugging by print statements", on-call burnout, enterprise overkill |
| Цели | Developer experience, здоровая engineering культура |

### 4. Михаил — Владелец Fintech стартапа

| Параметр | Значение |
|----------|----------|
| Бюджет | $2K-3K/мес |
| Боли | Технический язык как барьер, ответственность без контроля |
| Цели | Peace of mind, отчёты для партнёров |

### 5. Виктория — DevOps Engineer

| Параметр | Значение |
|----------|----------|
| Бюджет | $500-1K/мес |
| Боли | Ручная рутина, missing SRE knowledge, инциденты в нерабочее время |
| Цели | Автоматизация, карьерный рост |

### Pain Points (Cross-segment)

| Боль | Проявление | Impact |
|-----|------------|--------|
| Даунтайм = деньги | $5,600-9,000/минута потери | Критично |
| Data silos | 5+ инструментов в 65% компаний | Сложность диагностики |
| False positives | Alert fatigue, игнорирование алертов | Пропущенные инциденты |
| Сложные решения | Enterprise overkill для mid-market | Низкое adoption |
| "Звонок клиента раньше алерта" | MTTD 20-30 минут | Репутационный риск |

---

## 4. Functional Requirements

### v1.0 MVP (MUST HAVE)

| # | Feature | Priority | Effort | RICE Score |
|---|---------|----------|--------|------------|
| 1 | HTTP/HTTPS monitoring | #1 | 2 мес | 750 |
| 2 | Email + Telegram alerts | #2 | 1 мес | 950 |
| 3 | Dashboard | #3 | 1 мес | 800 |
| 4 | Response time (P50/P95/P99) | #4 | 1 мес | 1080 |
| 5 | Status code checking | #5 | 0.5 мес | 2400 |
| 6 | Check intervals (30сек/1/2/5/10/30 мин) | #6 | 0.5 мес | 1530 |
| 7 | Basic auth/headers | #7 | 0.5 мес | 720 |
| 8 | History (7-90 days tiered) | #8 | 1 мес | 720 |
| 9 | Webhook notifications (POST JSON) | #9 | 0.75 мес | 960 |
| 10 | СБП/ЮKassa оплата | #10 | 1.5 мес | 1000 |
| 11 | Free tier (25 monitors) | #11 | 0.5 мес | 2400 |
| 12 | Russian localization | #12 | 0.5 мес | — |

**Всего:** ~12 человеко-месяцев → 4-6 месяцев для команды 2-3 человека.

#### Детализация MUST HAVE

**1. HTTP/HTTPS Monitoring**

| Параметр | Значение | Обоснование |
|----------|----------|-------------|
| Check intervals | 30 сек, 1, 2, 5, 10, 30 минут | Tiered: Free=5мин, Starter=2мин, Pro=1мин, Business=30сек |
| Timeout | 10-30 секунд | Баланс детекции и false positives |
| Redirects | Follow до 5 | Стандарт для web monitoring |
| Request methods | GET, HEAD | HEAD быстрее, GET для keyword |
| Max response size | 1 MB | Защита от oversize |

**Status Code Handling:**
- 2xx (200-299) → UP
- 3xx (300-399) → UP (follow redirect)
- 4xx (400-499) → DOWN (client error)
- 5xx (500-599) → DOWN (server error)
- Timeout → DOWN
- DNS error → DOWN

**Response Time Tracking:**
- P50 (медиана)
- P95 (95th percentile)
- P99 (99th percentile)
- Min/Max/Avg

**2. Alerting**

| Канал | Параметры | Приоритет |
|--------|-----------|-----------|
| Email | SMTP/API, throttling, customisable | Стандарт |
| Telegram | Bot API, personal/group/channel, Markdown, quick actions | **#1 для РФ** |
| Webhook | POST JSON, retry 3x, Basic/Bearer auth | Интеграции |

**Alert Logic:**
- Trigger: N consecutive failures (configurable, default 2)
- Recovery: 1 consecutive success
- Rate limiting: Max 1 alert/monitor/5 минут
- Grouping: Similar alerts grouped

**3. Dashboard**
- Quick view — текущее состояние всех мониторов
- Real-time status (UP/DOWN/DEGRADED)
- Overall uptime percentage
- Simple, clean design

---

### v1.5 (SHOULD HAVE)

| Feature | Зачем | Сложность |
|---------|-------|-----------|
| SSL мониторинг | Prevention истечения сертификатов | Низкая |
| Public Status Pages | Прозрачность для клиентов, reduction support tickets | Средняя |
| Keyword monitoring | "Server up but page broken" | Низкая |
| Ping (ICMP) monitoring | Network connectivity checks | Низкая |
| Port (TCP) monitoring | Проверка доступности сервисов | Низкая |
| SMS уведомления (paid add-on) | Critical alerts когда Telegram недоступен | Средняя |
| Maintenance windows | Prevention false alerts | Низкая |
| REST API access | Automation, CI/CD интеграции | Средняя |
| Team seats (3 users) | Collaboration в команде | Средняя |
| SLA reporting | Отчёты для менеджмента/партнёров | Средняя |

---

### v2.0 (COULD HAVE)

| Feature | Зачем | Сложность |
|---------|-------|-----------|
| Multi-location monitoring | Проверка из разных регионов РФ/СНГ | Средняя |
| Domain expiration monitoring | Prevention потери доменов | Низкая |
| Cron job monitoring | Heartbeat для scheduled tasks | Низкая |
| AI-powered smart alerting | Reduction false positives | Высокая |
| Incident history/post-mortem templates | Organisational learning | Средняя |
| Slack/Discord integrations | Каналы для международных команд | Низкая |
| Private locations | Мониторинг за firewall | Высокая |

---

### WON'T HAVE (out of scope)

| Feature | Почему |
|---------|--------|
| RUM (Real User Monitoring) | Требует client-side instrumentation, v3.0+ |
| Multi-step transaction monitoring | Дорогая реализация, niche, v2.5+ |
| APM/Infrastructure monitoring | Другой рынок — отдельный продукт |
| Log management | Requires significant infrastructure, v3.0+ |
| Distributed tracing | Complex, enterprise feature, out of scope |
| Malware scanning | Niche, competitors уже делают |

---

## 5. Non-Functional Requirements

### Performance

| Метрика | Target | Обоснование |
|---------|--------|-------------|
| Check execution latency | <5 секунд | Быстрая детекция проблем |
| Alert delivery time | <30 секунд | MTTD <5 минут (fintech requirement) |
| Dashboard load time | <2 секунды | UX requirement |
| API response time | <200ms (P95) | Developer experience |

### Reliability

| Метрика | Target | Обоснование |
|---------|--------|-------------|
| Uptime сервиса | 99.9% | Доверие к мониторингу |
| Data loss | 0% | Критично для истории |
| Check accuracy | >99% | False positives kill trust |

### Security

| Метрика | Target | Обоснование |
|---------|--------|-------------|
| Data encryption | TLS 1.3+ | Standard |
| Auth | JWT + OAuth2 | Modern |
| Logs retention | 7-90 дней tiered by plan (Free=7d, Starter=30d, Pro=90d, Business=1y) | PCI DSS: 3 месяца online |

### Scalability

| Метрика | Target |
|---------|--------|
| Monitors | 10K+ |
| Checks/min | 100K+ |
| Users | 5K+ (Year 1) |

---

## 6. Technology Stack

| Компонент | Технология | Обоснование |
|-----------|------------|-------------|
| Backend | Go / Python | Производительность, экосистема |
| Frontend | React / Next.js | DX, экосистема |
| Database | PostgreSQL / ClickHouse | Надёжность, аналитика |
| Queue | Redis / RabbitMQ | Performance |
| Infrastructure | Yandex Cloud | Локация в РФ (3 ЦОД) |
| Monitoring | Prometheus + VictoriaMetrics | Best practices |

### Architecture (High-Level)

```
┌─────────────────────────────────────────────────────────────┐
│                         Frontend                            │
│  - Dashboard (React/Next.js)                               │
│  - Monitor management UI                                    │
│  - Settings/Profile                                        │
└────────────────────┬────────────────────────────────────────┘
                     │ REST API
┌────────────────────▼────────────────────────────────────────┐
│                      API Gateway                            │
│  - Auth (JWT)                                              │
│  - Rate limiting                                           │
│  - Request routing                                         │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                    Core Services                            │
│  - Monitor management                                      │
│  - Check scheduling                                        │
│  - Alert engine                                            │
│  - Status aggregation                                      │
└─────┬─────────────┬─────────────┬──────────────┬───────────┘
      │             │             │              │
┌─────▼─────┐ ┌────▼────┐ ┌──────▼─────┐ ┌────▼────────┐
│  Checks   │ │ Alerts  │ │  Storage   │ │ Queue      │
│  (workers)│ │(sender) │ │ (Postgres) │ │ (Redis/RMQ) │
└───────────┘ └─────────┘ └────────────┘ └─────────────┘
       │              │
┌──────▼──────────────▼──────────────────────────────────────┐
│              Infrastructure (RUS regions)                    │
│  - Moscow (Yandex Cloud/Selectel)                          │
│  - St. Petersburg                                          │
│  - Expand v1.5: Kazan, Ekaterinburg, Almaty               │
└─────────────────────────────────────────────────────────────┘
```

---

## 7. Pricing

### Тарифная сетка

| Тариф | Цена | Мониторы | Интервал | Ключевые фичи |
|-------|------|----------|----------|---------------|
| **Free** | 0 ₽ | 25 | 5 мин | Email, Telegram, 7d retention |
| **Starter** | 500 ₽/мес | 50 | 2 мин | +Webhook notifications, 3 status pages, 30d retention |
| **Professional** | 2 500 ₽/мес | 150 | 1 мин | +5 team seats (from v1.5), SLA, REST API, 90d retention |
| **Business** | 7 500 ₽/мес | 500+ | 30 сек | +White-label, Private location, 1y retention |

### vs Конкуренты

| Мы | UptimeRobot | Pingdom | Site24x7 |
|-----|-------------|---------|----------|
| Free: 25 monitors | 50 monitors | 0 (только trial) | 500 metrics |
| Starter: 500 ₽ | $7 (~700 ₽) | $10 (~1000 ₽) | ~$50 |
| **Серверы в РФ** | Нет | Нет | Нет |
| **СБП оплата** | Нет | Нет | Нет |

### Преимущества

- На 20-30% дешевле конкурентов
- Серверы в РФ (точность для RuNet)
- СБП оплата (уникально)
- Telegram-first
- Цены в рублях (нет курсовых рисков)

### Payment Methods

- **СБП** — комиссия 0.5-0.7%
- **Банковская карта** (ЮKassa) — Visa/Mastercard, МИР
- **Платёжные поручения** — для B2B
- **Годовая оплата** — скидка 20%

---

## 8. Roadmap

### Q2 2026: v1.0 MVP

**Цель:** Запуск базового сервиса для ранних adopters

**Features:**
- HTTP/HTTPS monitoring
- Check intervals: 30сек/1/2/5/10/30 минут (tiered by plan)
- Response time tracking (P50/P95/P99)
- Status code checking
- Email + Telegram alerts
- Webhook notifications (POST JSON for integrations)
- Dashboard с состоянием мониторов
- История (7-90 дней tiered by plan: Free=7d, Starter=30d, Pro=90d, Business=1y)
- Basic auth/headers support
- Free tier (25 monitors, 5 min intervals)
- Оплата через СБП/ЮKassa
- Серверы проверки в РФ (Москва, СПб)

### Q3 2026: v1.5

**Цель:** Feature parity с базовыми конкурентами

**New Features:**
- SSL monitoring
- Public status pages (custom domain + branding)
- Keyword monitoring
- Ping/Port monitoring
- Maintenance windows
- REST API access (automation, CI/CD integrations)
- SMS alerts (paid)
- SLA reporting
- Team seats (3 users)
- Professional tier (150 monitors, 1 min) — 2 500 ₽/мес

### Q4 2026: v2.0

**Цель:** Дифференциация на рынке

**New Features:**
- Multi-location (Москва, СПб, Екатеринбург, Алматы)
- Domain expiration monitoring
- Cron job monitoring
- Incident history
- Post-mortem templates
- Slack + Discord integrations
- Advanced alerting (escalation policies)
- Private status pages
- Custom alert rules
- Business tier (500 monitors, 30 sec) — 7 500 ₽/мес

### v2.5+ (2027)

**New Features:**
- AI-powered smart alerting (anomaly detection)
- PagerDuty + OpsGenie + MS Teams integrations
- Private location monitoring (on-premise agent)
- White-label status pages
- SSO (SAML)
- Advanced analytics и trend prediction
- Mobile app (MVP)

---

## 9. Compliance

### Обязательные требования

| Требование | Описание | Штрафы |
|------------|----------|--------|
| **152-ФЗ** | Локализация персональных данных в РФ | До 3 млн ₽ (обычные), 5 млн ₽ (не локализовано) |
| **Реестр ПО** | Рекомендуется для B2B (Минцифры) | — |

### Чек-лист compliance

**Запуск продукта:**
- [ ] Политика обработки ПД (152-ФЗ)
- [ ] Согласие на обработку ПД (отдельный документ)
- [ ] Пользовательское соглашение
- [ ] Выбрать провайдера с ЦОД в РФ
- [ ] Privacy Policy на русском

### Data centers

| Провайдер | Доля рынка | Особенности |
|-----------|------------|-------------|
| Yandex Cloud | 47.4% | 3 ЦОД (Москва, СПб, Новосибирск) |
| SberCloud | — | 37 облачных сервисов |
| Selectel | 17.5% | ЦОД в СПб и Москве |
| Cloud.ru (Sber) | 14.3% | Экосистема Сбера |

---

## 10. Unit Economics

### Ключевые метрики

| Метрика | Значение | Target | Статус |
|---------|----------|--------|--------|
| **LTV/CAC** | **40.4** | >3 | Отлично |
| **Payback period** | **1.2 мес** | <12 мес | Отлично |
| **Churn (платящие)** | **4.5%/мес** | <5%/мес | В норме |
| **ARPU** | **2,167 ₽** | — | — |

### Прогнозы

| Пользователи | Paying | MRR | ARR |
|--------------|--------|-----|-----|
| 100 | 30 | 65K ₽ | 780K ₽ |
| 1,000 | 300 | 650K ₽ | 7.8M ₽ |
| 10,000 | 3,000 | 6.5M ₽ | 78M ₽ |

### Break-even

- **Monthly expenses** (команда 3 чел): ~600K ₽
- **Break-even MRR**: ~650K ₽
- **Break-even users**: ~300 paying users

### Sensitivity Analysis

**Пессимистичный сценарий (всё сразу):**
- Churn +2%
- CAC +50%
- Conversion Free → Paid = 10%

**Результат:** LTV/CAC ~12 (всё ещё healthy!)

---

## 11. Go-to-Market

### Каналы привлечения

#### Launch (0-100 users)

| Канал | CAC (руб) | Leads | Бюджет |
|-------|-----------|-------|--------|
| Telegram (sysadmin_ru) | 50-100 | 20-40 | 0-5K ₽ |
| Habr статья | 2000-5000* | 30-60 | Время |

#### Growth (100-1000 users)

| Канал | CAC (руб) | Leads | Бюджет |
|-------|-----------|-------|--------|
| 8-10 Telegram каналов | 70-200 | 150-300 | 20-50K ₽ |
| Habr + vc.ru контент | 15-40K ₽/статья | 100-250 | 15-40K ₽ |
| Онлайн-митапы | 500-1500 | 50-100 | 5-15K ₽ |

### Партнёрская сеть

| Партнёр | Аудитория | Интеграция |
|---------|-----------|------------|
| Bitrix24 | 15M+ пользователей | Marketplace приложение |
| Yandex Cloud | 47.4% рынка РФ | Публикация в маркетплейсе |
| Tilda | Стандарт в РФ | Встроенный виджет |
| WordPress | 726K+ клиентов в СНГ | Плагин |

---

## 12. Next Steps

### Ближайшие действия (1-2 недели)

1. [ ] Валидация цен через landing page + pre-registration
2. [ ] Юридическая структура (ИП/ООО)
3. [ ] Открытие расчётного счёта (Т-Банк, Сбер)
4. [ ] Регистрация в ЮKassa
5. [ ] Найм 2-3 разработчиков

### Phase 1: MVP (3-4 месяца)

1. [ ] Backend architecture (Go/Python)
2. [ ] Frontend (React/Vue)
3. [ ] Infrastructure (Yandex Cloud)
4. [ ] Monitoring infrastructure (Prometheus, VictoriaMetrics)
5. [ ] Payment integration (ЮKassa)
6. [ ] Telegram bot

### Phase 2: Launch (1-2 месяца)

1. [ ] Beta testing (20-30 users)
2. [ ] Content creation (5-7 статей)
3. [ ] Telegram channel launch
4. [ ] Public launch

---

## Sources

- `docs/product-analysis/product_strategy.md` — основная стратегия
- `docs/product-analysis/agent_01_global_competitors.md` — глобальные конкуренты
- `docs/product-analysis/agent_02_local_competitors.md` — локальные конкуренты
- `docs/product-analysis/agent_03_swot.md` — SWOT анализ
- `docs/product-analysis/agent_04_ecommerce_pains.md` — боли e-commerce
- `docs/product-analysis/agent_05_fintech_requirements.md` — требования финтеха
- `docs/product-analysis/agent_07_core_features.md` — детализация фич
- `docs/product-analysis/agent_08_differentiators.md` — дифференциаторы
- `docs/product-analysis/agent_09_rice_scoring.md` — RICE скоринг
- `docs/product-analysis/agent_13_price_positioning.md` — ценовое позиционирование
- `docs/product-analysis/agent_14_pricing_model.md` — модель ценообразования
- `docs/product-analysis/agent_15_unit_economics.md` — юнит-экономика

---

*Документ создан на основе синтеза данных от 17 агентов (Март 2026)*
