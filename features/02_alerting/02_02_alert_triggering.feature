@epic=02_alerting
@user_story=02_02_alert_triggering
# Description: Триггеры алертов и правила

Feature: Триггеры алертов
  Как пользователь
  Я хочу настраивать правила алертов
  Чтобы получать уведомления при проблемах

  Background:
    Given пользователь авторизован
    And пользователь имеет монитор "API Service"
    And пользователь имеет канал "Ops Team"

  # Integration: Subscribes to monitor state change events from Epic 01 (Monitoring)
  # See: Epic 01, uc_01_02_02
  @use_case=uc_02_02_01
  @critical
  @audit
  Scenario: Создание правила алерта
    When пользователь создаёт правило алерта с параметрами:
      | monitor_id           | API Service |
      | channel_ids          | Ops Team    |
      | consecutive_failures | 2           |
    Then правило алерта создано успешно
    And действие в аудит лог записано как "alert_rule_created" с полями:
      | field                | value                |
      | monitor_id           | API Service          |
      | channel_ids          | Ops Team             |
      | consecutive_failures | 2                    |
      | user_id              | текущий пользователь |

  @use_case=uc_02_02_02
  @critical
  Scenario: Настройка consecutive failures (1-5, default 2)
    When пользователь создаёт правило алерта с параметрами:
      | monitor_id           | API Service |
      | channel_ids          | Ops Team    |
      | consecutive_failures | 3           |
    Then правило алерта создано успешно
    And consecutive failures установлено в "3"

  @use_case=uc_02_02_03
  @critical
  @validation
  Scenario: Попытка установки consecutive failures вне диапазона 1-5
    When пользователь создаёт правило алерта с параметрами:
      | monitor_id           | API Service |
      | channel_ids          | Ops Team    |
      | consecutive_failures | 10          |
    Then возвращена ошибка "ALERT_INVALID_CONSECUTIVE_FAILURES"
    And описание ошибки "consecutive_failures must be between 1 and 5"

  @use_case=uc_02_02_04
  @critical
  Scenario: Создание правила с default consecutive failures (2)
    When пользователь создаёт правило алерта с параметрами:
      | monitor_id  | API Service |
      | channel_ids | Ops Team    |
    Then правило алерта создано успешно
    And consecutive failures установлено в "2" по умолчанию

  @use_case=uc_02_02_05
  @critical
  Scenario: Список правил алертов для монитора
    Given пользователь имеет правила алертов для монитора "API Service"
    When пользователь запрашивает правила алертов
    Then пользователь получает список правил
    And каждое правило содержит канал оповещений

  @use_case=uc_02_02_06
  @critical
  @audit
  Scenario: Подтверждение активного алерта
    Given активный алерт для монитора "API Service"
    When пользователь подтверждает алерт
    Then статус алерта "ACKNOWLEDGED"
    And уведомления не отправляются
    And действие в аудит лог записано как "alert_acknowledged" с полями:
      | field        | value                |
      | alert_id     | id алерта            |
      | monitor_id   | API Service          |
      | user_id      | текущий пользователь |
      | timestamp    | время подтверждения  |

  @use_case=uc_02_02_07
  @critical
  @business_rule
  @audit
  Scenario: Новый алерт при повторном падении во время ACKNOWLEDGED
    # Alert state transitions:
    # - Monitor DOWN → Create new alert with status TRIGGERED
    # - Monitor UP during TRIGGERED → Update alert to RESOLVED
    # - Monitor DOWN during ACKNOWLEDGED → Create new alert with status TRIGGERED
    # - Monitor UP during ACKNOWLEDGED → Update alert to RESOLVED
    Given алерт для монитора "API Service" со статусом "ACKNOWLEDGED"
    And монитор снова падает (consecutive failures достигнуто)
    When система создаёт новый алерт
    Then создан новый алерт со статусом "TRIGGERED"
    And старый алерт остаётся "ACKNOWLEDGED"
    And отправлено новое уведомление
    And действие в аудит лог записано как "alert_retriggered" с полями:
      | field                | value                      |
      | new_alert_id         | id нового алерта           |
      | previous_alert_id    | id старого алерта          |
      | monitor_id           | API Service                |
      | previous_alert_status| ACKNOWLEDGED               |

  @use_case=uc_02_02_08
  @critical
  @audit
  Scenario: Автоматический переход в RESOLVED после 1 успешной проверки
    Given активный алерт для монитора "API Service"
    When монитор успешно проверяется (1 раз)
    Then статус алерта "RESOLVED"
    And отправлено уведомление о восстановлении
    And действие в аудит лог записано как "alert_resolved" с полями:
      | field        | value                |
      | alert_id     | id алерта            |
      | monitor_id   | API Service          |
      | old_status   | TRIGGERED            |
      | new_status   | RESOLVED             |

  @use_case=uc_02_02_09
  @critical
  @boundary
  Scenario: Хранение RESOLVED алертов 90 дней
    Given алерт со статусом "RESOLVED" создан "91 days" назад
    When система выполняет очистку старых алертов
    Then алерт удалён из базы

  @use_case=uc_02_02_10
  @critical
  @boundary
  Scenario: RESOLVED алерт младше 90 дней не удаляется
    Given алерт со статусом "RESOLVED" создан "89 days" назад
    When система выполняет очистку старых алертов
    Then алерт не удалён

  @use_case=uc_02_02_11
  @critical
  Scenario: Список алертов для монитора
    Given монитор "API Service" имеет алерты за последние "7 days"
    When пользователь запрашивает алерты
    Then пользователь получает список алертов
    And каждый алерт содержит время срабатывания
    And каждый алерт содержит статус

  @use_case=uc_02_02_12
  @critical
  Scenario: Фильтрация алертов по статусу
    Given монитор "API Service" имеет алерты
    And алерты имеют статусы "TRIGGERED" и "RESOLVED"
    When пользователь запрашивает алерты со статусом "TRIGGERED"
    Then возвращаются только алерты со статусом "TRIGGERED"

  @use_case=uc_02_02_13
  @critical
  @business_rule
  Scenario: Cooldown период между алертами (15 минут) - иерархия throttling
    # THROTTLING HIERARCHY (highest to lowest priority):
    # 1. Alert Storm Detection: System-wide, 5+ monitors fail in 5 minutes (rolling window)
    # 2. Rate Limiting: Per-monitor, per-status, max 1 alert per 5 minutes
    # 3. Cooldown: Per-monitor, all statuses, min 15 minutes between alerts
    #
    # ALGORITHM:
    # ON monitor failure (consecutive_failures >= threshold):
    #   IF (storm_count_in_5min >= 5) THEN
    #     APPLY alert storm grouping (overrides all other limits)
    #   ELSE IF (time_since_last_alert_same_status < 5min) THEN
    #     QUEUE alert for delivery (rate limit)
    #   ELSE IF (time_since_last_alert_any_status < 15min) THEN
    #     DO NOT create new alert, update existing (cooldown)
    #   ELSE
    #     CREATE and send alert immediately
    #
    # KEY DISTINCTIONS:
    # - Rate limit: Per monitor, per status (DOWN and DEGRADED tracked separately)
    # - Cooldown: Per monitor, all statuses (any alert resets 15min cooldown)
    # - Alert storm: System-wide, can override rate limit and cooldown
    Given алерт для монитора "API Service" со статусом "DOWN" создан "5 minutes" назад
    And монитор снова падает (consecutive failures достигнуто)
    When система проверяет cooldown период (любой статус монитора)
    Then новый алерт не создан (cooldown период: 15 минут с последнего алерта любого статуса)
    And предыдущий алерт со статусом "DOWN" остаётся активным
    And incident counter обновлён (total failures увеличен)

  @use_case=uc_02_02_14
  @critical
  @business_rule
  Scenario: Создание нового алерта после cooldown периода
    Given алерт для монитора "API Service" создан "16 minutes" назад
    And cooldown период (15 минут) истёк
    And монитор снова падает
    And consecutive failures достигнуто
    When система проверяет cooldown и rate limit
    Then создан новый алерт (cooldown истёк, rate limit не применяется)
    And отправлено новое уведомление
    And timestamp последнего алерта обновлён

  @use_case=uc_02_02_15
  @critical
  Scenario: Mute алертов - пользователь отключает уведомления себе
    Given пользователь имеет активное правило алерта для монитора "API Service"
    When пользователь отключает алерты для себя
    Then пользователь не получает уведомления
    And правило алерта остаётся активным для других пользователей

  @use_case=uc_02_02_16
  @critical
  Scenario: Mute алертов - админ отключает все уведомления
    Given пользователь с ролью "ADMIN"
    And активное правило алерта для монитора "API Service"
    When админ отключает все алерты
    Then никто не получает уведомления
    And все правила алерта временно отключены
    And действие в аудит лог записано как "alerts_muted_global"
      | field        | value                |
      | monitor_id   | API Service          |
      | mute_scope   | global               |
      | admin_id     | <admin_id>           |
      | timestamp    | <iso8601>            |

  @use_case=uc_02_02_17
  @critical
  @business_rule
  Scenario: Определение FLAPPING статуса (5+ переключений за 10 минут)
    Given монитор "API Service" переключается между UP и DOWN
    And произошло "6" переключений за "10 minutes"
    When система детектирует flapping
    Then статус монитора "FLAPPING"
    And создан специальный алерт "FLAPPING"
    And действие в аудит лог записано как "flapping_detected"
      | field              | value                |
      | monitor_id         | API Service          |
      | flap_count         | 6                    |
      | detection_period   | 10 minutes           |
      | new_status         | FLAPPING             |
      | timestamp          | <iso8601>            |

  @use_case=uc_02_02_18
  @critical
  @integration
  Scenario: Отправка уведомления о FLAPPING статусе
    Given статус монитора изменился на "FLAPPING"
    When отправляется уведомление
    Then уведомление содержит статус "FLAPPING"
    And уведомление содержит количество переключений
    And уведомление содержит рекомендацию проверить настройки

  @use_case=uc_02_02_19
  @critical
  @business_rule
  Scenario: Выход из FLAPPING статуса
    Given монитор "API Service" в статусе "FLAPPING"
    And монитор стабилен в течение "15 minutes"
    When система обновляет статус
    Then статус монитора изменился на "UP"
    And отправлено уведомление о восстановлении стабильности

  @use_case=uc_02_02_20
  @critical
  @business_rule
  @audit
  Scenario: Алерт срабатывает точно при N consecutive failures
    # ALERT TRIGGERING ALGORITHM:
    #
    # Preconditions:
    # - consecutive_failures counter >= threshold
    # - No active alert already exists for this monitor+status combination
    # - Rate limit check passed (5 min since last alert of same status)
    # - Cooldown check passed (15 min since last alert of any status)
    # - Alert storm not detected (less than 5 monitors failed in 5 min)
    #
    # ON all checks passed:
    #   CREATE new alert with status = TRIGGERED
    #   SEND notification to all configured channels
    #   LOG audit event "alert_triggered"
    #   UPDATE last_alert_timestamp[monitor_id][status] = NOW()
    #   UPDATE last_alert_timestamp[monitor_id][any_status] = NOW()
    Given правило алерта с consecutive_failures = "3"
    And монитор успешно проверился (counter = 0)
    And cooldown период истёк (более 15 min с последнего алерта)
    When монитор падает ровно "3" раза подряд
    Then consecutive failures counter = 3
    And создан алерт со статусом "TRIGGERED"
    And отправлено уведомление
    And действие в аудит лог записано как "alert_triggered" с полями:
      | field                | value                |
      | alert_id             | id нового алерта      |
      | monitor_id           | API Service          |
      | consecutive_failures | 3                    |
      | status               | TRIGGERED            |
      | cooldown_remaining   | 15 minutes           |

  @use_case=uc_02_02_21
  @critical
  @business_rule
  @validation
  Scenario: Отказ в создании алерта при N-1 consecutive failures
    Given правило алерта с consecutive_failures = "3"
    And монитор успешно проверился (counter = 0)
    When монитор падает "2" раза подряд
    Then consecutive failures counter = 2
    And алерт не создан (threshold: 3, current: 2)
    And монитор помечен как "FAILED" но без алерта
    And возвращена ошибка "ALERT_INVALID_CONSECUTIVE_FAILURES" при попытке создания

  @use_case=uc_02_02_36
  @critical
  @business_rule
  Scenario: Счётчик consecutive failures - полный цикл сброса
    # CONSECUTIVE FAILURES COUNTER RESET RULES:
    #
    # ON each check result:
    #   IF (check.status = 'FAILED') THEN
    #     counter = counter + 1
    #     IF (counter >= threshold) THEN
    #       CREATE alert (if not already created)
    #       // Alert creation does NOT reset counter
    #     END IF
    #   ELSE IF (check.status = 'SUCCESS') THEN
    #     counter = 0  // First successful check resets counter to 0
    #     IF (alert exists AND status in [TRIGGERED, ACKNOWLEDGED]) THEN
    #       alert.status = 'RESOLVED'
    #       SEND recovery notification
    #     END IF
    #   END IF
    #
    # KEY RULES:
    # 1. Counter resets to 0 on FIRST successful check (not gradual)
    # 2. Alert creation does NOT reset counter (counter continues counting)
    # 3. After reset, counter starts fresh from 0 on next failure
    # 4. Acknowledge does NOT reset counter (only monitor success resets it)
    Given правило алерта с consecutive_failures = "3"
    And монитор успешно проверился (counter = 0)
    When монитор падает "1" раз
    Then consecutive failures counter = 1
    And алерт не создан (threshold не достигнут)

    When монитор падает "2" раз подряд
    Then consecutive failures counter = 2
    And алерт не создан (threshold не достигнут)

    When монитор падает "3" раз подряд
    Then consecutive failures counter = 3
    And создан алерт со статусом "TRIGGERED"
    And counter НЕ сброшен после создания алерта

    When монитор падает "4" раз подряд
    Then consecutive failures counter = 4
    And новый алерт не создан (уже есть активный для этой streak)

    When следующая проверка успешна
    Then consecutive failures counter сброшен в "0"
    And алерт помечен как "RESOLVED"
    And отправлено уведомление о восстановлении

    When монитор снова падает
    Then consecutive failures counter = 1 (начинается с 1, не продолжается)

  @use_case=uc_02_02_22
  @critical
  @business_rule
  Scenario: Переход монитора из DEGRADED в DOWN
    Given монитор "API Service" в статусе "DEGRADED"
    And правило алерта настроено на оба статуса
    When монитор переходит в статус "DOWN"
    Then создан новый алерт для статуса "DOWN"
    And предыдущий алерт "DEGRADED" закрыт
    And отправлено уведомление об ухудшении статуса

  @use_case=uc_02_02_23
  @critical
  @business_rule
  Scenario: FLAPPING не детектируется при 4 переключениях
    Given монитор переключается между UP и DOWN
    And произошло "4" переключения за "10 minutes"
    When система детектирует flapping
    Then статус монитора не изменён на "FLAPPING"
    And созданы обычные алерты для каждого падения

  @use_case=uc_02_02_24
  @critical
  @business_rule
  Scenario: Mute алертов с автоматическим отключением
    Given активное правило алерта для монитора "API Service"
    When пользователь отключает алерты на "2 hours"
    Then алерты отключены
    And через "2 hours" алерты автоматически включены
    And пользователь уведомлён о повторной активации

  @use_case=uc_02_02_25
  @critical
  @business_rule
  Scenario: Автоматическое escalating при отсутствии ACK
    Given критический алерт для монитора "API Service"
    And алерт не подтверждён в течение "30 minutes"
    When система проверяет время подтверждения
    Then алерт автоматически escalates на следующий уровень
    And отправлено уведомление escalation

  @use_case=uc_02_02_26
  @critical
  @security
  @audit
  Scenario: Отказ в подтверждении чужого алерта неавторизованным пользователем
    Given активный алерт для монитора "API Service" принадлежит пользователю "user1"
    And пользователь "user2" с ролью "USER"
    When пользователь "user2" пытается подтвердить алерт
    Then возвращена ошибка "FORBIDDEN"
    And алерт остаётся со статусом "TRIGGERED"
    And действие в аудит лог записано как "unauthorized_alert_ack_attempt" с полями:
      | field        | value                |
      | user_id      | user2                |
      | alert_owner  | user1                |
      | alert_id     | id алерта            |
      | action       | acknowledge_alert    |
      | status       | denied               |

  @use_case=uc_02_02_27
  @critical
  @security
  @audit
  Scenario: Отказ в глобальном отключении алертов для роли USER
    Given пользователь с ролью "USER"
    And активное правило алерта для монитора "API Service"
    When пользователь пытается отключить все алерты
    Then возвращена ошибка "FORBIDDEN"
    And алерты продолжают отправляться
    And действие в аудит лог записано как "unauthorized_global_mute_attempt" с полями:
      | field        | value                |
      | user_role    | USER                 |
      | action       | global_mute_alerts   |
      | status       | denied               |

  @use_case=uc_02_02_28
  @critical
  @security
  @audit
  Scenario: Отказ в доступе к чужим алертам при изоляции пользователей
    Given пользователь "user1" имеет монитор "Service1"
    And пользователь "user2" имеет монитор "Service2"
    And настроено правило алерта для монитора "Service1"
    When монитор "Service1" падает
    Then создан алерт для "user1"
    And пользователь "user2" не видит этот алерт
    And действие в аудит лог записано как "alert_access_control" с полями:
      | field        | value                |
      | alert_id     | id алерта            |
      | monitor_id   | Service1             |
      | owner        | user1                |
      | access_level | private              |

  @use_case=uc_02_02_30
  @critical
  @integration
  @audit
  Scenario: Недоступен сервис очереди сообщений при создании алерта
    Given активное правило алерта для монитора "API Service"
    And очередь сообщений недоступна
    When монитор падает (consecutive failures достигнуто)
    Then алерт создан в базе данных со статусом "TRIGGERED"
    And отправка уведомления провалена
    And алерт помещён в очередь для отправки при восстановлении
    And ошибка содержит код "ALERT_MESSAGE_QUEUE_UNAVAILABLE"
    And действие в аудит лог записано как "alert_queue_deferred" с полями:
      | field                | value                              |
      | alert_id             | id алерта                          |
      | error_code           | ALERT_MESSAGE_QUEUE_UNAVAILABLE    |
      | queue_status         | deferred_for_retry                 |

  @use_case=uc_02_02_31
  @critical
  @integration
  @audit
  Scenario: Таймаут базы данных при создании алерта
    Given активное правило алерта для монитора "API Service"
    And база данных не отвечает в течение таймаута (30 seconds)
    When монитор падает (consecutive failures достигнуто)
    Then алерт не создан
    And попытка создания повторена через "30 seconds" с экспоненциальной задержкой
    And ошибка содержит код "ALERT_DATABASE_TIMEOUT"
    And действие в аудит лог записано как "alert_creation_retry_scheduled" с полями:
      | field                | value                              |
      | monitor_id           | API Service                        |
      | error_code           | ALERT_DATABASE_TIMEOUT             |
      | retry_delay          | 30 seconds                         |
      | retry_strategy       | exponential_backoff                |

  @use_case=uc_02_02_32
  @critical
  @boundary
  @business_rule
  @audit
  Scenario: Превышение rate limit при отправке алертов
    # RATE LIMITING RULES:
    # - Per monitor, per status (DOWN, DEGRADED tracked separately)
    # - Max 1 alert per 5 minutes per monitor+status combination
    # - Queued alerts sent after rate limit expires, not discarded
    # - Rate limit checked AFTER alert storm detection (lower priority)
    Given правило алерта для монитора "API Service"
    And алерт со статусом "DOWN" отправлен "1 minute" назад
    When монитор снова падает (consecutive failures достигнуто)
    Then алерт создан в базе данных со статусом "TRIGGERED"
    And отправка отложена (rate limit: 1 alert per 5 minutes per status)
    And алерт поставлен в очередь для отправки через "4 minutes"
    And действие в аудит лог записано как "alert_rate_limited" с полями:
      | field                | value                        |
      | alert_id             | id алерта                    |
      | monitor_id           | API Service                  |
      | alert_status         | DOWN                         |
      | rate_limit           | 1 per 5 minutes              |
      | time_until_delivery   | 4 minutes                    |
      | queue_position       | <position>                   |

  @use_case=uc_02_02_33
  @critical
  @concurrency
  @audit
  Scenario: Конкурентный запрос на подтверждение одного алерта несколькими пользователями
    Given активный алерт для монитора "API Service"
    And пользователь "user1" начинает подтверждение алерта
    And пользователь "user2" начинает подтверждение того же алерта
    When оба пользователя одновременно отправляют подтверждение
    Then только один запрос на подтверждение выполнен успешно
    And второй запрос отклонён с ошибкой "ALERT_ALREADY_ACKNOWLEDGED"
    And статус алерта "ACKNOWLEDGED" с первым пользователем
    And действие в аудит лог записано как "concurrent_acknowledge_conflict" с полями:
      | field              | value                      |
      | alert_id           | id алерта                  |
      | conflict_type      | simultaneous_acknowledge   |
      | requests_count     | 2                          |
      | acknowledged_by    | <first_user_id>            |
      | rejected_requests  | 1                          |
      | timestamp          | <iso8601>                  |

  @use_case=uc_02_02_34
  @critical
  @boundary
  @audit
  Scenario: Таймаут автоматического escalation при отсутствии подтверждения
    Given критический алерт для монитора "API Service" со статусом "TRIGGERED"
    And время подтверждения истекло "30 minutes" назад
    When система проверяет время подтверждения
    Then алерт автоматически escalates на следующий уровень
    And отправлено уведомление escalation на канал "L2"
    And уровень escalation увеличен на "1"
    And действие в аудит лог записано как "alert_auto_escalated" с полями:
      | field              | value                |
      | alert_id           | id алерта            |
      | escalation_level   | 2                    |
      | previous_level     | 1                    |
      | escalation_reason  | timeout_no_ack       |
      | timeout_duration   | 30 minutes           |
      | new_channel        | L2                   |
      | timestamp          | <iso8601>            |

  @use_case=uc_02_02_35
  @critical
  @boundary
  @audit
  Scenario: Превышение порога FLAPPING при ровно 5 переключениях за 10 минут
    # FLAPPING DETECTION ALGORITHM:
    #
    # Use rolling window (last 10 minutes from current time):
    #   window_start = NOW() - 10 minutes
    #   flap_count = COUNT(status_changes WHERE timestamp >= window_start)
    #
    # IF (flap_count >= 5) THEN
    #   monitor.status = "FLAPPING"
    #   CREATE special alert with status "FLAPPING"
    #   Supress normal alerts until stable for 15 minutes
    # END IF
    #
    # Exit FLAPPING when:
    #   - No status changes for 15 minutes
    #   - monitor.status = last known status (UP or DOWN)
    Given монитор "API Service" переключается между UP и DOWN
    And произошло "5" переключений за "10 minutes" (rolling window)
    When система выполняет проверку на flapping
    Then статус монитора изменён на "FLAPPING"
    And создан специальный алерт "FLAPPING"
    And последующие переключения не создают новые алерты (подавлены)
    And действие в аудит лог записано как "flapping_threshold_boundary" с полями:
      | field              | value                |
      | monitor_id         | API Service          |
      | flap_count         | 5                    |
      | threshold          | 5                    |
      | detection_period   | 10 minutes           |
      | window_type        | rolling              |
      | boundary_status    | exact_threshold_met  |
      | timestamp          | <iso8601>            |

  @use_case=uc_02_02_37
  @critical
  @integration
  @audit
  Scenario: Сбой сервиса уведомлений при создании алерта
    Given активное правило алерта для монитора "API Service"
    And сервис уведомлений недоступен
    When монитор падает (consecutive failures достигнуто)
    Then алерт создан в базе данных со статусом "TRIGGERED"
    And отправка уведомления провалена
    And алерт помещён в очередь для отправки при восстановлении
    And ошибка содержит код "DELIVERY_NOTIFICATION_SERVICE_UNAVAILABLE"
    And действие в аудит лог записано как "alert_created_notification_failed" с полями:
      | field                | value                                    |
      | alert_id             | id алерта                                |
      | status               | TRIGGERED                                |
      | notification_status  | queued_for_retry                         |
      | error_code           | DELIVERY_NOTIFICATION_SERVICE_UNAVAILABLE|
      | queue_position       | <position_in_queue>                      |
      | timestamp            | <iso8601>                                |

  @use_case=uc_02_02_39
  @critical
  Scenario: Включение правила алерта
    Given пользователь имеет правило алерта для монитора "API Service"
    And правило алерта отключено
    When пользователь включает правило алерта
    Then правило алерта активировано
    And алерты снова отправляются

  @use_case=uc_02_02_40
  @critical
  Scenario: Отключение правила алерта
    Given пользователь имеет правило алерта для монитора "API Service"
    And правило алерта включено
    When пользователь отключает правило алерта
    Then правило алерта деактивировано
    And алерты не отправляются

  @use_case=uc_02_02_38
  @critical
  @integration
  @audit
  Scenario: Сбой при частичной отправке алерта в несколько каналов
    Given алерт для монитора "API Service"
    And настроены каналы "Ops Team" (Telegram) и "Incident" (Webhook)
    And Telegram канал доступен
    And Webhook канал возвращает ошибку "500 Internal Server Error"
    When алерт отправляется во все каналы
    Then алерт успешно доставлен в канал "Ops Team"
    And доставка в канал "Incident" помечена как FAILED
    And выполнена retry попытка для webhook
    And действие в аудит лог записано как "alert_partial_delivery" с полями:
      | field                | value                        |
      | alert_id             | id алерта                    |
      | successful_channels  | 1                            |
      | failed_channels      | 1                            |
      | total_channels       | 2                            |
      | delivery_status      | partial_success              |
      | retry_scheduled      | true                         |

  @use_case=uc_02_02_41
  @critical
  @validation
  Scenario: Попытка создания правила алерта с несуществующим монитором
    Given монитор "NonExistent" не существует
    When пользователь создаёт правило алерта с параметрами:
      | monitor_id  | NonExistent |
      | channel_ids | Ops Team    |
    Then возвращена ошибка "MONITOR_NOT_FOUND"
    And правило алерта не создано

  @use_case=uc_02_02_42
  @critical
  @validation
  Scenario: Попытка создания правила алерта с несуществующим каналом
    When пользователь создаёт правило алерта с параметрами:
      | monitor_id  | API Service     |
      | channel_ids | NonExistentChan |
    Then возвращена ошибка "ALERT_CHANNEL_NOT_FOUND"
    And правило алерта не создано

  @use_case=uc_02_02_43
  @critical
  @security
  @audit
  Scenario: Попытка создания правила для монитора другого пользователя
    Given монитор "Service X" принадлежит пользователю "user1"
    And пользователь "user2" авторизован
    When пользователь "user2" создаёт правило алерта для монитора "Service X"
    Then возвращена ошибка "FORBIDDEN"
    And правило алерта не создано
    And действие в аудит лог записано как "unauthorized_rule_creation_attempt" с полями:
      | field        | value                |
      | user_id      | user2                |
      | monitor_owner| user1                |
      | action       | create_alert_rule    |
      | status       | denied               |

  @use_case=uc_02_02_44
  @critical
  @security
  @audit
  Scenario: Попытка удаления правила другого пользователя
    Given пользователь "user1" имеет правило алерта для монитора "API Service"
    And пользователь "user2" авторизован
    When пользователь "user2" пытается удалить правило пользователя "user1"
    Then возвращена ошибка "FORBIDDEN"
    And правило алерта не удалено
    And действие в аудит лог записано как "unauthorized_rule_deletion_attempt" с полями:
      | field        | value                |
      | user_id      | user2                |
      | rule_owner   | user1                |
      | action       | delete_alert_rule    |
      | status       | denied               |

  @use_case=uc_02_02_45
  @critical
  @validation
  Scenario: Попытка установки consecutive_failures = 0 (ниже минимума)
    When пользователь создаёт правило алерта с параметрами:
      | monitor_id           | API Service |
      | channel_ids          | Ops Team    |
      | consecutive_failures | 0           |
    Then возвращена ошибка "ALERT_INVALID_CONSECUTIVE_FAILURES"
    And описание ошибки "consecutive_failures must be between 1 and 5"
    And правило алерта не создано

  @use_case=uc_02_02_46
  @critical
  @validation
  Scenario: Попытка установки consecutive_failures = 6 (выше максимума)
    When пользователь создаёт правило алерта с параметрами:
      | monitor_id           | API Service |
      | channel_ids          | Ops Team    |
      | consecutive_failures | 6           |
    Then возвращена ошибка "ALERT_INVALID_CONSECUTIVE_FAILURES"
    And описание ошибки "consecutive_failures must be between 1 and 5"
    And правило алерта не создано

  @use_case=uc_02_02_47
  @critical
  @integration
  @audit
  Scenario: Ошибка базы данных при создании алерта
    Given активное правило алерта для монитора "API Service"
    And база данных возвращает ошибку при записи
    When монитор падает (consecutive failures достигнуто)
    Then алерт не создан
    And ошибка содержит код "ALERT_DATABASE_ERROR"
    And действие в аудит лог записано как "alert_creation_failed" с полями:
      | field        | value                |
      | monitor_id   | API Service          |
      | error_code   | ALERT_DATABASE_ERROR |
      | retry_queued | true                 |

  @use_case=uc_02_02_48
  @critical
  @validation
  Scenario: Попытка создания дублирующегося правила алерта
    Given пользователь имеет правило алерта для монитора "API Service"
    When пользователь создаёт ещё одно правило алерта для монитора "API Service"
    Then возвращена ошибка "ALERT_RULE_ALREADY_EXISTS"
    And второе правило не создано

  @use_case=uc_02_02_49
  @critical
  @boundary
  @audit
  Scenario: Alert storm detection - ровно 5 мониторов за 5 минут (граничное значение)
    Given мониторы "S1", "S2", "S3", "S4", "S5" упали в течение "5 minutes"
    And это ровно "5" уникальных мониторов (граничное значение)
    When система проверяет alert storm
    Then детектирован alert storm (порог = 5, текущее = 5)
    And все алерты сгруппированы в одно сообщение
    And действие в аудит лог записано как "alert_storm_boundary_detected" с полями:
      | field              | value                |
      | monitors_affected  | 5                    |
      | threshold          | 5                    |
      | boundary_status    | exact_threshold      |
      | timestamp          | <iso8601>            |

  @use_case=uc_02_02_50
  @critical
  @boundary
  @audit
  Scenario: Alert storm detection - 4 монитора НЕ вызывают storm (N-1)
    Given мониторы "S1", "S2", "S3", "S4" упали в течение "5 minutes"
    And это "4" уникальных монитора (N-1 от порога 5)
    When система проверяет alert storm
    Then alert storm НЕ детектирован (4 < 5)
    And алерты отправляются индивидуально
    And действие в аудит лог записано как "alert_storm_not_triggered" с полями:
      | field              | value                |
      | monitors_count     | 4                    |
      | threshold          | 5                    |
      | storm_triggered    | false                |

  @use_case=uc_02_02_51
  @critical
  @validation
  Scenario: Попытка подтверждения уже RESOLVED алерта
    Given алерт для монитора "API Service" со статусом "RESOLVED"
    When пользователь пытается подтвердить алерт
    Then возвращена ошибка "ALERT_NOT_ACKNOWLEDGEABLE"
    And статус алерта остаётся "RESOLVED"

  @use_case=uc_02_02_52
  @critical
  @validation
  Scenario: Попытка подтверждения алерта в статусе MUTED
    Given алерт для монитора "API Service" со статусом "MUTED"
    When пользователь пытается подтвердить алерт
    Then возвращена ошибка "ALERT_NOT_ACKNOWLEDGEABLE"
    And статус алерта остаётся "MUTED"

  @use_case=uc_02_02_53
  @critical
  @business_rule
  Scenario: Алерт не отправляется если монитор заглушён (muted)
    Given активный мute для монитора "API Service" в области "USER"
    And правило алерта активно для монитора "API Service"
    When монитор падает (consecutive failures достигнуто)
    Then алерт создан с статусом "MUTED"
    And уведомление не отправлено пользователю с активным mute

  @use_case=uc_02_02_54
  @critical
  @integration
  @audit
  Scenario: Rate limit DOWN и DEGRADED статусы независимы
    Given алерт "DOWN" отправлен "1 minute" назад для монитора "API Service"
    When монитор переходит в статус "DEGRADED"
    Then алерт "DEGRADED" отправляется успешно (независимый rate limit)
    And rate limit для "DOWN" остаётся активным
    And действие в аудит лог записано как "alert_degraded_independent_rate_limit" с полями:
      | field              | value                |
      | monitor_id         | API Service          |
      | down_rate_limited  | true                 |
      | degraded_sent      | true                 |

  @use_case=uc_02_02_55
  @critical
  @concurrency
  @audit
  Scenario: Конкурентный запрос на создание правила алерта дважды
    Given пользователь начинает создание правила алерта для монитора "API Service"
    And параллельный запрос начинает создание того же правила
    When оба запроса выполняются одновременно
    Then только одно правило алерта создано успешно
    And второй запрос отклонён с ошибкой "ALERT_RULE_ALREADY_EXISTS"
    And действие в аудит лог записано как "concurrent_rule_creation_conflict" с полями:
      | field              | value                |
      | monitor_id         | API Service          |
      | conflict_type      | duplicate_creation   |
      | created            | 1                    |
      | rejected           | 1                    |

  @use_case=uc_02_02_56
  @critical
  @business_rule
  Scenario: Сброс cooldown при успешном восстановлении монитора
    Given алерт для монитора "API Service" создан "5 minutes" назад (cooldown активен)
    When монитор успешно восстанавливается (статус UP)
    And алерт помечен как RESOLVED
    And монитор снова падает через "1 minute"
    Then cooldown НЕ применяется (cooldown сбрасывается при RESOLVED)
    And создан новый алерт со статусом "TRIGGERED"

  @use_case=uc_02_02_57
  @critical
  @boundary
  Scenario: Попытка создания правила без указания каналов
    When пользователь создаёт правило алерта с параметрами:
      | monitor_id  | API Service |
      | channel_ids |             |
    Then возвращена ошибка "MISSING_REQUIRED_FIELD"
    And описание ошибки "channel_ids cannot be empty"
    And правило алерта не создано

  @use_case=uc_02_02_58
  @critical
  @integration
  @audit
  Scenario: Недоступен сервис алертов при обработке события мониторинга
    Given активное правило алерта для монитора "API Service"
    And сервис алертов недоступен
    When монитор падает (consecutive failures достигнуто)
    Then событие помещено в retry очередь
    And ошибка содержит код "ALERT_SERVICE_UNAVAILABLE"
    And действие в аудит лог записано как "alert_service_unavailable" с полями:
      | field        | value                      |
      | monitor_id   | API Service                |
      | error_code   | ALERT_SERVICE_UNAVAILABLE  |
      | queued       | true                       |

  @use_case=uc_02_02_59
  @critical
  @security
  @audit
  Scenario: Отказ в просмотре чужих правил алертов
    Given пользователь "user1" имеет правила алертов
    And пользователь "user2" авторизован
    When пользователь "user2" запрашивает правила пользователя "user1"
    Then возвращена ошибка "FORBIDDEN"
    And правила пользователя "user1" не возвращены
    And действие в аудит лог записано как "unauthorized_rules_access_attempt" с полями:
      | field        | value                |
      | user_id      | user2                |
      | target_user  | user1                |
      | action       | list_alert_rules     |
      | status       | denied               |

  @use_case=uc_02_02_60
  @critical
  @validation
  Scenario: Попытка создания правила с пустым monitor_id
    When пользователь создаёт правило алерта с параметрами:
      | monitor_id  |          |
      | channel_ids | Ops Team |
    Then возвращена ошибка "MISSING_REQUIRED_FIELD"
    And описание ошибки "monitor_id cannot be empty"
    And правило алерта не создано

  @use_case=uc_02_02_61
  @critical
  @business_rule
  @audit
  Scenario: Автоматическое эскалирование с несколькими уровнями
    Given критический алерт для монитора "API Service" на уровне эскалации "1"
    And уровень "1" не подтверждён в течение "30 minutes"
    And уровень эскалации "2" настроен с каналом "L2 Team"
    When система проверяет время эскалации
    Then алерт автоматически escalates на уровень "2"
    And отправлено уведомление в канал "L2 Team"
    And уровень эскалации увеличен до "2"
    And действие в аудит лог записано как "alert_multi_level_escalated" с полями:
      | field              | value                |
      | alert_id           | id алерта            |
      | escalation_level   | 2                    |
      | previous_level     | 1                    |
      | escalation_reason  | timeout_no_ack       |
      | timeout_duration   | 30 minutes           |
      | new_channel        | L2 Team              |
