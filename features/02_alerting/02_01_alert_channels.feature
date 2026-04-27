@epic=02_alerting
@user_story=02_01_alert_channels
# Description: Управление каналами оповещений

Feature: Каналы оповещений
  Как пользователь
  Я хочу настраивать каналы оповещений
  Чтобы получать уведомления о проблемах с моими сервисами

  Background:
    Given пользователь авторизован

  @use_case=uc_02_01_01
  @critical
  @audit
  Scenario: Создание Telegram канала
    When пользователь создаёт Telegram канал с параметрами:
      | name    | Ops Team           |
      | chat_id | -1001234567890     |
    Then канал должен быть создан успешно
    And отправлено проверочное сообщение
    And действие в аудит лог записано как "channel_created" с полями:
      | field        | value             |
      | channel_type | telegram          |
      | channel_name | Ops Team          |
      | chat_id      | -1001234567890    |
      | user_id      | текущий пользователь |

  @use_case=uc_02_01_02
  @critical
  @audit
  Scenario: Создание Email канала
    When пользователь создаёт Email канал с параметрами:
      | name | Admin Email |
      | email| admin@example.com |
    Then канал должен быть создан успешно
    And отправлено подтверждение на email
    And действие в аудит лог записано как "channel_created" с полями:
      | field        | value             |
      | channel_type | email             |
      | channel_name | Admin Email       |
      | email        | admin@example.com |
      | user_id      | текущий пользователь |

  @use_case=uc_02_01_03
  @critical
  @audit
  Scenario: Создание Webhook канала
    When пользователь создаёт Webhook канал с параметрами:
      | name   | Incident Webhook |
      | url    | https://hooks.example.com/incidents |
      | method | POST             |
    Then канал должен быть создан успешно
    And действие в аудит лог записано как "channel_created" с полями:
      | field        | value                                |
      | channel_type | webhook                              |
      | channel_name | Incident Webhook                     |
      | url          | https://hooks.example.com/incidents  |
      | method       | POST                                 |
      | user_id      | текущий пользователь                  |

  @use_case=uc_02_01_06
  @critical
  Scenario: Список всех каналов оповещений
    Given пользователь имеет следующие каналы:
      | name       | type    |
      | Ops Team   | Telegram |
      | Admin Email| Email    |
      | Webhook    | Webhook  |
    When пользователь запрашивает список каналов
    Then пользователь получает "3" канала

  @use_case=uc_02_01_08
  @critical
  @audit
  Scenario: Обновление канала оповещений
    Given пользователь имеет канал "Ops Team"
    When пользователь обновляет канал с параметрами:
      | name | New Ops Team |
    Then канал обновлён успешно
    And действие в аудит лог записано как "channel_updated" с полями:
      | field        | value                |
      | channel_id   | id канала Ops Team   |
      | old_name     | Ops Team             |
      | new_name     | New Ops Team         |
      | user_id      | текущий пользователь |

  @use_case=uc_02_01_09
  @critical
  @audit
  Scenario: Удаление канала оповещений
    Given пользователь имеет канал "Ops Team"
    When пользователь удаляет канал
    Then канал удалён
    And канал больше не появляется в списке
    And действие в аудит лог записано как "channel_deleted" с полями:
      | field        | value              |
      | channel_id   | id удалённого канала|
      | channel_name | Ops Team           |
      | user_id      | текущий пользователь |

  @use_case=uc_02_01_10
  @critical
  Scenario: Проверка работоспособности канала
    Given пользователь имеет канал "Ops Team"
    When пользователь проверяет канал
    Then статус канала "verified"
    And проверочное сообщение доставлено

  @use_case=uc_02_01_11
  @critical
  @validation
  Scenario: Невалидный chat_id при создании Telegram канала
    When пользователь создаёт Telegram канал с параметрами:
      | name    | Invalid Channel |
      | chat_id | invalid_format  |
    Then возвращена ошибка "INVALID_CHAT_ID"
    And канал не создан
    And отправлено сообщение с подсказкой о правильном формате

  @use_case=uc_02_01_12
  @critical
  @integration
  @audit
  Scenario: Сбой верификации Telegram канала
    Given пользователь создаёт Telegram канал с параметрами:
      | name    | Blocked Channel |
      | chat_id | -1001234567890  |
    And Telegram API возвращает ошибку "chat not found"
    When система пытается отправить проверочное сообщение
    Then канал не создан
    And пользователь видит ошибку "VERIFICATION_FAILED"
    And пользователь видит причину ошибки
    And действие в аудит лог записано как "channel_verification_failed" с полями:
      | field        | value                |
      | channel_type | telegram             |
      | chat_id      | -1001234567890       |
      | error        | chat not found       |
      | user_id      | текущий пользователь |

  @use_case=uc_02_01_13
  @critical
  Scenario: Настройка приоритетов каналов для монитора
    Given пользователь имеет канал "Ops Team L1"
    And пользователь имеет канал "Ops Team L2"
    When пользователь настраивает правила алерта для монитора "API Service":
      | channel_ids        | priority |
      | Ops Team L1        | 1        |
      | Ops Team L2        | 2        |
    Then при алерте сначала уведомляется "Ops Team L1"
    And если нет подтверждения через "15 minutes", уведомляется "Ops Team L2"

  @use_case=uc_02_01_14
  @critical
  @validation
  Scenario: Попытка создания дубликата канала
    Given пользователь имеет канал "Ops Team" с chat_id "-1001234567890"
    When пользователь создаёт Telegram канал с параметрами:
      | name    | Ops Team Copy |
      | chat_id | -1001234567890 |
    Then возвращена ошибка "DUPLICATE_CHANNEL"
    And предлагается использовать существующий канал

  @use_case=uc_02_01_15
  @critical
  @validation
  Scenario: Невалидный URL при создании Webhook канала
    When пользователь создаёт Webhook канал с параметрами:
      | name   | Invalid Webhook |
      | url    | not-a-url       |
      | method | POST            |
    Then возвращена ошибка "INVALID_URL"
    And канал не создан

  @use_case=uc_02_01_16
  @critical
  @integration
  Scenario: Таймаут проверки Webhook канала
    Given пользователь создаёт Webhook канал
      | url    | https://slow-endpoint.example.com/webhook |
    And endpoint не отвечает в течение "10 seconds"
    When система выполняет проверку
    Then проверка помечена как FAILED
    And пользователь видит предупреждение "ENDPOINT_TIMEOUT"

  @use_case=uc_02_01_17
  @critical
  @integration
  @audit
  Scenario: Ошибка доставки Email при множественных bounce вызывает отключение канала
    Given канал "Admin Email" с email "admin@example.com"
    And последние "5" отправок завершились с ошибкой "bounce"
    When система обнаруживает проблему доставки
    Then канал автоматически отключён
    And пользователь уведомлён о проблеме
    And канал имеет статус "DELIVERY_FAILED"
    And действие в аудит лог записано как "channel_auto_disabled" с полями:
      | field           | value                       |
      | channel_id      | id канала Admin Email       |
      | channel_type    | email                       |
      | failure_count   | 5                           |
      | failure_reason  | bounce                      |
      | new_status      | DELIVERY_FAILED             |

  @use_case=uc_02_01_18
  @critical
  @security
  @audit
  Scenario: Отказ в создании канала для GUEST пользователя
    Given пользователь с ролью "GUEST"
    When пользователь пытается создать Telegram канал с параметрами:
      | name    | Unauthorized Channel |
      | chat_id | -1001234567890       |
    Then возвращена ошибка "FORBIDDEN"
    And канал не создан
    And действие в аудит лог записано как "unauthorized_channel_creation_attempt" с полями:
      | field        | value                |
      | user_role    | GUEST                |
      | action       | create_channel       |
      | channel_type | telegram             |
      | status       | denied               |

  @use_case=uc_02_01_19
  @critical
  @security
  @audit
  Scenario: Отказ в обновлении чужого канала пользователем
    Given пользователь "user1" имеет канал "User1 Channel"
    And пользователь "user2" с ролью "USER"
    When пользователь "user2" пытается обновить канал "User1 Channel"
    Then возвращена ошибка "FORBIDDEN"
    And канал не обновлён
    And действие в аудит лог записано как "unauthorized_channel_update_attempt" с полями:
      | field        | value                    |
      | user_id      | user2                    |
      | channel_owner| user1                    |
      | channel_id   | id канала User1 Channel  |
      | action       | update_channel           |
      | status       | denied                   |

  @use_case=uc_02_01_20
  @critical
  @security
  @audit
  Scenario: Отказ в удалении чужого канала пользователем
    Given пользователь "user1" имеет канал "User1 Channel"
    And пользователь "user2" с ролью "USER"
    When пользователь "user2" пытается удалить канал "User1 Channel"
    Then возвращена ошибка "FORBIDDEN"
    And канал не удалён
    And действие в аудит лог записано как "unauthorized_channel_deletion_attempt" с полями:
      | field        | value                    |
      | user_id      | user2                    |
      | channel_owner| user1                    |
      | channel_id   | id канала User1 Channel  |
      | action       | delete_channel           |
      | status       | denied                   |

  @use_case=uc_02_01_21
  @critical
  @security
  @audit
  Scenario: Инъекция в webhook URL блокируется при валидации
    When пользователь создаёт Webhook канал с параметрами:
      | name   | Malicious Webhook          |
      | url    | https://evil.com/x SSTIInject |
      | method | POST                       |
    Then система валидирует URL
    And возвращена ошибка "INVALID_URL" если обнаружена инъекция
    And канал не создан
    And действие в аудит лог записано как "suspicious_webhook_url" с полями:
      | field        | value                       |
      | url          | https://evil.com/x SSTIInject|
      | reason       | potential_injection         |
      | user_id      | текущий пользователь        |
      | status       | blocked                     |

  @use_case=uc_02_01_26
  @critical
  @security
  @audit
  Scenario: Ошибка webhook не раскрывает API key в логах
    Given Webhook канал "Secure Webhook" с API key "sk_live_1234567890abcdef"
    And webhook возвращает ошибку "401 Unauthorized"
    When доставка алерта завершается с ошибкой
    Then результат доставки содержит ошибку
    And API key замаскирован в логах как "sk_live_****"
    And API key не виден в audit logs
    And действие в аудит лог записано как "webhook_delivery_failed" с полями:
      | field        | value                    |
      | channel_id   | id канала Secure Webhook |
      | api_key_mask | sk_live_****             |
      | error        | Unauthorized             |
      | api_key_safe | true                     |

  @use_case=uc_02_01_22
  @critical
  @concurrency
  @audit
  Scenario: Конкурентный запрос на создание каналов с одинаковыми параметрами
    Given пользователь "user1" начинает создание Telegram канала с chat_id "-1001234567890"
    And пользователь "user2" начинает создание Telegram канала с chat_id "-1001234567890"
    When оба пользователя одновременно завершают создание
    Then только один канал создан успешно
    And второй запрос отклонён с ошибкой "DUPLICATE_CHANNEL"
    And действие в аудит лог записано как "concurrent_channel_creation_conflict" с полями:
      | field           | value                      |
      | chat_id         | -1001234567890             |
      | conflict_type   | duplicate_creation         |
      | requests_count  | 2                          |
      | created         | 1                          |
      | rejected        | 1                          |
      | timestamp       | <iso8601>                  |

  @use_case=uc_02_01_23
  @critical
  @boundary
  @audit
  Scenario: Превышение лимита каналов при попытке создания пользователем
    Given пользователь с тарифом "Free" имеет "10" каналов
    And лимит каналов для тарифа "Free" равен "10"
    When пользователь пытается создать 11-й канал
    Then возвращена ошибка "CHANNEL_LIMIT_REACHED"
    And канал не создан
    And сообщение содержит информацию о текущем лимите
    And действие в аудит лог записано как "channel_limit_exceeded" с полями:
      | field        | value                |
      | user_tier    | Free                 |
      | current_count| 10                   |
      | limit        | 10                   |
      | error        | CHANNEL_LIMIT_REACHED|

  @use_case=uc_02_01_24
  @critical
  @concurrency
  @audit
  Scenario: Конкурентный запрос на обновление приоритетов каналов
    Given пользователь имеет канал "Ops Team L1" с приоритетом "1"
    And пользователь имеет канал "Ops Team L2" с приоритетом "2"
    When пользователь "user1" обновляет приоритет "Ops Team L1" на "3"
    And пользователь "user2" одновременно обновляет приоритет "Ops Team L2" на "1"
    Then оба обновления применены успешно
    And приоритеты каналов не конфликтуют
    And действие в аудит лог записано как "concurrent_priority_update" с полями:
      | field              | value                    |
      | channels_updated   | 2                        |
      | update_type        | priority_change          |
      | conflicts_resolved | true                     |
      | timestamp          | <iso8601>                |

  @use_case=uc_02_01_25
  @critical
  @boundary
  @audit
  Scenario: Блокировка доставки при обновлении канала во время активной отправки
    Given канал "Ops Team" используется для активной доставки алерта
    And алерт находится в процессе отправки
    When пользователь обновляет параметры канала "Ops Team"
    Then текущая доставка завершена со старыми параметрами
    And новые доставки используют обновлённые параметры
    And действие в аудит лог записано как "channel_updated_during_delivery" с полями:
      | field                | value                |
      | channel_id           | id канала Ops Team   |
      | update_timing        | during_active_delivery|
      | pending_deliveries   | 1                    |
      | update_strategy      | apply_on_next        |

  @use_case=uc_02_01_27
  @critical
  @validation
  Scenario: Попытка создания Email канала с невалидным адресом
    When пользователь создаёт Email канал с параметрами:
      | name  | Invalid Email |
      | email | not-an-email  |
    Then возвращена ошибка "INVALID_EMAIL"
    And канал не создан

  @use_case=uc_02_01_28
  @critical
  @integration
  @audit
  Scenario: Ошибка SMTP при верификации Email канала
    Given пользователь создаёт Email канал с email "test@example.com"
    And SMTP сервер недоступен
    When система пытается отправить верификационное письмо
    Then канал не создан
    And пользователь видит ошибку "VERIFICATION_FAILED"
    And действие в аудит лог записано как "channel_verification_smtp_error" с полями:
      | field        | value                |
      | channel_type | email                |
      | email        | test@example.com     |
      | error        | smtp_unavailable     |

  @use_case=uc_02_01_29
  @critical
  @validation
  Scenario: Попытка создания Webhook канала с невалидным HTTP методом
    When пользователь создаёт Webhook канал с параметрами:
      | name   | Invalid Method Webhook |
      | url    | https://hooks.example.com |
      | method | INVALID                |
    Then возвращена ошибка "INVALID_FORMAT"
    And описание ошибки "method must be one of: GET, POST, PUT, PATCH"
    And канал не создан

  @use_case=uc_02_01_30
  @critical
  @validation
  Scenario: Попытка создания канала с пустым именем
    When пользователь создаёт Telegram канал с параметрами:
      | name    |                |
      | chat_id | -1001234567890 |
    Then возвращена ошибка "MISSING_REQUIRED_FIELD"
    And описание ошибки "name cannot be empty"
    And канал не создан

  @use_case=uc_02_01_31
  @critical
  @boundary
  @validation
  Scenario: Попытка создания канала с именем длиннее максимальной длины
    When пользователь создаёт Telegram канал с параметрами:
      | name    | Очень длинное имя канала которое превышает максимально допустимую длину в 255 символов и не должно быть принято системой потому что система должна защищать пользователей от слишком длинных названий каналов которые могут вызвать проблемы в интерфейсе и базе данных |
      | chat_id | -1001234567890 |
    Then возвращена ошибка "FIELD_TOO_LONG"
    And описание ошибки "name exceeds maximum length of 255 characters"
    And канал не создан

  @use_case=uc_02_01_32
  @critical
  @integration
  Scenario: Таймаут при верификации Email канала
    Given пользователь создаёт Email канал с email "test@slowmail.com"
    And SMTP сервер отвечает медленно (>30 seconds)
    When система пытается отправить верификационное письмо
    Then верификация помечена как TIMEOUT
    And пользователь видит ошибку "VERIFICATION_TIMEOUT"
    And канал не создан

  @use_case=uc_02_01_33
  @critical
  @validation
  Scenario: Попытка установки дублирующегося приоритета каналов
    Given пользователь имеет каналы "Channel A" и "Channel B"
    When пользователь настраивает приоритеты:
      | channel    | priority |
      | Channel A  | 1        |
      | Channel B  | 1        |
    Then возвращена ошибка "DUPLICATE_PRIORITY"
    And приоритеты не обновлены

  @use_case=uc_02_01_34
  @critical
  @integration
  @audit
  Scenario: Автоматическое отключение канала после N последовательных ошибок
    Given канал "Ops Team" с failure_count = "4"
    When следующая доставка завершается с ошибкой
    Then failure_count увеличивается до "5"
    And канал автоматически отключён (failure threshold reached)
    And статус канала изменён на "DELIVERY_FAILED"
    And действие в аудит лог записано как "channel_auto_disabled" с полями:
      | field           | value                |
      | channel_id      | id канала Ops Team   |
      | failure_count   | 5                    |
      | threshold       | 5                    |
      | new_status      | DELIVERY_FAILED      |

  @use_case=uc_02_01_35
  @critical
  @validation
  Scenario: Попытка удаления канала с несуществующим ID
    When пользователь удаляет канал с ID "non-existent-uuid"
    Then возвращена ошибка "ALERT_CHANNEL_NOT_FOUND"
    And никакие данные не удалены

  @use_case=uc_02_01_36
  @critical
  @integration
  @audit
  Scenario: Блокировка Telegram бота при верификации канала
    Given пользователь создаёт Telegram канал с chat_id "-1001234567890"
    And Telegram API возвращает ошибку "403 Forbidden - bot was blocked by the user"
    When система пытается отправить верификационное сообщение
    Then канал не создан
    And пользователь видит ошибку "VERIFICATION_FAILED"
    And действие в аудит лог записано как "channel_verification_bot_blocked" с полями:
      | field        | value                         |
      | channel_type | telegram                      |
      | chat_id      | -1001234567890                |
      | error        | 403 Forbidden - bot blocked   |

  @use_case=uc_02_01_37
  @critical
  @validation
  Scenario: Попытка обновления канала с невалидными параметрами
    Given пользователь имеет Telegram канал "Ops Team"
    When пользователь обновляет канал с параметрами:
      | name |  |
    Then возвращена ошибка "MISSING_REQUIRED_FIELD"
    And канал не обновлён

  @use_case=uc_02_01_38
  @critical
  @validation
  Scenario: Попытка верификации несуществующего канала
    When пользователь проверяет канал с ID "non-existent-uuid"
    Then возвращена ошибка "ALERT_CHANNEL_NOT_FOUND"
    And никакие действия не выполнены

  @use_case=uc_02_01_39
  @critical
  @security
  @audit
  Scenario: Попытка создания канала без авторизации
    Given пользователь не авторизован
    When пользователь пытается создать Telegram канал с параметрами:
      | name    | Unauthorized Channel |
      | chat_id | -1001234567890       |
    Then возвращена ошибка "AUTH_REQUIRED"
    And канал не создан

  @use_case=uc_02_01_40
  @critical
  @boundary
  @audit
  Scenario: Попытка получить список каналов другого пользователя
    Given пользователь "user1" имеет каналы
    And пользователь "user2" авторизован
    When пользователь "user2" запрашивает каналы пользователя "user1"
    Then возвращена ошибка "FORBIDDEN"
    And каналы пользователя "user1" не возвращены
    And действие в аудит лог записано как "unauthorized_channel_list_attempt" с полями:
      | field        | value                |
      | user_id      | user2                |
      | target_user  | user1                |
      | action       | list_channels        |
      | status       | denied               |

  @use_case=uc_02_01_41
  @critical
  @validation
  Scenario: Попытка создания Telegram канала с пустым chat_id
    When пользователь создаёт Telegram канал с параметрами:
      | name    | Empty ChatID Channel |
      | chat_id |                      |
    Then возвращена ошибка "MISSING_REQUIRED_FIELD"
    And описание ошибки "chat_id cannot be empty"
    And канал не создан

  @use_case=uc_02_01_42
  @critical
  @integration
  @audit
  Scenario: Удаление канала, который используется в активной доставке
    Given канал "Ops Team" используется для текущей доставки алерта
    When пользователь пытается удалить канал "Ops Team"
    Then возвращена ошибка "CHANNEL_IN_USE"
    And канал не удалён
    And текущая доставка завершена успешно
    And действие в аудит лог записано как "channel_deletion_blocked" с полями:
      | field            | value                |
      | channel_id       | id канала Ops Team   |
      | blocking_reason  | active_delivery      |

  @use_case=uc_02_01_43
  @critical
  @integration
  Scenario: Webhook канал возвращает нечитаемый ответ при верификации
    Given пользователь создаёт Webhook канал с URL "https://hooks.example.com"
    And endpoint возвращает непarseable ответ (не JSON и не HTTP 200)
    When система выполняет верификационный запрос
    Then верификация помечена как FAILED
    And пользователь видит ошибку "VERIFICATION_INVALID_RESPONSE"
    And канал не создан

  @use_case=uc_02_01_44
  @critical
  @validation
  Scenario: Попытка создания Email канала без указания email адреса
    When пользователь создаёт Email канал с параметрами:
      | name  | No Email Channel |
      | email |                  |
    Then возвращена ошибка "MISSING_REQUIRED_FIELD"
    And описание ошибки "email cannot be empty"
    And канал не создан
