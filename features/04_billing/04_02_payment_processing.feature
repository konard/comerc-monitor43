@epic=04_billing
@user_story=04_02_payment_processing
# Description: Обработка платежей через СБП и карты

Feature: Обработка платежей
  Как пользователь
  Я хочу оплачивать подписку через СБП или карту
  Чтобы иметь доступ к платным функциям

  @use_case=uc_04_02_01
  @critical
  Scenario: Инициация платежа через ЮMoney
    Given пользователь имеет подписку "Starter" стоимостью "500" рублей
    When пользователь инициирует платеж через СБП
    Then пользователь получает инструкции по оплате
    And формируется уникальный идентификатор платежа
    And создана запись audit log об инициации платежа
    And audit log содержит:
      | action          | payment_initiated |
      | amount          | 500 |
      | method          | SBP |
      | subscription_id | <uuid> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_02
  @critical
  Scenario: Инициация платежа через СБП
    Given платеж находится в обработке
    When платёжная система подтверждает транзакцию
    Then статус платежа "COMPLETED"
    And подписка активна на следующий период
    And создана запись audit log о успешном платеже
    And audit log содержит:
      | action          | payment_completed |
      | payment_id      | <uuid> |
      | amount          | <amount> |
      | method          | <method> |
      | subscription_id | <uuid> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_03
  @critical
  Scenario: Обработка успешного платежа
    Given платеж инициирован
    When платёжная система возвращает ошибку
    Then статус платежа "FAILED"
    And пользователю отправлено уведомление
    And подписка переходит в grace period на "14" дней
    And создана запись audit log о неудачном платеже
    And audit log содержит:
      | action          | payment_failed |
      | payment_id      | <uuid> |
      | error_code      | <code> |
      | error_message   | <message> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_04
  @critical
  Scenario: Оплата в течение grace period
    Given подписка в статусе "GRACE_PERIOD"
    When пользователь оплачивает "500" рублей
    Then подписка переходит в статус "ACTIVE"
    And grace period отменяется
    And создана запись audit log о grace period payment
    And audit log содержит:
      | action          | grace_period_payment_completed |
      | payment_id      | <uuid> |
      | amount          | 500 |
      | grace_period_days_used | <days> |
      | subscription_id | <uuid> |
      | timestamp       | <iso8601> |
    And действие в аудит лог записано как "payment_updated"
      | field | value |
      | payment_id | <uuid> |
      | status | ACTIVE |

  @use_case=uc_04_02_05
  Scenario: Инициация платежа через СБП с deeplink
    Given пользователь выбирает оплату через СБП
    And сумма платежа составляет "500" рублей
    When система формирует платеж
    Then генерируется уникальный deeplink для СБП
    And пользователь перенаправляется в банковское приложение
    And создается транзакция со статусом "PENDING"

  @use_case=uc_04_02_06
  Scenario: Успешный платеж через СБП
    Given платеж через СБП находится в статусе "PENDING"
    When банк подтверждает успешную транзакцию
    Then статус платежа обновляется на "COMPLETED"
    And подписка активируется на следующий период
    And пользователю отправлено подтверждение оплаты
    And действие в аудит лог записано как "payment_updated"
      | field | value |
      | payment_id | <uuid> |
      | status | COMPLETED |

  @use_case=uc_04_02_07
  @critical
  Scenario: Оплата через ЮMoney
    Given пользователь выбирает оплату через ЮMoney
    And сумма платежа составляет "2000" рублей
    When пользователь подтверждает платеж в ЮMoney
    Then происходит перенаправление на страницу ЮMoney
    And после оплаты пользователь возвращается в систему
    And подписка активируется при успешном платеже
    And создана запись audit log о YuMoney платеже
    And audit log содержит:
      | action          | payment_initiated |
      | amount          | 2000 |
      | method          | YuMoney |
      | subscription_id | <uuid> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_08
  @critical
  Scenario: Оплата банковской картой
    Given пользователь выбирает оплату картой
    And сумма платежа составляет "500" рублей
    When пользователь вводит данные карты
    Then происходит привязка карты перед оплатой
    And инициируется транзакция через платежный шлюз
    And при успешной оплате подписка активируется
    And создана запись audit log о card платеже
    And audit log содержит:
      | action          | payment_initiated |
      | amount          | 500 |
      | method          | card |
      | card_last4      | <****> |
      | subscription_id | <uuid> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_09
  @critical
  Scenario: Возврат платежа
    Given платеж успешно завершен
    And прошло менее "30" дней с момента оплаты
    When пользователь запрашивает возврат
    And администратор подтверждает возврат
    Then инициируется возврат средств
    And подписка переходит в статус "CANCELED"
    And пользователю отправлено уведомление о возврате
    And создана запись audit log о возврате
    And audit log содержит:
      | action          | refund_initiated |
      | payment_id      | <uuid> |
      | refund_amount   | <amount> |
      | reason          | <reason> |
      | approved_by     | <admin_id> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_10
  Scenario: Повторный платеж после неудачи
    Given предыдущий платеж не прошел
    And подписка находится в grace period
    When пользователь инициирует повторный платеж
    Then создается новая транзакция
    And при успешной оплате подписка возобновляется
    And grace период отменяется

  @use_case=uc_04_02_11
  Scenario: Проверка статуса платежа
    Given платеж инициирован
    When пользователь запрашивает статус платежа
    Then система возвращает текущий статус
    And возможные статусы включают "PENDING", "COMPLETED", "FAILED"
    And предоставляется дата и время последнего обновления

  @use_case=uc_04_02_12
  Scenario: Получение истории платежей
    Given пользователь имеет историю из "5" платежей
    When пользователь запрашивает историю платежей
    Then возвращается список всех транзакций
    And каждая транзакция содержит дату, сумму, статус и метод оплаты
    And список отсортирован по дате по убыванию

  @use_case=uc_04_02_13
  @critical
  Scenario: Обработка webhook от платежной системы
    Given платеж инициирован через СБП
    When платежная система отправляет webhook о статусе платежа
    Then система обрабатывает webhook асинхронно
    And статус платежа обновляется в базе данных
    And пользователю отправляется уведомление о результате
    And создана запись audit log об обработке webhook
    And audit log содержит:
      | action          | webhook_processed |
      | webhook_id      | <uuid> |
      | payment_id      | <uuid> |
      | old_status      | <status> |
      | new_status      | <status> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_14
  @critical
  Scenario: Ошибка при обработке webhook a
    Given платежная система отправляет webhook
    And данные webhook некорректны или подпись неверна
    Then система отклоняет webhook
    And логируется ошибка безопасности
    And статус платежа не изменяется
    And создана запись audit log security_incident
    And audit log содержит:
      | action          | webhook_rejected |
      | reason          | invalid_signature |
      | webhook_id      | <uuid> |
      | source_ip       | <ip> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_15
  @critical
  @integration
  Scenario: Payment timeout handling
    Given платеж через СБП инициирован
    And платеж находится в статусе "PENDING"
    And прошло "30" минут без ответа от банка
    When платеж истекает по таймауту
    Then статус платежа обновляется на "TIMEOUT"
    And пользователю предлагается повторить платеж
    And создается новая транзакция
    And создана запись audit log о timeout
    And audit log содержит:
      | action          | payment_timeout |
      | payment_id      | <uuid> |
      | timeout_minutes | 30 |
      | method          | SBP |
      | timestamp       | <iso8601> |
    And действие в аудит лог записано как "payment_updated"
      | field | value |
      | payment_id | <uuid> |
      | status | TIMEOUT |

  @use_case=uc_04_02_16
  @critical
  Scenario: Duplicate payment prevention
    Given пользователь имеет подписку "Starter" за "500" рублей
    And пользователь инициировал платеж
    And платеж находится в статусе "PENDING"
    When пользователь случайно инициирует повторный платеж
    Then система обнаруживает дубликат
    And возвращается ошибка с кодом "DUPLICATE_PAYMENT"
    And ответ содержит структуру:
      | error.code                    | DUPLICATE_PAYMENT |
      | error.details.existing_payment_id | <uuid> |
      | error.details.status          | PENDING |
      | error.request_id              | <uuid> |
    And инициированный ранее платеж остается в обработке
    And возвращается информация о существующем платеже
    And создана запись audit log о duplicate attempt
    And audit log содержит:
      | action          | duplicate_payment_prevented |
      | original_payment_id | <uuid> |
      | attempted_amount | 500 |
      | user_id         | <user_id> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_17
  @critical
  @integration
  Scenario: Webhook delivery failure and retry - попытка 1
    Given платежная система отправляет webhook об успешном платеже
    When webhook не доставлен (network error)
    Then система регистрирует неудачную попытку с кодом "CONNECTION_ERROR"
    And повторяет запрос webhook через "30 seconds" используя exponential backoff
    And ошибка является retryable (временная сетевая ошибка)
    And создана запись audit log о webhook failure
    And audit log содержит:
      | action          | webhook_delivery_failed |
      | webhook_id      | <uuid> |
      | payment_id      | <uuid> |
      | error_code      | CONNECTION_ERROR |
      | error_type      | network_error |
      | retry_scheduled | true |
      | retry_delay     | 30 seconds |
      | retry_strategy  | exponential_backoff |
      | timestamp       | <iso8601> |
    When повторная попытка успешна
    Then статус платежа обновляется на "COMPLETED"
    And подписка активируется
    And пользователю отправлено подтверждение
    And создана запись audit log о webhook retry success
    And audit log содержит:
      | action          | webhook_retry_succeeded |
      | webhook_id      | <uuid> |
      | retry_attempts  | 1 |
      | timestamp       | <iso8601> |
    And действие в аудит лог записано как "payment_updated"
      | field | value |
      | payment_id | <uuid> |
      | status | COMPLETED |

  @use_case=uc_04_02_17a
  @critical
  @integration
  Scenario: Webhook delivery failure - попытка 2
    Given платежная система отправляет webhook об успешном платеже
    And retry попытка 1 завершилась с ошибкой "CONNECTION_ERROR"
    When выполняется retry попытка 2
    Then повторный запрос выполнен через "60 seconds" (удвоенная задержка)
    And используется exponential backoff стратегия
    And создана запись audit log:
      | action          | webhook_delivery_retry |
      | webhook_id      | <uuid> |
      | payment_id      | <uuid> |
      | retry_attempt   | 2 |
      | retry_delay     | 60 seconds |
      | error_code      | CONNECTION_ERROR |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_18
  @critical
  Scenario: Card payment 3DS authentication failure
    Given пользователь оплачивает банковской картой
    And карта требует 3DS аутентификацию
    When пользователь не завершает 3DS аутентификацию
    Then платеж отменяется
    And статус платежа "FAILED"
    And возвращается ошибка с кодом "3DS_AUTHENTICATION_FAILED"
    And ответ содержит структуру:
      | error.code            | 3DS_AUTHENTICATION_FAILED |
      | error.details.payment_id | <uuid> |
      | error.request_id      | <uuid> |
    And пользователю предложено повторить попытку
    And создана запись audit log о 3DS failure
    And audit log содержит:
      | action          | 3ds_authentication_failed |
      | payment_id      | <uuid> |
      | card_last4      | <****> |
      | reason          | user_canceled |
      | timestamp       | <iso8601> |
    And действие в аудит лог записано как "payment_updated"
      | field | value |
      | payment_id | <uuid> |
      | status | FAILED |

  @use_case=uc_04_02_19
  @critical
  @business_rule
  Scenario: Auto-renewal with expired card
    Given пользователь имеет годовую подписку с автопродлением
    And привязанная карта истекает за "1" месяц до окончания подписки
    When система определяет скорое истечение карты
    Then пользователю отправлено предупреждение за "30" дней
    And предложено обновить платежные данные
    And создана запись audit log о expiring card warning
    And audit log содержит:
      | action          | payment_method_expiring_soon |
      | subscription_id | <uuid> |
      | card_last4      | <****> |
      | expiry_date     | <date> |
      | days_until_expiry | 30 |
      | timestamp       | <iso8601> |
    When наступает дата автопродления с истекшей картой
    Then платеж не проходит
    And подписка переходит в статус "GRACE_PERIOD"
    And пользователю отправлено уведомление с предложением обновить данные
    And создана запись audit log о failed auto-renewal
    And audit log содержит:
      | action          | auto_renewal_failed |
      | subscription_id | <uuid> |
      | reason          | card_expired |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_20
  @critical
  Scenario: Chargeback processing
    Given платеж успешно завершен "30" дней назад
    When банк инициирует chargeback по транзакции
    Then подписка переходит в статус "SUSPENDED"
    And администратор уведомлен о chargeback
    And пользователю отправлено уведомление о приостановке
    And предоставлена возможность повторить оплату
    And создана запись audit log о chargeback
    And audit log содержит:
      | action          | chargeback_initiated |
      | payment_id      | <uuid> |
      | amount          | <amount> |
      | reason          | <reason> |
      | bank_reference  | <ref> |
      | timestamp       | <iso8601> |
    And действие в аудит лог записано как "payment_updated"
      | field | value |
      | payment_id | <uuid> |
      | status | SUSPENDED |

  @use_case=uc_04_02_21
  @critical
  @security
  Scenario: Webhook replay attack prevention
    Given платежная система отправляет webhook об успешном платеже
    And webhook обработан, статус платежа "COMPLETED"
    When злоумышленник повторно отправляет идентичный webhook
    Then система проверяет idempotency key
    And обнаруживает дубликат webhook
    And webhook отклоняется
    And логируется попытка replay атаки
    And статус платежа не изменяется
    And создана запись audit log security incident
    And audit log содержит:
      | action          | webhook_replay_attack_prevented |
      | webhook_id      | <uuid> |
      | payment_id      | <uuid> |
      | source_ip       | <ip> |
      | idempotency_key | <key> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_22
  @critical
  Scenario: Payment method update during active subscription
    Given пользователь имеет активную подписку с автопродлением
    And привязана банковская карта
    When пользователь обновляет платежные данные на СБП
    Then новые данные сохраняются для будущего автопродления
    And текущая подписка остается активной
    And следующее автопродление использует новый метод
    And пользователю отправлено подтверждение обновления
    And создана запись audit log о payment method update
    And audit log содержит:
      | action          | payment_method_updated |
      | subscription_id | <uuid> |
      | old_method      | card |
      | new_method      | SBP |
      | initiated_by    | <user_id> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_23
  @critical
  Scenario: SBP payment limit handling
    Given пользователь пытается оплатить годовую подписку за "24000" рублей
    And лимит СБП составляет "15000" рублей за транзакцию
    When система формирует платеж через СБП
    Then возвращается ошибка с кодом "SBP_LIMIT_EXCEEDED"
    And ответ содержит структуру:
      | error.code           | SBP_LIMIT_EXCEEDED |
      | error.details.amount | 24000 |
      | error.details.limit  | 15000 |
      | error.details.excess | 9000 |
      | error.request_id     | <uuid> |
    And пользователю предложено разбить платеж на части
    And пользователю предложено оплатить банковской картой
    And создана запись audit log о limit exceeded
    And audit log содержит:
      | action          | payment_limit_exceeded |
      | attempted_amount | 24000 |
      | limit           | 15000 |
      | excess          | 9000 |
      | method          | SBP |
      | user_id         | <user_id> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_24
  @critical
  Scenario: Webhook with missing required fields
    Given платежная система отправляет webhook
    And webhook payload не содержит обязательное поле "transaction_id"
    When система обрабатывает webhook
    Then webhook отклоняется с ошибкой "INVALID_PAYLOAD"
    And ответ содержит структуру:
      | error.code            | INVALID_PAYLOAD |
      | error.details.missing_fields | transaction_id |
      | error.request_id      | <uuid> |
    And логируется ошибка с деталями отсутствующих полей
    And статус платежа не изменяется
    And администратор уведомлен о некорректном webhook
    And создана запись audit log security incident
    And audit log содержит:
      | action          | webhook_rejected |
      | reason          | missing_required_fields |
      | missing_fields  | transaction_id |
      | source_ip       | <ip> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_25
  @critical
  Scenario: PCI DSS compliance validation - card data encryption
    Given пользователь оплачивает банковской картой
    And карта требует 3DS аутентификацию
    When пользователь вводит данные карты
    Then данные карты никогда не сохраняются в системе
    And данные карты передаются напрямую в платежный шлюз
    And используется TLS 1.3 для шифрования
    And PAN маскируется во всех логах
    And CVV/CVC никогда не логируется
    And создана запись audit log:
      | action          | pci_compliance_check |
      | encryption      | TLS_1.3 |
      | pan_masked      | true |
      | cvv_logged      | false |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_26
  @critical
  Scenario: Payment fraud detection - suspicious pattern
    Given пользователь имеет подписку "Starter"
    And пользователь инициировал "5" платежей за последний час
    When пользователь пытается инициировать новый платеж
    Then система обнаруживает подозрительную активность
    And платеж блокируется
    And возвращается ошибка с кодом "SUSPICIOUS_ACTIVITY_DETECTED"
    And ответ содержит структуру:
      | error.code                 | SUSPICIOUS_ACTIVITY_DETECTED |
      | error.details.payment_count | 6 |
      | error.details.timeframe    | 1 hour |
      | error.request_id           | <uuid> |
    And создана запись audit log о fraud attempt
    And audit log содержит:
      | action          | fraud_detected |
      | fraud_type      | suspicious_payment_pattern |
      | payment_count   | 6 |
      | timeframe       | 1 hour |
      | user_id         | <user_id> |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |
    And администратор уведомлен о подозрительной активности

  @use_case=uc_04_02_27
  @critical
  Scenario: Rate limiting on payment endpoints
    Given пользователь имеет подписку "Pro"
    And пользователь отправил "10" запросов к payment API за минуту
    When пользователь отправляет новый запрос к payment API
    Then система применяет rate limiting
    And запрос отклоняется с ошибкой "RATE_LIMIT_EXCEEDED"
    And ответ содержит структуру:
      | error.code                    | RATE_LIMIT_EXCEEDED |
      | error.details.limit           | 10 |
      | error.details.window          | 60 |
      | error.details.retry_after     | <seconds> |
      | error.request_id              | <uuid> |
    And создана запись audit log о rate limit
    And audit log содержит:
      | action          | rate_limit_exceeded |
      | endpoint        | payment_api |
      | limit           | 10 |
      | window          | 60 |
      | ip_address      | <ip> |
      | user_id         | <user_id> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_28
  @critical
  Scenario: AML check for large payments
    Given пользователь пытается оплатить годовую подписку "Enterprise" за "100000" рублей
    And сумма превышает порог AML проверки
    When система обрабатывает платеж
    Then инициируется AML проверка
    And платеж помещается в статус "PENDING_REVIEW"
    And пользователю отправлено уведомление о дополнительной проверке
    And создана запись audit log о AML check
    And audit log содержит:
      | action          | aml_check_initiated |
      | amount          | 100000 |
      | threshold       | <threshold> |
      | user_risk_score | <score> |
      | timestamp       | <iso8601> |
    And compliance команда уведомлена
    And действие в аудит лог записано как "payment_updated"
      | field | value |
      | payment_id | <uuid> |
      | status | PENDING_REVIEW |

  @use_case=uc_04_02_29
  @critical
  Scenario: Payment webhook signature verification
    Given платежная система отправляет webhook
    And webhook содержит цифровую подпись
    When система получает webhook
    Then верифицируется подпись webhook
    And проверяется соответствие payload подписи
    And при неверной подписи:
      | webhook отклоняется с ошибкой "INVALID_SIGNATURE" |
      | создается запись audit log о security breach |
      | IP адрес источника логируется |
      | администратор получает немедленное уведомление |
    And audit log содержит:
      | action          | signature_verification_failed |
      | webhook_id      | <uuid> |
      | source_ip       | <ip> |
      | signature_type  | <type> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_30
  @critical
  Scenario: Card tokenization security
    Given пользователь сохраняет карту для автопродления
    When система обрабатывает данные карты
    Then используется токенизация вместо реальных данных карты
    And токен генерируется платежным шлюзом
    And система хранит только токен
    And реальные данные карты недоступны системе
    And создана запись audit log:
      | action          | card_tokenized |
      | token_id        | <uuid> |
      | last4           | <****> |
      | expiry_masked   | <**/****> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_31
  @critical
  Scenario: Payment data retention policy
    Given платеж успешно завершен
    And прошло "3" года с момента платежа
    When система выполняет scheduled cleanup
    Then детальные данные платежа anonymizируются
    And сохраняются только:
      | payment_id     | <uuid> |
      | amount         | <amount> |
      | date           | <date> |
      | status         | COMPLETED |
    And PAN, CVV и sensitive данные удалены
    And создана запись audit log о data cleanup
    And audit log содержит:
      | action          | payment_data_anonymized |
      | payment_id      | <uuid> |
      | retention_period | 3 years |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_32
  @critical
  @integration
  Scenario: Payment gateway connection timeout
    Given пользователь инициирует платеж через СБП
    When платежный шлюз не отвечает в течение "30" секунд
    Then платеж помечается как "TIMEOUT" с кодом "PAYMENT_GATEWAY_TIMEOUT"
    And ошибка является retryable (временная ошибка подключения)
    And предлагается повторить попытку через "1 hour" используя linear backoff
    And возвращается ошибка с кодом "PAYMENT_GATEWAY_TIMEOUT"
    And ответ содержит структуру:
      | error.code                | PAYMENT_GATEWAY_TIMEOUT |
      | error.details.timeout     | 30s |
      | error.details.method      | SBP |
      | error.retry_scheduled     | true |
      | error.retry_after         | 1 hour |
      | error.retry_strategy      | linear_backoff |
      | error.request_id          | <uuid> |
    And создается запись audit log:
      | action            | payment_gateway_timeout |
      | timeout_duration  | 30s |
      | payment_method    | SBP |
      | error_code        | PAYMENT_GATEWAY_TIMEOUT |
      | retry_scheduled   | true |
      | retry_delay       | 1 hour |
      | retry_strategy    | linear_backoff |
      | timestamp         | <iso8601> |
    And действие в аудит лог записано как "payment_updated"
      | field | value |
      | payment_id | <uuid> |
      | status | TIMEOUT |

  @use_case=uc_04_02_33
  @critical
  @integration
  Scenario: Webhook delivery failure with retry exhaustion
    Given платежная система отправляет webhook об успешном платеже
    And webhook не доставлен (network error)
    When система повторяет запрос "3" раза безуспешно используя exponential backoff
    Then платеж остается в статусе "PENDING"
    And все retry попытки исчерпаны
    And webhook перемещен в dead letter queue
    And создается alert для администратора
    And создается запись audit log:
      | action                | webhook_delivery_failed |
      | retry_attempts        | 3 |
      | webhook_id            | <uuid> |
      | payment_id            | <uuid> |
      | error_code            | CONNECTION_ERROR |
      | retry_strategy        | exponential_backoff |
      | dlq_status            | queued |
      | requires_manual_review | true |
      | timestamp             | <iso8601> |
    And администратор уведомлен о необходимости ручной обработки

  @use_case=uc_04_02_34
  @critical
  @boundary
  Scenario: Payment amount exactly at boundary
    Given пользователь имеет подписку "Pro" стоимостью "2000" рублей
    When пользователь инициирует платеж ровно на "2000" рублей
    Then платеж обрабатывается успешно
    And подписка активируется на полный период
    And создается запись audit log:
      | action          | payment_completed |
      | amount          | 2000 |
      | subscription_id | <uuid> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_35
  @critical
  @boundary
  Scenario: Payment amount one ruble below minimum
    Given минимальная сумма платежа составляет "100" рублей
    When пользователь пытается оплатить "99" рублей
    Then возвращается ошибка с кодом "PAYMENT_AMOUNT_TOO_LOW"
    And ответ содержит структуру:
      | error.code       | PAYMENT_AMOUNT_TOO_LOW |
      | error.message    | Payment amount below minimum |
      | error.details    | {"amount": 99, "minimum": 100} |
      | error.request_id | <uuid> |
    And платеж не инициируется
    And пользователю предложено увеличить сумму

  @use_case=uc_04_02_36
  @critical
  @boundary
  Scenario: Payment amount one ruble above maximum
    Given максимальная сумма платежа составляет "100000" рублей
    When пользователь пытается оплатить "100001" рублей
    Then возвращается ошибка с кодом "PAYMENT_AMOUNT_TOO_HIGH"
    And ответ содержит структуру:
      | error.code       | PAYMENT_AMOUNT_TOO_HIGH |
      | error.message    | Payment amount exceeds maximum |
      | error.details    | {"amount": 100001, "maximum": 100000} |
      | error.request_id | <uuid> |
    And платеж не инициируется
    And предлагается разбить платеж на части

  @use_case=uc_04_02_37
  @critical
  @state_transition
  Scenario: Payment state transition from PENDING to COMPLETED
    Given платеж через СБП находится в статусе "PENDING"
    And пользователь ожидает подтверждения
    When банк подтверждает успешную транзакцию
    Then платеж переходит в статус "COMPLETED"
    And подписка активируется на следующий период
    And пользователю отправляется подтверждение оплаты
    And создается запись audit log:
      | action          | payment_state_transition |
      | payment_id      | <uuid> |
      | previous_status | PENDING |
      | new_status      | COMPLETED |
      | timestamp       | <iso8601> |
    And действие в аудит лог записано как "payment_updated"
      | field | value |
      | payment_id | <uuid> |
      | status | COMPLETED |

  @use_case=uc_04_02_38
  @critical
  @state_transition
  Scenario: Payment state transition from PENDING to FAILED
    Given платеж через СБП находится в статусе "PENDING"
    And пользователь ожидает подтверждения
    When банк отклоняет транзакцию
    Then платеж переходит в статус "FAILED"
    And пользователю отправляется уведомление об отказе
    And подписка переходит в grace period
    And создается запись audit log:
      | action          | payment_state_transition |
      | payment_id      | <uuid> |
      | previous_status | PENDING |
      | new_status      | FAILED |
      | reason          | <bank_reason> |
      | timestamp       | <iso8601> |
    And действие в аудит лог записано как "payment_updated"
      | field | value |
      | payment_id | <uuid> |
      | status | FAILED |

  @use_case=uc_04_02_39
  @critical
  @state_transition
  Scenario: Monitor status during payment processing
    Given пользователь имеет подписку в статусе "GRACE_PERIOD"
    And мониторы активны
    And платеж находится в статусе "PENDING"
    When платеж успешно завершается
    Then подписка переходит в статус "ACTIVE"
    And мониторы остаются активными
    And исторические данные сохранены
    And пользователю отправляется подтверждение
    And создается запись audit log:
      | action          | subscription_reactivated_after_payment |
      | payment_id      | <uuid> |
      | monitors_count  | <count> |
      | timestamp       | <iso8601> |

  @use_case=uc_04_02_40
  @critical
  @state_transition
  Scenario: Monitor pause after payment failure
    Given пользователь имеет подписку в статусе "GRACE_PERIOD"
    And grace период истекает
    And платеж не поступил
    When подписка переходит в статус "SUSPENDED"
    Then все мониторы переходят в статус "PAUSED"
    And данные сохраняются "30" дней
    And создается запись audit log:
      | action          | monitors_paused_after_payment_failure |
      | subscription_id | <uuid> |
      | monitors_paused | <count> |
      | data_retention_until | <date> |
      | timestamp       | <iso8601> |
