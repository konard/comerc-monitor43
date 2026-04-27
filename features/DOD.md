# Definition of Done (DoD) для Feature файлов

> **Версия:** 1.3
> **Дата:** 2025-03-12
> **Статус:** Active
> **Changes:** Enhanced scenario classification rules (outcome-based, not parameter-based)

---

## Обязательные требования (MUST)

### 1. Структура Feature файла

Каждый feature файл ДОЛЖЕН содержать:

```gherkin
@epic=XX_epic_name
@user_story=XX_YY_user_story_name
# Description: Описание User Story

Feature: Название функции
  Как [роль]
  Я хочу [действие]
  Чтобы [бизнес-ценность]

  Background:
    Given общее предусловие 1
    And общее предусловие 2

  @use_case=uc_XX_YY_ZZ
  @critical
  Scenario: Название сценария
    Given контекст
    When действие
    Then ожидаемый результат
```

**Требования:**
- [ ] Заголовок `Feature:` с описанием на русском
- [ ] User Story в формате "Как... Я хочу... Чтобы..."
- [ ] Теги `@epic={epic_num}_{epic_name}` и `@user_story={epic_num}_{us_num}_{us_name}`
- [ ] Каждая строка отделена пустой строкой для читаемости

---

### 2. Синтаксис Gherkin

**Ключевые слова (английские):**
- `Feature` - определяет тестируемую функцию
- `Scenario` - конкретный тестовый случай
- `Scenario Outline` - шаблон для data-driven тестов
- `Background` - общие шаги для всех сценариев
- `Examples` - данные для Scenario Outline
- `Given` - начальный контекст/предусловие
- `When` - действие/событие
- `Then` - ожидаемый результат
- `And` - синоним Given/When/Then
- `But` - альтернатива And

**Требования:**
- [ ] Ключевые слова на английском
- [ ] Описания шагов на русском языке
- [ ] Отступы в 2 пробела для вложенности
- [ ] Пустые строки между сценариями
- [ ] `godog` или `cucumber` парсит без ошибок

**Анти-паттерны (❌ ЗАПРЕЩЕНО):**
- Смешивание языков: `И` вместо `And`
- Hardcoded текст сообщений в Then (используйте error codes)
- Отсутствие пустых строк между сценариями
- Вложенные сценарии (Scenario Outline без Examples)

---

### 3. Идентификация сценариев

Каждый сценарий ДОЛЖЕН иметь:

```gherkin
  @use_case=uc_XX_YY_ZZ
  @tag1
  @tag2
  Scenario: Описательное название сценария на русском
```

**Требования:**
- [ ] Комментарий `@use_case=uc_XX_YY_ZZ` перед каждым сценарием
- [ ] Название сценария описывает суть (что тестируем)
- [ ] Имена сценариев уникальны в пределах feature файла
- [ ] Теги для категоризации: `@critical`, `@validation`, `@security`, `@integration`
- [ ] **Основной сценарий (Happy Path) НЕ заканчивается на латинскую букву**
- [ ] **Альтернативные сценарий ДОЛЖЕН заканчиваться на маленькую латинскую букву (a, b, c, ...)**

**Правила использования тегов:**
- [ ] **Теги НЕ должны дублироваться в одном сценарии**
- [ ] Каждый тег (`@use_case`, `@critical`, `@validation`, etc.) должен появляться только один раз
- [ ] Дублирование тегов считается ошибкой форматирования

**Примеры правильного именования:**

```gherkin
# Основной сценарий (без буквы в конце)
@use_case=uc_01_01_01
Scenario: Создание нового HTTP монитора
  ...

# Альтернативные сценарии (с буквой в конце)
@use_case=uc_01_01_02
Scenario: Попытка создания монитора с невалидным URL a
  ...

@use_case=uc_01_01_03
Scenario: Превышение лимита мониторов на Free тарифе b
  ...

@use_case=uc_01_01_04
Scenario: Попытка создания монитора с дублирующимся именем c
  ...
```

**Примеры НЕправильного использования тегов:**

```gherkin
# ❌ НЕПРАВИЛЬНО: Дублирующийся @use_case тег
@use_case=uc_01_01_02
@use_case=uc_01_01_02
@critical
Scenario: Список всех мониторов
  ...

# ❌ НЕПРАВИЛЬНО: Дублирующиеся теги
@critical
@critical
@validation
Scenario: Создание монитора
  ...

# ❌ НЕПРАВИЛЬНО: @use_case дублируется с другими тегами
@use_case=uc_01_01_03
@use_case=uc_01_01_03
@critical
@integration
Scenario: Тестирование интеграции
  ...
```

**Примеры ПРАВИЛЬНОГО использования тегов:**

```gherkin
# ✓ ПРАВИЛЬНО: Один @use_case тег
@use_case=uc_01_01_02
@critical
Scenario: Список всех мониторов
  ...

# ✓ ПРАВИЛЬНО: Несколько разных тегов (без дублирования)
@use_case=uc_01_01_05
@critical
@validation
@integration
Scenario: Создание монитора с валидацией
  ...

# ✓ ПРАВИЛЬНО: Единственный экземпляр каждого тега
@use_case=uc_01_01_10
@critical
@security
@boundary
Scenario: Проверка граничных условий
  ...
```

**Правило:**
- Основной сценарий = без суффикса (например: "Создание монитора")
- Альтернативный сценарий = суффикс "a", "b", "c"... (например: "Создание монитора с невалидным URL a")
- Буквы присваиваются последовательно в порядке появления сценариев

