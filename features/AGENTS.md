# BDD Конвенции и Руководство для Агентов

> **Версия:** 1.0
> **Дата:** Март 2026
> **Для:** AI агентов и разработчиков

---

## 📚 Definition of Done (DoD)

👉 **Полные требования к feature файлам: [DOD.md](./DOD.md)**

В DOD.md вы найдете:
- Обязательные требования к структуре
- Типы сценариев (Happy Path, Alternative, Edge Cases)
- Checklist для альтернативных сценариев
- Стандарты для error codes, статусов, audit logging
- Метрики качества
- Примеры полного покрытия

---

## Структура директорий

```
features/
├── XX_epic_name/                    # Epic: XX_название_эпика
│   ├── XX_YY_user_story_name.feature    # User Story: XX_YY_название
```

## Нумерация

| Уровень | Формат | Пример |
|---------|--------|--------|
| Epic | `XX_epic_name` | `01_monitoring` |
| User Story | `XX_YY_user_story_name.feature` | `01_01_monitor_management.feature` |
| Use Case | `uc_XX_YY_ZZ` | `uc_01_01_01` |

## Epics

| # | Epic ID | Описание | Статус | Сервис |
|---|---------|----------|--------|--------|
| 1 | `01_monitoring` | Управление мониторами и выполнение проверок | DONE | monitor-service |
| 2 | `02_alerting` | Каналы, триггеры и доставка алертов | DONE | alert-service |
| 3 | `03_dashboard` | Отображение статусов и история | DONE | dashboard-service |
| 4 | `04_billing` | Подписки и платежи | DONE | billing-service |
| 5 | `05_security` | Аутентификация, авторизация, команды и роли | IN PROGRESS | auth-service |
| 6 | `06_reporting` | SLA отчёты и аналитика | TODO | — |
| 7 | `07_integrations` | Webhooks, API, внешние сервисы | DONE | integration-service |
| 8 | `08_maintenance` | Maintenance windows и настройки | DONE | monitor-service |
| 9 | `09_scheduler` | Scheduler Service: планирование проверок | TODO | — |
| 10 | `10_check_worker` | Check Worker: выполнение проверок | TODO | — |

## Формат Feature файла

```gherkin
@epic=01_monitoring
@user_story=01_01_monitor_management
# Description: CRUD операции для HTTP/HTTPS мониторов

Feature: Управление мониторами
  Как пользователь
  Я хочу управлять своими мониторами
  Чтобы отслеживать доступность моих сервисов

  @use_case=uc_01_01_01
  Scenario: Создание нового HTTP монитора
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name     | API Service            |
      | url      | https://api.example.com |
      | interval | 5 minutes               |
    Then монитор должен быть создан успешно
    And монитор должен иметь статус "UP"
```

## Ключевые слова Gherkin (английские)

- `Feature` - определяет тестируемую функцию
- `Scenario` - конкретный тестовый случай
- `Scenario Outline` - шаблон для data-driven тестов
- `Background` - общие шаги для всех сценариев
- `Examples` - данные для Scenario Outline
- `Given` - начальный контекст/предусловия
- `When` - действие/событие
- `Then` - ожидаемый результат
- `And` - синоним Given/When/Then
- `But` - альтернатива And

## Definition of Done (DoD) для Feature файлов

### Обязательные требования

- [ ] **Структура файла:**
  - [ ] Заголовок `Feature:` с описанием на русском
  - [ ] Теги `@epic={epic_num}_{epic_name}` и `@user_story={epic_num}_{us_num}_{us_name}`
  - [ ] Описание User Story в формате "Как... Я хочу... Чтобы..."
  - [ ] Каждая строка отделена пустой строкой для читаемости

- [ ] **Синтаксис Gherkin:**
  - [ ] Ключевые слова на английском: `Feature`, `Scenario`, `Given`, `When`, `Then`, `And`, `But`
  - [ ] Описания шагов на русском языке
  - [ ] Отступы в 2 пробела для вложенности
  - [ ] Пустые строки между сценариями
  - [ ] godog/cucumber парсит без ошибок

- [ ] **Идентификация сценариев:**
  - [ ] Комментарий `@use_case=uc_XX_YY_ZZ` перед каждым сценарием
  - [ ] Название сценария описывает суть (на русском)
  - [ ] Имена сценариев уникальны в пределах feature файла

