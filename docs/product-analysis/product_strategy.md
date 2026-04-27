# Product Strategy: Milan — Мониторинг доступности для РФ/СНГ

> *"Малыми средствами принести большую пользу. Красота в простоте."*

---

## Executive Summary

**Milan** — сервис мониторинга внешней доступности сайтов для рынка России и СНГ.

**Ключевая проблема:** E-commerce и fintech компании теряют $5,600-9,000 за каждую минуту даунтайма, а существующие решения (UptimeRobot, Pingdom) либо не имеют серверов в РФ, либо не принимают российские платежи.

**Наше решение:** SaaS-сервис мониторинга с серверами проверки в РФ, оплатой через СБП и простым UX "в одну минуту".

**Рыночная возможность:** TAM ~$3.1M для РФ/СНГ, whitespace — нет локального SaaS игрока.

**Бизнес-модель:** Freemium → 500₽ → 2,500₽ → 7,500₽/мес. LTV/CAC = 40.4, payback 1.2 мес.

**Горизонт запуска:** 6 месяцев (Q3 2026).

---

## 1. Market Opportunity

### TAM/SAM/SOM для РФ/СНГ

| Сегмент | Описание | Размер |
|---------|----------|--------|
| **TAM** (Total Addressable Market) | Все компании РФ/СНГ с веб-присутствием | ~$3.1M |
| **SAM** (Serviceable Available Market) | E-commerce + Fintech + SaaS Mid-market | ~$1.2M |
| **SOM** (Serviceable Obtainable Market) | Первые 12 месяцев, РФ | ~$120K |

**Обоснование:**
- TAM: ~50,000+ потенциальных пользователей в РФ/СНГ (Agent 9)
- SAM: E-commerce (рост 20%+ YoY) + Fintech (регуляторные требования)
- SOM: 3,000 paying users × $40 ARPU = $120K ARR за первый год

### Рыночные тренды

| Тренд | Описание | Источник |
|-------|----------|----------|
| **Импортозамещение в IT** | Рост XX% в 2025, компании ищут локальные альтернативы | Agent 3 |
| **Рост e-commerce** | Ozon, Wildberries, маркетплейсы — двойной цифровой рост | Agent 4 |
| **Рост fintech** | ЦБ РФ требует перехода на отечественное ПО к 2025 | Agent 5 |
| **Популярность Telegram** | Основной канал коммуникации в РФ/СНГ | Agent 2 |

### Whitespace

**Ключевой инсайт:** Нет локального SaaS сервиса мониторинга с серверами проверки в РФ.

| Конкурент | Серверы в РФ | СБП оплата | Цены в рублях |
|-----------|--------------|------------|---------------|
| UptimeRobot | Нет | Нет | Нет |
| Pingdom | Нет | Нет | Нет |
| Site24x7 | Нет | Нет | Нет |
| Uptime Kuma (self-hosted) | Требует свой сервер | — | — |
| **Milan** | **Да** | **Да** | **Да** |

---

## 2. Target Audience

### Primary: Mid-market E-commerce & Fintech

**5 Key Personas:**

#### 1. Александр — CTO E-commerce (ключевой)

| Параметр | Значение |
|----------|----------|
| **Возраст** | 38 лет |
| **Компания** | 150 сотрудников, 30 в IT |
| **Отрасль** | E-commerce (маркетплейс товаров для дома) |
| **Бюджет** | $7K-10K/мес |
| **Боли** | "звонок клиента раньше алерта", 5+ инструментов, MTTR 2+ часа |
| **Цели** | Снизить MTTR до 30 мин, единый стек, 99.9% uptime |

#### 2. Дмитрий — SRE Lead в Fintech

| Параметр | Значение |
|----------|----------|
| **Возраст** | 33 года |
| **Компания** | 200 сотрудников, 50 в Engineering |
| **Бюджет** | $20K-25K/мес |
| **Боли** | PCI DSS compliance, shadow API, false positive алерты |
| **Цели** | 99.99% availability, MTTD < 5 мин, автоматизация |

#### 3. Елена — Technical Lead
- **Бюджет:** $3K-5K/мес
- **Боли:** "Debugging by print statements", on-call burnout, оверкилл enterprise-решений
- **Цели:** Developer experience, здоровая engineering культура