---

#### 3.1. Критерии классификации сценариев (Enhanced v1.3)

**Сценарий является HAPPY_PATH (основной), если:**
- Описывает **УСПЕШНОЕ** выполнение операции (без error codes)
- Является базовой CRUD операцией (Create, Read, Update, Delete)
- **Демонстрирует работу функции** с НЕстандартными параметрами (feature demo)
- Переход состояния завершается **успешно** (UP → PAUSED → UP)
- Тестирует основную функциональность при идеальных условиях

**Сценарий является ALTERNATIVE (альтернативный), если:**
- Возвращает **error code** (INVALID_URL, MONITOR_LIMIT_REACHED, FORBIDDEN, etc.)
- Описывает обработку ошибок, валидацию, edge cases
- Тестирует **граничные условия** (boundary: N-1, N, N+1)
- Проверяет нарушения безопасности (unauthorized access, SQL injection)
- Тестирует сбои зависимостей (database timeout, service unavailable)
- Имеет теги: @validation, @security, @boundary, @integration, @business_rule

**Ключевой принцип:** Основывается на **ИСХОДЕ** (outcome), а не параметрах
- **SUCCESS** (даже с нестандартными параметрами) → Happy Path (без суффикса)
- **FAILURE** или **error code** → Alternative (суффикс a, b, c, ...)

**Ключевые слова для автоматической классификации ALTERNATIVE:**
```
Попытка, Ошибка, Неудачный, Превышение, Блокировка, Отказ,
Невалидный, Неверный, Сбой, Таймаут, Недоступен, Отклоняет,
Инъекция, Атака, Конкурентный
```

**Таблица принятия решений:**

| Тип сценария | Пример названия | Суффикс | Теги |
|-------------|----------------|---------|------|
| CRUD операция | "Создание монитора" | Нет (HAPPY_PATH) | - |
| CRUD операция | "Получение деталей" | Нет (HAPPY_PATH) | - |
| Feature demo (SUCCESS) | "Создание с интервалом 15с (платный)" | Нет (HAPPY_PATH) | - |
| Feature demo (SUCCESS) | "Создание с таймаутом" | Нет (HAPPY_PATH) | - |
| Feature demo (SUCCESS) | "Создание с рабочими часами" | Нет (HAPPY_PATH) | - |
| State transition (SUCCESS) | "Приостановка монитора" | Нет (HAPPY_PATH) | - |
| Ошибка валидации | "Попытка создания с невалидным URL a" | "a" | @validation |
| Нарушение лимита | "Превышение лимита мониторов b" | "b" | @business_rule |
| Security breach | "SQL инъекция в поле URL c" | "c" | @security |
| Boundary test | "Ровно 25 мониторов на Free d" | "d" | @boundary |
| Business rule violation | "Интервал 15с на Free тарифе e" | "e" | @business_rule |

**Алгоритм применения правил:**

Для каждого feature файла:

```
1. Сгруппировать сценарии по user_story (XX_YY в use_case)
2. Внутри каждой группы:
   a) Определить тип каждого сценария (HAPPY_PATH или ALTERNATIVE)
   b) HAPPY_PATH сценарии = без суффикса
   c) ALTERNATIVE сценарии = добавить a, b, c, ... по порядку
3. Scenario Outline всегда = HAPPY_PATH (без суффикса)
```

**Пример для 01_01_monitor_management.feature:**

```
Группа: 01_01 (Monitor Management)

HAPPY_PATH (без суффикса):
  uc_01_01_01: Создание нового HTTP монитора
  uc_01_01_02: Получение деталей монитора
  uc_01_01_03: Список всех мониторов
  uc_01_01_04: Обновление монитора
  uc_01_01_05: Удаление монитора
  uc_01_01_06: Приостановка монитора
  uc_01_01_07: Возобновление монитора
  uc_01_01_08: Создание монитора с интервалом 15 секунд (платный тариф)
  uc_01_01_12: Создание монитора с таймаутом
  uc_01_01_13: Создание монитора с рабочими часами
  # ↑ Это HAPPY_PATH потому что они успешные и демонстрируют возможности функции

ALTERNATIVE (с суффиксами):
  uc_01_01_01a: Попытка создания монитора с дублирующимся именем a
  uc_01_01_01b: Превышение лимита мониторов на Free тарифе b
  uc_01_01_01c: Попытка создания монитора с невалидным URL c
  uc_01_01_01d: Попытка создания монитора с пустым именем d
  uc_01_01_01e: Попытка создания монитора с именем длиннее 255 символов e
  uc_01_01_01f: Попытка создания монитора с нулевым интервалом f
  uc_01_01_01g: Попытка создания монитора с таймаутом больше интервала g
  uc_01_01_01h: Попытка создания монитора с невалидными рабочими часами h
  uc_01_01_02a: Попытка просмотра чужого монитора a
  uc_01_01_04a: Попытка обновления монитора на дублирующееся имя a
  uc_01_01_04b: Попытка обновления чужого монитора b
  uc_01_01_05a: Попытка удаления чужого монитора a
  # ↑ Это ALTERNATIVE потому что возвращают error codes
```

---

### 4. Типы сценариев

Каждый feature file ДОЛЖЕН содержать следующие типы сценариев:

#### 4.1 Happy Path Scenarios (Основной сценарий)

**Цель:** Проверить базовую функциональность при идеальных условиях.

**Правило именования:** Основной сценарий НЕ заканчивается на латинскую букву

