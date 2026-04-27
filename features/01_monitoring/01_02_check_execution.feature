@epic=01_monitoring
@user_story=01_02_check_execution
# Description: Выполнение HTTP проверок и измерение response times
#
# Architecture Note:
#   Начиная с v1.0, проверка выполняется распределённо:
#   - scheduler-service (Epic 09) планирует проверки и назначает воркерам
#   - check-worker (Epic 10) выполняет HTTP-запрос и отправляет результат
#   - monitor-service принимает результат через SubmitCheckResult и публикует события
#
#   Бизнес-правила статусов (UP/DOWN/DEGRADED) описаны в этом файле.
#   Операционное выполнение описано в Epic 10 (10_02_check_execution).
#   Планирование описано в Epic 09 (09_02_check_distribution).

Feature: Выполнение проверок
  Как система
  Я хочу выполнять запланированные проверки
  Чтобы обнаруживать проблемы с доступностью сервисов

  @use_case=uc_01_02_01
  @critical
  Scenario: Успешная HTTP проверка
    Given монитор для "https://api.example.com" с интервалом "5 minutes"
    And сервис отвечает со статусом 200 OK
    When выполняется проверка
    Then результат должен быть "UP"
    And время ответа записано
    And статус код равен 200
    And действие в аудит лог записано как "check_completed"
      | field        | value                |
      | monitor_url  | https://api.example.com |
      | status       | UP                   |
      | status_code  | 200                  |
      | response_time| <response_time>      |
      | timestamp    | <iso8601>            |

  # Integration: Emits state change event to Epic 02 (Alerting) for alert evaluation
  # See: Epic 02, uc_02_02_20
  @use_case=uc_01_02_02
  @critical
  Scenario: Неудачная HTTP проверка с ошибкой 500
    Given монитор для "https://api.example.com"
    And сервис отвечает со статусом 500 Internal Server Error
    When выполняется проверка
    Then результат должен быть "DOWN"
    And алерт должен быть инициирован
    And действие в аудит лог записано как "check_failed"
      | field        | value                |
      | monitor_url  | https://api.example.com |
      | status       | DOWN                 |
      | status_code  | 500                  |
      | error        | Internal Server Error|
      | timestamp    | <iso8601>            |

  @use_case=uc_01_02_03
  @critical
  Scenario: Проверка с таймаутом
    Given монитор для "https://api.example.com" с таймаутом "10 seconds"
    And сервис не отвечает в течение таймаута
    When выполняется проверка
    Then результат должен быть "DOWN"
    And ошибка содержит код "CONNECTION_TIMEOUT"
    And действие в аудит лог записано как "check_timeout"
      | field        | value                |
      | monitor_url  | https://api.example.com |
      | status       | DOWN                 |
      | error        | CONNECTION_TIMEOUT   |
      | timeout_value| 10 seconds           |
      | timestamp    | <iso8601>            |

  @use_case=uc_01_02_04
  Scenario: Health check для checker worker'а
    Given checker worker запущен
    When запрашивается health check
    Then worker должен быть "healthy"

  @use_case=uc_01_02_01a
  @critical
  Scenario: Проверка с кодом 4xx по умолчанию
    Given монитор для "https://api.example.com"
    And настройка 4xx кодов = "400+ as DOWN"
    And сервис отвечает со статусом 404 Not Found
    When выполняется проверка
    Then результат должен быть "DOWN"
    And ошибка содержит "404"

  @use_case=uc_01_02_01b
  Scenario: Проверка с кодом 4xx как UP (настраиваемо)
    Given монитор для "https://api.example.com"
    And настройка 4xx кодов = "4xx as UP"
    And сервис отвечает со статусом 404 Not Found
    When выполняется проверка
    Then результат должен быть "UP"
    And статус код равен 404

  # DEGRADED TRIGGER LOGIC (OR condition):
  # Monitor becomes DEGRADED when EITHER condition is true:
  # 1. Response time > degraded_response_time threshold (default: 5s)
  # 2. Failure rate > degraded_failure_rate threshold (default: 10%)
  # If BOTH conditions met → still DEGRADED (show both reasons)
  # Priority: UP < DEGRADED < DOWN
  @use_case=uc_01_02_05
  Scenario: DEGRADED статус при медленном ответе (OR condition #1)
    Given монитор для "https://api.example.com"
    And degraded_response_time threshold = "5 seconds"
    And degraded_failure_rate threshold = "10%"
    And сервис отвечает с задержкой "6 seconds"
    And failure rate = "5%" (below threshold)
    When выполняется проверка
    Then результат должен быть "DEGRADED"
    And причина = "Response time exceeded threshold (6s > 5s)"
    And время ответа записано
    And ошибка содержит код "DEGRADED_SLOW_RESPONSE"

  @use_case=uc_01_02_06
  Scenario: DEGRADED статус при проценте failed проверок (OR condition #2)
    Given монитор для "https://api.example.com"
    And degraded_response_time threshold = "5 seconds"
    And degraded_failure_rate threshold = "10%"
    And сервис отвечает с задержкой "2 seconds" (below threshold)
    And за последние "10" проверок failed "2" (20% failure rate)
    When выполняется проверка
    Then результат должен быть "DEGRADED"
    And причина = "Failure rate exceeded threshold (20% > 10%)"
    And ошибка содержит код "DEGRADED_HIGH_FAILURE_RATE"

  @use_case=uc_01_02_07
  Scenario: Working hours - проверка вне рабочего времени
    Given монитор для "https://api.example.com"
    And рабочие часы настроены: "09:00-18:00" по будням
    And текущее время "2026-03-10T20:00:00Z" (воскресенье)
    When планируется проверка
    Then проверка не запланирована
    And статус монитора не изменяется

  @use_case=uc_01_02_08
  Scenario: Working hours - проверка в рабочее время
    Given монитор для "https://api.example.com"
    And рабочие часы настроены: "09:00-18:00" по будням
    And текущее время "2026-03-08T14:00:00Z" (понедельник)
    When выполняется проверка
    Then проверка выполняется
    And результат записан

  @use_case=uc_01_02_03a
  @critical
  Scenario: Проверка с ошибкой соединения (DNS failure)
    Given монитор для "https://nonexistent-domain-12345.com"
    When выполняется проверка
    Then результат должен быть "DOWN"
    And ошибка содержит код "DNS_RESOLUTION_FAILED"

  @use_case=uc_01_02_03b
  @critical
  Scenario: Проверка с истекшим SSL сертификатом
    Given монитор для "https://expired-cert.example.com"
    And SSL сертификат истек
    When выполняется проверка
    Then результат должен быть "DOWN"
    And ошибка содержит код "SSL_CERTIFICATE_EXPIRED"

  @use_case=uc_01_02_01c
  @critical
  Scenario: Проверка с пустым ответом
    Given монитор для "https://api.example.com"
    And сервис отвечает со статусом 200
    And тело ответа пустое
    When выполняется проверка
    Then результат должен быть "UP"
    And ошибка содержит код "EMPTY_RESPONSE"

  @use_case=uc_01_02_03c
  @critical
  Scenario: Проверка с ответом превышающим лимит
    Given монитор для "https://api.example.com"
    And сервис возвращает "100 MB" данных
    And максимальный размер ответа "10 MB"
    When выполняется проверка
    Then результат должен быть "DOWN"
    And ошибка содержит код "RESPONSE_TOO_LARGE"

  @use_case=uc_01_02_03d
  @critical
  Scenario: Проверка с циклическим редиректом
    Given монитор для "https://api.example.com"
    And сервис возвращает бесконечный редирект
    When выполняется проверка
    Then результат должен быть "DOWN"
    And ошибка содержит код "TOO_MANY_REDIRECTS"

  # Integration: Checks Epic 08 (Maintenance) for active maintenance windows
  # See: Epic 08, uc_08_01_26
  @use_case=uc_01_02_09
  @critical
  Scenario: Проверка во время окна обслуживания
    Given монитор для "https://api.example.com"
    And активно окно обслуживания для монитора
    When выполняется проверка
    Then проверка пропущена
    And статус не изменяется
    And ошибка содержит код "CHECK_SKIPPED_MAINTENANCE"
    And причина пропуска записана в лог

  @use_case=uc_01_02_10
  Scenario: Health check для неработающего checker worker'а
    Given checker worker запущен
    And worker имеет внутреннюю ошибку
    When запрашивается health check
    Then worker должен быть "unhealthy"
    And возвращена диагностика ошибки
    And ошибка содержит код "WORKER_UNHEALTHY"

  @use_case=uc_01_02_03e
  @critical
  @integration
  Scenario: Сбой подключения к базе данных во время выполнения проверки e
    Given монитор для "https://api.example.com"
    And подключение к базе данных потеряно
    When выполняется проверка
    Then результат проверки не сохранён
    And ошибка содержит код "DATABASE_CONNECTION_FAILED"
    And результат помещён в очередь для повторной попытки

  @use_case=uc_01_02_03f
  @critical
  @integration
  Scenario: Таймаут сервиса хранения результатов f
    Given монитор для "https://api.example.com"
    And сервис хранения результатов не отвечает в течение таймаута
    When выполняется проверка
    Then результат не сохранён
    And ошибка содержит код "STORAGE_TIMEOUT"
    And результат помещён в очередь для повторной попытки

  @use_case=uc_01_02_03g
  @critical
  @recovery
  Scenario: Восстановление после кратковременного сетевого сбоя
    Given монитор для "https://api.example.com"
    And предыдущая проверка завершилась с ошибкой "NETWORK_ERROR"
    When сетевое подключение восстановлено
    And выполняется следующая проверка
    Then проверка выполняется успешно
    And результат сохранён
    And ошибка содержит код "NETWORK_RECOVERY_SUCCESS"

  @use_case=uc_01_02_03h
  @critical
  @recovery
  Scenario: Повторная попытка выполнения неудачной проверки
    Given монитор для "https://api.example.com"
    And проверка завершилась с ошибкой "CONNECTION_ERROR"
    When выполняется retry попытка
    Then проверка повторяется через "1 minute"
    And максимальное количество retry попыток "3"
    And ошибка содержит код "RETRY_SCHEDULED"
    And retry attempt записан в лог с номером попытки

  @use_case=uc_01_02_11
  @critical
  @boundary
  Scenario: Проверка с таймаутом точно на границе лимита
    Given монитор для "https://api.example.com" с таймаутом "30 seconds"
    And сервис отвечает ровно через "30 seconds"
    When выполняется проверка
    Then результат должен быть "DOWN"
    And ошибка содержит код "CONNECTION_TIMEOUT"

  @use_case=uc_01_02_01d
  @critical
  @integration
  Scenario: Недействительность кеша конфигурации монитора
    Given монитор для "https://api.example.com"
    And конфигурация монитора изменена
    And кеш конфигурации устарел
    When выполняется проверка
    Then используется актуальная конфигурация
    And кеш обновлён
    And результат проверен с новой конфигурацией
    And ошибка содержит код "CACHE_INVALIDATED"
    And событие инвалидации кеша записано в лог

  @use_case=uc_01_02_02a
  @critical
  @boundary
  Scenario: Проверка при N-1 неудачных попытках перед алертом
    Given монитор для "https://api.example.com"
    And правило алерта с consecutive_failures = "3"
    When монитор падает "2" раза подряд (N-1)
    Then результат должен быть "DOWN"
    And алерт не создан (требуется ещё 1 неудача)
    And монитор помечен как "FAILED"
    And ошибка содержит код "ALERT_NOT_TRIGGERED"
    And счётчик неудач сохранён в контексте монитора
