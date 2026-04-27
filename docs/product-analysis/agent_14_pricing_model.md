# Агент 14: Модель ценообразования

## Анализ моделей

### Freemium

**Плюсы:**
- Низкий барьер входа для пользователей
- Быстрый рост user base
- Виральный маркетинг (word-of-mouth)
- Сбор данных о поведении пользователей
- Конверсия в paid occurs organically при росте потребностей

**Минусы:**
- Free пользователи генерируют издержки (серверы, поддержка)
- Риск "бесплатных riders" которые никогда не конвертируются
- Сложность баланса между "слишком щедрым" и "слишком ограниченным"
- Cannibalization потенциальных paying customers
- Необходимо attrition management (удаление неактивных)

**Для нас:**
- UptimeRobot имеет 50 free monitors — это industry standard
- Наша цель — привлечь пользователей из РФ/СНГ, где платёжная дисциплина отличается
- Free tier критичен для initial traction
- Рекомендация: 25-50 monitors на Free, с ограничениями которые "зудят" при росте

### Tiered pricing

**Плюсы:**
- Предсказуемый revenue (MRR)
- Понятен для клиентов и sales team
- Естественный upgrade path (рост = смена tier)
- Простота в billing и ops
- Хорошо работает для B2B SaaS

**Минусы:**
- Risk of overpricing (клиент платит за неиспользуемые фичи)
- Risk of underpricing (power users "сидят" на дешёвых тарифах)
- Требует тщательного дизайна ограничений
- Может быть "жёстким" для non-linear growth

**Для нас:**
- Mid-market аудитория (из personas) предпочитает предсказуемые расходы
- 4-5 tiers покрывают все сегменты: Free → Starter → Professional → Business → Enterprise
- Это best fit для нашей целевой аудитории (CTO, SRE, Founders с бюджетами $500-25K/мес)

### Usage-based

**Плюсы:**
- Fair pricing (платишь за то, что используешь)
- Привлекает price-sensitive пользователей
- Масштабируется вместе с клиентом
- Отсутствие "sticker shock" от фиксированной цены

**Минусы:**
- Непредсказуемость расходов для клиента (и для нас)
- Сложность в прогнозировании revenue
- Требует sophisticated billing system
- Клиенты боятся "счётов-сюрпризов"
- Не работает для enterprise (нужны бюджеты, POs)

**Для нас:**
- Не рекомендуется как primary модель
- Может работать как hybrid: tiered + overage charges
- Опционально: usage-based для крупных enterprise клиентов

## Рекомендованная модель

### Выбор: Tiered pricing с Freemium

**Обоснование:**

1. **Целевая аудитория (personas):**
   - Александр (CTO): $7K-10K/месяц, предсказуемость критична
   - Дмитрий (SRE): $20K-25K/месяц, enterprise-контракт
   - Елена (Tech Lead): $3K-5K/месяц, средний бизнес
   - Михаил (Founder): $2K-3K/месяц, стартап
   - Виктория (DevOps): $500-1K/месяц, small team

   Все предпочитают предсказуемые бюджеты.

2. **Рынок РФ/СНГ:**
   - Платёжная дисциплина требует простой биллинг
   - Annual contracts более популярны чем monthly (закупки)
   - Сложные модели воспринимаются с недоверием

3. **Конкурентный ландшафт:**
   - UptimeRobot, Pingdom, StatusCake — все используют tiered
   - Это industry standard для мониторинга

4. **Удобство operations:**
   - Проще строить прогнозы revenue
   - Проще управлять capacity

### Детали модели

**Hybrid подход:**
- Primary: 4-tier structure (Free, Starter, Professional, Business)
- Secondary: Enterprise для крупных клиентов (custom pricing)
- Add-ons: SMS packs, additional status pages, white-label removal
- Overage: "Soft limits" с уведомлением, но не жёстким cutoff

**Умные ограничения:**
- Мониторы: primary limiter
- Check frequency: differentiation factor
- Team seats: для Premium tiers
- Retention дней: storage proxy
- API calls: rate limiting для защиты

## Финальная структура тарифов

### Free

