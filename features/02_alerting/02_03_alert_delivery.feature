@epic=02_alerting
@user_story=02_03_alert_delivery
# Description: Доставка алертов в каналы оповещений

Feature: Доставка алертов
  Как система
  Я хочу доставлять алерты в настроенные каналы
  Чтобы пользователи получали уведомления

  Background:
    Given настроенный канал "Ops Team" типа Telegram
    And настроенный канал "Admin Email" типа Email

  @use_case=uc_02_03_01
  @critical
  @audit
  Scenario: Отправка алерта в Telegram канал
    Given алерт для монитора "API Service"
    And алерт имеет статус "TRIGGERED"
    When алерт отправляется в канал "Ops Team"
    Then Telegram сообщение доставлено
    And сообщение содержит имя монитора
    And сообщение содержит статус
    And сообщение получен message_id
    And действие в аудит лог записано как "alert_delivered" с полями:
      | field        | value                |
      | alert_id     | id алерта            |
      | channel_id   | id канала Ops Team   |
      | channel_type | telegram             |
      | message_id   | id сообщения         |
      | status       | delivered            |

  @use_case=uc_02_03_02
  @critical
  Scenario: Отправка алерта в Email канал
    Given алерт для монитора "API Service"
    And алерт имеет статус "TRIGGERED"
    When алерт отправляется в канал "Admin Email"
    Then email отправлен
    And email содержит подробности алерта

  @use_case=uc_02_03_03
  @critical
  Scenario: Отправка алерта в Webhook канал
    Given настроенный Webhook канал "Incident Webhook"
    And алерт для монитора "API Service"
    When алерт отправляется в канал "Incident Webhook"
    Then POST запрос выполнен
    And payload содержит детали алерта
    And получен HTTP ответ 200

  @use_case=uc_02_03_04
  @critical
  Scenario: Уведомление о восстановлении
    Given активный алерт для монитора "API Service"
    And монитор возвращается в состояние "UP"
    When отправляется уведомление о восстановлении
    Then отправлено сообщение "Recovery"
    And сообщение содержит текущий статус "UP"

  @use_case=uc_02_03_05
  @critical
  @business_rule
  @audit
  Scenario: Блокировка отправки алерта rate limiting (1 алерт/5 минут/статус)
    # RATE LIMITING RULES:
    # - Per monitor, per status (DOWN and DEGRADED tracked separately)
    # - Max 1 alert per 5 minutes per monitor+status combination
    # - Queued alerts sent after rate limit expires, not discarded
    # - Checked AFTER alert storm detection (lower priority in hierarchy)
    Given алерт со статусом "DOWN" для монитора "API Service" отправлен "1 minute" назад
    And монитор снова падает (consecutive failures достигнуто)
    When система проверяет rate limit для статуса "DOWN"
    Then алерт не отправлен немедленно (rate limit)
    And алерт поставлен в очередь для отправки через "4 minutes"
    And в очереди alert storm проверка не выполняется (ещё не 5 алертов)
    And действие в аудит лог записано как "alert_rate_limited"
      | field           | value                |
      | alert_id        | id алерта            |
      | monitor_id      | API Service          |
      | alert_status    | DOWN                 |
      | rate_limit      | 1 per 5 minutes      |
      | queued          | true                 |
      | time_until_delivery | 4 minutes       |
      | timestamp       | <iso8601>            |

  @use_case=uc_02_03_06
  @critical
  @business_rule
  Scenario: Отправка алерта после окончания rate limit периода
    Given алерт для монитора "API Service" отправлен "5 minutes" назад
    And rate limit период истёк
    And монитор снова падает
    When consecutive failures достигнуто
    Then алерт отправлен успешно (rate limit истёк)

  @use_case=uc_02_03_07
  @critical
  @business_rule
  Scenario: Отказ в повторной отправке DOWN алерта при активном rate limit
    # Rate limiting is per-monitor, per-status
    # DOWN and DEGRADED alerts are rate limited independently
    Given монитор "API Service" в статусе "DOWN"
    And алерт "DOWN" уже отправлен для этого статуса "1 minute" назад
    When монитор продолжает быть в статусе "DOWN"
    Then новые алерты "DOWN" не отправляются (rate limit active)
    And один flow отправки до изменения статуса

  @use_case=uc_02_03_08
  @critical
  Scenario: Доставка в разные типы каналов
    Given алерт для монитора "API Service"
    And настроены каналы:
      | name       | type    |
      | Ops Team   | Telegram |
      | Admin Email| Email    |
      | Incident   | Webhook  |
    When алерт отправляется во все каналы
    Then Telegram сообщение доставлено
    And email отправлен
    And webhook POST запрос выполнен успешно
    And действие в аудит лог записано как "alert_multi_type_delivered"
      | field           | value                      |
      | alert_id        | id алерта                  |
      | channel_types   | telegram, email, webhook   |
      | delivery_status | all_successful             |
      | timestamp       | <iso8601>                  |

  @use_case=uc_02_03_09
  @critical
  @audit
  Scenario: Неудачный retry доставки в Telegram с экспоненциальной задержкой
    # RETRY BACKOFF STRATEGY FOR TRANSIENT ERRORS:
    #
    # Error types: TIMEOUT, CONNECTION_ERROR, SERVICE_UNAVAILABLE
    # Strategy: EXPONENTIAL_BACKOFF_WITH_JITTER
    # Base delay: 1 minute
    # Max retries: 3
    # Formula: delay = MIN(base_delay * (2 ^ attempt), 10 minutes) + random_jitter(±20%)
    #
    # Example:
    #   Attempt 0: delay = 1 min + jitter = ~50-70 seconds
    #   Attempt 1: delay = 2 min + jitter = ~96-144 seconds
    #   Attempt 2: delay = 4 min + jitter = ~192-288 seconds
    Given алерт для монитора "API Service"
    And канал "Ops Team" типа Telegram временно недоступен (TIMEOUT)
    When первая попытка доставки провалена
    Then retry #1 scheduled через "~60 seconds" (1 min + jitter)
    And действие в аудит лог записано как "alert_delivery_retry" с полями:
      | field           | value                |
      | alert_id        | id алерта            |
      | channel_id      | id канала Ops Team   |
      | channel_type    | telegram             |
      | retry_attempt   | 1                    |
      | max_retries     | 3                    |
      | retry_delay     | ~60 seconds          |
      | retry_strategy  | exponential_backoff  |
      | timestamp       | <iso8601>            |

    When retry #1 провален
    Then retry #2 scheduled через "~120 seconds" (2 min + jitter)

    When retry #2 провален
    Then retry #3 scheduled через "~240 seconds" (4 min + jitter)

    When retry #3 провален
    Then все retry исчерпаны (max 3)
    And создан incident о недоставленном алерте
    And действие в аудит лог записано как "delivery_retry_exhausted" с полями:
      | field           | value                |
      | alert_id        | id алерта            |
      | channel_id      | id канала Ops Team   |
      | retry_count     | 3                    |
      | incident_status | created              |

  @use_case=uc_02_03_10
  @critical
  @business_rule
  @audit
  Scenario: Отказ в retry доставки Webhook при постоянной 4xx ошибке
    # PERMANENT ERRORS - NO RETRY:
    # Error codes: 400, 401, 403, 404, INVALID_EMAIL, AUTH_FAILED, SSL_CERTIFICATE_INVALID
    # Action: Immediate failure, mark channel as DISABLED or INVALID
    Given алерт для монитора "API Service"
    And канал "Incident" типа Webhook
    When webhook возвращает HTTP "400"
    Then отправка помечена как FAILED
    And retry не выполняется (клиентская ошибка)
    And действие в аудит лог записано как "delivery_no_retry_permanent_error" с полями:
      | field           | value                |
      | alert_id        | id алерта            |
      | channel_id      | id канала Incident   |
      | http_status     | 400                  |
      | error_category  | permanent            |
      | retry_scheduled | false                |

  @use_case=uc_02_03_11
  @critical
  @boundary
  @audit
  Scenario: Alert storm detection - rolling window algorithm
    # ALERT STORM DETECTION ALGORITHM:
    #
    # WINDOW_TYPE = Rolling window (last 5 minutes from current time)
    # STORM_THRESHOLD = 5 unique monitors failed
    # STORM_DURATION = 10 minutes (once triggered)
    #
    # ON each monitor failure (consecutive_failures >= threshold):
    #   current_time = NOW()
    #   window_start = current_time - 5 minutes
    #
    #   failed_monitors = QUERY(
    #     SELECT DISTINCT monitor_id
    #     FROM monitor_checks
    #     WHERE status = 'FAILED'
    #     AND timestamp >= window_start
    #     AND timestamp <= current_time
    #     AND consecutive_failures >= alert_threshold
    #   )
    #
    #   IF (COUNT(failed_monitors) >= 5) THEN
    #     TRIGGER alert storm
    #     storm_state = ACTIVE
    #     storm_start_time = current_time
    #     affected_monitors = failed_monitors
    #   END IF
    #
    # KEY SPECIFICATIONS:
    # - Rolling window: [NOW-5min, NOW], evaluated in real-time
    # - Count unique monitor IDs (same monitor failing twice counts once)
    # - Storm duration: 10 minutes once triggered
    # - Grouping: All alerts during storm grouped into single notification
    # - Priority: HIGHEST (overrides rate limit and cooldown)
    Given монитор "API Service" упал в "12:00"
    And монитор "Database" упал в "12:01"
    And монитор "Cache" упал в "12:02"
    And монитор "Frontend" упал в "12:03"
    And монитор "CDN" упал в "12:04"
    When текущее время "12:04"
    And система проверяет alert storm (rolling window: 12:04-5min to 12:04)
    Then детектирован alert storm (5 уникальных мониторов за 5 минут)
    And storm_state установлен в "ACTIVE"
    And storm_duration равен "10 minutes"
    And все последующие алерты группируются в одно сообщение
    And количество отправленных сообщений "1"
    And действие в аудит лог записано как "alert_storm_detected" с полями:
      | field              | value                |
      | monitors_affected  | 5                    |
      | storm_trigger      | 5+ failures in 5 min |
      | storm_duration     | 10 minutes           |
      | window_type        | rolling              |
      | grouped_alert_id   | id группированного алерта |
      | messages_sent      | 1                    |
      | timestamp          | <iso8601>            |

  @use_case=uc_02_03_12
  @critical
  @integration
  Scenario: Логирование результатов доставки
    Given алерт для монитора "API Service"
    And настроены каналы "Ops Team" и "Admin Email"
    When алерт отправляется в каналы
    Then результаты доставки залогированы
    And результаты содержат статус для каждого канала
    And результаты содержат timestamp отправки
    And результаты содержат message_id/идентификатор

  @use_case=uc_02_03_13
  @critical
  @integration
  @audit
  Scenario: Ошибка доставки Telegram при превышении квоты
    # RATE LIMIT ERRORS - RETRY WITH FIXED LONG DELAY:
    # Error codes: 429, RATE_LIMIT_EXCEEDED
    # Strategy: FIXED_DELAY with jitter
    # Base delay: 5 minutes
    # Max retries: 5
    # Formula: delay = 5 minutes + random_jitter(±10%)
    Given алерт для монитора "API Service"
    And канал "Ops Team" типа Telegram
    When Telegram API возвращает ошибку "429 Too Many Requests"
    Then отправка помечена как FAILED
    And выполнена retry попытка через "~5 minutes" (fixed delay + jitter)
    And пользователь видит ошибку "DELIVERY_RATE_LIMIT_EXCEEDED"
    And действие в аудит лог записано как "alert_delivery_rate_limit_exceeded" с полями:
      | field           | value                       |
      | alert_id        | id алерта                   |
      | channel_id      | id канала Ops Team          |
      | http_status     | 429                         |
      | error           | Too Many Requests           |
      | error_code      | DELIVERY_RATE_LIMIT_EXCEEDED|
      | retry_scheduled | ~5 minutes                  |
      | retry_strategy  | fixed_delay                 |
      | timestamp       | <iso8601>                   |

  @use_case=uc_02_03_14
  @critical
  @integration
  @audit
  Scenario: Блокировка Telegram бота пользователем при доставке алерта
    Given алерт для монитора "API Service"
    And канал "Ops Team" типа Telegram
    When Telegram API возвращает ошибку "403 Forbidden - bot was blocked"
    Then отправка помечена как FAILED
    And retry не выполняется (перманентная ошибка)
    And канал автоматически отключён
    And пользователь уведомлён о проблеме с каналом
    And действие в аудит лог записано как "channel_disabled_permanent" с полями:
      | field           | value                      |
      | channel_id      | id канала Ops Team         |
      | channel_type    | telegram                   |
      | error           | 403 Forbidden - bot blocked|
      | error_code      | DELIVERY_PERMANENT_ERROR   |
      | disable_reason  | permanent_failure          |
      | new_status      | DISABLED                   |

  @use_case=uc_02_03_15
  @critical
  @integration
  @audit
  Scenario: Ошибка доставки Email при несуществующем ящике
    Given алерт для монитора "API Service"
    And канал "Admin Email" с email "nonexistent@example.com"
    When email provider возвращает "550 Mailbox not found"
    Then отправка помечена как FAILED
    And retry не выполняется (перманентная ошибка)
    And канал помечен как "INVALID_EMAIL"
    And действие в аудит лог записано как "alert_delivery_email_failed" с полями:
      | field           | value                       |
      | alert_id        | id алерта                   |
      | channel_id      | id канала Admin Email       |
      | channel_type    | email                       |
      | smtp_response   | 550 Mailbox not found       |
      | error_code      | DELIVERY_PERMANENT_ERROR    |
      | channel_status  | INVALID_EMAIL               |
      | timestamp       | <iso8601>                   |

  @use_case=uc_02_03_16
  @critical
  @integration
  @audit
  Scenario: Таймаут Webhook при ожидании ответа от endpoint
    Given алерт для монитора "API Service"
    And канал "Incident" типа Webhook
    And webhook timeout равен "10 seconds"
    When endpoint не отвечает в течение "10 seconds"
    Then отправка помечена как TIMEOUT
    And выполнена retry попытка (временная ошибка)
    And лог содержит информацию о таймауте
    And действие в аудит лог записано как "alert_delivery_webhook_timeout" с полями:
      | field           | value                       |
      | alert_id        | id алерта                   |
      | channel_id      | id канала Incident          |
      | channel_type    | webhook                     |
      | timeout_value   | 10 seconds                  |
      | error           | connection_timeout          |
      | error_code      | DELIVERY_TIMEOUT            |
      | retry_scheduled | true                        |
      | timestamp       | <iso8601>                   |

  @use_case=uc_02_03_17
  @critical
  @integration
  @audit
  Scenario: Ошибка Webhook 5xx вызывает retry с экспоненциальной задержкой
    # TRANSIENT ERRORS - RETRY WITH EXPONENTIAL BACKOFF:
    # Error types: 5xx errors, TIMEOUT, CONNECTION_ERROR
    # Strategy: EXPONENTIAL_BACKOFF_WITH_JITTER
    # Base delay: 1 minute
    # Max retries: 3
    # Formula: delay = MIN(base_delay * (2 ^ attempt), 10 min) + jitter(±20%)
    Given алерт для монитора "API Service"
    And канал "Incident" типа Webhook
    When webhook возвращает HTTP "500 Internal Server Error"
    Then отправка помечена как FAILED
    And выполнена retry попытка через "~1 minute" (exponential backoff)
    And максимальное количество retry "3"
    And действие в аудит лог записано как "alert_delivery_webhook_5xx_retry" с полями:
      | field           | value                       |
      | alert_id        | id алерта                   |
      | channel_id      | id канала Incident          |
      | channel_type    | webhook                     |
      | http_status     | 500                         |
      | retry_attempt   | 1                           |
      | max_retries     | 3                           |
      | retry_delay     | ~60 seconds                 |
      | retry_strategy  | exponential_backoff        |
      | timestamp       | <iso8601>                   |

  @use_case=uc_02_03_18
  @critical
  @integration
  @audit
  Scenario: Сбой доставки при исчерпании всех retry попыток
    Given алерт для монитора "API Service"
    And канал "Ops Team" временно недоступен
    When выполнены "3" неудачные retry попытки
    Then алерт помечен как "DELIVERY_FAILED"
    And создан incident о недоставленном алерте
    And администратор уведомлён о проблеме доставки
    And действие в аудит лог записано как "alert_delivery_incident" с полями:
      | field           | value                |
      | alert_id        | id алерта            |
      | channel_id      | id канала Ops Team   |
      | retry_count     | 3                    |
      | incident_status | created              |

  @use_case=uc_02_03_19
  @critical
  @security
  @audit
  Scenario: Ошибка аутентификации Webhook
    Given алерт для монитора "API Service"
    And канал "Incident" типа Webhook с API key
    When webhook возвращает HTTP "401 Unauthorized"
    Then отправка помечена как FAILED
    And retry не выполняется (ошибка конфигурации)
    And канал помечен как "AUTH_FAILED"
    And администратор уведомлён о необходимости проверить credentials
    And действие в аудит лог записано как "webhook_auth_failed" с полями:
      | field        | value                |
      | alert_id     | id алерта            |
      | channel_id   | id канала Incident   |
      | http_status  | 401                  |
      | error        | Unauthorized         |
      | error_code   | DELIVERY_PERMANENT_ERROR |
      | channel_status| AUTH_FAILED         |

  @use_case=uc_02_03_20
  @critical
  @integration
  Scenario: Подавление алертов во время maintenance window
    Given активное maintenance window для монитора "API Service"
    And maintenance window продлится ещё "10 minutes"
    When монитор падает
    Then алерт не отправляется
    And алерт сохранён как "SUPPRESSED_BY_MAINTENANCE"
    And после окончания maintenance отправлен summary

  @use_case=uc_02_03_21
  @critical
  @security
  @audit
  Scenario: Невалидный SSL/TLS сертификат Webhook блокирует доставку
    Given алерт для монитора "API Service"
    And канал "Incident" типа Webhook с URL "https://self-signed-cert.example.com/webhook"
    And endpoint использует self-signed сертификат
    When алерт отправляется в канал "Incident"
    Then отправка помечена как FAILED
    And возвращена ошибка "DELIVERY_SSL_CERTIFICATE_INVALID"
    And действие в аудит лог записано как "webhook_ssl_error" с полями:
      | field        | value                                      |
      | alert_id     | id алерта                                  |
      | channel_id   | id канала Incident                         |
      | url          | https://self-signed-cert.example.com/webhook|
      | error        | SSL certificate verification failed        |
      | error_code   | DELIVERY_SSL_CERTIFICATE_INVALID           |
      | status       | failed                                     |

  @use_case=uc_02_03_22
  @critical
  @integration
  @audit
  Scenario: Сбой Email сервиса при отправке алерта
    # EMAIL SERVICE ERRORS - RETRY WITH EXPONENTIAL BACKOFF:
    # Error types: SERVICE_UNAVAILABLE, temporary SMTP errors
    # Strategy: EXPONENTIAL_BACKOFF_WITH_JITTER
    # Base delay: 2 minutes
    # Max retries: 5
    # Formula: delay = MIN(base_delay * (2 ^ attempt), 15 min) + jitter(±20%)
    Given алерт для монитора "API Service"
    And канал "Admin Email" типа Email
    When Email сервис возвращает ошибку "Service Unavailable"
    Then отправка помечена как FAILED
    And выполнена retry попытка через "~2 minutes" (exponential backoff)
    And максимальное количество retry "5"
    And ошибка содержит код "DELIVERY_EMAIL_SERVICE_UNAVAILABLE"
    And действие в аудит лог записано как "alert_delivery_email_service_unavailable" с полями:
      | field           | value                                 |
      | alert_id        | id алерта                             |
      | channel_id      | id канала Admin Email                 |
      | channel_type    | email                                 |
      | error_code      | DELIVERY_EMAIL_SERVICE_UNAVAILABLE    |
      | retry_scheduled | ~2 minutes                            |
      | max_retries     | 5                                     |
      | retry_strategy  | exponential_backoff                   |
      | timestamp       | <iso8601>                             |

  @use_case=uc_02_03_23
  @critical
  @concurrency
  @audit
  Scenario: Конкурентный запрос доставки одного алерта в несколько каналов
    Given алерт для монитора "API Service"
    And настроены каналы "Ops Team", "Admin Email", "Incident"
    And все три канала начинают доставку одновременно
    When система обрабатывает параллельную доставку
    Then все три доставки выполнены без конфликтов
    And каждая доставка имеет уникальный message_id
    And результат доставки содержит статусы для всех каналов
    And действие в аудит лог записано как "alert_concurrent_delivery" с полями:
      | field                  | value                        |
      | alert_id               | id алерта                    |
      | concurrent_channels    | 3                            |
      | delivery_strategy      | parallel                     |
      | successful_deliveries  | 3                            |
      | conflicts              | 0                            |
      | timestamp              | <iso8601>                    |

  @use_case=uc_02_03_24
  @critical
  @boundary
  @audit
  Scenario: Превышение лимита очереди доставки при alert storm
    Given система обрабатывает alert storm
    And очередь доставки достигла лимита "1000" сообщений
    When новые алерты продолжают поступать
    Then новые доставки помещены в очередь с приоритетом
    And самые старые доставки обрабатываются в первую очередь
    And возвращено предупреждение "DELIVERY_QUEUE_HIGH_LOAD"
    And действие в аудит лог записано как "delivery_queue_overflow" с полями:
      | field                | value                        |
      | queue_size           | 1000                         |
      | queue_limit          | 1000                         |
      | pending_alerts       | <pending_count>              |
      | priority_strategy    | fifo_with_priority           |
      | warning_code         | DELIVERY_QUEUE_HIGH_LOAD     |
      | timestamp            | <iso8601>                    |

  @use_case=uc_02_03_25
  @critical
  @integration
  @audit
  Scenario: Сбой частичной доставки при пакетной отправке алертов
    Given "10" алертов готовы к отправке в канал "Ops Team"
    And канал "Ops Team" поддерживает пакетную отправку
    When система выполняет пакетную отправку
    Then "7" алертов из "10" доставлены успешно
    And "3" алерта не доставлены из-за временного сбоя
    And недоставленные алерты помещены в retry очередь
    And действие в аудит лог записано как "alert_batch_partial_delivery" с полями:
      | field                | value                        |
      | channel_id           | id канала Ops Team           |
      | batch_size           | 10                           |
      | successful_count     | 7                            |
      | failed_count         | 3                            |
      | delivery_status      | partial_success              |
      | retry_scheduled      | true                         |
      | timestamp            | <iso8601>                    |

  @use_case=uc_02_03_26
  @critical
  @concurrency
  @audit
  Scenario: Конкурентный запрос на обновление статуса доставки алерта
    Given алерт для монитора "API Service" доставляется в канал "Ops Team"
    And процесс доставки возвращает статус "SENT"
    When параллельный процесс обновляет статус того же алерта
    Then только один статус обновления применён
    And финальный статус доставки соответствует последнему обновлению
    And действие в аудит лог записано как "concurrent_delivery_status_update" с полями:
      | field                | value                        |
      | alert_id             | id алерта                    |
      | channel_id           | id канала Ops Team           |
      | conflict_type        | concurrent_status_update     |
      | update_strategy      | last_write_wins              |
      | timestamp            | <iso8601>                    |

  @use_case=uc_02_03_27
  @critical
  @boundary
  @audit
  Scenario: Таймаут доставки при медленном webhook endpoint
    Given алерт для монитора "API Service"
    And канал "Incident" типа Webhook
    And webhook timeout равен "10 seconds"
    And endpoint отвечает за "15 seconds"
    When алерт отправляется в канал "Incident"
    Then отправка помечена как TIMEOUT
    And выполнена retry попытка
    And лог содержит информацию о превышении timeout
    And действие в аудит лог записано как "alert_delivery_timeout" с полями:
      | field                | value                        |
      | alert_id             | id алерта                    |
      | channel_id           | id канала Incident           |
      | timeout_value        | 10 seconds                   |
      | actual_duration      | 15 seconds                   |
      | error_code           | DELIVERY_TIMEOUT             |
      | retry_scheduled      | true                         |
      | timestamp            | <iso8601>                    |
