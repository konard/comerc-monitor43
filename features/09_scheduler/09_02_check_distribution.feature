@epic=09_scheduler
@user_story=09_02_check_distribution
# Description: Планирование проверок и их распределение по воркерам

Feature: Распределение проверок
  Как scheduler-service
  Я хочу планировать проверки мониторов и распределять их по воркерам
  Чтобы обеспечить регулярную проверку всех сервисов с учётом зон и приоритетов

  # Integration: Scheduler запрашивает активные мониторы из monitor-service
  # See: Epic 01, uc_01_01_03
  @use_case=uc_09_02_01
  @critical
  Scenario: Планирование проверки для монитора
    Given монитор "API Service" с интервалом "5 minutes"
    And монитор в статусе "UP"
    And зарегистрирован воркер "worker-msk-01" в зоне "moscow" со статусом "IDLE"
    When наступает время следующей проверки
    Then scheduler-service создаёт запланированную проверку
    And проверка назначена воркеру "worker-msk-01"
    And приоритет проверки "NORMAL"
    And время планирования зафиксировано
    And действие в аудит лог записано как "check_scheduled"
      | field       | value          |
      | monitor_id  | <monitor_id>   |
      | worker_id   | <worker_id>    |
      | scheduled_at| <iso8601>      |
      | priority    | NORMAL         |

  @use_case=uc_09_02_02
  @critical
  Scenario: Распределение проверки к доступному воркеру
    Given монитор "API Service" с интервалом "5 minutes"
    And зарегистрированы воркеры:
      | name          | zone   | status |
      | worker-msk-01 | moscow | IDLE   |
      | worker-msk-02 | moscow | BUSY   |
      | worker-spb-01 | spb    | IDLE   |
    When scheduler-service распределяет проверку
    Then проверка назначена воркеру "worker-msk-01" или "worker-spb-01"
    And проверка НЕ назначена воркеру "worker-msk-02"
    And выбран воркер с наименьшей нагрузкой

  @use_case=uc_09_02_03
  @critical
  Scenario: Получение запланированных проверок за временной интервал
    Given существует "10" запланированных проверок за следующие "10 minutes"
    When запрашиваются проверки за "10 minutes"
    Then получен список из "10" проверок
    And каждая проверка содержит monitor_id, worker_id, scheduled_at, priority

  @use_case=uc_09_02_04
  @critical
  Scenario: Получение следующей проверки для монитора
    Given монитор "API Service" с интервалом "5 minutes"
    And проверка запланирована
    When запрашивается следующая проверка для монитора
    Then получена информация о запланированной проверке
    And указано время выполнения

  # Integration: Ручной триггер проверки от пользователя через monitor-service
  # See: Epic 01, uc_01_01_08 (TriggerCheck)
  @use_case=uc_09_02_05
  @critical
  Scenario: Ручной запуск проверки
    Given монитор "API Service" существует
    And зарегистрирован воркер "worker-msk-01" со статусом "IDLE"
    When поступает запрос на ручной запуск проверки
    Then проверка создана с приоритетом "HIGH"
    And проверка назначена воркеру "worker-msk-01"
    And время выполнения = "now"
    And действие в аудит лог записано как "check_manually_triggered"
      | field       | value          |
      | monitor_id  | <monitor_id>   |
      | worker_id   | <worker_id>    |
      | priority    | HIGH           |
      | timestamp   | <iso8601>      |

  @use_case=uc_09_02_06
  @critical
  Scenario: Завершение запланированной проверки
    Given проверка назначена воркеру "worker-msk-01"
    And статус проверки "IN_PROGRESS"
    When воркер сообщает о завершении проверки
    Then статус проверки обновлён на "COMPLETED"
    And время завершения зафиксировано
    And воркер "worker-msk-01" снова доступен для новых проверок

  @use_case=uc_09_02_07
  @critical
  Scenario: Массовое планирование проверок по расписанию
    Given существует "100" активных мониторов
    And интервалы проверок от "30 seconds" до "30 minutes"
    And зарегистрированы "5" воркеров со статусом "IDLE"
    When scheduler-service выполняет цикл планирования
    Then все просроченные проверки запланированы
    And проверки равномерно распределены между воркерами
    And время планирования не превышает "2 seconds"
    And действие в аудит лог записано как "batch_scheduling_completed"
      | field          | value    |
      | checks_planned | <count>  |
      | workers_used   | 5        |
      | duration_ms    | <number> |

  @use_case=uc_09_02_08
  @critical
  @recovery
  Scenario: Переназначение проверок при потере воркера
    Given воркер "worker-msk-01" выполняет "5" проверок
    And воркер помечен как "OFFLINE"
    When scheduler-service обнаруживает offline воркер
    Then все "5" проверок перенесены в очередь
    And проверки назначены другим доступным воркерам
    And действие в аудит лог записано как "checks_reassigned"
      | field            | value           |
      | offline_worker   | worker-msk-01   |
      | reassigned_count | 5               |
      | new_workers      | <worker_ids>    |
      | timestamp        | <iso8601>       |

  @use_case=uc_09_02_09
  @critical
  Scenario: Приоритетное планирование критических мониторов
    Given монитор "Payment Gateway" с интервалом "30 seconds"
    And монитор "Blog Feed" с интервалом "5 minutes"
    And зарегистрирован "1" воркер со статусом "IDLE"
    When поступают обе проверки одновременно
    Then проверка "Payment Gateway" назначена первой
    And приоритет "Payment Gateway" = "CRITICAL"
    And приоритет "Blog Feed" = "NORMAL"

  @use_case=uc_09_02_10
  @critical
  Scenario: Распределение с учётом зоны монитора
    Given монитор "API Service" настроен на зону "moscow"
    And зарегистрированы воркеры:
      | name          | zone   | status |
      | worker-msk-01 | moscow | IDLE   |
      | worker-spb-01 | spb    | IDLE   |
    When scheduler-service распределяет проверку
    Then проверка назначена воркеру "worker-msk-01"
    And выбор обусловлен совпадением зоны

  # Integration: Проверки для приостановленного монитора не планируются
  # See: Epic 01, uc_01_01_06 (PauseMonitor)
  @use_case=uc_09_02_01a
  @critical
  @validation
  Scenario: Планирование проверки для приостановленного монитора
    Given монитор "API Service" в статусе "PAUSED"
    When наступает время следующей проверки
    Then проверка не планируется
    And статус расписания "not scheduled"

  @use_case=uc_09_02_01b
  @critical
  @validation
  Scenario: Планирование проверки для удалённого монитора
    Given монитор был удалён
    When наступает время следующей проверки
    Then система возвращает ошибку "MONITOR_NOT_FOUND"
    And проверка не планируется

  @use_case=uc_09_02_02a
  @critical
  @integration
  Scenario: Распределение проверки когда нет доступных воркеров
    Given монитор "API Service" требует проверку
    And нет зарегистрированных воркеров со статусом "IDLE"
    When scheduler-service пытается распределить проверку
    Then проверка помещена в очередь ожидания
    And статус проверки "PENDING"
    And ошибка содержит код "NO_WORKERS_AVAILABLE"
    And действие в аудит лог записано как "check_queued_no_workers"
      | field       | value         |
      | monitor_id  | <monitor_id>  |
      | reason      | no_idle_workers |
      | timestamp   | <iso8601>     |

  @use_case=uc_09_02_06a
  @critical
  @validation
  Scenario: Ошибка завершения проверки воркером a
    Given проверка назначена воркеру "worker-msk-01"
    And статус проверки "IN_PROGRESS"
    When воркер сообщает об ошибке выполнения
    Then статус проверки обновлён на "FAILED"
    And ошибка записана
    And воркер "worker-msk-01" снова доступен для новых проверок

  @use_case=uc_09_02_05a
  @critical
  @validation
  Scenario: Ручной запуск проверки для неизвестного монитора
    Given монитор с идентификатором "unknown-id" не существует
    When поступает запрос на ручной запуск проверки
    Then система возвращает ошибку "MONITOR_NOT_FOUND"

  @use_case=uc_09_02_11
  @critical
  @boundary
  Scenario: Предотвращение дублирования проверки
    Given монитор "API Service" имеет запланированную проверку в статусе "IN_PROGRESS"
    When наступает время следующей регулярной проверки
    Then новая проверка не создаётся
    And ожидается завершение текущей проверки

  @use_case=uc_09_02_12
  @critical
  @performance
  Scenario: Планирование 1000+ проверок в минуту
    Given существует "500" мониторов с интервалом "1 minute"
    And зарегистрированы "10" воркеров
    When scheduler-service выполняет цикл планирования
    Then все проверки запланированы
    And время планирования не превышает "5 seconds"
    And нагрузка распределена равномерно между воркерами

  @use_case=uc_09_02_13
  @critical
  @recovery
  Scenario: Восстановление расписания после перезапуска scheduler-service
    Given scheduler-service был перезапущен
    And существовали запланированные проверки
    When scheduler-service восстанавливает состояние
    Then все расписания восстановлены из базы данных
    And незавершённые проверки переназначены
    And новые проверки продолжают планироваться
    And действие в аудит лог записано как "scheduler_recovered"
      | field               | value    |
      | recovered_schedules | <count>  |
      | reassigned_checks   | <count>  |
      | timestamp           | <iso8601>|

  @use_case=uc_09_02_14
  @critical
  @recovery
  Scenario: Обработка задержки выполнения проверки
    Given проверка запланирована на "2026-03-29T10:00:00Z"
    And текущее время "2026-03-29T10:05:00Z"
    And проверка всё ещё в статусе "PENDING"
    When scheduler-service обнаруживает просроченную проверку
    Then проверка переназначена другому воркеру
    And приоритет повышен до "HIGH"
    And действие в аудит лог записано как "check_overdue_reassigned"
      | field           | value         |
      | check_id        | <check_id>    |
      | original_time   | 2026-03-29T10:00:00Z |
      | new_priority    | HIGH          |
      | timestamp       | <iso8601>     |

  @use_case=uc_09_02_15
  @critical
  @integration
  Scenario: Интеграция с maintenance windows
    Given монитор "API Service" имеет активное окно обслуживания
    # Integration: Maintenance windows из Epic 08
    # See: Epic 08, uc_08_01_26
    When наступает время проверки
    Then проверка не планируется
    And статус "CHECK_SKIPPED_MAINTENANCE"

  @use_case=uc_09_02_16
  @critical
  @validation
  Scenario: Запрос проверок с невалидным временным интервалом
    When запрашиваются проверки с интервалом "from" позже "to"
    Then система возвращает ошибку "INVALID_TIME_RANGE"