```gherkin
  @use_case=uc_01_01_01
  @critical
  Scenario: Создание нового HTTP монитора
    Given пользователь авторизован
    And пользователь имеет тариф "Free" с лимитом "25" мониторов
    When пользователь создаёт монитор с параметрами:
      | name     | API Service            |
      | url      | https://api.example.com |
      | interval | 5 minutes               |
    Then монитор создан успешно
    And монитор имеет статус "PENDING"
    And первая проверка запланирована через "5 minutes"
```

**Проверка:**
- [ ] Основной сценарий успеха присутствует
- [ ] Все нормальные шаги покрыты
- [ ] Название сценария НЕ заканчивается на латинскую букву

#### 4.2 Alternative Scenarios (Альтернативные сценарии)

**Цель:** Проверить обработку ошибок, граничные условия и edge cases.

**Правило именования:** Все альтернативные сценарии ДОЛЖНЫ заканчиваться на маленькую латинскую букву (a, b, c, ...)

**Категории альтернативных сценариев:**

**a) Validation Errors** — ошибки валидации ввода:
```gherkin
  @use_case=uc_01_01_12
  @critical
  @validation
  Scenario: Попытка создания монитора с невалидным URL
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name | Invalid Monitor |
      | url  | not-a-valid-url |
    Then возвращается ошибка "INVALID_URL"
    And монитор не создан
```

**b) Business Rule Violations** — нарушение бизнес-правил:
```gherkin
  @use_case=uc_04_01_07
  @critical
  @business_rule
  Scenario: Превышение лимита мониторов для тарифа
    Given пользователь имеет тариф "Free"
    And пользователь создал "25" мониторов
    When пользователь создаёт 26-й монитор
    Then возвращается ошибка "MONITOR_LIMIT_REACHED"
    And монитор не создан
    And сообщение содержит информацию о лимите
```

**c) State Transitions** — переходы между состояниями:
```gherkin
  @use_case=uc_01_01_06
  @critical
  @state_transition
  Scenario: Приостановка активного монитора
    Given монитор "API Service" в статусе "UP"
    When пользователь приостанавливает монитор
    Then монитор переходит в статус "PAUSED"
    And новые проверки не планируются
    And существующие проверки завершаются
```

**d) Edge Cases** — граничные условия:
```gherkin
  @use_case=uc_02_02_12
  @critical
  @boundary
  Scenario: Алерт срабатывает точно при N consecutive failures
    Given правило алерта с consecutive_failures = "3"
    When монитор падает "3" раза подряд
    Then создан алерт со статусом "TRIGGERED"
```

**e) Boundary Conditions** — проверки на границах:
```gherkin
  @use_case=uc_02_02_13
  @critical
  @boundary
  Scenario: Алерт не срабатывает при N-1 consecutive failures
    Given правило алерта с consecutive_failures = "3"
    When монитор падает "2" раза подряд
    Then алерт не создан
```

**f) Integration Failures** — сбои зависимостей:
```gherkin
  @use_case=uc_04_02_16
  @critical
  @integration
  Scenario: Webhook delivery failure and retry
    Given платежная система отправляет webhook
    When webhook не доставлен (network error)
    Then система регистрирует неудачную попытку
    And повторяет запрос webhook через "5" минут
```

**g) Security Scenarios** — сценарии безопасности:
```gherkin
  @use_case=uc_05_01_22
  @critical
  @security
  @session_fixation_prevention
  Scenario: Предотвращение фиксации сессии
    Given злоумышленник создает сессию с идентификатором "compromised"
    When пользователь авторизуется с существующей сессией
    Then идентификатор сессии изменен на новый
    And старый идентификатор недействителен
```

**h) Performance Edge Cases** — производительностные сценарии:
```gherkin
  @use_case=uc_03_03_20
  @performance
  Scenario: Дросселирование высокочастотных обновлений
    Given пользователь имеет "100" мониторов
    And все мониторы обновляются каждую "1 second"
    When поступает более "50" обновлений в секунду
    Then обновления группируются в пакеты
    And UI обновляется не чаще чем "20 times per second"
```

**Проверка:**
- [ ] Все validation ошибки покрыты (@critical)
- [ ] Граничные условия (boundary conditions) протестированы
- [ ] Основные state transitions покрыты
- [ ] Критичные integration failures обработаны
- [ ] Безопасные сценарии (authz, authn) протестированы
- [ ] Цель: >80% альтернативных сценариев от общего числа

---

### 5. Error Codes и Статусы

#### 5.1 Error Codes

**🔑 Канонический источник:** `/Users/raul/go/src/monitor/features/ERROR_CODES.md`

**Формат:** `UPPERCASE_SNAKE_CASE`

**Canonical Error Code Registry:**
Все error codes ДОЛЖНЫ соответствовать реестру в ERROR_CODES.md. При создании новых сценариев:
1. Сначала проверьте ERROR_CODES.md
2. Используйте существующий error code если доступен
3. Только если error code не существует, добавьте в ERROR_CODES.md