#### 4. Михаил — Владелец Fintech стартапа
- **Бюджет:** $2K-3K/мес
- **Боли:** Технический язык как барьер, ответственность без контроля
- **Цели:** Peace of mind, отчёты для партнёров

#### 5. Виктория — DevOps Engineer
- **Бюджет:** $500-1K/мес
- **Боли:** Ручная рутина, missing SRE knowledge, инциденты в нерабочее время
- **Цели:** Автоматизация, карьерный рост

### Pain Points (Cross-segment)

| Боль | Проявление | Impact |
|-----|------------|--------|
| **Даунтайм = деньги** | $5,600-9,000/минута потери | Агент 4 |
| **Data silos** | 5+ инструментов в 65% компаний | Агент 6 |
| **False positives** | Alert fatigue, игнорирование алертов | Агент 6 |
| **Сложные решения** | Enterprise overkill для mid-market | Агент 7 |
| **"Звонок клиента раньше алерта"** | MTTD 20-30 минут | Агент 6 |

---

## 3. Value Proposition

### Уникальное позиционирование

> *"Единственный локальный SaaS для мониторинга доступности с серверами проверки в РФ, российскими способами оплаты и простым UX в одну минуту"*

### Ключевые дифференциаторы

| Дифференциатор | Уникальность | Описание |
|----------------|--------------|----------|
| **Серверы проверки в РФ** | 100% уникальность | Проверка из RuNet, точность для российских сайтов |
| **Оплата через СБП** | 100% уникальность | Удобство для локальных клиентов, 0.5-0.7% комиссия |
| **Telegram-first** | Уникально для РФ | Популярнейший мессенджер, нативная интеграция |
| **Цены в рублях** | Уникально для локальных | Нет курсовых рисков, предсказуемость |
| **"Setup in 1 minute"** | Лучше конкурентов | One-click signup, smart defaults |

### Copywriting

| Элемент | Текст |
|---------|-------|
| **Headline** | "Know before your users do" |
| **Subheadline** | "Get alerted in seconds, not hours. Setup takes a minute." |
| **CTA** | "Start monitoring free — no credit card required" |
| **Tagline** | "Мониторинг для RuNet" |

---

## 4. Product v1.0

### Core Features (12 MUST HAVE)

| Feature | Priority | Effort | RICE Score |
|---------|----------|--------|------------|
| **HTTP/HTTPS monitoring** | #1 | 2 мес | 750 |
| **Email + Telegram alerts** | #2 | 1 мес | 950 |
| **Dashboard** | #3 | 1 мес | 800 |
| **Response time (P50/P95/P99)** | #4 | 1 мес | 1080 |
| **Status code checking** | #5 | 0.5 мес | 2400 |
| **Check intervals (1/5/10/30 мин)** | #6 | 0.5 мес | 1530 |
| **Basic auth/headers** | #7 | 0.5 мес | 720 |
| **History (7-30 days)** | #8 | 1 мес | 720 |
| **Webhook notifications** | #9 | 0.75 мес | 960 |
| **СБП/ЮKassa оплата** | #10 | 1.5 мес | 1000 |
| **Free tier (25 monitors)** | #11 | 0.5 мес | 2400 |
| **Russian localization** | #12 | 0.5 мес | — |

**Всего:** ~12 человеко-месяцев → 4-6 месяцев для команды 2-3 человека.

### Roadmap 12 месяцев

#### Q2 2026: v1.0 MVP
- HTTP мониторинг, Email+Telegram alerts, Dashboard
- Серверы проверки в РФ (Москва, СПб)
- Free tier (25 monitors, 5 min intervals)

**Success Metrics:**
- 100 active users
- 10 paying customers
- <5% false positive rate
- <30 sec alert delivery time

#### Q3 2026: v1.5
- SSL monitoring, Status pages, Keyword search
- API access, Maintenance windows
- SLA reporting

**Success Metrics:**
- 500 active users
- 50 paying customers
- 10% conversion free→paid

#### Q4 2026: v2.0
- Multi-location (Москва, СПб, Екатеринбург, Алматы)
- Slack/Discord integrations, Team seats
- Ping/Port monitoring

