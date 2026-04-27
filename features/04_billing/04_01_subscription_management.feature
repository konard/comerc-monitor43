@epic=04_billing
@user_story=04_01_subscription_management
# Description: Управление тарифными планами и подписками

Feature: Управление подписками
  Как пользователь
  Я хочу управлять своей подпиской
  Чтобы получать доступ к функциям мониторинга

  @use_case=uc_04_01_01
  @critical
  Scenario: Просмотр текущей подписки
    Given пользователь авторизован
    And пользователь имеет подписку "Starter"
    When пользователь запрашивает информацию о подписке
    Then система возвращает статус "ACTIVE"
    And система возвращает название тарифа "Starter"
    And система возвращает дату окончания периода

  @use_case=uc_04_01_02
  @critical
  Scenario: Получение списка тарифных планов
    Given пользователь авторизован
    When пользователь запрашивает список тарифных планов
    Then возвращается не менее "4" планов
    And каждый план содержит название, описание и цену

  @use_case=uc_04_01_03
  @critical
  Scenario: Upgrade тарифного плана
    Given пользователь имеет подписку "Free"
    And тариф "Starter" позволяет "50" мониторов
    When пользователь обновляется до тарифа "Starter"
    Then подписка обновлена успешно
    And лимит мониторов обновлён
    And создана запись audit log о смене тарифа
    And audit log содержит:
      | action         | subscription_upgraded |
      | old_tier       | Free |
      | new_tier       | Starter |
      | initiated_by   | <user_id> |
      | timestamp      | <iso8601> |

  @use_case=uc_04_01_04
  @critical
  Scenario: Отмена подписки
    Given пользователь имеет активную подписку
    When пользователь отменяет подписку
    Then статус подписки "CANCELED"
    And доступ к функциям сохраняется до конца оплаченного периода
    And создана запись audit log об отмене
    And audit log содержит:
      | action         | subscription_canceled |
      | tier           | <plan_name> |
      | reason         | user_initiated |
      | initiated_by   | <user_id> |
      | timestamp      | <iso8601> |

  @use_case=uc_04_01_05
  @critical
  Scenario: Начало trial периода
    Given новый пользователь регистрируется в системе
    When пользователь завершает регистрацию
    Then автоматически создается trial подписка на "14" дней
    And trial предоставляет функции тарифа "Starter"
    And возвращается информация о trial периоде
      | field       | value    |
      | duration    | 14 days  |
      | tier        | Starter  |

  @use_case=uc_04_01_06
  @critical
  @state_transition
  Scenario: Grace period после неудачного платежа
    Given пользователь имеет платную подписку
    And текущий период оплаты завершен
    And платеж не прошел
    Then подписка переходит в статус "GRACE_PERIOD"
    And доступ к функциям сохраняется "14" дней
    And пользователю отправлено уведомление о необходимости оплаты
    And создана запись audit log о grace period
    And audit log содержит:
      | action          | subscription_grace_period |
      | subscription_id | <uuid> |
      | reason          | payment_failed |
      | grace_period_days | 14 |
      | timestamp       | <iso8601> |

  # Integration: Enforces monitor limits that affect Epic 01 (Monitoring) operations
  # See: Epic 01, uc_01_01_01
  @use_case=uc_04_01_07
  @critical
  Scenario: Блокировка при превышении лимита мониторов a
    Given пользователь имеет подписку "Free" с лимитом "25" мониторов
    And пользователь создал "25" мониторов
    When пользователь пытается создать новый монитор
    Then возвращается ошибка с кодом "MONITOR_LIMIT_REACHED"
    And ответ содержит структуру:
      | error.code        | MONITOR_LIMIT_REACHED |
      | error.details.limit | 25 |
      | error.details.current | 25 |
      | error.request_id | <uuid> |
    And создание монитора запрещено
    And создана запись audit log о limit exceeded
    And audit log содержит:
      | action          | monitor_limit_exceeded |
      | user_id         | <user_id> |
      | current_limit   | 25 |
      | current_count   | 25 |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_08
  @critical
  Scenario: Блокировка при превышении лимита пользователей a
    Given пользователь имеет подписку "Starter" с лимитом "5" пользователей
    And в аккаунте "5" пользователей
    When владелец пытается добавить нового пользователя
    Then возвращается ошибка с кодом "USER_LIMIT_EXCEEDED"
    And ответ содержит структуру:
      | error.code        | USER_LIMIT_EXCEEDED |
      | error.details.limit | 5 |
      | error.details.current | 5 |
      | error.request_id | <uuid> |
    And добавление пользователя запрещено
    And создана запись audit log о limit exceeded
    And audit log содержит:
      | action          | user_limit_exceeded |
      | account_id      | <account_id> |
      | current_limit   | 5 |
      | current_count   | 5 |
      | initiated_by    | <user_id> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_09
  @critical
  Scenario: Downgrade тарифного плана
    Given пользователь имеет подписку "Pro" с лимитом "20" пользователей
    And в аккаунте "15" пользователей
    When пользователь понижает тариф до "Starter" с лимитом "5" пользователей
    Then возвращается ошибка с кодом "TOO_MANY_USERS"
    And ответ содержит структуру:
      | error.code          | TOO_MANY_USERS |
      | error.details.limit | 5 |
      | error.details.current | 15 |
      | error.request_id    | <uuid> |
    And downgrade запрещен
    And создана запись audit log о failed downgrade
    And audit log содержит:
      | action          | subscription_downgrade_failed |
      | from_tier       | Pro |
      | to_tier         | Starter |
      | reason          | too_many_users |
      | user_count      | 15 |
      | limit           | 5 |
      | initiated_by    | <user_id> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_10
  Scenario: Получение тарифных планов
    Given пользователь авторизован
    When пользователь запрашивает список тарифных планов
    Then возвращаются планы "Free", "Starter", "Pro", "Enterprise"
    And план "Free" стоит "0" рублей и включает "25" мониторов и "1" пользователя
    And план "Starter" стоит "500" рублей и включает "50" мониторов и "5" пользователей
    And план "Pro" стоит "2000" рублей и включает "20" пользователей
    And план "Enterprise" имеет цену "custom" и безлимитные пользователи

  @use_case=uc_04_01_11
  Scenario: Годовая оплата со скидкой
    Given пользователь выбирает тариф "Starter" за "500" рублей в месяц
    When пользователь выбирает годовую оплату
    Then применяется скидка "20" процентов
    And итоговая стоимость составляет "4800" рублей за год

  @use_case=uc_04_01_12
  Scenario: Proration при upgrade тарифа
    Given пользователь имеет подписку "Starter" с оплаченным периодом
    And до конца оплаченного периода осталось "15" дней
    When пользователь обновляется до тарифа "Pro"
    Then рассчитывается proration (пропорциональное списание)
    And период подписки продляется на полный месяц тарифа "Pro"
    And предыдущая оплата учитывается в расчете

  @use_case=uc_04_01_13
  @critical
  Scenario: Ограничения Free tier
    Given пользователь имеет подписку "Free"
    When пользователь пытается настроить webhook уведомления
    Then возвращается ошибка с кодом "FEATURE_NOT_AVAILABLE_IN_TIER"
    And ответ содержит структуру:
      | error.code              | FEATURE_NOT_AVAILABLE_IN_TIER |
      | error.details.feature   | webhook_notifications |
      | error.details.required_tier | Starter |
      | error.request_id        | <uuid> |
    And доступны только "Email" и "Telegram" уведомления
    And пользователь видит предложение upgrade до платного тарифа
    And создана запись audit log о feature restriction
    And audit log содержит:
      | action          | feature_restricted |
      | user_id         | <user_id> |
      | current_tier    | Free |
      | feature         | webhook_notifications |
      | required_tier   | Starter |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_14
  @critical
  @state_transition
  Scenario: Завершение grace периода
    Given пользователь имеет подписку в статусе "GRACE_PERIOD"
    And grace период активен "14" дней
    And платеж не поступил
    When grace период истекает
    Then подписка переходит в статус "SUSPENDED"
    And все мониторы переходят в статус "PAUSED"
    And пользователю отправлено финальное уведомление
    And создана запись audit log о suspension
    And audit log содержит:
      | action          | subscription_suspended |
      | subscription_id | <uuid> |
      | reason          | grace_period_expired |
      | grace_period_days | 14 |
      | monitors_paused | <count> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_15
  Scenario: Автоматическое продление подписки
    Given пользователь имеет подписку "Starter"
    And период оплаты составляет "1" месяц
    And привязан способ оплаты
    When текущий период завершается
    Then автоматически инициируется платеж
    And при успешной оплате подписка продляется на следующий период
    And пользователю отправлено уведомление о списании средств

  @use_case=uc_04_01_16
  @critical
  @business_rule
  Scenario: Trial expiration with active monitors
    Given пользователь имеет trial подписку на "Starter"
    And пользователь создал "30" мониторов в течение trial
    And trial период истекает
    When trial период завершается
    Then мониторы превышающие лимит Free тарифа приостанавливаются
    And пользователю отправлено предупреждение за "3" дня до окончания
    And пользователю предложено upgrade до платного тарифа
    And минимально важные мониторы остаются активными
    And создана запись audit log о trial expiration
    And audit log содержит:
      | action          | trial_expired |
      | subscription_id | <uuid> |
      | trial_tier      | Starter |
      | monitors_paused | <count> |
      | monitors_active | <count> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_17
  Scenario: Payment on last day of grace period
    Given пользователь имеет подписку в статусе "GRACE_PERIOD"
    And grace период активен "14" дней
    And сегодня "15"й день grace периода
    When пользователь успешно оплачивает в 23:59
    Then подписка переходит в статус "ACTIVE"
    And мониторы не были приостановлены
    And период подписки продлен

  @use_case=uc_04_01_18
  @critical
  @integration
  Scenario: Multiple failed payment retries
    Given пользователь имеет подписку "Starter" с автопродлением
    When первая попытка автоплатежа не удалась
    Then система повторяет попытку через "3" дня
    And создана запись audit log о failed retry
    And audit log содержит:
      | action          | payment_retry_failed |
      | attempt_number  | 1 |
      | subscription_id | <uuid> |
      | next_retry_at   | <date> |
      | timestamp       | <iso8601> |
    When вторая попытка не удалась
    Then система повторяет попытку через "3" дня
    And создана новая запись audit log с attempt_number 2
    And пользователю отправлены уведомления о каждой попытке
    When третья попытка не удалась
    Then подписка переходит в статус "GRACE_PERIOD"
    And создана финальная запись audit log:
      | action          | payment_retries_exhausted |
      | failed_attempts | 3 |
      | grace_period_until | <date> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_19
  Scenario: Proration on mid-cycle downgrade
    Given пользователь имеет подписку "Pro" за "2500" рублей/месяц
    And до конца оплаченного периода осталось "15" дней
    And пользователь имеет "10" пользователей (в лимите Pro)
    When пользователь понижает тариф до "Starter" за "500" рублей/месяц
    Then рассчитывается proration за unused days
    And кредит на будущие периоды составляет "1000" рублей
    And downgrade выполняется успешно
    And пользователь уведомлен о кредите

  @use_case=uc_04_01_20
  Scenario: Cancellation during grace period
    Given пользователь имеет подписку в статусе "GRACE_PERIOD"
    And неоплаченный период за "500" рублей
    When пользователь отменяет подписку
    Then подписка переходит в статус "CANCELED"
    And требование оплаты отменяется
    And доступ сохраняется до конца предыдущего оплаченного периода
    And пользователю не выставляется счет за неоплаченный период

  @use_case=uc_04_01_21
  @critical
  @state_transition
  Scenario: Reactivation after suspension
    Given пользователь имеет подписку в статусе "SUSPENDED"
    And все мониторы в статусе "PAUSED"
    And данные мониторов сохранены "30" дней
    When пользователь инициирует повторный платеж
    Then подписка переходит в статус "ACTIVE"
    And мониторы автоматически возобновляются
    And исторические данные сохранены
    And создана запись audit log о reactivation
    And audit log содержит:
      | action          | subscription_reactivated |
      | subscription_id | <uuid> |
      | payment_amount  | <amount> |
      | monitors_resumed | <count> |
      | initiated_by    | <user_id> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_22
  Scenario: Partial refund for annual subscription
    Given пользователь имеет годовую подписку "Pro" за "24000" рублей
    And прошло "6" месяцев с момента оплаты
    When пользователь запрашивает возврат
    And администратор подтверждает частичный возврат
    Then рассчитывается пропорциональная сумма возврата
    And сумма возврата составляет "12000" рублей
    And подписка переходит в статус "CANCELED"
    And возврат инициируется

  @use_case=uc_04_01_23
  @critical
  Scenario: Upgrade attempt to lower limit plan
    Given пользователь имеет подписку "Pro" с лимитом "150" мониторов
    And пользователь создал "100" мониторов
    When пользователь пытается понизить тариф до "Starter" с лимитом "50"
    Then возвращается ошибка с кодом "TOO_MANY_MONITORS"
    And ответ содержит структуру:
      | error.code           | TOO_MANY_MONITORS |
      | error.details.limit  | 50 |
      | error.details.current | 100 |
      | error.details.excess | 50 |
      | error.request_id     | <uuid> |
    And требуется удалить или приостановить "50" мониторов
    And downgrade запрещен до соответствия лимитам
    And создана запись audit log о failed downgrade
    And audit log содержит:
      | action          | subscription_downgrade_failed |
      | from_tier       | Pro |
      | to_tier         | Starter |
      | reason          | too_many_monitors |
      | monitor_count   | 100 |
      | limit           | 50 |
      | excess          | 50 |
      | initiated_by    | <user_id> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_24
  Scenario: Auto-renewal with expired payment method
    Given пользователь имеет подписку "Starter" с автопродлением
    And привязанная карта истекла
    When наступает дата автопродления
    Then платеж не проходит
    And подписка переходит в статус "GRACE_PERIOD"
    And пользователю отправлено уведомление об истекшей карте
    And предложено обновить платежные данные
    And сохранены данные мониторинга

  @use_case=uc_04_01_25
  @critical
  Scenario: Upgrade during trial period
    Given пользователь имеет trial подписку на "14" дней
    And прошло "7" дней trial периода
    When пользователь обновляется до тарифа "Pro"
    Then trial период завершается досрочно
    And начинается платный период тарифа "Pro"
    And оплата рассчитывается с учетом оставшихся trial дней
    And создана запись audit log о trial upgrade
    And audit log содержит:
      | action          | trial_upgraded |
      | trial_days_used | 7 |
      | trial_days_remaining | 7 |
      | new_tier        | Pro |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_26
  @critical
  Scenario: Audit log for subscription changes
    Given пользователь имеет подписку "Starter"
    When пользователь изменяет параметры подписки
    Then каждое изменение подписки логируется в audit log
    And audit log содержит:
      | action          | subscription_modified |
      | subscription_id | <uuid> |
      | old_parameters  | <old_values> |
      | new_parameters  | <new_values> |
      | changed_by      | <user_id> |
      | timestamp       | <iso8601> |
      | ip_address      | <ip> |
    And audit log является immutable
    And audit log хранится минимум "5" лет

  @use_case=uc_04_01_27
  @critical
  Scenario: Billing cycle completion logging
    Given пользователь имеет годовую подписку "Pro"
    And годовой период оплаты завершается
    When происходит автоматическое продление
    Then создается запись audit log о billing cycle
    And audit log содержит:
      | action          | billing_cycle_completed |
      | subscription_id | <uuid> |
      | period_start    | <date> |
      | period_end      | <date> |
      | amount_billed   | <amount> |
      | payment_method  | <method> |
      | timestamp       | <iso8601> |
    And пользователю отправлен receipt
    And receipt доступен в billing history

  @use_case=uc_04_01_28
  @critical
  Scenario: Failed payment retry audit
    Given пользователь имеет подписку "Starter" с автопродлением
    When первая попытка автоплатежа не удалась
    Then создается запись audit log о failed payment
    And audit log содержит:
      | action          | payment_retry_failed |
      | attempt_number  | 1 |
      | error_code      | <code> |
      | next_retry_at   | <date> |
      | timestamp       | <iso8601> |
    And пользователю отправлено уведомление
    When вторая попытка не удалась
    Then создается новая запись audit log
    And attempt_number равен 2
    When третья попытка не удалась
    Then подписка переходит в статус "GRACE_PERIOD"
    And создается финальная запись audit log:
      | action          | subscription_grace_period |
      | failed_attempts | 3 |
      | grace_period_until | <date> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_29
  @critical
  @integration
  Scenario: Payment gateway timeout
    Given пользователь имеет подписку "Starter" с автопродлением
    And наступает дата автопродления
    When платежный шлюз не отвечает в течение "60" секунд
    Then платеж помечается как "TIMEOUT"
    And подписка переходит в статус "GRACE_PERIOD"
    And пользователю отправлено уведомление о timeout
    And система повторяет попытку платежа через "3" дня
    And создается запись audit log:
      | action          | payment_timeout |
      | timeout_duration | 60s |
      | retry_scheduled | <date> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_30
  @critical
  @integration
  Scenario: Yandex.Kassa service unavailable
    Given пользователь инициирует платеж через ЮMoney
    When сервис ЮMoney недоступен
    Then возвращается ошибка с кодом "PAYMENT_SERVICE_UNAVAILABLE"
    And ответ содержит структуру:
      | error.code       | PAYMENT_SERVICE_UNAVAILABLE |
      | error.message    | Payment service temporarily unavailable |
      | error.details    | {"service": "Yandex.Kassa", "retry_after": "300"} |
      | error.request_id | <uuid> |
    And пользователю предложено повторить попытку позже
    And система автоматически повторяет запрос через "5" минут
    And ошибка логируется в audit log

  @use_case=uc_04_01_31
  @critical
  @integration
  Scenario: Database transaction failure during payment - successful retry
    Given платеж через СБП находится в обработке
    And банк подтверждает успешную транзакцию
    When база данных недоступна при сохранении статуса платежа
    Then транзакция помечается как "PENDING"
    And система повторяет попытку сохранения через "1" минуту
    When повторная попытка сохранения успешна
    Then платеж сохраняется в базе данных
    And подписка активируется
    And пользователю отправляется подтверждение
    And действие в аудит лог записано как "payment_saved_after_retry"

  @use_case=uc_04_01_32
  @critical
  @integration
  Scenario: Database transaction failure during payment - retry exhausted
    Given платеж через СБП находится в обработке
    And банк подтверждает успешную транзакцию
    When база данных недоступна при сохранении статуса платежа
    Then транзакция помечается как "PENDING"
    And система повторяет попытку сохранения через "1" минуту
    When повторная попытка также завершается ошибкой
    Then создается alert для администратора
    And платеж помечается для обработки вручную
    And все события логируются в audit log
    And действие в аудит лог записано как "payment_retry_failed"

  @use_case=uc_04_01_33
  @critical
  @boundary
  Scenario: Ровно 25 мониторов на Free tier
    Given пользователь имеет подписку "Free" с лимитом "25" мониторов
    And пользователь создал ровно "25" мониторов
    When пользователь пытается создать новый монитор
    Then возвращается ошибка с кодом "MONITOR_LIMIT_REACHED"
    And ответ содержит структуру:
      | error.code        | MONITOR_LIMIT_REACHED |
      | error.details.limit | 25 |
      | error.details.current | 25 |
      | error.request_id | <uuid> |
    And создание монитора запрещено
    And пользователю предложено upgrade до платного тарифа

  @use_case=uc_04_01_34
  @critical
  @boundary
  Scenario: Grace period expiration at exact boundary
    Given пользователь имеет подписку в статусе "GRACE_PERIOD"
    And grace период начался "14" дней назад в "10:00:00"
    And текущее время "10:00:00" на "15"й день
    When grace период истекает точно в этот момент
    Then подписка переходит в статус "SUSPENDED"
    And все мониторы переходят в статус "PAUSED"
    And пользователю отправлено финальное уведомление
    And данные сохраняются "30" дней
    And создается запись audit log о приостановке

  @use_case=uc_04_01_35
  @critical
  @boundary
  Scenario: Trial expiration at exact boundary
    Given пользователь имеет trial подписку на "14" дней
    And trial начался "2026-03-01 00:00:00"
    And текущее время "2026-03-15 00:00:00"
    When trial период истекает точно в этот момент
    Then trial подписка завершается
    And пользователь переходит на Free тариф
    And мониторы превышающие лимит Free приостанавливаются
    And пользователю отправлено уведомление о завершении trial
    And предложено upgrade до платного тарифа
    And создается запись audit log о trial expiration

  @use_case=uc_04_01_36
  @critical
  @state_transition
  Scenario: Переход подписки из ACTIVE в GRACE_PERIOD
    Given пользователь имеет подписку в статусе "ACTIVE"
    And текущий период оплаты завершается
    And платеж не проходит
    When подписка переходит в статус "GRACE_PERIOD"
    Then доступ к функциям сохраняется "14" дней
    And мониторы остаются активными
    And пользователю отправлено уведомление о необходимости оплаты
    And создается запись audit log:
      | action          | subscription_grace_period_started |
      | previous_status | ACTIVE |
      | new_status      | GRACE_PERIOD |
      | grace_period_until | <date> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_37
  @critical
  @state_transition
  Scenario: Переход подписки из GRACE_PERIOD в SUSPENDED
    Given пользователь имеет подписку в статусе "GRACE_PERIOD"
    And grace период активен "14" дней
    And платеж не поступил
    When grace период истекает
    Then подписка переходит в статус "SUSPENDED"
    And все мониторы переходят в статус "PAUSED"
    And данные сохраняются "30" дней
    And пользователю отправлено финальное уведомление
    And создается запись audit log:
      | action          | subscription_suspended |
      | previous_status | GRACE_PERIOD |
      | new_status      | SUSPENDED |
      | data_retention_until | <date> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_38
  @critical
  @state_transition
  Scenario: Переход подписки из SUSPENDED в ACTIVE
    Given пользователь имеет подписку в статусе "SUSPENDED"
    And все мониторы в статусе "PAUSED"
    When пользователь инициирует успешный платеж
    Then подписка переходит в статус "ACTIVE"
    And мониторы автоматически возобновляются
    And исторические данные сохранены
    And пользователю отправлено подтверждение активации
    And создается запись audit log:
      | action          | subscription_reactivated |
      | previous_status | SUSPENDED |
      | new_status      | ACTIVE |
      | payment_id      | <uuid> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_39
  @critical
  @state_transition
  Scenario: Изменение статуса мониторов при изменении подписки
    Given пользователь имеет подписку "Pro" с лимитом "150" мониторов
    And пользователь создал "100" мониторов
    And все мониторы имеют статус "UP"
    When пользователь понижает тариф до "Starter" с лимитом "50"
    And система обнаруживает превышение лимита
    Then система автоматически приостанавливает "50" мониторов
    And приостанавливаемые мониторы выбираются по критериям:
      | критерий                  | приоритет |
      | последние созданные       | 1         |
      | наименьшее использование  | 2         |
    And пользователю отправлено уведомление о приостановке
    And создается запись audit log:
      | action          | monitors_paused_due_to_downgrade |
      | paused_count    | 50 |
      | reason          | limit_exceeded |
      | timestamp       | <iso8601> |

  @use_case=uc_04_01_40
  @critical
  @performance
  Scenario: Обработка множественных одновременных изменений подписки
    Given тенант имеет "10" пользователей с правом изменения подписки
    And все пользователи одновременно инициируют изменения подписки
    When система обрабатывает одновременные запросы
    Then применяется только первое изменение
    And последующие запросы возвращают ошибку "CONFLICT"
    And ответ содержит структуру:
      | error.code       | SUBSCRIPTION_CONFLICT |
      | error.message    | Subscription is being modified by another user |
      | error.details    | {"conflicting_user": <user_id>, "retry_after": "5s"} |
      | error.request_id | <uuid> |
    And система использует optimistic locking
    And все попытки логируются в audit log

  @use_case=uc_04_01_41
  @critical
  @performance
  Scenario: Mass recalculation of limits after tier change
    Given пользователь имеет подписку "Pro" с "100" мониторами
    And пользователь имеет "15" пользователей в аккаунте
    When пользователь понижает тариф до "Starter"
    And лимиты составляют "50" мониторов и "5" пользователей
    Then система выполняет пересчет лимитов асинхронно
    And пересчет завершается не более чем за "30" секунд
    And пользователь получает уведомление о завершении
    And мониторы и пользователи за пределами лимита приостанавливаются
    And создается запись audit log:
      | action          | limits_recalculated |
      | duration        | <actual_duration> |
      | monitors_affected | <count> |
      | users_affected  | <count> |
      | timestamp       | <iso8601> |
