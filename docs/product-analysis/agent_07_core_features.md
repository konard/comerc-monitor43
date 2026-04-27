# Агент 7: Core features v1.0

## MoSCoW анализ

### MUST HAVE (для v1.0)

| Feature | Зачем пользователю | Источник | Оценка сложности |
|---------|-------------------|----------|------------------|
| HTTP/HTTPS monitoring | Базовый функционал — проверка доступности сайтов/API | Agent 1, 2, 3, 4, 5 | Низкая |
| Email уведомления | Стандартный канал — первый сигнал о проблеме | Agent 1, 2, 4 | Низкая |
| Telegram уведомления | Основной канал в РФ/СНГ — популярнее Email для алертов | Agent 2, 3, 6 | Низкая |
| Dashboard с состоянием мониторов | Quick view — текущее состояние всех ресурсов | Agent 4, 6 | Низкая |
| Response time tracking | Понимание degrade vs down — критично для UX | Agent 1, 4, 5 | Низкая |
| Status code checking (2xx/3xx/4xx/5xx) | Детализация типа проблемы | Agent 1, 5 | Низкая |
| Настройка check intervals (1/5/10/30 мин) | Flexibility для разных типов ресурсов | Agent 1, 4 | Низкая |
| Basic auth/headers support | Мониторинг приватных API/endpoints | Agent 5 | Низкая |
| История проверок (7-30 дней) | Post-mortem анализ, тренды | Agent 4, 5, 6 | Низкая |
| Webhook уведомления | Интеграции с существующими системами | Agent 1, 2 | Низкая |
| Оплата в рублях (СБП/ЮKassa) | Удобство для РФ клиентов — дифференциатор | Agent 2, 3, 13 | Средняя |
| Free tier (25-50 monitors) | Привлечение пользователей, competitive с UptimeRobot | Agent 1, 2, 13 | Низкая |

### SHOULD HAVE (для v1.5)

| Feature | Зачем пользователю | Источник | Сложность |
|---------|-------------------|----------|-----------|
| SSL мониторинг | Стандарт индустрии — prevention истечения сертификатов | Agent 1, 3 | Низкая |
| Public Status Pages | Прозрачность для клиентов — reduction support tickets | Agent 1, 2, 4 | Средняя |
| Keyword monitoring | Проверка контента — "server up but page broken" | Agent 1, 4 | Низкая |
| Ping (ICMP) monitoring | Network connectivity checks | Agent 1, 2 | Низкая |
| Port (TCP) monitoring | Проверка доступности сервисов | Agent 1 | Низкая |
| SMS уведомления (оплачиваемые) | Critical alerts — когда Telegram недоступен | Agent 1, 5 | Средняя |
| Maintenance windows | Prevention false alerts во время плановых работ | Agent 1, 6 | Низкая |
| API access | Automation, интеграции с CI/CD | Agent 1, 3, 6 | Средняя |
| SLA reporting | Отчёты для менеджмента/партнёров | Agent 5, 6 | Средняя |

### COULD HAVE (для v2.0)

| Feature | Зачем пользователю | Источник | Сложность |
|---------|-------------------|----------|-----------|
| Multi-location monitoring | Проверка из разных регионов РФ/СНГ | Agent 1, 2, 3 | Средняя |
| Domain expiration monitoring | Prevention потери доменов | Agent 1 | Низкая |
| Cron job monitoring | Heartbeat для scheduled tasks | Agent 1 | Низкая |
| AI-powered smart alerting | Reduction false positives, alert fatigue | Agent 3, 4, 5 | Высокая |
| Incident history/post-mortem templates | Organisational learning | Agent 6 | Средняя |
| Team seats (многопользовательский доступ) | Collaboration в команде | Agent 1, 6 | Средняя |
| Private locations | Мониторинг за firewall | Agent 1, 5 | Высокая |
| Integrations (Slack, Discord) | Каналы для международных команд | Agent 1 | Низкая |

### WON'T HAVE (out of scope)

| Feature | Почему не сейчас | Когда возможно |
|---------|------------------|----------------|
| RUM (Real User Monitoring) | Требует client-side instrumentation, сложная реализация | v3.0+ |
| Multi-step transaction monitoring | Дорогая реализация, niche feature | v2.5+ |
| APM/Infrastructure monitoring | Другой рынок — нужно отдельное решение | Never (separate product) |
| Log management | Requires significant infrastructure | v3.0+ или separate product |
| Distributed tracing | Complex, enterprise feature | Never (out of scope) |
| Malware scanning | Niche, StatusCake уже делает | Never |

---

## Детализация MUST HAVE для v1.0

### 1. HTTP/HTTPS Monitoring

**Что:**
Проверка доступности HTTP/HTTPS ресурсов (сайты, API endpoints) с серверов расположенных в РФ.

**Почему:**
- Базовый функционал любого uptime monitoring сервиса
- Критичная метрика для e-commerce (каждая минута даунтайма = $5,000+ потерь)
- Fintech требует 99.99% availability для критичных систем
- **Дифференциатор:** проверка из RuNet даёт точные результаты для российских сайтов