**Домены error codes:**
- **Authentication:** `AUTH_REQUIRED`, `AUTH_FAILED`, `AUTH_TOKEN_EXPIRED`, `AUTH_PASSWORD_TOO_WEAK`
- **Authorization:** `FORBIDDEN`, `INSUFFICIENT_PERMISSIONS`, `ROLE_NOT_FOUND`, `PERMISSION_DENIED`
- **Validation:** `INVALID_URL`, `INVALID_EMAIL`, `MISSING_REQUIRED_FIELD`, `FIELD_TOO_LONG`, `INVALID_FORMAT`
- **Monitor:** `MONITOR_NOT_FOUND`, `MONITOR_ALREADY_EXISTS`, `MONITOR_CONFIGURATION_INVALID`, `MONITOR_DEGRADED`
- **Alert:** `ALERT_NOT_FOUND`, `ALERT_RATE_LIMITED`, `ALERT_STORM_DETECTED`, `ALERT_COOLDOWN_ACTIVE`
- **Billing:** `PAYMENT_FAILED`, `PAYMENT_GATEWAY_ERROR`, `SUBSCRIPTION_NOT_FOUND`, `SUBSCRIPTION_LIMIT_EXCEEDED`
- **Integration:** `WEBHOOK_DELIVERY_FAILED`, `WEBHOOK_SIGNATURE_INVALID`, `API_KEY_INVALID`, `RATE_LIMIT_EXCEEDED`
- **System:** `INTERNAL_SERVER_ERROR`, `SERVICE_UNAVAILABLE`, `TIMEOUT`, `CONFLICT`

**Требования:**
- [ ] Все ошибки используют формат `UPPERCASE_SNAKE_CASE`
- [ ] Error codes соответствуют ERROR_CODES.md (canonical source)
- [ ] Error codes уникальны в пределах системы
- [ ] Error codes согласованы cross-epic
- [ ] Error responses содержат структуру:
  ```json
  {
    "error": {
      "code": "MONITOR_LIMIT_REACHED",
      "message": "Detailed message here",
      "details": {...},
      "request_id": "uuid"
    }
  }
  ```

#### 5.2 Статусы сущностей

**Мониторы:** `PENDING`, `UP`, `DOWN`, `DEGRADED`, `PAUSED`

**Алерты:** `TRIGGERED`, `ACKNOWLEDGED`, `RESOLVED`, `MUTED`, `SUPPRESSED_BY_MAINTENANCE`

**Подписки:** `ACTIVE`, `GRACE_PERIOD`, `SUSPENDED`, `CANCELED`

**Платежи:** `PENDING`, `COMPLETED`, `FAILED`, `TIMEOUT`, `REFUNDED`

**Пользователи:** `ACTIVE`, `INACTIVE`, `SUSPENDED`

**Требования:**
- [ ] Статусы соответствуют единому источнику правды
- [ ] Статусы согласованы cross-epic
- [ ] Все state transitions покрыты сценариями

---

### 6. API Response Format

**Правило:** Бекенд не возвращает отображаемый текст.

**Требования:**
- [ ] Error responses: только коды и данные (без localized сообщений)
- [ ] Success responses: только данные (без success сообщений)
- [ ] Вся i18n на стороне frontend/mobile

**Правильный пример:**
```json
{
  "error": {
    "code": "MONITOR_LIMIT_REACHED",
    "message": "Detailed technical message",
    "details": {
      "limit": 25,
      "current": 26,
      "tier": "Free"
    },
    "request_id": "req_123"
  }
}
```

**Неправильный пример:**
```json
{
  "error": {
    "message": "Вы превысили лимит мониторов для тарифа Free"  // ❌ localized on backend
  }
}
```

---

### 7. Аудит логирование

Критические действия ДОЛЖНЫ логироваться:

**Когда логировать:**
- Создание/обновление/удаление мониторов
- Изменение подписки/тарифа
- Платежные операции
- Изменение ролей/разрешений
- Авторизация/деавторизация
- Failed attempts (passwords, limits)
- Security события (session fixation, privilege escalation)

**Формат лога:**
```gherkin
  Then действие в аудит лог записано как "action_name"
  And запись содержит:
    | user_id     |
    | timestamp   |
    | ip_address  |
    | action_type |
    | details     |
```

**Требования:**
- [ ] Все критические действия логируются
- [ ] Логи содержат enough information для forensic analysis
- [ ] Логи защищены от подделы (cryptographic signature)
- [ ] Логи имеют retention period (30 дней онлайн, 1 год архив)

---

### 7.1 Retry и Recovery логика

**📋 Канонический источник:** `/Users/raul/go/src/monitor/features/RETRY_STRATEGIES.md`

**Retry Strategies:**

1. **Exponential Backoff with Jitter** (default for webhooks, API calls)
   - Formula: `delay_ms = base_delay_ms * (2 ^ attempt_number) + random_jitter(-25%, +25%)`
   - Base delay: 30 seconds
   - Max delay: 60 seconds
   - Max attempts: 3

2. **Linear Backoff** (for payments, idempotent operations)
   - Formula: `delay_ms = base_delay_ms * attempt_number`
   - Base delay: 1 hour
   - Max attempts: 3

3. **Fixed Delay** (for database, cache retries)
   - Formula: `delay_ms = 1000` (fixed 1 second)
   - Max attempts: 5

**Retryable Error Codes:**
- `CONNECTION_TIMEOUT`, `CONNECTION_REFUSED`, `GATEWAY_TIMEOUT` (504)
- `SERVICE_UNAVAILABLE` (503), `RATE_LIMIT_EXCEEDED` (429)
- `WEBHOOK_DELIVERY_FAILED`, `PAYMENT_GATEWAY_TIMEOUT`

**Non-Retryable Error Codes:**
- Validation errors (4xx): `INVALID_URL`, `MISSING_REQUIRED_FIELD`, `FIELD_TOO_LONG`
- Auth errors: `UNAUTHORIZED`, `FORBIDDEN`, `WEBHOOK_SIGNATURE_INVALID`
- Client errors: `NOT_FOUND` (404), `CONFLICT` (409)

