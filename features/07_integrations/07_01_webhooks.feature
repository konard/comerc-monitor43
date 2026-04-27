@epic=07_integrations
@user_story=07_01_webhooks
# Description: Управление вебхуками для внешних интеграций

Feature: Вебхуки
  Как пользователь
  Я хочу настраивать вебхуки
  Чтобы интегрировать мониторинг с внешними сервисами

  @use_case=uc_07_01_01
  @critical
  @implemented
  Scenario: Создание webhook интеграции
    Given пользователь авторизован
    When пользователь создаёт вебхук с параметрами:
      | name   | Custom Integration    |
      | url    | https://api.example.com/webhook |
      | method | POST                   |
    Then вебхук создан успешно
    And вебхук активен
    And действие в аудит лог записано как "webhook_created"
      | field     | value                      |
      | webhook_id | <webhook_id>               |
      | url       | https://api.example.com/webhook |
      | timestamp | <iso8601>                  |
      | user_id   | <user_id>                  |

  @use_case=uc_07_01_02
  @critical
  Scenario: Вебхук с настраиваемыми headers
    Given пользователь авторизован
    When пользователь создаёт вебхук с параметрами:
      | name    | Secured Webhook           |
      | url     | https://api.example.com/alerts |
      | method  | POST                      |
      | headers | {"Authorization": "Bearer xyz"} |
    Then вебхук создан с custom headers
    And запросы содержат указанные headers
    And действие в аудит лог записано как "webhook_created"
      | field     | value                      |
      | webhook_id | <webhook_id>               |
      | url       | https://api.example.com/alerts |
      | timestamp | <iso8601>                  |
      | user_id   | <user_id>                  |

  @use_case=uc_07_01_03a
  @critical
  Scenario: Ограничение вебхуков для Free тарифа
    Given пользователь имеет тариф "Free"
    And пользователь имеет "2" webhook
    When пользователь пытается создать третий вебхук
    Then система возвращает ошибку "WEBHOOK_LIMIT_REACHED"
    And ответ содержит структуру:
      | error.code    | WEBHOOK_LIMIT_REACHED |
      | error.message | Webhooks not available on free tier |
      | error.details | {...} |
      | error.request_id | uuid |
    And вебхук не создан
    And в audit log записано событие "webhook_limit_reached" с тарифом пользователя и количеством вебхуков

  @use_case=uc_07_01_04
  @critical
  Scenario: Конкурентный доступ при создании вебхуков a
    Given пользователь авторизован
    When пользователь отправляет "3" одновременных запроса на создание вебхуков
    Then все "3" вебхука созданы успешно
    And все вебхуки имеют уникальные ID
    And ни один вебхук не перезаписал другой
    And в audit log записаны "3" события "webhook_created"

  @use_case=uc_07_01_05
  @critical
  Scenario: Клонирование настроек вебхука
    Given пользователь имеет вебхук "Production Integration" с параметрами:
      | url    | https://api.production.com/webhook |
      | method | POST |
      | headers | {"Authorization": "Bearer prod-token"} |
    When пользователь клонирует вебхук с именем "Staging Integration"
    Then создан новый вебхук "Staging Integration"
    And все параметры скопированы кроме URL
    And пользователь может изменить URL клонированного вебхука
    And действие в аудит лог записано как "webhook_cloned"
      | field           | value           |
      | original_id     | <original_id>   |
      | cloned_id       | <cloned_id>     |
      | timestamp       | <iso8601>       |
      | user_id         | <user_id>       |

  @use_case=uc_07_01_06
  @critical
  Scenario: Отправка webhook при алерте
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Alerts Integration" настроен как канал уведомлений
    When срабатывает алерт
    Then POST запрос отправлен на webhook URL
    And тело запроса содержит детали алерта
    And тело запроса содержит детали монитора
    And в audit log записано событие "webhook_alert_sent" с webhook ID, alert ID и HTTP статусом

  @use_case=uc_07_01_07a
  @critical
  @integration
  Scenario: Обработка 4xx ошибок при отправке вебхука
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Broken Integration" настроен как канал уведомлений
    And endpoint webhook возвращает "404 Not Found"
    When срабатывает алерт
    Then webhook помечен как "failed" с кодом "NOT_FOUND"
    And попытки повтора не выполняются (non-retryable - 4xx client error)
    And пользователю отправлено уведомление о неработающем вебхуке
    And в логах записана ошибка с кодом ответа
    And в audit log записано событие "webhook_delivery_failed" с кодом ответа и webhook ID
    And retry_scheduled = false

  @use_case=uc_07_01_07b
  @critical
  @integration
  Scenario: Ошибка DNS при отправке вебхука b
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Invalid Domain" имеет URL "https://nonexistent-domain.example/webhook"
    When срабатывает алерт
    Then webhook помечен как "failed" с кодом "DNS_RESOLUTION_FAILED"
    And попытки повтора не выполняются (non-retryable - permanent DNS error)
    And пользователю отправлено уведомление "DNS resolution failed"
    And в логах записана ошибка DNS
    And в audit log записано событие "webhook_dns_error" с доменным именем и webhook ID
    And retry_scheduled = false

  @use_case=uc_07_01_07c
  @critical
  @integration
  Scenario: Ошибка валидации SSL сертификата c
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Invalid Cert" имеет URL с истекшим SSL сертификатом
    When срабатывает алерт
    Then webhook помечен как "failed" с кодом "SSL_CERTIFICATE_INVALID"
    And в логах записана ошибка "certificate expired"
    And в audit log записано событие "webhook_ssl_error" с типом ошибки SSL и webhook ID
    And пользователю отправлено уведомление о проблеме SSL
    And retry_scheduled = false (non-retryable - permanent SSL error)

  @use_case=uc_07_01_07d
  @critical
  @integration
  Scenario: Обработка таймаута вебхука - попытка 1
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Slow Endpoint" настроен как канал уведомлений
    And timeout вебхука составляет "10 seconds"
    And endpoint отвечает через "30 seconds"
    When срабатывает алерт
    Then соединение разорвано после "10 seconds"
    And webhook помечен как "failed" с кодом "CONNECTION_TIMEOUT"
    And выполнена retry попытка через "30 seconds" используя exponential backoff
    And ошибка является retryable (временный timeout)
    And в логах записана ошибка "timeout"
    And в audit log записано событие "webhook_timeout" с timeout значением и webhook ID
    And retry_scheduled = true
    And retry_attempt = 1
    And max_retries = 3

  @use_case=uc_07_01_07e
  @critical
  @integration
  Scenario: Обработка разрыва соединения при отправке
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Unstable Endpoint" настроен как канал уведомлений
    And соединение разрывается во время отправки
    When срабатывает алерт
    Then попытка отправки помечена как failed
    And выполнена повторная попытка
    And в логах записана ошибка "connection reset"
    And в audit log записано событие "webhook_connection_reset" с webhook ID и деталями ошибки

  @use_case=uc_07_01_07f
  @critical
  @integration
  Scenario: Переполнение очереди вебхуков
    Given монитор "API Service" находится в состоянии флаппинга
    And webhook очередь имеет лимит "1000" сообщений
    And в очереди уже находится "1000" неотправленных вебхуков
    When срабатывает новый алерт
    Then новый webhook отклонён с ошибкой "WEBHOOK_QUEUE_OVERFLOW"
    And ответ содержит структуру:
      | error.code    | WEBHOOK_QUEUE_OVERFLOW |
      | error.message | Webhook queue is full |
      | error.details | {...} |
      | error.request_id | uuid |
    And в логах записано "webhook queue full"
    And алерт сохранён в истории как "not delivered"
    And пользователю отправлено уведомление о потере алерта
    And в audit log записано событие "webhook_queue_overflow" с размером очереди и лимитом

  @use_case=uc_07_01_08
  @critical
  Scenario: Ограничение размера payload вебхука с обрезанием
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Size Limited" имеет лимит payload "1MB"
    And алерт содержит данные размером "2MB"
    And webhook настроен на обрезание payload при превышении лимита
    When срабатывает алерт
    Then payload обрезан до лимита "1MB"
    And webhook отправлен успешно с обрезанным payload
    And ответ содержит структуру:
      | success | true |
      | payload_truncated | true |
      | original_size | 2MB |
      | truncated_size | 1MB |
    And в audit log записано событие "webhook_payload_truncated" с размером payload и webhook ID

  @use_case=uc_07_01_09
  @critical
  Scenario: Ограничение размера payload вебхука с отклонением
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Size Limited" имеет лимит payload "1MB"
    And алерт содержит данные размером "2MB"
    And webhook настроен на отклонение при превышении лимита
    When срабатывает алерт
    Then webhook отклонён с ошибкой "PAYLOAD_TOO_LARGE"
    And ответ содержит структуру:
      | error.code | PAYLOAD_TOO_LARGE |
      | error.message | Payload size exceeds limit |
      | error.details.limit | 1MB |
      | error.details.actual_size | 2MB |
      | error.request_id | uuid |
    And webhook не отправлен
    And в audit log записано событие "webhook_payload_size_exceeded" с размером payload и webhook ID

  @use_case=uc_07_01_10
  Scenario: Вебхук с условной отправкой
    Given webhook "Critical Only" настроен на отправку только для critical алертов
    And монитор "API Service" имеет статус "DOWN" с severity "warning"
    When срабатывает алерт
    Then webhook не отправлен
    And в логах записано "webhook skipped: severity not critical"
    And в audit log записано событие "webhook_skipped_condition" с webhook ID и условием

  @use_case=uc_07_01_11
  @critical
  Scenario: Приоритет вебхуков при отправке
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Critical Alerts" имеет приоритет "high"
    And webhook "Low Priority Logs" имеет приоритет "low"
    And оба вебхука настроены как каналы уведомлений
    When срабатывает алерт
    Then webhook "Critical Alerts" отправлен первым
    And webhook "Low Priority Logs" отправлен вторым
    And время отправки приоритетного webhook минимально
    And в audit log записано событие "webhooks_sent_by_priority" с количеством и порядком

  @use_case=uc_07_01_12
  @critical
  Scenario: Ошибка доставки webhook с повторными Попытками a
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Flaky Endpoint" настроен как канал уведомлений
    And endpoint webhook возвращает "503 Service Unavailable"
    When срабатывает алерт
    Then первая попытка отправки завершается с ошибкой
    And выполнена повторная попытка через "30 seconds"
    And выполнена вторая попытка через "60 seconds"
    And выполнена третья попытка через "120 seconds"
    And после исчерпания попыток webhook помечен как "failed"
    And в логах записаны все попытки с кодами ответов
    And в audit log записано событие "webhook_retry_attempts_exhausted" с количеством попыток и webhook ID

  @use_case=uc_07_01_13a
  Scenario: Исчерпание попыток повтора вебхука
    Given монитор "API Service" имеет статус "DOWN"
    And webhook "Unreachable Endpoint" настроен как канал уведомлений
    And все "3" попытки отправки завершились ошибкой
    When срабатывает алерт
    Then webhook автоматически деактивирован
    And пользователю отправлено уведомление о деактивации
    And алерт не удаляется из очереди доставки
    And в истории монитора записано "webhook delivery failed"

  @use_case=uc_07_01_13b
  Scenario: Экспоненциальная задержка между повторами
    Given webhook "Flaky Endpoint" не отвечает
    And первая попытка отправки выполнена в "00:00:00"
    When первая попытка завершилась ошибкой "503"
    Then вторая попытка выполнена через "30 seconds"
    And третья попытка выполнена через "60 seconds" после второй
    And четвёртая попытка выполнена через "120 seconds" после третьей
    And время между попытками удваивается

  @use_case=uc_07_01_14
  Scenario: Исправление неотвечающего вебхука
    Given webhook "Broken Integration" имеет статус "failed"
    And endpoint webhook возвращает "503 Service Unavailable"
    When endpoint исправлен и отвечает "200 OK"
    And пользователь повторно активирует вебхук
    Then webhook имеет статус "active"
    And следующая попытка отправки успешна
    And в audit log записано событие "webhook_reactivated" с webhook ID и timestamp

  @use_case=uc_07_01_15
  @critical
  @implemented
  Scenario: Удаление webhook интеграции
    Given пользователь имеет вебхук "Test Webhook"
    When пользователь удаляет вебхук
    Then вебхук удалён
    And вебхук больше не появляется в списке
    And действие в аудит лог записано как "webhook_deleted"
      | field     | value     |
      | webhook_id | <webhook_id> |
      | timestamp | <iso8601> |
      | user_id   | <user_id> |

  @use_case=uc_07_01_16
  Scenario: Массовое удаление вебхуков
    Given пользователь имеет "10" вебхуков
    When пользователь выбирает "5" вебхуков для массового удаления
    Then выбранные "5" вебхуков удалены
    And остальные "5" вебхуков не затронуты
    And в audit log записано событие "webhooks_bulk_deleted" с количеством "5"

  @use_case=uc_07_01_17
  @implemented
  Scenario: Список вебхуков
    Given пользователь имеет "3" вебхука
    When пользователь запрашивает список вебхуков
    Then возвращается "3" вебхука
    And каждый содержит имя и URL

  @use_case=uc_07_01_18
  Scenario: Тестирование вебхука
    Given существует вебхук "Custom Integration"
    When пользователь тестирует вебхук
    Then отправлен тестовый POST запрос
    And получен статус ответа
    And получено время ответа

  @use_case=uc_07_01_19
  Scenario: Настройка вебхука как канала уведомлений
    Given пользователь имеет вебхук "Alerts Integration"
    And пользователь имеет монитор "API Service"
    When пользователь добавляет вебхук как канал уведомлений
    Then алерты монитора отправляются на webhook URL
    And формат запроса соответствует спецификации

  @use_case=uc_07_01_20
  @critical
  Scenario: Подпись webhook с HMAC
    Given пользователь создаёт webhook с секретным ключом
    When отправляется алерт на webhook
    Then запрос содержит заголовок "X-Webhook-Signature"
    And подпись вычислена как HMAC-SHA256 от тела запроса
    And подпись закодирована в hex
    And в audit log записано событие "webhook_hmac_signed" с webhook ID и алгоритмом подписи

  @use_case=uc_07_01_21a
  @critical
  Scenario: Проверка подписи webhook получателем
    Given webhook настроен с секретным ключом
    When получатель получает webhook запрос с неверной подписью
    Then получатель отклоняет запрос с ошибкой "INVALID_SIGNATURE"
    And в audit log записано событие "webhook_signature_verification_failed" с webhook ID и timestamp

  @use_case=uc_07_01_21b
  @critical
  Scenario: Защита от replay атак
    Given webhook отправляет алерт с ID "alert-123"
    And запрос содержит заголовок "X-Webhook-Timestamp"
    When получатель получает webhook с тем же ID и timestamp
    Then webhook с дублирующимся ID отклоняется
    And в логах записано "duplicate webhook ignored"
    And timestamp проверяется на актуальность (не старше 5 minutes)
    And в audit log записано событие "webhook_replay_attack_blocked" с webhook ID, alert ID и timestamp

  @use_case=uc_07_01_22
  Scenario: Статистика использования вебхука
    Given webhook "Analytics Integration" создан "30" дней назад
    And webhook использовался для "150" отправок
    When пользователь запрашивает статистику вебхука
    Then возвращена статистика:
      | total_sent      | 150 |
      | successful      | 142 |
      | failed          | 8 |
      | avg_response_ms | 245 |
      | last_used       | <timestamp> |
    And в audit log записано событие "webhook_statistics_accessed" с webhook ID