- [ ] **Содержание:**
  - [ ] Сценарии не содержат противоречий (проверено cross-epic анализом)
  - [ ] Error codes в формате UPPERCASE_SNAKE_CASE (например, `MONITOR_LIMIT_REACHED`)
  - [ ] Статусы и состояния соответствуют единому источнику правды
  - [ ] Все критические альтернативные сценарии покрыты

- [ ] **API Response формат:**
  - [ ] Бекенд не возвращает отображаемый текст: только коды и данные
  - [ ] Error responses содержат `{ error: { code, message, details, request_id } }`
  - [ ] Успешные ответы содержат только данные (без localized сообщений)
  - [ ] Вся i18n на стороне frontend/mobile

### Типы сценариев

Каждый feature файл ДОЛЖЕН содержать следующие типы сценариев:

#### 1. Happy Path Scenarios (Основной сценарий)
**Цель:** Проверить базовую функциональность при идеальных условиях.

**Пример:**
```gherkin
  @use_case=uc_01_01_01
  Scenario: Создание нового HTTP монитора
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name     | API Service            |
      | url      | https://api.example.com |
      | interval | 5 minutes               |
    Then монитор создан успешно
    And монитор имеет статус "PENDING"
```

#### 2. Alternative Scenarios (Альтернативные сценарии)
**Цель:** Проверить обработку ошибок, граничные условия и edge cases.

**Категории альтернативных сценариев:**

**a) Validation Errors** — ошибки валидации ввода:
```gherkin
  @use_case=uc_01_01_12
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
  Scenario: Попытка превышения лимита мониторов
    Given пользователь имеет тариф "Free" с лимитом "25" мониторов
    And пользователь создал "25" мониторов
    When пользователь создаёт 26-й монитор
    Then возвращается ошибка "MONITOR_LIMIT_REACHED"
    And монитор не создан
```

**c) State Transitions** — переходы между состояниями:
```gherkin
  @use_case=uc_01_01_06
  Scenario: Приостановка активного монитора
    Given монитор "API Service" в статусе "UP"
    When пользователь приостанавливает монитор
    Then монитор переходит в статус "PAUSED"
    And новые проверки не планируются
```

**d) Edge Cases** — граничные условия:
```gherkin
  @use_case=uc_02_02_12
  Scenario: Алерт срабатывает точно при N consecutive failures
    Given правило алерта с consecutive_failures = "3"
    When монитор падает "3" раза подряд
    Then создан алерт со статусом "TRIGGERED"
```

**e) Boundary Conditions** — проверки на границах:
```gherkin
  @use_case=uc_02_02_13
  Scenario: Алерт не срабатывает при N-1 consecutive failures
    Given правило алерта с consecutive_failures = "3"
    When монитор падает "2" раза подряд
    Then алерт не создан
```

**f) Integration Failures** — сбои зависимостей:
```gherkin
  @use_case=uc_04_02_16
  Scenario: Webhook delivery failure and retry
    Given платежная система отправляет webhook
    When webhook не доставлен (network error)
    Then система регистрирует неудачную попытку
    And повторяет запрос через "5" минут
```

**g) Security Scenarios** — сценарии безопасности:
```gherkin
  @use_case=uc_05_01_22
  Scenario: Предотвращение фиксации сессии
    Given злоумышленник создает сессию с идентификатором "compromised"
    When пользователь авторизуется с существующей сессией
    Then идентификатор сессии изменен на новый
    And старый идентификатор недействителен
```

**h) Performance Edge Cases** — производительностные сценарии:
```gherkin
  @use_case=uc_03_03_20
  Scenario: Дросселирование высокочастотных обновлений
    Given пользователь имеет "100" мониторов
    And все мониторы обновляются каждую "1 second"
    When поступает более "50" обновлений в секунду
    Then обновления группируются в пакеты
    And UI обновляется не чаще чем "20 times per second"
```

### Checklist для альтернативных сценариев

Для каждого feature файла проверьте наличие:

#### Критические (Critical — должны быть):
- [ ] Все validation ошибки покрыты
- [ ] Граничные условия (boundary conditions) протестированы
- [ ] Основные state transitions покрыты
- [ ] Критичные integration failures обработаны
- [ ] Безопасные сценарии (authz, authn) протестированы

#### Важные (High — настоятельно рекомендуются):
- [ ] Edge cases для бизнес-логики
- [ ] Сценарии восстановления после ошибок (recovery)
- [ ] Конфликтные ситуации (race conditions, duplicates)
- [ ] Производительные лимиты (timeouts, limits)

