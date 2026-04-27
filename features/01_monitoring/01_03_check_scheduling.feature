@epic=01_monitoring
@user_story=01_03_check_scheduling
# Description: Планирование и управление расписанием проверок
#
# Architecture Note:
#   Начиная с v1.0, планирование перенесено в scheduler-service:
#   - scheduler-service (Epic 09) управляет расписанием и распределением
#   - check-worker (Epic 10) получает назначения и выполняет проверки
#   - monitor-service хранит конфигурацию мониторов (интервалы, рабочие часы)
#
#   Бизнес-правила расписания (интервалы, рабочие часы) описаны в этом файле.
#   Операционное планирование описано в Epic 09 (09_02_check_distribution).

Feature: Планирование проверок
  Как система
  Я хочу планировать проверки для мониторов
  Чтобы регулярно проверять доступность сервисов

  @use_case=uc_01_03_01
  @critical
  Scenario: Планирование проверки для монитора
    Given монитор "API Service" с интервалом "5 minutes"
    When планируется проверка
    Then проверка запланирована
    And получен ID проверки
    And время планирования зафиксировано

  @use_case=uc_01_03_02
  @critical
  Scenario: Отмена проверок для монитора
    Given монитор "API Service" с запланированными проверками
    When отменяются все проверки для монитора
    Then проверки больше не планируются
    And статус расписания "not scheduled"

  @use_case=uc_01_03_03
  @critical
  Scenario: Получение статуса расписания
    Given монитор "API Service" с интервалом "5 minutes"
    And проверки запланированы
    When запрашивается статус расписания
    Then получен статус "scheduled"
    And получено время следующей проверки
    And интервал равен "5 minutes"

  @use_case=uc_01_03_04
  @critical
  Scenario: Планирование с интервалом 30 секунд
    Given монитор "Critical Service" с интервалом "30 seconds"
    When планируется проверка
    Then проверка запланирована
    And следующая проверка через "30 seconds"

  @use_case=uc_03_05
  @critical
  Scenario: Планирование с интервалом 1 минута
    Given монитор "Regular Service" с интервалом "1 minute"
    When планируется проверка
    Then проверка запланирована
    And следующая проверка через "1 minute"

  @use_case=uc_01_03_06
  @critical
  Scenario: Планирование с интервалом 10 минут
    Given монитор "Low Priority Service" с интервалом "10 minutes"
    When планируется проверка
    Then проверка запланирована
    And следующая проверка через "10 minutes"

  @use_case=uc_01_03_07
  @critical
  Scenario: Планирование с интервалом 30 минут
    Given монитор "Background Service" с интервалом "30 minutes"
    When планируется проверка
    Then проверка запланирована
    And следующая проверка через "30 minutes"

  @use_case=uc_01_03_08
  @critical
  Scenario: Планирование с рабочими часами
    Given монитор "Business Hours API"
    And интервал "5 minutes"
    And рабочие часы "09:00-18:00" по будням
    And текущее время "2026-03-08T09:00:00Z" (понедельник)
    When планируется проверка
    Then проверка запланирована
    And следующая проверка запланирована

  @use_case=uc_01_03_09
  @critical
  Scenario: Пропуск планирования вне рабочих часов
    Given монитор "Business Hours API"
    And интервал "5 minutes"
    And рабочие часы "09:00-18:00" по будням
    And текущее время "2026-03-08T20:00:00Z" (понедельник)
    When планируется проверка
    Then проверка не запланирована
    And следующая проверка на "2026-03-09T09:00:00Z" (вторник)

  @use_case=uc_01_03_10
  @critical
  Scenario: Пропуск планирования в выходные
    Given монитор "Business Hours API"
    And интервал "5 minutes"
    And рабочие часы "09:00-18:00" по будням
    And текущее время "2026-03-10T14:00:00Z" (воскресенье)
    When планируется проверка
    Then проверка не запланирована
    And следующая проверка на "2026-03-11T09:00:00Z" (понедельник)

  @use_case=uc_01_03_01a
  @critical
  Scenario: Планирование для приостановленного монитора
    Given монитор "API Service" в статусе "PAUSED"
    When планируется проверка
    Then проверка не запланирована
    And статус "not scheduled"

  @use_case=uc_01_03_01b
  @critical
  Scenario: Попытка планирования с интервалом менее 30 секунд b
    Given монитор "Critical Service"
    When планируется проверка с интервалом "10 seconds"
    Then система возвращает ошибку "INVALID_INTERVAL"
    And проверка не запланирована

  @use_case=uc_01_03_01c
  @critical
  Scenario: Планирование с интервалом более 1 часа
    Given монитор "Background Service"
    When планируется проверка с интервалом "2 hours"
    Then система возвращает ошибку "INVALID_INTERVAL"

  @use_case=uc_01_03_01d
  @critical
  Scenario: Планирование для удаленного монитора
    Given монитор был удален
    When планируется проверка для монитора
    Then система возвращает ошибку "MONITOR_NOT_FOUND"
    And проверка не запланирована

  @use_case=uc_01_03_03a
  @critical
  Scenario: Восстановление расписания после перезапуска scheduler'а
    Given scheduler worker был перезапущен
    And существовали запланированные проверки
    When worker восстанавливает состояние
    Then все расписания восстановлены
    And проверки продолжают выполняться
    And ошибка содержит код "SCHEDULE_RECOVERED"
    And количество восстановленных расписаний записано в лог

  @use_case=uc_01_03_11
  Scenario: Планирование с timezone пользователя
    Given монитор "Business Hours API"
    And рабочие часы "09:00-18:00" по будням
    And часовой пояс пользователя "Europe/Moscow"
    And текущее время "2026-03-08T09:00:00Z" по UTC
    When планируется проверка
    Then проверка запланирована с учетом часового пояса

  @use_case=uc_01_03_03b
  @critical
  @recovery
  Scenario: Восстановление scheduler worker'а после краша
    Given scheduler worker упал
    And существовали запланированные проверки
    When worker перезапускается
    Then все расписания восстановлены из персистентного хранилища
    And проверки продолжают выполняться по расписанию
    And пропущенные проверки выполняются
    And ошибка содержит код "SCHEDULE_RESTORED"
    And время простоя worker'а записано в лог

  @use_case=uc_01_03_12
  @critical
  @performance
  Scenario: Планирование проверок для 100+ мониторов одновременно
    Given система имеет "100" активных мониторов
    And все мониторы имеют интервал "1 minute"
    When выполняется массовое планирование проверок
    Then все проверки запланированы
    And время планирования не превышает "5 seconds"
    And ошибка содержит код "BATCH_SCHEDULE_COMPLETE"
    And количество запланированных проверок записано в лог

  @use_case=uc_01_03_01e
  @critical
  @integration
  Scenario: Ошибка очереди сообщений для планирования e
    Given монитор "API Service" с интервалом "5 minutes"
    And очередь сообщений недоступна
    When планируется проверка
    Then проверка не запланирована
    And ошибка содержит код "MESSAGE_QUEUE_UNAVAILABLE"
    And попытка повторяется через "30 seconds"