**Требования:**
- [ ] Все integration failure сценарии указывают retry strategy
- [ ] Retryable error codes соответствуют RETRY_STRATEGIES.md
- [ ] Non-retryable error codes немедленно возвращают ошибку
- [ ] Exhausted retries перемещаются в dead letter queue (DLQ)
- [ ] Retry попытки логируются с attempt number и strategy

**Пример сценария с retry логикой:**
```gherkin
@use_case=uc_02_03_05
@critical
@integration
Scenario: Webhook delivery retry with exponential backoff a
  Given webhook endpoint returns SERVICE_UNAVAILABLE
  When system delivers webhook
  Then system retries webhook delivery after 30 seconds using exponential backoff
  And system logs retry attempt 1 of 3
  And webhook delivery succeeds on retry
```

**Dead Letter Queue (DLQ) Handling:**
- DLQ для exhausted retries
- Alert ops team при DLQ size > 1000
- Manual replay from DLQ via admin interface

---

### 8. Кросс-эпик консистентность

**📋 Канонические источники:**
- **Integration Matrix:** `/Users/raul/go/src/monitor/features/INTEGRATION_MATRIX.md`
- **Retry Strategies:** `/Users/raul/go/src/monitor/features/RETRY_STRATEGIES.md`
- **Error Code Registry:** `/Users/raul/go/src/monitor/features/ERROR_CODES.md`

**Требования:**
- [ ] Сценарии внутри эпика не содержат противоречий
- [ ] Error codes согласованы между эпиками (см. ERROR_CODES.md)
- [ ] Статусы сущностей согласованы между эпиками
- [ ] Бизнес-правила согласованы (лимиты, интервалы, пороги)
- [ ] Data models согласованы (поля, типы, форматы)
- [ ] Cross-epic интеграции задокументированы (см. INTEGRATION_MATRIX.md)
- [ ] Retry логика согласована между эпиками (см. RETRY_STRATEGIES.md)

**Cross-Epic Integration Points (из INTEGRATION_MATRIX.md):**
- Epic 01 → Epic 02: Monitor state changes trigger alert evaluation
- Epic 01 → Epic 03: Monitor updates displayed on dashboard
- Epic 01 → Epic 08: Maintenance windows pause monitoring
- Epic 02 → Epic 03: Alert status displayed on dashboard
- Epic 04 → Epic 01: Billing limits enforce monitor constraints
- Epic 06 → Epic 01: SLA reports query monitor formulas
- Epic 08 → Epic 02: Maintenance suppresses alerts
- Epic 08 → Epic 06: Maintenance time excluded from SLA

**Использование:**
При добавлении cross-epic операций:
1. Проверьте INTEGRATION_MATRIX.md для существующих integration points
2. Добавьте integration комментарий к сценарию: `# Integration: Emits event to Epic XX (Description)`
3. Ссылайтесь на конкретные scenario IDs: `# See: Epic XX, uc_XX_YY_ZZ`

---

## Рекомендуемые практики (SHOULD)

### 1. Data Tables

Andспользуйте data tables для параметров вместо длинных списков:

**Хорошо:**
```gherkin
When пользователь создаёт монитор с параметрами:
  | name     | API Service            |
  | url      | https://api.example.com |
  | interval | 5 minutes               |
  | timeout  | 30 seconds             |
```

**Плохо:**
```gherkin
When пользователь создаёт монитор с параметрами: name = "API Service", url = "https://api.example.com", interval = "5 minutes", timeout = "30 seconds"
```

### 2. Scenario Outline

Andспользуйте Scenario Outline для повторяющихся сценариев:

```gherkin
  Scenario Outline: Создание монитора с разными интервалами
    Given пользователь авторизован
    And пользователь имеет тариф "<tier>"
    When пользователь создаёт монитор с интервалом "<interval>"
    Then монитор создан с интервалом "<interval>"

    Examples:
      | tier     | interval  |
      | Free     | 5 minutes |
      | Starter  | 2 minutes |
      | Pro      | 1 minute  |
```

### 3. Background

Andспользуйте Background для общих шагов:

```gherkin
  Background:
    Given пользователь авторизован
    And пользователь имеет тариф "Free"

  @use_case=uc_01_01_01
  Scenario: Создание монитора
    When пользователь создаёт монитор ...

  @use_case=uc_01_01_02
  Scenario: Создание второго монитора
    When пользователь создаёт ещё один монитор ...
```

### 4. Теги для группировки

Andспользуйте теги для категоризации сценариев:

```gherkin
  @critical
  @validation
  @security
  @integration
  @performance
  @state_transition
  @boundary
  @edge_case
```

---

## Метрики качества

Для каждого feature файла отслеживайте:

| Метрика | Цель | Как измерить |
|---------|------|--------------|
| **Покрытие альтернативных сценариев** | >80% | (Alternative / Total scenarios) × 100% |
| **Критические сценарии покрыты** | 100% | Все @critical tagged scenarios реализованы |
| **Противоречия** | 0 | Cross-epic validation pass |
| **Синтаксические ошибки** | 0 | `godog parse` — нет ошибок |
| **Audit logging** | 100% | Все критические действия логируются |

**Команда для проверки метрик:**
```bash
# Подсчет сценариев
grep -c "^  Scenario:" features/XX_YY/*.feature

# Подсчет альтернативных сценариев (оценка вручную)

# Проверка синтаксиса
godog parse features/

# Поиск противоречий (cross-epic validation)
```

---

## Checklist для Review

Перед merge feature файла проверьте:

### Структура
- [ ] Feature header с описанием
- [ ] User Story в формате "Как... Я хочу... Чтобы..."
- [ ] Теги @epic и @user_story присутствуют
- [ ] Все сценарии имеют @use_case комментарии
- [ ] Теги НЕ дублируются в одном сценарии (каждый тег появляется только один раз)

### Синтаксис
- [ ] Gherkin ключевые слова на английском
- [ ] Описания на русском
- [ ] Пустые строки между сценариями
- [ ] Отступы 2 пробела
- [ ] `godog parse` успешен

### Содержание
- [ ] Happy Path сценарий присутствуют (НЕ заканчиваются на латинскую букву)
- [ ] Validation ошибки покрыты (@critical)
- [ ] Boundary conditions протестированы (@critical)
- [ ] State transitions покрыты (@critical)
- [ ] Security сценарии присутствуют (@critical)
- [ ] Integration failures обработаны (@critical)
- [ ] Error codes в формате UPPERCASE_SNAKE_CASE
- [ ] Аудит логирование для критических действий
- [ ] API responses не содержат localized текст
- [ ] Альтернативные сценарии заканчиваются на латинскую букву (a, b, c, ...)
- [ ] Основные сценарии НЕ заканчиваются на латинскую букву

### Консистентность
- [ ] Нет противоречий внутри эпика
- [ ] Error codes согласованы cross-epic
- [ ] Статусы согласованы cross-epic
- [ ] Бизнес-правила согласованы

### Покрытие
- [ ] >80% альтернативных сценариев
- [ ] Все @critical сценарии покрыты
- [ ] Основные edge cases протестированы

---

## Полезные команды

```bash
# Проверка синтаксиса всех feature файлов
godog parse features/

# Запуск конкретного сценария
godog features/01_monitoring/01_01_monitor_management.feature \
  --name "Создание нового HTTP монитора"

# Запуск всех сценариев с тегом @critical
godog features/ --tags @critical

# Запуск всех сценариев с тегом @validation
godog features/ --tags @validation

# Запуск всех сценариев с тегом @security
godog features/ --tags @security

# Генерация отчета о покрытии
godog features/ --format junit > coverage.xml

# Подсчет количества сценариев в файле
grep -c "^  Scenario:" features/XX_YY/ZZ.feature

# Поиск конкретных use case
grep "uc_01_01_01" features/ -R

# Валидация всех feature файлов
find features/ -name "*.feature" -exec godog parse {} \;

# Проверка на дублирование тегов в файле
awk '/^  @/ {tag=$0; count[tag]++} END {for (t in count) if (count[t] > 1) print t, "дублируется", count[t], "раз"}' features/XX_YY/ZZ.feature

# Проверка на дублирование @use_case тегов во всех файлах
for file in features/*/*.feature; do echo "=== $file ==="; awk '/^  @use_case=/ {print $0}' "$file" | sort | uniq -d; done
```

---

## Словарь терминов

| Термин | Определение | Пример |
|--------|------------|--------|
| **Happy Path** | Основной сценарий успеха при идеальных условиях. НЕ заканчивается на латинскую букву | "Создание монитора" |
| **Alternative Scenario** | Альтернативный сценарий (error path, edge case). Заканчивается на маленькую латинскую букву (a, b, c, ...) | "Создание монитора с невалидным URL a" |
| **Critical Scenario** | Критический сценарий, который должен быть реализован до релиза | Validation errors, security breaches |
| **Boundary Condition** | Проверка на границе диапазона (N, N-1, N+1) | Exactly 3 failures vs 2 failures |
| **State Transition** | Переход сущности между состояниями | UP → PAUSED → UP |
| **Edge Case** | Маловероятное, но возможное условие | 1000+ мониторов, concurrent updates |
| **Integration Failure** | Сбой внешней зависимости или сервиса | Database down, API timeout |
| **Validation Error** | Ошибка валидации входных данных | Invalid URL, missing required field |
| **Business Rule** | Правило, ограничивающее бизнес-логику | Monitor limits per tier |
| **Recovery Scenario** | Сценарий восстановления после сбоя | Retry logic, fallback behavior |

---

## Автоматизированная валидация

### Обзор

Для обеспечения соответствия DOD требованиям создан набор автоматизированных инструментов валидации, которые проверяют feature файлы на соответствие всем обязательным требованиям.

### Инструменты валидации

#### 1. Скрипт валидации DOD

**Файл:** `/Users/raul/go/src/monitor/scripts/validate_dod.sh`

**Проверки:**
- Структура Feature файлов (Feature header, User Story, теги)
- Наличие @use_case комментариев у всех сценариев
- Формат error codes (UPPERCASE_SNAKE_CASE)
- Отсутствие локализованных сообщений в backend ответах
- Подсчет @critical tagged сценариев
- Проверка покрытия audit logging
- Расчет покрытия альтернативными сценариями
- Проверка синтаксиса Gherkin

**Запуск локально:**

```bash
# Базовый запуск
./scripts/validate_dod.sh

# С自定义 директорией feature файлов
./scripts/validate_dod.sh --features-dir ./features

# С自定义 файлом отчета
./scripts/validate_dod.sh --output /tmp/my_report.txt

# Показать справку
./scripts/validate_dod.sh --help
```

**Выходные данные:**

Скрипт генерирует:
- Консольный вывод с цветными сообщениями о статусе
- Детальный отчет в файле (по умолчанию: `/tmp/dod_validation_report.txt`)
- Процент покрытия альтернативными сценариями
- Список всех найденных нарушений
- Код возврата: 0 (успех) или 1 (ошибки)

