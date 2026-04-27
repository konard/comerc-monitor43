@epic=06_reporting
@user_story=06_02_analytics
# Description: Агрегированная аналитика по всем мониторам аккаунта (тренды, сравнения, MTTR/MTTF)

Feature: Агрегированная аналитика
  Как пользователь
  Я хочу просматривать агрегированную аналитику по всем мониторам
  Чтобы понимать общую картину доступности и производительности сервисов

  # Отличие от Epic 01 (per-monitor) и Epic 03 (dashboard UI):
  # Здесь — кросс-мониторная агрегация: тренды, сравнения, MTTR/MTTF, сводная статистика

  @use_case=uc_06_02_01
  @critical
  Scenario: Сводная аналитика по всем мониторам за период
    Given пользователь имеет "10" мониторов
    And "8" мониторов в статусе "UP"
    And "2" монитора в статусе "DOWN"
    When пользователь запрашивает сводную аналитику за период "24h"
    Then возвращается общее количество мониторов
    And возвращается количество UP мониторов
    And возвращается количество DOWN мониторов
    And возвращается средний uptime по всем мониторам
    And возвращается среднее время ответа по всем мониторам
    And данные возвращаются в UTC
    And действие в аудит лог записано как "analytics_summary_requested"
    And запись содержит:
      | field     | value |
      | user_id   | <user_id> |
      | period    | 24h |
      | timestamp | <iso8601> |

  @use_case=uc_06_02_02
  @critical
  Scenario: Тренд uptime по дням за период
    Given пользователь имеет "5" мониторов за последние "30 days"
    When пользователь запрашивает тренд uptime по дням
    Then возвращается массив ежедневных значений uptime
    And каждая точка содержит дату и средний uptime по всем мониторам
    And данные упорядочены по дате (возрастание)
    And дата представлена в UTC

  @use_case=uc_06_02_03
  @critical
  Scenario: Тренд response time по дням за период
    Given пользователь имеет "5" мониторов за последние "30 days"
    When пользователь запрашивает тренд response time по дням
    Then возвращается массив ежедневных значений P50, P95, P99
    And percentiles рассчитываются только для успешных проверок (2xx)
    And неудачные проверки (5xx) исключены из расчёта percentile
    And каждая точка содержит дату и значения P50, P95, P99

  @use_case=uc_06_02_04
  @critical
  Scenario: Инцидент-аналитика (MTTR, MTTF, частота)
    Given пользователь имеет "10" мониторов
    And было "15" инцидентов за последние "30 days"
    When пользователь запрашивает инцидент-аналитику за период "30d"
    Then возвращается MTTR (Mean Time To Recovery) в минутах
    And возвращается MTTF (Mean Time To Failure) в минутах
    And возвращается общее количество инцидентов
    And возвращается распределение инцидентов по мониторам
    And возвращается средняя длительность инцидента

  @use_case=uc_06_02_05
  @critical
  Scenario: Сравнение мониторов по uptime
    Given пользователь имеет "5" мониторов
    And мониторы имеют разный uptime за "7 days"
    When пользователь запрашивает сравнение мониторов за период "7d"
    Then возвращается список мониторов с uptime за период
    And список упорядочен по uptime (возрастание)
    And каждый элемент содержит идентификатор монитора и процент uptime
    And сравнение позволяет выявить наименее доступные сервисы

  @use_case=uc_06_02_06
  @critical
  Scenario: Сравнение мониторов по response time
    Given пользователь имеет "5" мониторов
    And мониторы имеют разный P95 response time за "7 days"
    When пользователь запрашивает сравнение response time за период "7d"
    Then возвращается список мониторов с P95 за период
    And список упорядочен по P95 (убывание)
    And каждый элемент содержит идентификатор монитора и P95 response time
    And сравнение позволяет выявить самые медленные сервисы

  @use_case=uc_06_02_07
  @critical
  Scenario: Экспорт агрегированной аналитики в CSV
    Given пользователь имеет "10" мониторов
    And данные доступны за "30 days"
    When пользователь экспортирует агрегированную аналитику в CSV
    Then файл CSV загружен
    And файл содержит колонки: date, monitor_id, uptime_pct, p50_ms, p95_ms, p99_ms, status
    And timestamp в UTC
    And действие в аудит лог записано как "analytics_exported"
    And запись содержит:
      | field   | value |
      | user_id | <user_id> |
      | format  | csv |
      | period  | 30d |
      | monitors | 10 |
      | timestamp | <iso8601> |

  @use_case=uc_06_02_08
  @critical
  Scenario: Распределение инцидентов по часам суток
    Given пользователь имеет "5" мониторов
    And было "20" инцидентов за последние "30 days"
    When пользователь запрашивает распределение инцидентов по часам
    Then возвращается массив из "24" значений (по количеству часов)
    And каждое значение содержит количество инцидентов в данный час
    And данные усреднены за весь период
    And позволяет выявить часы с наибольшей нагрузкой

  @use_case=uc_06_02_01a
  @critical
  @validation
  Scenario: Попытка запроса аналитики с невалидным форматом периода a
    Given пользователь имеет "5" мониторов
    When пользователь запрашивает аналитику за период "invalid"
    Then возвращается ошибка "INVALID_PERIOD_FORMAT"
    And ответ содержит структуру:
      | error.code       | INVALID_PERIOD_FORMAT |
      | error.message    | Invalid period format |
      | error.details    | {"period": "invalid"} |
      | error.request_id | <uuid> |

  @use_case=uc_06_02_01b
  @critical
  @validation
  Scenario: Попытка запроса аналитики с отрицательным периодом b
    Given пользователь имеет "5" мониторов
    When пользователь запрашивает аналитику за период "-7d"
    Then возвращается ошибка "INVALID_PERIOD_FORMAT"
    And ответ содержит структуру:
      | error.code       | INVALID_PERIOD_FORMAT |
      | error.message    | Period must be a positive number |
      | error.details    | {"period": "-7d"} |
      | error.request_id | <uuid> |

  @use_case=uc_06_02_01c
  @critical
  @validation
  Scenario: Попытка запроса аналитики за период превышающий retention policy c
    Given retention policy составляет "90" дней
    And пользователь имеет "5" мониторов
    When пользователь запрашивает аналитику за период "100d"
    Then возвращается ошибка "PERIOD_EXCEEDS_RETENTION"
    And ответ содержит структуру:
      | error.code       | PERIOD_EXCEEDS_RETENTION |
      | error.message    | Period exceeds data retention policy of 90 days |
      | error.details    | {"requested_period": "100d", "retention_days": 90} |
      | error.request_id | <uuid> |

  @use_case=uc_06_02_01d
  @critical
  @validation
  Scenario: Попытка запроса аналитики с пустым диапазоном дат d
    Given пользователь имеет "5" мониторов
    When пользователь запрашивает аналитику с "" по ""
    Then возвращается ошибка "INVALID_DATE_RANGE"
    And ответ содержит структуру:
      | error.code       | INVALID_DATE_RANGE |
      | error.message    | Date range cannot be empty |
      | error.details    | {"start_date": "", "end_date": ""} |
      | error.request_id | <uuid> |

  @use_case=uc_06_02_07a
  @critical
  @security
  Scenario: Попытка экспорта аналитики чужого монитора a
    Given пользователь "user1@example.com" имеет "5" мониторов
    And пользователь "user2@example.com" авторизован
    When пользователь "user2@example.com" пытается экспортировать аналитику мониторов "user1"
    Then возвращается ошибка "FORBIDDEN"
    And ответ содержит структуру:
      | error.code       | FORBIDDEN |
      | error.message    | No access to this resource |
      | error.details    | {"user_id": "user2@example.com"} |
      | error.request_id | <uuid> |
    And действие в аудит лог записано как "analytics_access_denied"
    And запись содержит:
      | field   | value |
      | user_id | user2@example.com |
      | timestamp | <iso8601> |

  @use_case=uc_06_02_01e
  @critical
  @integration
  Scenario: Таймаут запроса для больших наборов данных e
    Given пользователь имеет "50" мониторов
    And монитор имеет "50000" проверок за "90 days"
    And запрос к базе данных превышает "30" секунд
    When пользователь запрашивает сводную аналитику
    Then возвращается ошибка "TIMEOUT"
    And ответ содержит структуру:
      | error.code       | TIMEOUT |
      | error.message    | Query execution timeout |
      | error.details    | {"timeout_seconds": 30} |
      | error.request_id | <uuid> |
    And предлагается уменьшить период запроса

  @use_case=uc_06_02_01f
  @critical
  @integration
  Scenario: Ошибка аналитики: time series сервис Недоступен f
    Given пользователь имеет "5" мониторов
    And time series сервис недоступен
    When пользователь запрашивает аналитику
    Then возвращается ошибка "SERVICE_UNAVAILABLE"
    And ответ содержит структуру:
      | error.code       | SERVICE_UNAVAILABLE |
      | error.message    | Time series service unavailable |
      | error.details    | {"service": "timeseries_db"} |
      | error.request_id | <uuid> |
    And предлагается повторить запрос позже

  @use_case=uc_06_02_07b
  @critical
  @integration
  Scenario: Таймаут сервиса генерации PDF b
    Given пользователь запрашивает экспорт аналитики в PDF
    And сервис генерации PDF не отвечает в течение "30" секунд
    When пользователь экспортирует отчёт в PDF
    Then возвращается ошибка "TIMEOUT"
    And ответ содержит структуру:
      | error.code       | TIMEOUT |
      | error.message    | PDF generation service timeout |
      | error.details    | {"timeout_seconds": 30} |
      | error.request_id | <uuid> |
    And файл PDF не загружен
    And пользователю предложено повторить попытку

  @use_case=uc_06_02_01g
  @critical
  @boundary
  Scenario: Запрос аналитики для аккаунта без мониторов
    Given пользователь не имеет ни одного монитора
    When пользователь запрашивает сводную аналитику за период "24h"
    Then возвращается пустой результат
    And общее количество мониторов "0"
    And средний uptime "N/A"
    And среднее время ответа "N/A"

  @use_case=uc_06_02_04a
  @critical
  @boundary
  Scenario: Инцидент-аналитика при отсутствии инцидентов
    Given пользователь имеет "5" мониторов
    And не было инцидентов за последние "30 days"
    When пользователь запрашивает инцидент-аналитику за период "30d"
    Then MTTR равен "N/A"
    And MTTF равен "N/A"
    And общее количество инцидентов "0"
    And распределение инцидентов пустое

  @use_case=uc_06_02_01h
  @critical
  @boundary
  Scenario: Переход через DST при запросе аналитики
    Given пользователь имеет timezone "Europe/Moscow"
    And происходит переход на летнее время
    And период аналитики включает момент перехода
    When пользователь запрашивает тренд uptime по дням
    Then временные метки корректно учитывают переход DST
    And период расчёта корректен
    And данные возвращаются в UTC
    And конвертация timezone происходит на frontend

  @use_case=uc_06_02_02a
  @critical
  @boundary
  Scenario: Запрос аналитики с нулевыми данными (все мониторы без проверок)
    Given пользователь имеет "3" монитора
    And ни один монитор не имеет проверок
    When пользователь запрашивает тренд uptime по дням
    Then возвращается пустой массив
    And статус ответа "200 OK"

  @use_case=uc_06_02_01i
  @performance
  Scenario: Агрегированная аналитика для 50+ мониторов
    Given пользователь имеет "50" мониторов
    And каждый монитор имеет "10000" проверок за "30 days"
    When пользователь запрашивает сводную аналитику за период "30d"
    Then отчёт сгенерирован в течение "15" секунд
    And все мониторы обработаны
    And метрики рассчитаны корректно

  @use_case=uc_06_02_07c
  @performance
  Scenario: Экспорт большой агрегированной аналитики в CSV
    Given пользователь имеет "50" мониторов
    And каждый монитор имеет данные за "90 days"
    When пользователь экспортирует агрегированную аналитику в CSV
    Then файл CSV генерируется в течение "30" секунд
    And файл содержит данные по всем мониторам
    And размер файла не превышает "50" MB
    And файл успешно загружен

  @use_case=uc_06_02_03a
  @performance
  Scenario: Параллельный запрос множественных метрик
    Given пользователь имеет "20" мониторов
    And данные доступны за "30 days"
    When пользователь запрашивает одновременно тренд uptime, тренд response time и инцидент-аналитику
    Then все метрики возвращены в течение "20" секунд
    And данные корректны и согласованы
    And нет деградации производительности