| Параметр | Значение |
|----------|----------|
| Мониторов | 25 |
| Частота проверок | 5 минут |
| Alerting | Email, Telegram |
| Status page | 1 публичная (с брендингом Milan) |
| Интеграции | Telegram, Email |
| Retention данных | 7 дней |
| Team seats | 1 |
| API access | Базовый read-only (100 requests/день) |

**Цель:** Привлечение пользователей, демонстрация ценности, сбор leads.

**Ограничения:**
- 5-минутный интервал (интервал升级 trigger)
- Брендированная status page (white-label upgrade trigger)
- 7 дней retention (для истории — upgrade trigger)

---

### Starter (500 ₽/мес или 5 000 ₽/год)

| Параметр | Значение |
|----------|----------|
| Мониторов | 50 |
| Частота проверок | 2 минуты |
| Alerting | Email, Telegram, SMS (10/мес), Webhook |
| Status page | 3 публичные, custom branding (с remover) |
| Интеграции | Telegram, Email, Webhook, Slack |
| Retention данных | 30 дней |
| Team seats | 1 |
| API access | Read-write (1 000 requests/день) |
| Поддержка | Email, 48h response |

**Целевая аудитория:** Freelancers, небольшие проекты, стартапы на ранней стадии (персона Михаил).

**Value proposition:**
- Доступный мониторинг с продвинутыми фичами
- Status pages без брендинга Milan
- SMS-оповещения для critical alerts
- Цены в рублях, оплата СБП

**Upgrade triggers:**
- Больше 50 мониторов
- Нужна 1-минутная проверка
- Больше 1 пользователя в команде

---

### Professional (2 500 ₽/мес или 25 000 ₽/год)

| Параметр | Значение |
|----------|----------|
| Мониторов | 150 |
| Частота проверок | 1 минута |
| Alerting | Все каналы + неограниченные SMS |
| Status page | 10 публичных + 5 приватных |
| Интеграции | Все + PagerDuty, Discord, MS Teams |
| Retention данных | 90 дней |
| Team seats | 5 |
| API access | Full API (10 000 requests/день) |
| Поддержка | Telegram/чат, 24h response |
| Доп. фичи | Maintenance windows, SLA reporting, Multi-location (5+ локаций РФ/СНГ) |

**Целевая аудитория:** E-commerce компании, mid-size tech teams (персоны Елена, Viktoriya).

**Value proposition:**
- Enterprise-фичи по mid-market цене
- Единая система для всей команды
- SLA reporting для партнёров
- Priority support

**Upgrade triggers:**
- Больше 150 мониторов
- Нужен 30-секундный интервал
- Transaction monitoring
- Private location

---

### Business (7 500 ₽/мес или 75 000 ₽/год)

| Параметр | Значение |
|----------|----------|
| Мониторов | 500+ |
| Частота проверок | 30 секунд |
| Alerting | Все каналы, custom escalation policies |
| Status page | Неограниченные, white-label |
| Интеграции | Все + custom integrations |
| Retention данных | 1 год |
| Team seats | Неограниченно |
| API access | Unlimited rate limits |
| Поддержка | Выделенный менеджер, телефон, 4h response |
| Доп. фичи | Transaction monitoring, API monitoring, Private location, SSO, Audit logs |

**Целевая аудитория:** Крупный e-commerce, Fintech, Enterprise (персоны Александр, Дмитрий).

**Value proposition:**
- Enterprise-grade мониторинг без enterprise complexity
- Полный контроль и кастомизация
- Compliance-ready (PCI DSS templates)
- Выделенная поддержка

---

## Billing strategy

### Billing cycle

**Monthly:**
- Стандартный billing cycle
- Оплата в начале месяца
- Flexibility для изменения тарифа

**Annual:**
- Скидка 20% при оплате за год
- Скидка 30% при оплате за 2 года
- Популярен в РФ (бюджетирование, закупки)

**Rationale:**
- Annual улучшает retention и predictable revenue
- Скидка commensurate с market standard (20-30%)
- Для enterprise: annual contracts mandatory

### Payment methods

**СБП (Система быстрых платежей):**
- Commission-free для малого бизнеса
- Популярен в РФ с 2022
- Мгновенная оплата

**Банковская карта (ЮKassa/CloudPayments):**
- Visa/Mastercard (работают в РФ)
- МИР (национальная система)
- Recurring payments

