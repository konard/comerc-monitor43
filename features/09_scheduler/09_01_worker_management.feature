@epic=09_scheduler
@user_story=09_01_worker_management
# Description: Управление жизненным циклом check-worker'ов: регистрация, heartbeat, отключение

Feature: Управление воркерами
  Как scheduler-service
  Я хочу управлять регистрацией и жизненным циклом check-worker'ов
  Чтобы распределять проверки между доступными агентами в разных зонах

  @use_case=uc_09_01_01
  @critical
  Scenario: Регистрация нового воркера
    Given check-worker запускается с параметрами:
      | name | worker-msk-01     |
      | zone | moscow            |
    When воркер регистрируется в scheduler-service
    Then воркер зарегистрирован успешно
    And воркер имеет статус "IDLE"
    And воркер имеет идентификатор
    And воркер привязан к зоне "moscow"
    And время регистрации зафиксировано
    And действие в аудит лог записано как "worker_registered"
      | field        | value         |
      | worker_id    | <worker_id>   |
      | worker_name  | worker-msk-01 |
      | zone         | moscow        |
      | timestamp    | <iso8601>     |

  @use_case=uc_09_01_02
  @critical
  Scenario: Heartbeat от активного воркера
    Given воркер "worker-msk-01" зарегистрирован в зоне "moscow"
    And воркер имеет статус "IDLE"
    When воркер отправляет heartbeat с метриками:
      | checks_completed      | 150           |
      | checks_failed         | 3             |
      | avg_check_duration_ms | 245.5         |
    Then heartbeat принят
    And время последнего heartbeat обновлено
    And метрики воркера обновлены

  @use_case=uc_09_01_03
  @critical
  Scenario: Graceful отключение воркера
    Given воркер "worker-msk-01" зарегистрирован и имеет статус "BUSY"
    And воркер выполняет "2" проверки
    When воркер отправляет запрос на отключение
    Then воркер получает разрешение на отключение
    And незавершённые проверки переназначены другим воркерам
    And статус воркера изменён на "OFFLINE"
    And действие в аудит лог записано как "worker_unregistered"
      | field         | value         |
      | worker_id     | <worker_id>   |
      | worker_name   | worker-msk-01 |
      | pending_tasks | 2             |
      | reason        | graceful      |
      | timestamp     | <iso8601>     |

  @use_case=uc_09_01_04
  @critical
  Scenario: Получение статуса воркера
    Given воркер "worker-msk-01" зарегистрирован в зоне "moscow"
    When запрашивается статус воркера
    Then получена информация о воркере:
      | name                  | worker-msk-01 |
      | zone                  | moscow        |
      | status                | IDLE          |
      | checks_completed      | <number>      |
      | checks_failed         | <number>      |
      | avg_check_duration_ms | <number>      |
      | last_heartbeat        | <iso8601>     |

  @use_case=uc_09_01_05
  @critical
  Scenario: Список всех зарегистрированных воркеров
    Given зарегистрированы воркеры:
      | name          | zone    | status |
      | worker-msk-01 | moscow  | IDLE   |
      | worker-msk-02 | moscow  | BUSY   |
      | worker-spb-01 | spb     | IDLE   |
    When запрашивается список воркеров
    Then получен список из "3" воркеров
    And каждый воркер содержит имя, зону и статус

  @use_case=uc_09_01_06
  @critical
  Scenario: Фильтрация воркеров по зоне
    Given зарегистрированы воркеры:
      | name          | zone    |
      | worker-msk-01 | moscow  |
      | worker-msk-02 | moscow  |
      | worker-spb-01 | spb     |
    When запрашивается список воркеров с фильтром по зоне "moscow"
    Then получен список из "2" воркеров
    And все воркеры принадлежат зоне "moscow"

  @use_case=uc_09_01_07
  @critical
  Scenario: Фильтрация воркеров по статусу
    Given зарегистрированы воркеры:
      | name          | status |
      | worker-msk-01 | IDLE   |
      | worker-msk-02 | BUSY   |
      | worker-spb-01 | IDLE   |
    When запрашивается список воркеров со статусом "IDLE"
    Then получен список из "2" воркеров
    And все воркеры имеют статус "IDLE"

  # Integration: Воркер не отправляет heartbeat в течение configured timeout
  # Scheduler помечает его как OFFLINE и переназначает проверки
  # See: Epic 09, uc_09_02_08
  @use_case=uc_09_01_08
  @critical
  @state_transition
  Scenario: Обнаружение offline воркера по истечению heartbeat timeout
    Given воркер "worker-msk-01" зарегистрирован и имеет статус "BUSY"
    And heartbeat timeout = "30 seconds"
    And воркер не отправлял heartbeat более "30 seconds"
    When scheduler-service обнаруживает просроченный heartbeat
    Then статус воркера изменён на "OFFLINE"
    And проверки назначенные воркеру перенесены в очередь
    And действие в аудит лог записано как "worker_marked_offline"
      | field        | value         |
      | worker_id    | <worker_id>   |
      | worker_name  | worker-msk-01 |
      | last_seen    | <iso8601>     |
      | pending_task | <count>       |
      | reason       | heartbeat_timeout |
      | timestamp    | <iso8601>     |

  # Integration: Воркер переподключается после offline
  # See: Epic 10, uc_10_01_07
  @use_case=uc_09_01_09
  @critical
  @recovery
  Scenario: Переподключение воркера после offline
    Given воркер "worker-msk-01" был помечен как "OFFLINE"
    And прошло менее "5 minutes" с момента offline
    When воркер отправляет heartbeat
    Then статус воркера восстановлен на "IDLE"
    And воркер может получать новые проверки
    And действие в аудит лог записано как "worker_reconnected"
      | field        | value         |
      | worker_id    | <worker_id>   |
      | worker_name  | worker-msk-01 |
      | offline_duration | <duration> |
      | timestamp    | <iso8601>     |

  @use_case=uc_09_01_10
  @critical
  Scenario: Регистрация воркера с метаданными
    Given check-worker запускается с параметрами:
      | name | worker-msk-01 |
      | zone | moscow        |
    And воркер передает метаданные:
      | key       | value              |
      | version   | 1.0.0              |
      | os        | linux              |
      | max_concurrent | 50            |
    When воркер регистрируется в scheduler-service
    Then воркер зарегистрирован успешно
    And метаданные сохранены

  @use_case=uc_09_01_01a
  @critical
  @validation
  Scenario: Регистрация воркера без имени
    Given check-worker запускается без имени
    When воркер отправляет запрос на регистрацию
    Then система возвращает ошибку "WORKER_NAME_REQUIRED"
    And воркер не зарегистрирован

  @use_case=uc_09_01_01b
  @critical
  @validation
  Scenario: Регистрация воркера с дублирующимся именем
    Given воркер "worker-msk-01" уже зарегистрирован
    When другой воркер регистрируется с именем "worker-msk-01"
    Then система возвращает ошибку "WORKER_NAME_EXISTS"
    And второй воркер не зарегистрирован

  @use_case=uc_09_01_01c
  @critical
  @validation
  Scenario: Ошибка регистрации воркера с невалидной зоной c
    Given check-worker запускается с параметрами:
      | name | worker-xyz-01 |
      | zone | unknown-zone  |
    When воркер отправляет запрос на регистрацию
    Then система возвращает ошибку "INVALID_ZONE"
    And воркер не зарегистрирован

  @use_case=uc_09_01_02a
  @critical
  @validation
  Scenario: Heartbeat от незарегистрированного воркера
    Given воркер с идентификатором "unknown-id" не зарегистрирован
    When воркер отправляет heartbeat
    Then система возвращает ошибку "WORKER_NOT_FOUND"
    And heartbeat отклонён

  @use_case=uc_09_01_02b
  @critical
  @validation
  Scenario: Ошибка heartbeat с невалидным статусом b
    Given воркер "worker-msk-01" зарегистрирован
    When воркер отправляет heartbeat со статусом "invalid_status"
    Then система возвращает ошибку "INVALID_WORKER_STATUS"
    And heartbeat отклонён

  @use_case=uc_09_01_03a
  @critical
  @validation
  Scenario: Отключение незарегистрированного воркера
    Given воркер с идентификатором "unknown-id" не зарегистрирован
    When воркер отправляет запрос на отключение
    Then система возвращает ошибку "WORKER_NOT_FOUND"
    And действие в аудит лог записано как "unregistered_worker_disconnect_attempt"
      | field      | value       |
      | worker_id  | unknown-id  |
      | timestamp  | <iso8601>   |

  @use_case=uc_09_01_04a
  @critical
  @validation
  Scenario: Запрос статуса незарегистрированного воркера
    Given воркер с идентификатором "unknown-id" не зарегистрирован
    When запрашивается статус воркера
    Then система возвращает ошибку "WORKER_NOT_FOUND"

  @use_case=uc_09_01_01d
  @critical
  @validation
  Scenario: Регистрация воркера с именем длиннее 255 символов
    Given check-worker запускается с именем из "300" символов
    When воркер отправляет запрос на регистрацию
    Then система возвращает ошибку "WORKER_NAME_TOO_LONG"
    And воркер не зарегистрирован

  @use_case=uc_09_01_08a
  @critical
  @recovery
  Scenario: Удаление offline воркера после extended timeout
    Given воркер "worker-msk-01" помечен как "OFFLINE"
    And прошло более "10 minutes" с момента offline
    When scheduler-service выполняет cleanup
    Then воркер удалён из реестра активных воркеров
    And все назначенные воркеру проверки переназначены
    And действие в аудит лог записано как "worker_removed"
      | field        | value         |
      | worker_id    | <worker_id>   |
      | worker_name  | worker-msk-01 |
      | reason       | extended_timeout |
      | timestamp    | <iso8601>     |

  @use_case=uc_09_01_11
  @critical
  @performance
  Scenario: Heartbeat от 50+ воркеров одновременно
    Given зарегистрированы "50" воркеров в зонах "moscow" и "spb"
    When все воркеры одновременно отправляют heartbeat
    Then все heartbeat обработаны успешно
    And время обработки не превышает "5 seconds"
    And действие в аудит лог записано как "batch_heartbeat_processed"
      | field  | value |
      | count  | 50    |

  @use_case=uc_09_01_12
  @critical
  @integration
  Scenario: Health check для scheduler-service
    Given scheduler-service запущен
    And подключение к базе данных активно
    When запрашивается health check
    Then scheduler-service возвращает статус "healthy"

  # =====================================================================
  # Реализованный срез: lifecycle воркеров через реальный gRPC scheduler API.
  # Мигрировано из backend/tests/integration/grpc/worker_scheduler_test.go.
  # =====================================================================

  @use_case=uc_09_01_20
  @implemented
  @integration
  Scenario: Lifecycle воркера через scheduler API
    When воркер регистрируется через scheduler API с именем "lifecycle-worker" в зоне "moscow"
    Then воркер успешно зарегистрирован и получил идентификатор
    When воркер отправляет heartbeat со статусом "IDLE"
    Then heartbeat принят без ошибки
    When воркер выполняет дерегистрацию
    Then воркер удалён из реестра scheduler-service

  @use_case=uc_09_01_21
  @implemented
  @integration
  Scenario: Повторная регистрация воркера с тем же именем отклоняется
    Given воркер с именем "duplicate-worker" уже зарегистрирован через scheduler API
    When второй воркер пытается зарегистрироваться с именем "duplicate-worker"
    Then scheduler возвращает ошибку с кодом AlreadyExists

  @use_case=uc_09_01_22
  @implemented
  @integration
  Scenario: Heartbeat от незарегистрированного воркера отклоняется
    When heartbeat отправляется для неизвестного worker_id "00000000-0000-0000-0000-000000000001"
    Then scheduler возвращает ошибку с кодом NotFound

  @use_case=uc_09_01_23
  @implemented
  @integration
  Scenario: Параллельная регистрация уникальных воркеров
    When одновременно регистрируются "3" воркеров с уникальными именами
    Then все "3" воркеров получают уникальные идентификаторы
    And в реестре scheduler-service не менее "3" воркеров