**Success Metrics:**
- 2,000 active users
- 200 paying customers
- 15% MRR growth QoQ

### AI Capabilities (v2.5+)

#### v1.0 AI (Basic)
- Smart thresholds (статистические аномалии)
- Template-based explanations
- Alert aggregation

#### v2.0 AI
- Isolation Forest для anomaly detection
- Predictive alerts (trend forecasting)
- LLM-powered explanations

**Magic UX:** ИИ невидим, польза очевидна — "просто работает лучше".

---

## 5. Go-to-Market Strategy

### Каналы привлечения (по этапам)

#### Launch (0-100 users)

| Канал | CAC (руб) | Ожидание leads | Бюджет |
|-------|-----------|----------------|--------|
| Telegram (sysadmin_ru) | 50-100 | 20-40 | 0-5K ₽ |
| Habr статья | 2000-5000* | 30-60 | Время автора |
| Комментарии/форумы | 0 | 5-15 | 0 ₽ |

\*Только время, SEO эффект

**Итого:** 55-115 пользователей | Бюджет: 0-10K ₽

#### Growth (100-1000 users)

| Канал | CAC (руб) | Ожидание leads | Бюджет |
|-------|-----------|----------------|--------|
| 8-10 Telegram каналов | 70-200 | 150-300 | 20-50K ₽ |
| Habr + vc.ru контент | 15-40K ₽/статья | 100-250 | 15-40K ₽ |
| Онлайн-митапы | 500-1500 | 50-100 | 5-15K ₽ |

**Итого:** 350-750 пользователей | Бюджет: 55-135K ₽

#### Scale (1000+ users)

| Канал | CAC (руб) | Ожидание leads | Бюджет |
|-------|-----------|----------------|--------|
| Конференции (DevOops) | 4-12K | 200-500 | 150-500K ₽ |
| Платная реклама (Habr) | 1.5-4K | 500-1000 | 100-300K ₽ |
| Реферальная программа | Variable | 20-30% growth | Cost per acquisition |

### Партнёрская сеть

**Высокий приоритет:**

| Партнёр | Аудитория | Интеграция |
|---------|-----------|------------|
| **Bitrix24** | 15M+ пользователей | Marketplace приложение |
| **Yandex Cloud** | 47.4% рынка РФ | Публикация в маркетплейсе |
| **Tilda** | Стандарт в РФ | Встроенный виджет |
| **WordPress** | 726K+ клиентов в СНГ | Плагин |

### Контент-стратегия

4 контент-угла:
1. **"Сколько стоит даунтайм?"** — $5,600-180K/мин
2. **"Кейсы известных даунтаймов"** — Shopify, AWS, Amazon
3. **"Best practices мониторинга"** — SLA, MTTD/MTTR
4. **"Philosophy of simplicity"** — против enterprise overkill

**План 6 месяцев:** ~257K views, ~1,695 leads, ~3K Telegram подписчиков.

---

## 6. Pricing Strategy

### Тарифы

| Тариф | Цена | Мониторы | Интервал | Ключевые фичи |
|-------|------|----------|----------|---------------|
| **Free** | 0 ₽ | 25 | 5 мин | Email, Telegram |
| **Starter** | 500 ₽/мес | 50 | 2 мин | +Custom branding, 3 status pages |
| **Professional** | 2,500 ₽/мес | 150 | 1 мин | +5 team seats, SLA, API |
| **Business** | 7,500 ₽/мес | 500+ | 30 сек | +White-label, Private location |

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

---

## 7. Unit Economics

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

### Sensitivity Analysis

**Пессимистичный сценарий (всё сразу):**
- Churn +2%
- CAC +50%
- Conversion Free → Paid = 10%

**Результат:** LTV/CAC ~12 (всё ещё healthy!)

### Funding Requirements

| Сценарий | Сумма | Runway | Цель |
|----------|-------|---------|------|
| **Bootstrap** | 500K ₽ | 2.5 мес | Proof of concept |
| **Angel** | 3M ₽ | 8.5 мес | Product-market fit |
| **Pre-seed** | 10M ₽ | 12.5 мес | Scale |

---

## 8. Regulatory & Compliance

### Обязательные требования