#### Желательные (Medium — добавляются по возможности):
- [ ] Мало вероятные edge cases
- [ ] UX улучшающие сценарии (empty states, hints)
- [ ] Диагностические сценарии (logging, monitoring)

### Стандарты для элементов сценариев

#### Error Codes
- Формат: `UPPERCASE_SNAKE_CASE`
- Примеры: `MONITOR_LIMIT_REACHED`, `INSUFFICIENT_PERMISSIONS`, `INVALID_URL`
- Обязательны для всех ошибок валидации и бизнес-правил

#### Статусы сущностей
- Мониторы: `PENDING`, `UP`, `DOWN`, `DEGRADED`, `PAUSED`
- Алерты: `TRIGGERED`, `ACKNOWLEDGED`, `RESOLVED`, `MUTED`, `SUPPRESSED_BY_MAINTENANCE`
- Подписки: `ACTIVE`, `GRACE_PERIOD`, `SUSPENDED`, `CANCELED`
- Платежи: `PENDING`, `COMPLETED`, `FAILED`, `TIMEOUT`, `REFUNDED`

#### Аудит логирование
Критические действия ДОЛЖНЫ логироваться:
```
действие в аудит лог записано как "action_name"
And запись содержит:
  | user_id     |
  | timestamp   |
  | ip_address  |
  | action_type |
  | details     |
```

### Анти-паттерны (избегать)

❌ **Запрещено:**
- Hardcoded текст сообщений в Then (используйте error codes)
- Смешивание языков (And/И — только And)
- Отсутствие альтернативных сценариев для validation
- Противоречивые статусы/состояния между feature files
- Missing audit logging для критических действий
- Отсутствие boundary testing (N, N-1, N+1)

✅ **Рекомендуется:**
- Data tables для параметров вместо длинных списков
- Scenario Outline для повторяющихся сценариев с разными данными
- Background для общих шагов
- Теги для группировки сценариев (@critical, @integration, @security)

### Пример полного DoD для сценария

```gherkin
  @use_case=uc_01_01_01
  @critical
  @validation
  Scenario: Создание нового HTTP монитора
    Given пользователь авторизован
    And пользователь имеет тариф "Free" с лимитом "25" мониторов
    And пользователь создал "10" мониторов
    When пользователь создаёт монитор с параметрами:
      | name     | API Service            |
      | url      | https://api.example.com |
      | interval | 5 minutes               |
    Then монитор создан успешно
    And монитор имеет идентификатор "monitor_123"
    And монитор имеет статус "PENDING"
    And первая проверка запланирована через "5 minutes"
    And действие в аудит лог записано как "monitor_created"
    And запись содержит user_id, monitor_id, timestamp

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
    And ошибка содержит подробное описание проблемы
    And действие в аудит лог записано как "monitor_creation_failed"
```

### Метрики качества

Для каждого feature файла отслеживайте:

| Метрика | Цель | Как измерить |
|---------|------|--------------|
| Покрытие альтернативных сценариев | >80% | (Alternative / Total scenarios) |
| Критические сценарии покрыты | 100% | Все @critical tagged scenarios |
| Противоречия | 0 | Cross-epic validation |
| Синтаксические ошибки | 0 | `godog parse` success |

---

## 🚀 Краткий старт для агентов

### 1. Создание нового feature файла

```bash
# 1. Создайте директорию эпика (если нет)
mkdir -p features/01_monitoring

# 2. Создайте feature файл
touch features/01_monitoring/01_01_new_functionality.feature
```

### 2. Минимальный шаблон

```gherkin
@epic=01_monitoring
@user_story=01_01_new_functionality
# Description: Краткое описание

Feature: Новая функциональность
  Как пользователь
  Я хочу новую функцию
  Чтобы получить ценность

  @use_case=uc_01_01_01
  @critical
  Scenario: Основной сценарий успеха
    Given предусловие
    When действие
    Then результат
```

### 3. Проверка синтаксиса

```bash
godog parse features/01_monitoring/01_01_new_functionality.feature
```

---

## Полезные команды

```bash
# Проверка синтаксиса всех feature файлов
godog parse features/

# Запуск конкретного сценария
godog features/01_monitoring/01_01_monitor_management.feature --name "Создание нового HTTP монитора"

# Запуск всех сценариев с тегом @critical
godog features/ --tags @critical

# Проверка покрытия сценариев
godog features/ --format junit > coverage.xml
```
