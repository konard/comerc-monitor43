@epic=01_monitoring
@user_story=01_05_monitor_history
# Description: Просмотр истории проверок и инцидентов монитора

Feature: История монитора
  Как пользователь
  Я хочу просматривать историю проверок и инцидентов
  Чтобы анализировать доступность своего сервиса

  @use_case=uc_01_05_01
  @critical
  Scenario: Просмотр истории проверок за период
    Given пользователь имеет монитор "API Service"
    And монитор имеет проверки за последние "7 days"
    When пользователь запрашивает историю за "7 days"
    Then пользователь получает список проверок
    And каждая проверка содержит статус
    And каждая проверка содержит время выполнения
    And список упорядочен по убыванию времени

  @use_case=uc_01_05_02
  @critical
  Scenario: Просмотр истории инцидентов
    Given пользователь имеет монитор "API Service"
    And за последние "30 days" было "3" инцидента
    When пользователь запрашивает историю инцидентов
    Then пользователь получает список инцидентов
    And каждый инцидент содержит время начала
    And каждый инцидент содержит время окончания
    And каждый инцидент содержит длительность

  @use_case=uc_01_05_03
  @critical
  Scenario: Фильтрация истории по статусу
    Given пользователь имеет монитор "API Service"
    And монитор имеет проверки за последние "7 days"
    When пользователь запрашивает проверки со статусом "DOWN"
    Then пользователь получает только неудачные проверки
    And каждая проверка имеет статус "DOWN"
    And действие в аудит лог записано как "history_filtered"
      | field        | value                |
      | monitor_id   | API Service          |
      | filter       | status=DOWN          |
      | user_id      | <user_id>            |
      | timestamp    | <iso8601>            |

  @use_case=uc_01_05_04
  @critical
  Scenario: Расчет uptime за 24 часа
    Given пользователь имеет монитор "API Service"
    And за последние "24 hours" было "288" проверок
    And "286" проверок успешны
    And "2" проверки неудачны
    When пользователь запрашивает uptime за "24 hours"
    Then uptime равен "99.31%"
    And общее количество проверок равно "288"

  @use_case=uc_01_05_05
  @critical
  Scenario: Расчет uptime за 7 дней
    Given пользователь имеет монитор "API Service"
    And за последние "7 days" было "2016" проверок
    And "2000" проверок успешны
    And "16" проверок неудачны
    When пользователь запрашивает uptime за "7 days"
    Then uptime равен "99.21%"
    And общее количество проверок равно "2016"

  @use_case=uc_01_05_06
  @critical
  Scenario: Расчет uptime за 30 дней
    Given пользователь имеет монитор "API Service"
    And за последние "30 days" было "8640" проверок
    And "8500" проверок успешны
    And "140" проверок неудачны
    When пользователь запрашивает uptime за "30 days"
    Then uptime равен "98.38%"
    And общее количество проверок равно "8640"

  # Business Logic: When monitor is entirely PAUSED for the requested period,
  # return 100% uptime (or null) - no checks were performed to measure actual uptime.
  # This is different from excluding PAUSED periods from mixed active/paused periods.
  # See uc_01_05_11 for mixed periods logic.
  @use_case=uc_01_05_07
  @critical
  Scenario: Uptime для полностью PAUSED монитора (edge case)
    Given пользователь имеет монитор "API Service"
    And монитор в статусе "PAUSED"
    And за последние "24 hours" не было проверок (все время PAUSED)
    When пользователь запрашивает uptime за "24 hours"
    Then uptime равен "100%" или null (нет данных для измерения)
    And статус монитора "PAUSED"
    And система показывает примечание "Monitor paused, no uptime data available"

  @use_case=uc_01_05_08
  @critical
  Scenario: Uptime с учетом DEGRADED статуса
    # Uptime calculation: (UP + DEGRADED*0.5 + DOWN*0) / Total
    # (280 + 4*0.5 + 4*0) / 288 = 282 / 288 = 97.92%
    # Note: DEGRADED counts as 50% uptime
    Given пользователь имеет монитор "API Service"
    And за последние "24 hours" было "288" проверок
    And "280" проверок успешны
    And "4" проверки DEGRADED
    And "4" проверки неудачны
    When пользователь запрашивает uptime за "24 hours"
    Then uptime равен "97.92%"
    And DEGRADED проверки учтены как частичный downtime

  @use_case=uc_01_05_09
  @critical
  Scenario: Получение метрик P50, P95, P99
    Given пользователь имеет монитор "API Service"
    And монитор имеет "1000" успешных проверок за период
    When пользователь запрашивает метрики response time
    Then получен P50 response time
    And получен P95 response time
    And получен P99 response time
    And получено среднее время ответа

  @use_case=uc_01_05_01a
  @critical
  Scenario: История проверок старше 90 дней не доступна
    Given пользователь имеет монитор "API Service"
    And retention период равен "90 days"
    When пользователь запрашивает историю за "120 days"
    Then система возвращает ошибку "DATA_NOT_AVAILABLE"
    And данные старше "90 days" удалены

  @use_case=uc_01_05_01b
  @critical
  Scenario: Просмотр истории для монитора без проверок
    Given пользователь имеет монитор "API Service"
    And монитор не имеет проверок
    When пользователь запрашивает историю за "7 days"
    Then возвращается пустой список
    And сообщение "NO_CHECKS_YET"

  @use_case=uc_01_05_01c
  @critical
  Scenario: Запрос истории с невалидным диапазоном дат
    Given пользователь имеет монитор "API Service"
    When пользователь запрашивает историю с "2026-03-10" по "2026-03-01"
    Then система возвращает ошибку "INVALID_DATE_RANGE"

  @use_case=uc_01_05_01d
  @critical
  @validation
  Scenario: Запрос истории с будущей датой - возврат ошибки
    Given пользователь имеет монитор "API Service"
    And текущая дата "2026-03-07"
    When пользователь запрашивает историю с "2026-03-10" по "2026-03-15"
    Then система возвращает ошибку "FUTURE_DATE_RANGE"
    And ответ содержит структуру ошибки:
      | field       | type               |
      | code        | FUTURE_DATE_RANGE  |
      | message     | string             |
      | request_id  | string             |
    And действие в аудит лог записано как "history_query_rejected"
      | field       | value              |
      | monitor_id  | API Service        |
      | error_code  | FUTURE_DATE_RANGE  |
      | user_id     | <user_id>          |
      | timestamp   | <iso8601>          |

  @use_case=uc_01_05_01e
  @critical
  @boundary
  Scenario: Запрос истории с будущей датой - возврат доступных данных
    Given пользователь имеет монитор "API Service"
    And текущая дата "2026-03-07"
    And монитор имеет проверки до "2026-03-07"
    When пользователь запрашивает историю с "2026-03-05" по "2026-03-15"
    Then возвращаются только данные до "2026-03-07"
    And данные после "2026-03-07" отсутствуют
    And действие в аудит лог записано как "history_query_partial"
      | field       | value              |
      | monitor_id  | API Service        |
      | date_range  | 2026-03-05:2026-03-15 |
      | actual_range| 2026-03-05:2026-03-07 |
      | user_id     | <user_id>          |
      | timestamp   | <iso8601>          |

  @use_case=uc_01_05_10
  @critical
  Scenario: Просмотр инцидента который еще не завершен
    Given монитор "API Service" в статусе "DOWN"
    And инцидент начался "2 hours" назад
    And инцидент еще не завершен
    When пользователь запрашивает историю инцидентов
    Then инцидент отображается с временем начала
    And время окончания "ongoing" или null
    And длительность рассчитана до текущего момента

  # Business Logic: PAUSED periods are EXCLUDED from uptime calculation
  # Formula: (UP + DEGRADED*0.5) / (UP + DEGRADED + DOWN) * 100
  # PAUSED checks don't count as uptime or downtime - they're neutral
  # This matches industry standard: Pingdom, UptimeRobot, StatusCake
  @use_case=uc_01_05_11
  @critical
  Scenario: Uptime с периодами PAUSED и UP/DOWN (PAUSED excluded from calculation)
    Given пользователь имеет монитор "API Service"
    And за последние "24 hours" было "144" UP проверок
    And "72" проверки были PAUSED
    And "72" проверки были DOWN
    When пользователь запрашивает uptime за "24 hours"
    Then uptime рассчитан без учета PAUSED
    And uptime = "66.67%" (144 UP / 216 active checks, 72 PAUSED excluded)
    And система показывает "72 checks paused, excluded from calculation"

  @use_case=uc_01_05_01f
  @critical
  @validation
  Scenario: Просмотр истории удаленного монитора - возврат ошибки
    Given пользователь имел монитор "API Service"
    And монитор был удален
    And история проверок еще не удалена (retention period)
    When пользователь запрашивает историю монитора
    Then система возвращает ошибку "MONITOR_NOT_FOUND"
    And ответ содержит структуру ошибки:
      | field       | type               |
      | code        | MONITOR_NOT_FOUND  |
      | message     | string             |
      | request_id  | string             |
    And история не показана
    And действие в аудит лог записано как "history_query_denied"
      | field       | value              |
      | monitor_id  | API Service        |
      | error_code  | MONITOR_NOT_FOUND  |
      | user_id     | <user_id>          |
      | timestamp   | <iso8601>          |

  @use_case=uc_01_05_01g
  @critical
  @boundary
  Scenario: Просмотр истории удаленного монитора - архив доступен
    Given пользователь имел монитор "API Service"
    And монитор был удален
    And история проверок доступна в архиве
    And retention период еще не истек
    When пользователь запрашивает историю монитора через архив
    Then история монитора возвращена из архива
    And данные доступны только для чтения
    And действие в аудит лог записано как "history_accessed_archive"
      | field       | value              |
      | monitor_id  | API Service        |
      | source      | archive            |
      | user_id     | <user_id>          |
      | timestamp   | <iso8601>          |

  @use_case=uc_01_05_03a
  @critical
  Scenario: Попытка просмотра истории чужого монитора a
    Given пользователь "UserA" имеет монитор "API Service" с историей проверок
    And пользователь "UserB" авторизован
    When пользователь "UserB" запрашивает историю монитора "API Service"
    Then система возвращает ошибку "FORBIDDEN"
    And история не показана

  @use_case=uc_01_05_03b
  @critical
  Scenario: Попытка просмотра статистики чужого монитора a
    Given пользователь "UserA" имеет монитор "API Service" с результатами проверок
    And пользователь "UserB" авторизован
    When пользователь "UserB" запрашивает статистику response time монитора "API Service"
    Then система возвращает ошибку "FORBIDDEN"
    And статистика не показана

  @use_case=uc_01_05_03c
  @critical
  Scenario: Попытка просмотра инцидентов чужого монитора a
    Given пользователь "UserA" имеет монитор "API Service" с инцидентами
    And пользователь "UserB" авторизован
    When пользователь "UserB" запрашивает историю инцидентов монитора "API Service"
    Then система возвращает ошибку "FORBIDDEN"
    And инциденты не показаны

  @use_case=uc_01_05_01h
  Scenario: Попытка запроса истории с нелегальным диапазоном дат a
    Given пользователь имеет монитор "API Service"
    When пользователь запрашивает историю с "2025-01-01" по "9999-12-31"
    Then система возвращает ошибку "INVALID_DATE_RANGE"
    And запрос отклонен

  @use_case=uc_01_05_12
  @critical
  @recovery
  Scenario: Graceful degradation при высокой нагрузке на историю
    Given пользователь имеет монитор "API Service" с "100000" результатов
    And система под высокой нагрузкой
    When пользователь запрашивает историю за "30 days"
    Then запрос выполнен с пагинацией
    And результаты возвращены постранично
    And система не degraded