**Пример вывода:**

```
========================================
DOD Validation Report
========================================
Generated: 2026-03-09 12:34:56

>>> Validating 18 feature files...

Validating: 01_01_monitor_management.feature
✓ No issues found (Coverage: 75%)

Validating: 02_01_alert_channels.feature
⚠ WARNING: Missing audit logging for 'создаёт' operations
✗ ERROR: Missing @use_case comments: 02_01_alert_channels.feature (2 scenarios)

>>> Validation Summary

Total Features: 18
Total Scenarios: 245
Critical Scenarios: 89
Features with Issues: 3
Estimated Alternative Coverage: 76%

>>> Compliance Check

✗ 3 feature(s) have issues
⚠ Alternative scenario coverage: 76% (target: >80%)

✗ DOD Validation FAILED
```

#### 2. Pre-commit Hook

**Файл:** `/Users/raul/go/src/monitor/.git/hooks/pre-commit`

Автоматически запускается перед каждым коммитом и блокирует коммит, если:

- Feature файлы не проходят валидацию DOD
- Найдены критические нарушения структуры или синтаксиса

**Использование:**

```bash
# Нормальный коммит (hook запускается автоматически)
git commit -m "Add new feature"

# Принудительный коммит (обход hook - не рекомендуется)
git commit --no-verify -m "WIP commit"
```

**Сообщения об ошибках:**

При ошибках валидации hook покажет:
- Список staged feature файлов
- Детальные ошибки валидации
- Инструкции по исправлению
- Путь к полному отчету

#### 3. CI/CD Integration

**Файл:** `/Users/raul/go/src/monitor/.github/workflows/dod-validation.yml`

GitHub Actions workflow, который:

**Триггеры:**
- Все Pull Requests (opened, synchronize, reopened)
- Push в main/develop ветки
- Ручной запуск (workflow_dispatch)

**Действия:**
1. Проверяет код и устанавливает зависимости
2. Запускает валидацию DOD
3. Парсит результаты и извлекает метрики
4. Генерирует комментарий к PR с результатами
5. Загружает артефакт с полным отчетом
6. Блокирует merge при неудачной валидации

**Пример комментария в PR:**

```markdown
## 🔍 DOD Validation Results

### Summary
- **Total Features:** 18
- **Total Scenarios:** 245
- **Critical Scenarios:** 89
- **Features with Issues:** 3
- **Alternative Coverage:** 76%

### Status
❌ **Validation FAILED**

### Issues Found

Please review the validation report below and fix the issues:

[Detailed validation report...]
```

### Требования к валидации

#### Успешная валидация требует:

1. **Структурная целостность:**
   - ✓ Feature header с описанием
   - ✓ User Story в формате "Как... Я хочу... Чтобы..."
   - ✓ Теги @epic и @user_story
   - ✓ @use_case комментарии на всех сценариях

2. **Синтаксическая правильность:**
   - ✓ Gherkin ключевые слова на английском
   - ✓ Правильные отступы (2 пробела)
   - ✓ Пустые строки между сценариями
   - ✓ Отсутствие синтаксических ошибок godog

3. **Качество контента:**
   - ✓ Error codes в формате UPPERCASE_SNAKE_CASE
   - ✓ Отсутствие локализованных сообщений в backend
   - ✓ Audit logging для критических операций
   - ✓ >80% покрытия альтернативными сценариями

### Устранение неполадок

#### Частые проблемы и решения

**1. Ошибка: "Missing Feature header"**

**Проблема:** Feature файл не начинается с `Feature:`

**Решение:**
```gherkin
@epic=01_monitoring
@user_story=01_01_monitor_management
# Description: Описание

Feature: Название функции  # ← Обязательно
  Как роль
  Я хочу действие
  Чтобы бизнес-ценность
```

**2. Ошибка: "Invalid User Story format"**

**Проблема:** Отсутствует формат "Как... Я хочу... Чтобы..."

**Решение:** Убедитесь, что все три компонента присутствуют:
```gherkin
Feature: Управление мониторами
  Как пользователь              # ← Как
  Я хочу управлять мониторами   # ← Я хочу
  Чтобы отслеживать сервисы     # ← Чтобы
```

**3. Ошибка: "Missing @use_case comments"**

**Проблема:** Сценарии не имеют @use_case идентификатора

**Решение:** Добавьте @use_case перед каждым сценарием:
```gherkin
@use_case=uc_01_01_01
Scenario: Создание монитора
  ...
```

**4. Ошибка: "Lowercase error codes"**

**Проблема:** Error codes не в формате UPPERCASE_SNAKE_CASE

**Решение:**
```gherkin
# Неправильно:
Then возвращается ошибка "monitor_limit_reached"

# Правильно:
Then возвращается ошибка "MONITOR_LIMIT_REACHED"
```

**5. Предупреждение: "Missing audit logging"**

**Проблема:** Критические операции не имеют audit logging

**Решение:** Добавьте audit logging шаги:
```gherkin
Then действие в аудит лог записано как "monitor_created"
  | field   | value    |
  | name    | Monitor  |
  | url     | http://... |
```

**6. Ошибка: "Alternative coverage < 80%"**

**Проблема:** Недостаточно альтернативных сценариев

**Решение:** Добавьте сценарии для:
- Validation errors
- Boundary conditions
- State transitions
- Integration failures
- Security scenarios

#### Отладка валидации

**Детальный режим:**