**Детали:**

| Параметр | Значение | Обоснование |
|----------|----------|-------------|
| Check intervals | 1, 5, 10, 30 минут | Standard: UptimeRobot (5 min free), Pingdom (1 min paid) |
| Timeout | 10-30 секунд | Баланс между детекцией и false positives |
| Redirects | Follow до 5 | Стандарт для web monitoring |
| Request methods | GET, HEAD | HEAD быстрее, GET для keyword checking |
| Max response size | 1 MB | Достаточно для большинства response, защита от oversize |

**Status Code Handling:**
- 2xx (200-299) → UP
- 3xx (300-399) → UP (follow redirect)
- 4xx (400-499) → DOWN (client error = problem)
- 5xx (500-599) → DOWN (server error = downtime)
- Timeout → DOWN
- DNS error → DOWN

**Response Time Tracking:**
- P50 (медиана)
- P95 (95th percentile)
- P99 (99th percentile)
- Min/Max/Avg

### 2. Alerting

**Что:**
Мультиканальная система уведомлений о проблемах с мониторами.

**Почему:**
- MTTR (Mean Time To Recovery) должен быть <30 минут для e-commerce
- Fintech требует MTTD (Mean Time To Detect) <5 минут
- "Звонок от клиента раньше алерта" — key pain point (Agent 6)

**Детали:**

#### Email (бесплатно, unlimited)
| Параметр | Значение |
|----------|----------|
| Тип | SMTP / API integration |
| Frequency | Throttling для prevention spam |
| Шаблоны | Customisable subject/body |

#### Telegram (приоритет для РФ)
| Параметр | Значение |
|----------|----------|
| Тип | Bot API |
| Channels | Personal chat, Group chat, Channel |
| Rich format | Markdown support (tables, bold, links) |
| Actions | Quick buttons (Acknowledge, Resolve) |

**Почему Telegram приоритет:**
- Популярен в РФ/СНГ как основной канал коммуникации
- Работает когда Email недоступен
- Push notifications на мобильных
- Agent 2: "Telegram-бот для уведомлений — популярнейший мессенджер в РФ/СНГ"

#### SMS (paid addon)
| Параметр | Значение |
|----------|----------|
| Цена | 5-10 ₽/SMS |
| Использование | Critical alerts только |
| Провайдеры | Интеграция с РФ SMS шлюзами |

#### Webhook (для интеграций)
| Параметр | Значение |
|----------|----------|
| Метод | POST |
| Format | JSON |
| Retry | 3 попытки с exponential backoff |
| Auth | Basic, Bearer token, headers |

**Alert Logic:**
- Trigger: N consecutive failures (configurable, default 2)
- Recovery: 1 consecutive success
- Rate limiting: Max 1 alert/monitor/5 минут
- Grouping: Similar alerts grouped (prevention spam)

### 3. Status Page

**Что:**
Публичная страница с текущим статусом всех мониторов.

**Почему:**
- Прозрачность для клиентов — reduction support tickets
- "Показать партнёру красивый отчёт" — Mikhail Founder needs this (Agent 6)
- Standard у всех конкурентов (Agent 1)
- Prevention "злой звонок от клиента"

**Детали:**

| Feature | v1.0 | v1.5 |
|---------|------|------|
| Public URL | ✅ | ✅ |
| Custom domain | ❌ | ✅ |
| Custom branding (logo, colors) | ❌ | ✅ |
| Incident history | ❌ | ✅ |
| Subscribe to updates | ❌ | ✅ |
| Private (password protected) | ❌ | v2.0 |

**v1.0 MVP Status Page:**
- Auto-generated URL: `status.milan.monitor/username`
- Real-time status (UP/DOWN/DEGRADED)
- List of monitors с status
- Overall uptime percentage
- Simple, clean design

### 4. Интеграции

**Что:**
Возможность отправлять алерты в внешние системы.

**Почему:**
- Developer experience — ключевой критерий для Elena Tech Lead (Agent 6)
- Automation — "everything as code" для Victoria DevOps (Agent 6)
- Enterprise requires SIEM/SOAR integration (Agent 5)

**Детали:**

#### v1.0
| Интеграция | Статус | Обоснование |
|------------|--------|-------------|
| Webhook | ✅ | Universal, работает с чем угодно |
| Telegram | ✅ | Priority #1 для РФ |

#### v1.5
| Интеграция | Статус | Обоснование |
|------------|--------|-------------|
| Slack | ✅ | International teams |
| Discord | ✅ | Dev communities |
| Email | ✅ | Standard |

#### v2.0
| Интеграция | Статус | Обоснование |
|------------|--------|-------------|
| PagerDuty | ✅ | Enterprise on-call |
| OpsGenie | ✅ | Enterprise on-call |
| MS Teams | ✅ | Enterprise comms |
| SIEM (via webhook) | ✅ | Fintech compliance |

---

## Roadmap на 12 месяцев