**Платёжные поручения (B2B):**
- Для Business/Enterprise tier
- Юрлица оплачивают по счёту
- Post-payment option (по договору)

**Криптовалюта:**
- Опционально для international clients
- USDT (TRC20) для borderless payments

---

## Сравнение с конкурентами

| Параметр | Milan | UptimeRobot | Pingdom | Site24x7 |
|----------|-------|-------------|---------|----------|
| Free monitors | **25** | 50 | 0 (только trial) | 500 metrics |
| Free interval | 5 мин | 5 мин | 1 мин (trial) | 1 мин |
| Starter цена | **500 ₽** (~$5) | $7 (~$8 с курсом) | $10 | ~$50 |
| Starter monitors | 50 | 10 | 10 | 10 |
| Professional цена | **2 500 ₽** (~$25) | $29 | ~$50 | ~$195 |
| Professional monitors | 150 | 100 | ~50 | ~50 |
| Business цена | **7 500 ₽** (~$75) | $54+ | $100+ | ~$500+ |
| Business monitors | 500+ | 200 | ~100 | ~200 |
| Серверы в РФ | **Да** | Нет | Нет | Опционально |
| Оплата в рублях | **Да** | Нет | Нет | Нет |
| Telegram native | **Да** | Да | Да | Да |
| СБП оплата | **Да** | Нет | Нет | Нет |

**Ключевые преимущества:**
1. Цена в рублях (нет currency risk для РФ клиентов)
2. Серверы проверки в РФ (точность для RuNet)
3. Большее значение monitors за меньшую цену (на Starter и Pro tiers)
4. Локальные платежи (СБП)
5. Русскоязычная поддержка

---

## Psychology of pricing

### Anchor pricing

**Эффект якоря:** первый видимый цена создаёт референс.

**Применение:**
- На pricing странице показываем Business (7 500 ₽) первым — якорь сверху
- Professional (2 500 ₽) воспринимается как "выгодная середина"
- Starter (500 ₽) кажется "очень дешёвым"

**Визуальная стратегия:**
```
┌─────────────────────────────────────┐
│  BUSINESS     7 500 ₽/мес           │  ← Якорь (высокий)
│  500+ monitors, 30s interval        │
├─────────────────────────────────────┤
│  PROFESSIONAL  2 500 ₽/мес  ★ POPULAR│  ← Рекомендуемый
│  150 monitors, 1min interval        │
├─────────────────────────────────────┤
│  STARTER      500 ₽/мес             │  ← Входной
│  50 monitors, 2min interval         │
├─────────────────────────────────────┤
│  FREE         0 ₽/мес               │  ← Hook
│  25 monitors, 5min interval         │
└─────────────────────────────────────┘
```

### Decoy effect

**Эффект приманки:** третий选项 делает target option более привлекательным.

**Применение:**
- Professional vs Business: Business в 3x дороже, но monitors всего в 3.3x больше
- Professional кажется "оптимальным" выбором

### Loss aversion

**Страх потери:** люди сильнее реагируют на potential loss чем на gain.

**Применение:**
- "Сэкономьте 6 000 ₽ при годовой оплате" (frame как saving, а не как discount)
- "Ограничение: 25 monitors" (frame как potential loss, а не как limitation)
- Free trial countdown: "Осталось 14 дней Premium"

### Price ending

**Магия числа:**
- 500 ₽ vs 499 ₽: 499 воспринимается как "400 с чем-то"
- 2 500 ₽ vs 2 499 ₽: для B2B SaaS круглые числа работают лучше (прозрачность)

**Наша стратегия:** Круглые числа (500, 2 500, 7 500) — для B2B это воспринимается как честный, прозрачный pricing.

---

## A/B тесты для валидации

### Test 1: Free tier monitors

**Гипотеза:** 50 free monitors (как UptimeRobot) привлекут больше пользователей, но конверсия будет ниже, чем при 25 monitors.

**Варианты:**
- A: 25 free monitors
- B: 50 free monitors

**Метрика:**
- Primary: Free-to-paid conversion rate
- Secondary: Sign-up rate, churn rate, MRR after 3 months

