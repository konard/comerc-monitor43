@epic=01_monitoring
@user_story=01_04_response_tracking
# Description: Отслеживание и хранение результатов проверок
#
# Architecture Note:
#   Начиная с v1.0, результаты поступают от check-worker'ов:
#   - check-worker (Epic 10) отправляет результаты через SubmitCheckResult API
#   - monitor-service сохраняет результат и публикует StatusChangeEvent
#   - События потребляются alert-service, dashboard-service, integration-service
#
#   Правила хранения и расчёта статистики описаны в этом файле.
#   Отправка результатов описана в Epic 10 (10_03_result_reporting).

Feature: Отслеживание результатов проверок
  Как система
  Я хочу отслеживать результаты всех проверок
  Чтобы иметь историю доступности сервисов

  @use_case=uc_01_04_01
  @critical
  Scenario: Сохранение результата успешной проверки
    Given выполнена проверка монитора "API Service"
    And результат проверки "UP"
    When результат сохраняется
    Then запись о проверке сохранена в хранилище
    And запись содержит статус "UP"
    And запись содержит время ответа

  @use_case=uc_01_04_02
  @critical
  Scenario: Сохранение результата неуспешной проверки
    Given выполнена проверка монитора "API Service"
    And результат проверки "DOWN"
    And ошибка "Connection refused"
    When результат сохраняется
    Then запись о проверке сохранена в хранилище
    And запись содержит статус "DOWN"
    And запись содержит описание ошибки
    And действие в аудит лог записано как "result_saved_failed"
      | field        | value                |
      | monitor_id   | API Service          |
      | status       | DOWN                 |
      | error        | Connection refused   |
      | timestamp    | <iso8601>            |

  @use_case=uc_01_04_03
  @critical
  Scenario: Получение статистики response time
    Given монитор "API Service" имеет "100" результатов проверок
    When запрашивается статистика response time
    Then получены значения P50, P95, P99
    And получено среднее время ответа

  @use_case=uc_01_04_04
  @critical
  Scenario: Сохранение результата DEGRADED проверки
    Given выполнена проверка монитора "API Service"
    And результат проверки "DEGRADED"
    And время ответа "6 seconds"
    When результат сохраняется
    Then запись о проверке сохранена в хранилище
    And запись содержит статус "DEGRADED"
    And запись содержит время ответа

  @use_case=uc_01_04_05
  @critical
  Scenario: Удаление старых результатов проверки (retention 90 дней)
    Given монитор "API Service" имеет результаты проверки старше "90 days"
    When выполняется очистка старых данных
    Then результаты старше "90 days" удалены
    And результаты новее "90 days" сохранены

  @use_case=uc_01_04_06
  @critical
  Scenario: Получение результатов за период
    Given монитор "API Service" имеет результаты за последние "30 days"
    When запрашиваются результаты за "7 days"
    Then получены результаты за "7 days"
    And результаты содержат временные метки
    And результаты упорядочены по времени

  @use_case=uc_01_04_07
  @critical
  Scenario: Пропуск сохранения для PAUSED монитора
    Given монитор "API Service" в статусе "PAUSED"
    When выполняется проверка
    Then результат не сохраняется
    And проверки не выполняются

  # Business Logic: When monitor is entirely PAUSED, return 100% or null
  # For mixed periods (UP/DOWN/PAUSED), exclude PAUSED from calculation
  # See 01_05_monitor_history.feature for complete uptime calculation rules
  @use_case=uc_01_04_08
  @critical
  Scenario: Расчет процентов для uptime (полностью PAUSED = 100% или null)
    Given монитор "API Service" в статусе "PAUSED"
    And монитор имеет "100" проверок за период
    And все проверки были "PAUSED"
    When рассчитывается uptime за период
    Then uptime равен "100%" или null (нет активных проверок для измерения)
    And PAUSED проверки не учитываются как downtime
    And система показывает что монитор был приостановлен

  @use_case=uc_01_04_02a
  @critical
  Scenario: Ошибка сохранения результата при недоступности БД a
    Given выполнена проверка монитора "API Service"
    And результат проверки "UP"
    And база данных недоступна
    When результат сохраняется
    Then результат помещен в очередь для повторной попытки
    And возвращена ошибка "STORAGE_ERROR"

  @use_case=uc_01_04_09
  @critical
  Scenario: Повторное сохранение того же результата
    Given результат проверки уже сохранен
    And ID проверки "12345"
    When результат с тем же ID сохраняется повторно
    Then дубликат не создается
    And существующая запись обновляется или игнорируется

  @use_case=uc_01_04_02b
  @critical
  Scenario: Попытка сохранения результата без статуса b
    Given выполнена проверка монитора "API Service"
    And результат проверки не содержит статуса
    When результат сохраняется
    Then система возвращает ошибку "MISSING_STATUS"
    And результат не сохранен

  @use_case=uc_01_04_10
  @critical
  Scenario: Получение статистики для монитора без проверок - пустая статистика
    Given монитор "API Service" не имеет результатов проверок
    When запрашивается статистика response time
    Then возвращается пустая статистика
    And статистика содержит нулевые значения
    And действие в аудит лог записано как "statistics_empty"
      | field        | value           |
      | monitor_id   | API Service     |
      | status       | no_data         |
      | timestamp    | <iso8601>       |

  @use_case=uc_01_04_11
  @critical
  Scenario: Получение статистики для монитора без проверок - ошибка
    Given монитор "API Service" не имеет результатов проверок
    And система настроена возвращать ошибку при отсутствии данных
    When запрашивается статистика response time
    Then возвращается ошибка "NO_DATA_AVAILABLE"
    And ответ содержит структуру:
      | error.code    | NO_DATA_AVAILABLE |
      | error.message | No data available |
      | error.details | {...} |
      | error.request_id | uuid |
    And действие в аудит лог записано как "statistics_not_available"
      | field        | value                |
      | monitor_id   | API Service          |
      | error_code   | NO_DATA_AVAILABLE    |
      | timestamp    | <iso8601>            |

  @use_case=uc_01_04_12
  @critical
  Scenario: Попытка сохранения результата во время обслуживания БД - очередь a
    Given выполнена проверка монитора "API Service"
    And активное окно обслуживания базы данных
    When результат сохраняется
    Then результат помещен в очередь
    And результат будет сохранен после окончания обслуживания
    And действие в аудит лог записано как "result_queued_during_maintenance"
      | field           | value                |
      | monitor_id      | API Service          |
      | queue_position  | <position>           |
      | timestamp       | <iso8601>            |

  @use_case=uc_01_04_13
  @critical
  Scenario: Попытка сохранения результата во время обслуживания БД - после обслуживания a
    Given выполнена проверка монитора "API Service"
    And активное окно обслуживания базы данных
    And очередь результатов переполнена
    When результат сохраняется
    Then результат сохранен после окончания обслуживания
    And ответ содержит статус "ACCEPTED_DELAYED"
    And действие в аудит лог записано как "result_saved_after_maintenance"
      | field        | value                      |
      | monitor_id   | API Service                |
      | delay_ms     | <delay>                    |
      | timestamp    | <iso8601>                  |

  @use_case=uc_01_04_03a
  @critical
  Scenario: Попытка доступа к результатам чужого монитора a
    Given пользователь "UserA" имеет монитор "API Service" с результатами проверок
    And пользователь "UserB" авторизован
    When пользователь "UserB" запрашивает результаты монитора "API Service"
    Then система возвращает ошибку "FORBIDDEN"
    And результаты не показаны

  @use_case=uc_01_04_03b
  @critical
  Scenario: Попытка доступа к статистике чужого монитора b
    Given пользователь "UserA" имеет монитор "API Service" с результатами проверок
    And пользователь "UserB" авторизован
    When пользователь "UserB" запрашивает статистику монитора "API Service"
    Then система возвращает ошибку "FORBIDDEN"
    And статистика не показана

  @use_case=uc_01_04_14
  @critical
  @performance
  Scenario: Обработка большого набора результатов (10000+ проверок)
    Given монитор "API Service" имеет "10000" результатов проверок
    When запрашиваются результаты за "30 days"
    Then результаты возвращены
    And время запроса не превышает "2 seconds"

  @use_case=uc_01_04_02c
  @critical
  @recovery
  Scenario: Консистентность данных после частичного сбоя сохранения
    Given выполняется пакетное сохранение "100" результатов
    And база данных становится недоступной после сохранения "50" результатов
    When происходит сбой
    Then первые "50" результатов сохранены
    And остальные "50" результатов помещены в очередь
    And данные консистентны

  @use_case=uc_01_04_02d
  @critical
  @recovery
  Scenario: Реконсиляция состояния после перезапуска системы
    Given система была перезапущена
    And существуют несохранённые результаты в памяти
    When система восстанавливается
    Then все несохранённые результаты записаны
    And состояние реконсиляции завершено
    And данные целостны

  @use_case=uc_01_04_02e
  @critical
  @performance
  Scenario: Высокочастотные изменения статуса монитора
    Given монитор "API Service" переключается между UP и DOWN
    And происходит "20" переключений за "1 minute"
    When каждое изменение статуса сохраняется
    Then все изменения записаны
    And система обрабатывает изменения без задержек
    And производительность не degraded