### Q1: v1.0 MVP
**Цель:** Запуск базового сервиса для ранних adopters

**Features:**
- HTTP/HTTPS monitoring
- Check intervals: 1/5/10/30 минут
- Response time tracking (P50/P95/P99)
- Status code checking
- Email + Telegram + Webhook alerts
- Dashboard с состоянием мониторов
- История 30 дней
- Basic auth/headers support
- Free tier (25 monitors, 5 min intervals)
- Starter tier (50 monitors, 2 min intervals) — 500 ₽/мес
- Оплата через СБП/ЮKassa
- Серверы проверки в РФ (Москва, СПб)

**Success Metrics:**
- 100 active users
- 10 paying customers
- <5% false positive rate
- <30 sec alert delivery time

### Q2: v1.5
**Цель:** Feature parity с базовыми конкурентами

**New Features:**
- SSL monitoring (expiration, validity)
- Public status pages (custom domain + branding)
- Keyword monitoring
- Ping (ICMP) monitoring
- Port (TCP) monitoring
- Maintenance windows
- API access (REST)
- SMS alerts (paid)
- SLA reporting
- Team seats (3 users)

**New Pricing:**
- Professional tier (150 monitors, 1 min) — 2000 ₽/мес

**Success Metrics:**
- 500 active users
- 50 paying customers
- 10% conversion free→paid
- API: 100K requests/month

### Q3: v2.0
**Цель:** Дифференциация на рынке

**New Features:**
- Multi-location monitoring (3-5 городов РФ)
- Domain expiration monitoring
- Cron job monitoring
- Incident history
- Post-mortem templates
- Slack + Discord integrations
- Advanced alerting (escalation policies)
- Private status pages
- Custom alert rules

**New Pricing:**
- Business tier (500 monitors, 30 sec) — 6000 ₽/мес

**Success Metrics:**
- 2000 active users
- 200 paying customers
- 15% MRR growth QoQ
- Enterprise pilot (1-2 contracts)

### Q4: v2.5
**Цель:** Enterprise features и AI

**New Features:**
- AI-powered smart alerting (anomaly detection, false positive reduction)
- PagerDuty + OpsGenie + MS Teams integrations
- Private location monitoring (on-premise agent)
- White-label status pages
- SSO (SAML)
- Advanced analytics и trend prediction
- Mobile app (MVP)

**Success Metrics:**
- 5000 active users
- 500 paying customers
- Enterprise revenue: 20% of total
- AI alert accuracy: >95%

---

## Non-functional requirements для v1.0

### Performance
| Метрика | Target | Обоснование |
|---------|--------|-------------|
| Check execution latency | <5 секунд | Быстрая детекция проблем |
| Alert delivery time | <30 секунд | MTTD <5 минут (Agent 5) |
| Dashboard load time | <2 секунды | UX requirement |
| API response time | <200ms (P95) | Developer experience |

### Reliability
| Метрика | Target | Обоснование |
|---------|--------|-------------|
| Uptime самого сервиса | 99.9% | Доверие к мониторингу |
| Data loss | 0% | Критично для истории |
| Check accuracy | >99% | False positives kill trust |

### Security
| Метрика | Target | Обоснование |
|---------|--------|-------------|
| Data encryption | TLS 1.3+ | Standard |
| Auth | JWT + OAuth2 | Modern |
| Logs retention | 30 дней (v1.0) | PCI DSS: 3 месяца online (Agent 5) |

### Scalability
| Метрика | Target | Обоснование |
|---------|--------|-------------|
| Monitors | 10K+ | Growth path |
| Checks/min | 100K+ | Peak load |
| Users | 5K+ | Year 1 target |

---

## Technical Architecture (high-level)

### Components для v1.0
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
│  - Moscow (Selectel/Yandex Cloud)                          │
│  - St. Petersburg                                          │
│  - (Expand in v1.5: Kazan, Ekaterinburg, Novosibirsk)     │
└─────────────────────────────────────────────────────────────┘
```

---

## Приоритеты разработки

### P0 (Критично для MVP)
1. HTTP check engine
2. PostgreSQL схема
3. Auth (JWT)
4. API endpoints (CRUD monitors)
5. Alert engine (Email + Telegram)
6. Basic dashboard

### P1 (Важно для launch)
1. Webhook integration
2. Response time tracking
3. History API
4. Payment integration (СБП/ЮKassa)
5. Status page MVP

### P2 (Можно отложить)
1. Multi-user teams
2. Advanced analytics
3. Mobile app
4. AI features

---

## Sources
- Agent 1: Global competitors (UptimeRobot, Pingdom, StatusCake, Uptime.com, Site24x7)
- Agent 2: Local competitors (Uptime Kuma, Zabbix, Upptime)
- Agent 3: SWOT analysis
- Agent 4: E-commerce pain points
- Agent 5: Fintech requirements (PCI DSS, SLA)
- Agent 6: User personas (Alexander, Dmitry, Elena, Mikhail, Victoria)
- Agent 13: Price positioning