**Duration:** 30 дней

**Success criteria:** Если conversion в B не ниже на 20%, то переходим на 50 monitors.

---

### Test 2: Anchor position

**Гипотеза:** Professional как первый тариф (сверху) даст более высокую конверсию в него, чем если он стоит вторым.

**Варианты:**
- A: Порядок: Business → Professional → Starter → Free
- B: Порядок: Professional → Business → Starter → Free

**Метрика:**
- Primary: Conversion в Professional
- Secondary: ARPU (Average Revenue Per User)

**Duration:** 14 дней

---

### Test 3: Annual discount

**Гипотеза:** 30% скидка за год даст более высокую LTV, чем 20%, даже с учётом более низкой маржинальности.

**Варианты:**
- A: 20% discount за год
- B: 30% discount за год

**Метрика:**
- Primary: LTV (Lifetime Value)
- Secondary: Annual payment adoption rate, churn rate

**Duration:** 60 дней (для накопления данных)

---

### Test 4: Add-on pricing

**Гипотеза:** SMS packs pricing влияет на adoption paid tiers.

**Варианты:**
- A: 10 SMS included в Starter, additional 5 ₽/SMS
- B: 5 SMS included в Starter, additional 3 ₽/SMS

**Метрика:**
- Primary: Starter adoption rate
- Secondary: SMS revenue per user

**Duration:** 30 дней

---

### Test 5: Status page strategy

**Гипотеза:** White-label status pages на Starter увеличат конверсию из Free.

**Варианты:**
- A: Status page только с брендингом Milan на Free, custom branding на Starter
- B: Custom branding уже на Free (но 1 page)

**Метрика:**
- Primary: Free-to-Starter conversion
- Secondary: Status page usage rate

**Duration:** 21 день

---

## Дополнительные рекомендации

### Tier migration strategy

**Gradual engagement:**
1. User sign-ups на Free
2. Onboarding email sequence (7 дней)
3. Day 7: "Upgrade to Starter for 2-minute checks"
4. Day 14: "Your Free tier has limits, upgrade for peace of mind"
5. Day 30: "You're hitting limits, upgrade to continue"

**In-app triggers:**
- "You have 23/25 monitors used. Upgrade for more."
- "5-minute check interval detected. Upgrade to Starter for 2-minute checks."

### Pricing page design

**Best practices:**
- Чёткая дифференциация tiers (feature comparison table)
- Highlight "Recommended" tier (Professional)
- Social proof: "Join 500+ companies monitoring with Milan"
- Trust signals: Серверы в РФ, ЮKassa payment
- FAQ section под pricing table

### Enterprise pricing approach

**Custom, not public:**
- Enterprise pricing не published на сайте
- "Contact us" CTA ведёт на форму
- Sales qualification перед custom quote

**Typical deal:**
- 1 000-10 000+ monitors
- 30-60 second intervals
- Custom SLA (99.9% for Milan service itself)
- Dedicated success manager
- Price: 15 000-100 000+ ₽/месяц

### Pricing evolution roadmap

**Phase 1 (Launch):**
- 4 tiers: Free, Starter (500 ₽), Professional (2 500 ₽), Business (7 500 ₽)
- Focus на acquisition и validation

**Phase 2 (6 месяцев):**
- A/B test результаты
- Possible price adjustment (+/- 20%)
- Add-on marketplace launch

**Phase 3 (12 месяцев):**
- Introduction of usage-based options для enterprise
- Platform pricing (для agencies/resellers)
- Geographic expansion pricing (если выходим за пределы РФ)

### Churn prevention

**Proactive monitoring:**
- Monitor usage patterns pre-churn
- Offer "pause account" вместо delete
- Win-back campaign с discounts

**Downgrade flexibility:**
- Allow downgrade без penalty
- Preserve data retention period
- "Pause subscription" option для seasonal businesses

---

## Sources

- Agent 1: Глобальные конкуренты — цены и фичи UptimeRobot, Pingdom, StatusCake, Site24x7
- Agent 6: User Personas — бюджеты и pain points целевой аудитории
- Agent 13: Price positioning — ценовой анализ и whitespace opportunities
- Industry research: SaaS pricing best practices, psychology of pricing
