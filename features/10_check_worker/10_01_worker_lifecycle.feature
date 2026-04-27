@epic=10_check_worker
@user_story=10_01_worker_lifecycle
# Description: Жизненный цикл check-worker'а: запуск, регистрация, heartbeat, отключение

Feature: Жизненный цикл воркера
  Как check-worker
  Я хочу управлять своим подключением к scheduler-service
  Чтобы получать задания и корректно завершать работу

  # Integration: Worker регистрируется через SchedulerService.RegisterWorker
  # See: Epic 09, uc_09_01_01
  @use_case=uc_10_01_01
  @critical
  Scenario: Запуск и регистрация воркера
    Given check-worker сконфигурирован:
      | scheduler_address | scheduler-service:9094 |
      | monitor_address   | monitor-service:9091   |
      | worker_name       | worker-msk-01          |
      | zone              | moscow                 |
    And scheduler-service доступен
    When check-worker запускается
    Then воркер успешно подключается к scheduler-service
    And воркер зарегистрирован и получает идентификатор
    And воркер переходит в состояние "idle"
    And начинается цикл heartbeat
    And воркер готов к получению проверок
    And действие в аудит лог записано как "worker_started"
      | field             | value              |
      | worker_id         | <worker_id>        |
      | worker_name       | worker-msk-01      |
      | zone              | moscow             |
      | scheduler_address | scheduler-service:9094 |
      | timestamp         | <iso8601>          |

  # Integration: Worker отправляет heartbeat через SchedulerService.WorkerHeartbeat
  # See: Epic 09, uc_09_01_02
  @use_case=uc_10_01_02
  @critical
  Scenario: Периодический heartbeat
    Given воркер "worker-msk-01" зарегистрирован и работает
    And heartbeat интервал = "10 seconds"
    When проходит "10 seconds" с последнего heartbeat
    Then воркер отправляет heartbeat с метриками:
      | status                | IDLE    |
      | checks_completed      | <count> |
      | checks_failed         | <count> |
      | avg_check_duration_ms | <ms>    |
    And scheduler-service подтверждает heartbeat

  # Integration: Worker отключается через SchedulerService.UnregisterWorker
  # See: Epic 09, uc_09_01_03
  @use_case=uc_10_01_03
  @critical
  Scenario: Graceful shutdown
    Given воркер "worker-msk-01" зарегистрирован и работает
    And воркер выполняет "2" проверки
    When поступает сигнал завершения (SIGTERM)
    Then воркер прекращает приём новых проверок
    And воркер дожидается завершения текущих проверок
    And воркер отправляет результаты в monitor-service
    And воркер отправляет запрос на отключение в scheduler-service
    And воркер завершает работу
    And действие в аудит лог записано как "worker_shutdown"
      | field             | value         |
      | worker_id         | <worker_id>   |
      | worker_name       | worker-msk-01 |
      | completed_checks  | 2             |
      | reason            | graceful      |
      | timestamp         | <iso8601>     |

  @use_case=uc_10_01_04
  @critical
  Scenario: Воркер получает назначенную проверку
    Given воркер "worker-msk-01" в состоянии "idle"
    And scheduler-service назначил проверку для монитора "API Service"
    When воркер получает назначение
    Then воркер переходит в состояние "busy"
    And воркер начинает выполнение проверки

  @use_case=uc_10_01_05
  @critical
  Scenario: Воркер завершает проверку и возвращается в idle
    Given воркер "worker-msk-01" в состоянии "busy"
    And выполняется проверка монитора "API Service"
    When проверка завершена
    And результат отправлен в monitor-service
    Then воркер уведомляет scheduler-service о завершении
    And воркер переходит в состояние "idle"
    And воркер готов к следующей проверке

  @use_case=uc_10_01_06
  @critical
  Scenario: Воркер обрабатывает несколько параллельных проверок
    Given воркер "worker-msk-01" сконфигурирован с max_concurrent = "10"
    And scheduler-service назначил "5" проверок
    When воркер получает назначения
    Then все "5" проверок выполняются параллельно
    And воркер не принимает более "10" одновременных проверок
    And статус воркера "BUSY"

  @use_case=uc_10_01_01a
  @critical
  @integration
  Scenario: Scheduler-service недоступен при запуске
    Given check-worker сконфигурирован для подключения к scheduler-service
    And scheduler-service недоступен
    When check-worker запускается
    Then воркер повторяет попытку подключения через "5 seconds"
    And воркер использует exponential backoff
    And максимальное количество попыток "10"
    And действие в аудит лог записано как "worker_connection_retry"
      | field       | value         |
      | attempt     | <number>      |
      | next_retry  | <duration>    |
      | timestamp   | <iso8601>     |

  @use_case=uc_10_01_02a
  @critical
  @integration
  Scenario: Scheduler-service становится недоступным во время работы
    Given воркер "worker-msk-01" зарегистрирован и работает
    And scheduler-service становится недоступен
    When воркер не может отправить heartbeat
    Then воркер продолжает выполнение текущих проверок
    And результаты сохраняются в локальную очередь
    And воркер повторяет попытку heartbeat через "5 seconds"
    And действие в аудит лог записано как "scheduler_unavailable"
      | field       | value       |
      | worker_id   | <worker_id> |
      | retry_in    | 5 seconds   |
      | timestamp   | <iso8601>   |

  # Integration: Scheduler восстанавливает воркера через heartbeat
  # See: Epic 09, uc_09_01_09
  @use_case=uc_10_01_07
  @critical
  @recovery
  Scenario: Переподключение после восстановления scheduler-service
    Given воркер "worker-msk-01" работал
    And scheduler-service был недоступен
    And прошло "30 seconds" без heartbeat
    When scheduler-service снова доступен
    Then воркер отправляет heartbeat
    And воркер re-регистрируется если необходимо
    And локальная очередь результатов отправлена в monitor-service
    And действие в аудит лог записано как "worker_reconnected"
      | field            | value       |
      | worker_id        | <worker_id> |
      | downtime         | <duration>  |
      | queued_results   | <count>     |
      | timestamp        | <iso8601>   |

  @use_case=uc_10_01_03a
  @critical
  @integration
  Scenario: Graceful shutdown при недоступном scheduler-service
    Given воркер "worker-msk-01" работает
    And scheduler-service недоступен
    When поступает сигнал завершения (SIGTERM)
    Then воркер дожидается завершения текущих проверок
    And результаты отправлены в monitor-service
    And запрос на отключение поставлен в очередь
    And воркер завершает работу с задержкой
    And действие в аудит лог записано как "worker_shutdown_degraded"
      | field       | value       |
      | worker_id   | <worker_id> |
      | reason      | graceful_scheduler_unavailable |
      | timestamp   | <iso8601>   |

  @use_case=uc_10_01_01b
  @critical
  @validation
  Scenario: Ошибка запуска с невалидной конфигурацией b
    Given check-worker сконфигурирован:
      | scheduler_address |              |
      | worker_name       | worker-msk-01|
    When check-worker запускается
    Then воркер возвращает ошибку конфигурации
    And воркер не запускается
    And действие в аудит лог записано как "worker_config_error"
      | field | value               |
      | error | SCHEDULER_ADDRESS_REQUIRED |
      | timestamp | <iso8601>       |

  @use_case=uc_10_01_08
  @critical
  @recovery
  Scenario: Forceful shutdown по SIGKILL
    Given воркер "worker-msk-01" работает и выполняет "3" проверки
    When поступает сигнал SIGKILL
    Then текущие проверки теряются
    And scheduler-service обнаруживает потерю по heartbeat timeout
    And проверки переназначены другим воркерам
    # Integration: Scheduler handles offline worker
    # See: Epic 09, uc_09_01_08

  @use_case=uc_10_01_09
  @critical
  @performance
  Scenario: Обработка 100+ параллельных проверок
    Given воркер "worker-msk-01" сконфигурирован с max_concurrent = "100"
    And scheduler-service назначил "100" проверок
    When воркер обрабатывает все проверки
    Then все "100" проверок выполнены
    And ни одна проверка не потеряна
    And среднее время выполнения проверки < "5 seconds"
