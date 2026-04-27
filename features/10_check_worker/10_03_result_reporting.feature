@epic=10_check_worker
@user_story=10_03_result_reporting
# Description: Отправка результатов проверок в monitor-service: отчёты, повторы, очереди

Feature: Отчётность результатов
  Как check-worker
  Я хочу отправлять результаты проверок в monitor-service
  Чтобы результаты сохранялись и инициировали события изменения статуса

  # Integration: Worker отправляет результат через MonitorService.SubmitCheckResult
  # Monitor-service сохраняет результат и публикует StatusChangeEvent в RabbitMQ
  # See: Epic 01, uc_01_04_01
  @use_case=uc_10_03_01
  @critical
  Scenario: Отправка успешного результата проверки
    Given воркер выполнил проверку монитора "API Service"
    And результат проверки:
      | status        | UP                    |
      | status_code   | 200                   |
      | response_time | 150ms                 |
    When воркер отправляет результат в monitor-service
    Then monitor-service подтверждает получение
    And результат сохранён в базе данных
    And действие в аудит лог записано как "result_reported"
      | field         | value                   |
      | monitor_id    | <monitor_id>            |
      | worker_id     | <worker_id>             |
      | status        | UP                      |
      | status_code   | 200                     |
      | response_time | 150ms                   |
      | zone          | moscow                  |
      | timestamp     | <iso8601>               |

  @use_case=uc_10_03_02
  @critical
  Scenario: Отправка неудачного результата проверки
    Given воркер выполнил проверку монитора "API Service"
    And результат проверки:
      | status        | DOWN                   |
      | status_code   | 500                    |
      | error         | Internal Server Error  |
    When воркер отправляет результат в monitor-service
    Then monitor-service подтверждает получение
    And результат сохранён
    # Integration: monitor-service публикует StatusChangeEvent в RabbitMQ
    # See: Epic 02, uc_02_02_20

  @use_case=uc_10_03_03
  @critical
  Scenario: Отправка результата DEGRADED
    Given воркер выполнил проверку монитора "API Service"
    And результат проверки:
      | status        | DEGRADED    |
      | status_code   | 200         |
      | response_time | 6000ms      |
      | reason        | SLOW_RESPONSE|
    When воркер отправляет результат в monitor-service
    Then monitor-service подтверждает получение
    And результат сохранён со статусом "DEGRADED"

  @use_case=uc_10_03_04
  @critical
  Scenario: Отправка результата с метриками response time
    Given воркер выполнил проверку монитора
    And результат содержит метрики:
      | dns_lookup_ms    | 15    |
      | tcp_connect_ms   | 30    |
      | tls_handshake_ms | 45    |
      | ttfb_ms          | 120   |
      | total_ms         | 250   |
    When воркер отправляет результат в monitor-service
    Then все метрики response time сохранены
    And данные доступны для расчёта P50/P95/P99

  @use_case=uc_10_03_05
  @critical
  Scenario: Пакетная отправка нескольких результатов
    Given воркер выполнил "10" проверок
    And результаты готовы к отправке
    When воркер отправляет пакет результатов в monitor-service
    Then monitor-service подтверждает получение всех "10" результатов
    And все результаты сохранены

  @use_case=uc_10_03_01a
  @critical
  @validation
  Scenario: Отправка результата для неизвестного монитора
    Given воркер выполнил проверку для monitor_id "unknown-id"
    When воркер отправляет результат в monitor-service
    Then monitor-service возвращает ошибку "MONITOR_NOT_FOUND"
    And результат отброшен

  @use_case=uc_10_03_01b
  @critical
  @integration
  Scenario: Ошибка отправки результата: monitor-service Недоступен b
    Given воркер выполнил проверку монитора "API Service"
    And результат готов к отправке
    And monitor-service недоступен
    When воркер пытается отправить результат
    Then результат помещён в локальную очередь
    And максимальный размер очереди "1000" результатов
    And повторная попытка через "5 seconds" с exponential backoff
    And действие в аудит лог записано как "result_queued"
      | field       | value         |
      | monitor_id  | <monitor_id>  |
      | queue_size  | <number>      |
      | retry_in    | 5 seconds     |
      | timestamp   | <iso8601>     |

  @use_case=uc_10_03_01c
  @critical
  @validation
  Scenario: Отправка дублирующего результата
    Given воркер отправил результат проверки с check_id "check-123"
    When воркер повторно отправляет результат с тем же check_id
    Then monitor-service игнорирует дубликат
    And ошибка не возвращена
    And действие в аудит лог записано как "duplicate_result_ignored"
      | field     | value      |
      | check_id  | check-123  |
      | timestamp | <iso8601>  |

  @use_case=uc_10_03_01d
  @critical
  @validation
  Scenario: Отправка результата с обязательными полями
    Given воркер подготовил результат без статуса
    When воркер пытается отправить результат
    Then monitor-service возвращает ошибку "MISSING_STATUS"
    And результат не сохранён

  @use_case=uc_10_03_06
  @critical
  @recovery
  Scenario: Повторная отправка после восстановления monitor-service
    Given в локальной очереди "15" результатов
    And monitor-service снова доступен
    When воркер начинает отправку из очереди
    Then все "15" результатов отправлены
    And результаты отправлены в порядке FIFO
    And локальная очередь очищена
    And действие в аудит лог записано как "queue_flushed"
      | field        | value    |
      | flushed_count| 15       |
      | duration_ms  | <number> |
      | timestamp    | <iso8601>|

  @use_case=uc_10_03_07
  @critical
  @boundary
  Scenario: Переполнение локальной очереди результатов
    Given в локальной очереди "1000" результатов
    And monitor-service недоступен
    And максимальный размер очереди "1000"
    When воркер получает новый результат
    Then самый старый результат удалён из очереди
    And новый результат добавлен
    And действие в аудит лог записано как "queue_overflow"
      | field        | value    |
      | dropped_count| 1        |
      | queue_size   | 1000     |
      | timestamp    | <iso8601>|

  @use_case=uc_10_03_01e
  @critical
  @validation
  Scenario: Отправка результата для удалённого монитора
    Given воркер выполнил проверку для монитора
    And монитор был удалён между назначением и выполнением
    When воркер отправляет результат в monitor-service
    Then monitor-service возвращает ошибку "MONITOR_NOT_FOUND"
    And результат отброшен
    And действие в аудит лог записано как "result_for_deleted_monitor"
      | field       | value         |
      | monitor_id  | <monitor_id>  |
      | worker_id   | <worker_id>   |
      | timestamp   | <iso8601>     |

  @use_case=uc_10_03_08
  @critical
  Scenario: Отправка результата с информацией о зоне
    Given воркер "worker-msk-01" в зоне "moscow"
    And воркер выполнил проверку
    When воркер отправляет результат в monitor-service
    Then результат содержит зону "moscow"
    And monitor-service сохраняет зону для multi-location анализа
    # Integration: v2.0 multi-location monitoring
    # See: PRD v2.0, Multi-location monitoring
