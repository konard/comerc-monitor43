@epic=06_reporting
@user_story=06_01_sla_reports
# Description: Генерация SLA отчётов для менеджмента и партнёров

Feature: SLA отчёты
  Как пользователь
  Я хочу генерировать SLA отчёты
  Чтобы отслеживать соблюдение соглашений с партнёрами и предоставлять отчёты менеджменту

  # Integration: Queries monitor check history from Epic 01 (Monitoring)
  # Uses uptime calculation formula from uc_01_05_03 (DEGRADED weight: 50%, PAUSED excluded)
  # See: Epic 01, uc_01_05_08 and uc_01_05_11 for uptime calculation rules

  @use_case=uc_06_01_01
  @critical
  Scenario: Генерация SLA отчёта за 24 часа
    Given монитор имеет "100" проверок за последние "24 hours"
    And "2" проверки неудачные
    When пользователь генерирует SLA отчёт за период "24h"
    Then отчёт содержит процент доступности "98.0%"
    And отчёт содержит общее количество проверок
    And отчёт содержит количество DOWN проверок
    And отчёт содержит время простоя в секундах
    And данные возвращаются в UTC
    And конвертация в timezone пользователя происходит на frontend
    And действие в аудит лог записано как "sla_report_generated"
    And запись содержит:
      | field   | value |
      | user_id | <user_id> |
      | monitor_id | <monitor_id> |
      | period  | 24h |
      | timestamp | <iso8601> |

  @use_case=uc_06_01_02
  @critical
  Scenario: Генерация SLA отчёта за 7 дней
    Given монитор имеет "700" проверок за последние "7 days"
    And "10" проверок неудачные
    And "5" проверок в статусе DEGRADED
    When пользователь генерирует SLA отчёт за период "7d"
    Then отчёт содержит процент доступности учитывая DEGRADED как частичный downtime
    And отчёт содержит общее количество проверок
    And отчёт содержит количество DOWN проверок
    And отчёт содержит количество DEGRADED проверок
    And формула расчёта: (UP + DEGRADED * 0.5) / (UP + DEGRADED + DOWN) * 100

  @use_case=uc_06_01_03
  @critical
  Scenario: Генерация SLA отчёта за 30 дней
    Given монитор имеет "3000" проверок за последние "30 days"
    And "50" проверок неудачные
    And монитор был в статусе PAUSED в течение "24 hours"
    When пользователь генерирует SLA отчёт за период "30d"
    Then отчёт содержит процент доступности "98.33%"
    And период PAUSED исключён из расчёта downtime
    And PAUSED проверки не учитываются как uptime и не учитываются как downtime
    And данные возвращаются в UTC

  @use_case=uc_06_01_04
  @critical
  @implemented
  Scenario: Получение списка SLA отчётов
    Given существует "5" SLA отчётов
    When пользователь запрашивает список отчётов
    Then возвращается "5" отчётов
    And отчёты отсортированы по дате создания (сначала новые)
    And каждый отчёт содержит:
      | поле        | описание                        |
      | id          | Идентификатор отчёта            |
      | monitor_id  | Идентификатор монитора          |
      | period      | Период отчёта                   |
      | availability | Процент доступности            |
      | created_at  | Дата генерации                  |

  @use_case=uc_06_01_05
  @critical
  Scenario: Экспорт SLA отчёта в PDF
    Given сгенерирован SLA отчёт
    When пользователь экспортирует отчёт в PDF
    Then файл PDF загружен
    And файл содержит все метрики SLA
    And файл содержит заголовок с периодом отчёта
    And файл содержит таблицу с проверками
    And действие в аудит лог записано как "sla_report_exported"
    And запись содержит:
      | field  | value |
      | user_id | <user_id> |
      | report_id | <report_id> |
      | format | pdf |
      | timestamp | <iso8601> |

  @use_case=uc_06_01_06
  @critical
  Scenario: Учёт 4xx кодов в SLA
    Given монитор имеет "100" проверок
    And "5" проверок вернули код "400"
    And пользователь настроил "400" как DOWN
    When пользователь генерирует SLA отчёт
    Then проверки с кодом "400" учитываются как downtime
    And отчёт содержит процент доступности "95.0%"

  @use_case=uc_06_01_07
  @critical
  Scenario: Настраиваемое отношение к 4xx кодам
    Given монитор имеет "100" проверок
    And "5" проверок вернули код "404"
    And пользователь настроил "404" как UP
    When пользователь генерирует SLA отчёт
    Then проверки с кодом "404" НЕ учитываются как downtime
    And отчёт содержит процент доступности "100.0%"

  @use_case=uc_06_01_08
  @critical
  Scenario: DEGRADED статус влияет на SLA
    Given монитор имеет "100" проверок
    And "10" проверок в статусе DEGRADED с response time > "5" секунд
    And пользователь настроил DEGRADED threshold как "5" секунд
    When пользователь генерирует SLA отчёт
    Then DEGRADED проверки учитываются как частичный downtime (вес 50%)
    And отчёт содержит процент доступности < "100.0%"
    And отчёт содержит количество DEGRADED проверок

  @use_case=uc_06_01_01a
  @critical
  @validation
  Scenario: Попытка генерации отчёта с невалидным форматом периода a
    Given монитор имеет данные
    When пользователь генерирует SLA отчёт за период "invalid"
    Then возвращается ошибка "INVALID_PERIOD_FORMAT"
    And ответ содержит структуру:
      | error.code       | INVALID_PERIOD_FORMAT |
      | error.message    | Invalid period format |
      | error.details    | {"period": "invalid"} |
      | error.request_id | <uuid> |

  @use_case=uc_06_01_01b
  @critical
  @validation
  @implemented
  Scenario: Попытка генерации отчёта с датой окончания раньше даты начала b
    Given монитор имеет данные
    When пользователь генерирует SLA отчёт с "2024-01-15" по "2024-01-01"
    Then возвращается ошибка "INVALID_DATE_RANGE"
    And ответ содержит структуру:
      | error.code       | INVALID_DATE_RANGE |
      | error.message    | End date cannot be before start date |
      | error.details    | {"start_date": "2024-01-15", "end_date": "2024-01-01"} |
      | error.request_id | <uuid> |

  @use_case=uc_06_01_01c
  @critical
  @validation
  @implemented
  Scenario: Попытка генерации отчёта за будущий период c
    Given монитор имеет данные
    When пользователь генерирует SLA отчёт с "2030-01-01" по "2030-12-31"
    Then возвращается ошибка "INVALID_DATE_RANGE"
    And ответ содержит структуру:
      | error.code       | INVALID_DATE_RANGE |
      | error.message    | Cannot generate report for future period |
      | error.details    | {"start_date": "2030-01-01", "end_date": "2030-12-31"} |
      | error.request_id | <uuid> |

  @use_case=uc_06_01_01d
  @critical
  @validation
  Scenario: Попытка генерации отчёта с отрицательным периодом d
    Given монитор имеет данные
    When пользователь генерирует SLA отчёт за период "-24h"
    Then возвращается ошибка "INVALID_PERIOD_FORMAT"
    And ответ содержит структуру:
      | error.code       | INVALID_PERIOD_FORMAT |
      | error.message    | Period must be a positive number |
      | error.details    | {"period": "-24h"} |
      | error.request_id | <uuid> |

  @use_case=uc_06_01_01e
  @critical
  @validation
  Scenario: Попытка генерации отчёта за период превышающий retention policy e
    Given retention policy составляет "90" дней
    And монитор имеет данные за "100" дней
    When пользователь генерирует SLA отчёт за период "100d"
    Then возвращается ошибка "PERIOD_EXCEEDS_RETENTION"
    And ответ содержит структуру:
      | error.code       | PERIOD_EXCEEDS_RETENTION |
      | error.message    | Period exceeds data retention policy of 90 days |
      | error.details    | {"requested_period": "100d", "retention_days": 90} |
      | error.request_id | <uuid> |

  @use_case=uc_06_01_04a
  @critical
  @security
  @implemented
  Scenario: Попытка доступа к чужому SLA отчёту a
    Given пользователь "user1@example.com" создал SLA отчёт
    And пользователь "user2@example.com" авторизован
    When пользователь "user2@example.com" пытается просмотреть отчёт
    Then возвращается ошибка "FORBIDDEN"
    And ответ содержит структуру:
      | error.code       | FORBIDDEN |
      | error.message    | No access to this report |
      | error.details    | {"report_id": <uuid>, "user_id": "user2@example.com"} |
      | error.request_id | <uuid> |
    And действие в аудит лог записано как "sla_report_access_denied"
    And запись содержит:
      | field     | value |
      | user_id   | user2@example.com |
      | report_id | <uuid> |
      | timestamp | <iso8601> |

  @use_case=uc_06_01_01f
  @critical
  @integration
  Scenario: Сбой подключения к базе данных при генерации отчёта f
    Given монитор имеет данные за "24 hours"
    And база данных недоступна
    When пользователь генерирует SLA отчёт за период "24h"
    Then возвращается ошибка "SERVICE_UNAVAILABLE"
    And ответ содержит структуру:
      | error.code       | SERVICE_UNAVAILABLE |
      | error.message    | Database connection failed |
      | error.details.retryable | true |
      | error.request_id | <uuid> |
    And система использует fixed delay retry стратегию
    And выполнена retry попытка через "1 second"
    And максимальное количество retry попыток "5"
    And событие ошибки записано в audit log

  @use_case=uc_06_01_05a
  @critical
  @integration
  Scenario: Сбой при экспорте отчёта a
    Given сгенерирован SLA отчёт
    And сервис экспорта недоступен
    When пользователь экспортирует отчёт в PDF
    Then возвращается ошибка "SERVICE_UNAVAILABLE"
    And ответ содержит структуру:
      | error.code       | SERVICE_UNAVAILABLE |
      | error.message    | Export service temporarily unavailable |
      | error.details.retryable | true |
      | error.request_id | <uuid> |
    And система использует exponential backoff retry стратегию
    And выполнена retry попытка через "5 seconds"
    And максимальное количество retry попыток "3"

  @use_case=uc_06_01_03a
  @critical
  @integration
  Scenario: Генерация отчёта при недоступности сервиса графиков
    Given монитор имеет данные за "7 days"
    And сервис графиков недоступен
    When пользователь генерирует SLA отчёт за период "7d"
    Then отчёт сгенерирован с текстовыми данными
    And график недоступен
    And пользователь видит предупреждение "CHARTS_TEMPORARILY_UNAVAILABLE"
    And остальная часть отчёта корректна

  @use_case=uc_06_01_03b
  @critical
  @boundary
  Scenario: Запрос отчёта точно на границе retention policy (90 дней)
    Given retention policy составляет "90" дней
    And монитор имеет данные за "90" дней
    When пользователь генерирует SLA отчёт за период "90d"
    Then отчёт сгенерирован успешно
    And период отчёта составляет ровно "90" дней
    And все данные за период включены

  @use_case=uc_06_01_03c
  @critical
  @boundary
  Scenario: Генерация отчёта для монитора со всеми проверками UP
    Given монитор имеет "1000" проверок за "24 hours"
    And все проверки успешные
    When пользователь генерирует SLA отчёт за период "24h"
    Then отчёт содержит процент доступности "100.0%"
    And время простоя "0" секунд
    And отчёт показывает нулевой downtime

  @use_case=uc_06_01_03d
  @critical
  @boundary
  Scenario: Дата создания монитора совпадает с началом периода отчёта
    Given монитор создан "7" дней назад в "2024-01-01 00:00:00 UTC"
    When пользователь генерирует SLA отчёт с "2024-01-01" по "2024-01-08"
    Then отчёт включает данные с момента создания
    And период расчёта точно совпадает с возрастом монитора
    And все данные включены

  @use_case=uc_06_01_01g
  @critical
  @boundary
  Scenario: Генерация отчёта для монитора без проверок
    Given монитор создан "1" час назад
    And монитор не имеет ни одной проверки
    When пользователь генерирует SLA отчёт за период "1h"
    Then возвращается пустой отчёт
    And процент доступности "N/A"
    And отчёт содержит:
      | report.availability | N/A |
      | report.total_checks | 0 |

  @use_case=uc_06_01_01h
  @performance
  Scenario: Генерация отчёта для большого набора данных (10000+ проверок)
    Given монитор имеет "10000" проверок за "30 days"
    When пользователь генерирует SLA отчёт за период "30d"
    Then отчёт сгенерирован в течение "10" секунд
    And все проверки обработаны
    And процент доступности рассчитан корректно

  @use_case=uc_06_01_04b
  @performance
  Scenario: Обработка параллельных запросов отчётов
    Given пользователь имеет "5" мониторов
    And пользователь запрашивает отчёты для всех мониторов одновременно
    When обрабатываются "5" параллельных запросов
    Then все отчёты сгенерированы успешно
    And время ответа каждого запроса не превышает "30" секунд
    And нет конфликтов между запросами

  @use_case=uc_06_01_01i
  @critical
  @integration
  Scenario: Конфликт при параллельной генерации отчёта
    Given пользователь инициировал генерацию отчёта для монитора
    And отчёт генерируется в данный момент
    When пользователь пытается генерировать второй отчёт для того же монитора
    Then возвращается ошибка "CONFLICT"
    And ответ содержит структуру:
      | error.code       | CONFLICT |
      | error.message    | Report generation already in progress |
      | error.details    | {"monitor_id": <uuid>} |
      | error.request_id | <uuid> |
    And вторая генерация не запущена