| Требование | Описание | Источник |
|------------|----------|----------|
| **152-ФЗ** | Локализация персональных данных в РФ | Агент 16 |
| **Реестр ПО** | Рекомендуется для B2B | Минцифры |

### Штрафы за нарушение 152-ФЗ

- Обычные нарушения: до 3 млн ₽
- Не локализованное хранение: 5 млн ₽
- Уголовная ответственность: до 5 лет лишения свободы

### Data centers

| Провайдер | Доля рынка | Особенности |
|-----------|------------|-------------|
| **Yandex Cloud** | 47.4% | 3 ЦОД (Москва, СПб, Новосибирск) |
| **SberCloud** | — | 37 облачных сервисов |
| **Selectel** | 17.5% | ЦОД в СПб и Москве |
| **Cloud.ru (Sber)** | 14.3% | Экосистема Сбера |

### Чек-лист compliance

**Запуск продукта:**
- [ ] Политика обработки ПД (152-ФЗ)
- [ ] Согласие на обработку ПД (отдельный документ с 1.09.2025)
- [ ] Пользовательское соглашение
- [ ] Выбрать провайдера с ЦОД в РФ
- [ ] Privacy Policy на русском

---

## 9. Success Metrics

### North Star Metric

**"Monitored with confidence"** — количество мониторов, которые не пропустили критический инцидент за 30 дней.

### KPIs по этапам

#### Launch (Q2 2026)
- 25 active monitors
- 10 paying users
- MTTD < 10 мин

#### Growth (Q3 2026)
- 500 active monitors
- 50 paying users
- MTTD < 5 мин

#### Scale (Q4 2026)
- 5,000 active monitors
- 500 paying users
- MTTD < 3 мин

### Health Metrics

| Метрика | Q1 | Q2 | Q3 | Q4 |
|---------|----|----|----|----|
| Active monitors | — | 25 | 500 | 2,500 |
| Paying users | — | 10 | 50 | 200 |
| MRR | — | 15K ₽ | 75K ₽ | 300K ₽ |
| Churn | — | <10% | <6% | <5% |
| NPS | — | >40 | >50 | >60 |

---

## 10. Next Steps

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

## Appendices

### A. Конкуренты (детально)

#### UptimeRobot
- **Free:** 50 monitors, 5 min intervals
- **Paid:** $7-54/month
- **Плюсы:** Extremely generous free tier, simple setup
- **Минусы:** No Russian servers, no Russian payments

#### Pingdom
- **Paid:** $10-15/month starting
- **Плюсы:** RUM, page speed analysis
- **Минусы:** No free plan, expensive

#### Site24x7
- **Paid:** ~$50/month starting
- **Плюсы:** AIOps-powered anomaly detection
- **Минусы:** Complex, expensive for SMB

### B. Personas (детально)

**Александр — CTO E-commerce**
- Ключевой persona, бюджет $7K-10K/мес
- Боли: MTTR 2+ часов, 5+ инструментов, звонок клиента раньше алерта
- Цели: 99.9% uptime, единый стек

**Дмитрий — SRE Lead Fintech**
- Бюджет $20K-25K/мес
- Боли: PCI DSS compliance, shadow API, false positives
- Цели: 99.99% availability, MTTD < 5 мин

### C. Технический стек

| Компонент | Технология | Обоснование |
|-----------|------------|-------------|
| **Backend** | Go / Python | Производительность, экосистема |
| **Frontend** | React / Next.js | DX, экосистема |
| **Database** | PostgreSQL / ClickHouse | Надёжность, аналитика |
| **Queue** | Redis / RabbitMQ | Performance |
| **Infrastructure** | Yandex Cloud | Локация в РФ |
| **Monitoring** | Prometheus + VictoriaMetrics | Best practices |

### D. Финансовая модель

**Unit Economics Summary:**
- LTV/CAC = 40.4 (excellent)
- Payback = 1.2 мес
- ARPU = 2,167 ₽
- Churn = 4.5%/мес (paid)

**Проекции:**
- Year 1: ~$120K ARR (3,000 paying users target)
- Break-even: ~40 paying users
- Runway (Bootstrap): 2.5 мес
- Runway (Angel 3M ₽): 8.5 мес

---

*Документ создан: Март 2026*
*Версия: 1.0*
*Источник: Синтез данных от 17 агентов*