```bash
# Запуск с максимальным выводом
bash -x scripts/validate_dod.sh

# Проверка одного feature файла
./scripts/validate_dod.sh 2>&1 | grep "01_01_monitor_management"
```

**Просмотр полного отчета:**

```bash
# Открыть отчет в редакторе
cat /tmp/dod_validation_report.txt

# Или в редакторе
vim /tmp/dod_validation_report.txt
```

**Временное отключение pre-commit hook:**

```bash
# Одноразовый обход (не рекомендуется для production)
git commit --no-verify -m "WIP"

# Полное отключение hook (не рекомендуется)
rm .git/hooks/pre-commit
```

### Метрики качества

Автоматизированная валидация отслеживает следующие метрики:

| Метрика | Цель | Как измеряется |
|---------|------|----------------|
| **Structural Compliance** | 100% | Автоматическая проверка структуры |
| **Use Case Coverage** | 100% | @use_case на всех сценариях |
| **Error Code Format** | 100% | UPPERCASE_SNAKE_CASE формат |
| **Audit Logging Coverage** | 100% | Критические операции логируются |
| **Alternative Scenario Coverage** | >80% | (Alternative / Total) × 100% |
| **Gherkin Syntax** | 100% | godog parse без ошибок |

### Интеграция в процесс разработки

#### Рекомендуемый workflow:

1. **Разработка:**
   ```bash
   # Создайте/измените feature файл
   vim features/01_monitoring/01_01_new.feature
   ```

2. **Локальная валидация:**
   ```bash
   # Запустите валидацию перед коммитом
   ./scripts/validate_dod.sh
   ```

3. **Исправление ошибок:**
   ```bash
   # Просмотрите отчет
   cat /tmp/dod_validation_report.txt

   # Исправьте найденные проблемы
   vim features/01_monitoring/01_01_new.feature
   ```

4. **Коммит:**
   ```bash
   # Pre-commit hook запустится автоматически
   git add features/01_monitoring/01_01_new.feature
   git commit -m "Add new monitoring feature"
   ```

5. **Push и PR:**
   ```bash
   # CI/CD автоматически запустит валидацию
   git push origin feature-branch
   ```

6. **Review и merge:**
   - Проверьте комментарий с результатами валидации в PR
   - Убедитесь, что валидация прошла успешно
   - Merge только после прохождения всех проверок

### Обновление инструментов валидации

При изменении требований DOD:

1. Обновите `/Users/raul/go/src/monitor/features/DOD.md`
2. Обновите проверки в `/Users/raul/go/src/monitor/scripts/validate_dod.sh`
3. Протестируйте изменения:
   ```bash
   ./scripts/validate_dod.sh
   ```
4. Закоммитьте изменения вместе с обновленными feature файлами

---

---

## Канонические стандарты (Canonical Standards)

### Обзор

Следующие документы являются **единственным источником правды** (single source of truth) для cross-epic стандартов:

1. **ERROR_CODES.md** - Канонический реестр error codes
   - 60+ error codes across 8 domains
   - Обязательный формат: UPPERCASE_SNAKE_CASE
   - Все error codes в feature files ДОЛЖНЫ соответствовать этому реестру
   - Ссылка: `/Users/raul/go/src/monitor/features/ERROR_CODES.md`

2. **INTEGRATION_MATRIX.md** - Матрица интеграций между эпиками
   - 8 documented integration points
   - Event-driven flows и implementation notes
   - Cross-epic зависимости и контракты
   - Ссылка: `/Users/raul/go/src/monitor/features/INTEGRATION_MATRIX.md`

3. **RETRY_STRATEGIES.md** - Спецификации retry и recovery логики
   - 3 retry strategies: Exponential backoff, Linear, Fixed delay
   - Retryable vs non-retryable error codes
   - Dead letter queue handling
   - Ссылка: `/Users/raul/go/src/monitor/features/RETRY_STRATEGIES.md`

### Использование канонических стандартов

**При создании новых сценариев:**
1. Проверьте ERROR_CODES.md перед созданием нового error code
2. Добавьте cross-epic ссылки из INTEGRATION_MATRIX.md
3. Используйте retry стратегии из RETRY_STRATEGIES.md

**При обновлении существующих сценариев:**
1. Замените нестандартные error codes на канонические
2. Добавьте integration комментарии для cross-epic операций
3. Уточните retry логику для integration failures

### Добавление новых стандартов

**Процесс:**
1. Обсудите изменение с командой
2. Добавьте/обновите запись в каноническом документе
3. Обновите все feature files для соответствия
4. Запустите `./scripts/validate_dod.sh` для проверки
5. Закоммитьте изменения вместе с feature files

---

## История изменений

| Версия | Дата | Изменения | Автор |
|--------|------|-----------|-------|
| 1.0 | 2026-03-07 | Initial version | Claude & User |
| 1.1 | 2026-03-09 | Добавлена секция автоматизированной валидации | Claude & User |
| 1.2 | 2026-03-10 | Добавлено правило именования сценариев: основные сценарии без буквы, альтернативные с буквой (a, b, c, ...) в конце названия | Claude & User |
| 1.3 | 2026-03-12 | Enhanced scenario classification rules (outcome-based, not parameter-based) | Claude & User |
| 1.4 | 2026-03-12 | P0-P1 fixes completed: Added canonical standards section, ERROR_CODES.md, INTEGRATION_MATRIX.md, RETRY_STRATEGIES.md references, removed Russian text, resolved business logic contradictions, added audit logging, specified retry logic | Claude & User |
