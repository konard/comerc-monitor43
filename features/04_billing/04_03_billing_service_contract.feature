@epic=04_billing
@user_story=04_03_billing_service_contract
# Description: Исполняемый контракт billing-service gRPC API (мигрирован из backend/billing-service/test)

Feature: Контракт billing-service gRPC API
  Как разработчик системы мониторинга
  Я хочу иметь исполняемый контракт gRPC API биллинга
  Чтобы регрессии в эндпоинтах ловились BDD-сьютой

  @use_case=uc_04_03_01
  @critical
  Scenario: Получение списка доступных тарифных планов
    Given пользователь является аутентифицированным
    When пользователь запрашивает список тарифных планов
    Then получает список планов содержащий минимум 4 плана
    And каждый план содержит id, name, description, price_kopeks
    And планы включают Free, Starter, Professional, Business

  @use_case=uc_04_03_02
  @critical
  Scenario: Получение информации о текущей подписке
    Given пользователь является аутентифицированным
    And пользователь имеет подписку "Free"
    When пользователь запрашивает информацию о подписке
    Then получает статус подписки "ACTIVE"
    And получает id тарифного плана "TIER_FREE"
    And получает дату истечения подписки

  @use_case=uc_04_03_03
  @critical
  Scenario: Создание checkout сессии для оплаты подписки
    Given пользователь является аутентифицированным
    And пользователь имеет подписку "Free"
    When пользователь создает checkout для плана "Starter"
    Then получает checkout_url для оплаты
    And получает payment_id
    And в системе создана подписка со статусом "PENDING"

  @use_case=uc_04_03_04
  @critical
  Scenario: Получение истории платежей
    Given пользователь является аутентифицированным
    And пользователь имеет 2 платежа
    When пользователь запрашивает историю платежей
    Then получает список из 2 платежей
    And каждый платеж содержит id, provider, status, amount_kopeks
    And получает общее количество платежей

  @use_case=uc_04_03_05
  @critical
  Scenario: Отмена активной подписки
    Given пользователь является аутентифицированным
    And пользователь имеет активную подписку "Starter"
    When пользователь отменяет подписку с причиной "too expensive"
    Then подписка получает статус "CANCELED"
    And подписка имеет дату отмены

  @use_case=uc_04_03_06
  @critical
  Scenario: Обработка webhook от платежной системы при успешной оплате
    Given существует подписка со статусом "PENDING"
    And существует платеж со статусом "PENDING"
    When приходит webhook от YOOKASSA с событием "payment.succeeded"
    Then платеж получает статус "SUCCESS"
    And подписка активируется со статусом "ACTIVE"

  @use_case=uc_04_03_07
  @critical
  Scenario: Обработка webhook от платежной системы при неудачной оплате
    Given существует подписка со статусом "PENDING"
    And существует платеж со статусом "PENDING"
    When приходит webhook от YOOKASSA с событием "payment.failed"
    Then платеж получает статус "FAILED"
    And подписка отменяется со статусом "CANCELED"

  @use_case=uc_04_03_08
  Scenario: Проверка лимитов тарифного плана Starter
    Given пользователь является аутентифицированным
    When пользователь проверяет лимиты своего плана
    Then получает max_monitors = 25
    And получает min_check_interval_seconds = 60
    And получает max_alerts_per_day = 100

  @use_case=uc_04_03_09
  Scenario: Валидация запроса на создание checkout с неверным plan_id
    Given пользователь является аутентифицированным
    When пользователь создает checkout с неверным plan_id "INVALID"
    Then получает ошибку "NotFound"
    And ошибка содержит описание проблемы

  @use_case=uc_04_03_10
  Scenario: Авторизация — пользователь не может читать чужую подписку
    Given пользователь "user1" является аутентифицированным
    When пользователь "user1" запрашивает подписку пользователя "user2"
    Then получает ошибку "PermissionDenied"
