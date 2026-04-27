@epic=07_integrations
@user_story=07_02_api_access
# Description: API ключи для программного доступа

Feature: API ключи
  Как разработчик
  Я хочу создавать API ключи
  Чтобы автоматизировать работу с мониторингом

  @use_case=uc_07_02_01
  @critical
  @implemented
  Scenario: Создание API ключа
    Given пользователь авторизован
    When пользователь создаёт API ключ с параметрами:
      | name   | Read-only key     |
      | scopes | read_monitors     |
    Then API ключ создан успешно
    And возвращён полный ключ только один раз
    And ключ имеет префикс "baku_ro_"
    And ключ активен
    And в audit log записано событие "api_key_created" с деталями ключа

  @use_case=uc_07_02_02
  @critical
  Scenario: Создание API ключа с описанием
    Given пользователь авторизован
    When пользователь создаёт API ключ с параметрами:
      | name        | Integration Key |
      | scopes      | read_monitors,write_monitors |
      | description | Used for CI/CD pipeline |
    Then API ключ создан с описанием
    And описание отображается в списке ключей
    And описание помогает идентифицировать назначение ключа
    And в audit log записано событие "api_key_created_with_description" с ID ключа и описанием

  @use_case=uc_07_02_03
  @critical
  Scenario: Создание API ключа с сроком действия
    Given пользователь авторизован
    When пользователь создаёт API ключ с параметрами:
      | name      | Temporary Key |
      | scopes    | read_monitors |
      | expires_at | 2026-04-01T00:00:00Z |
    Then API ключ создан с датой истечения
    And после "2026-04-01T00:00:00Z" ключ не работает
    And в audit log записано событие "api_key_created_with_expiration" с ID ключа и датой истечения

  @use_case=uc_07_02_04
  @critical
  Scenario: Конкурентное создание API ключей
    Given пользователь авторизован
    When пользователь отправляет "3" одновременных запроса на создание API ключей
    Then все "3" ключа созданы успешно
    And все ключи имеют уникальные ID
    And ни один ключ не перезаписал другой
    And в audit log записаны "3" события "api_key_created"

  @use_case=uc_07_02_05
  @critical
  Scenario: Использование API ключа для запросов
    Given пользователь имеет API ключ с scopes "read_monitors"
    When пользователь делает запрос с этим ключом
    Then запрос выполнен успешно
    And возвращены данные мониторов
    And в audit log записано событие "api_key_used" с ID ключа, endpoint, HTTP статусом и timestamp
    And в audit log записано событие "api_key_used" с ID ключа и endpoint

  @use_case=uc_07_02_06a
  @critical
  Scenario: Запрос с невалидным API ключом
    Given пользователь делает API запрос
    And заголовок Authorization содержит ключ "invalid_key"
    When запрос отправлен
    Then получен ответ с кодом "401"
    And тело ответа содержит ошибку "INVALID_API_KEY"
    And ответ содержит структуру:
      | error.code    | INVALID_API_KEY |
      | error.message | Invalid API key |
      | error.details | {...} |
      | error.request_id | uuid |
    And время ответа не превышает "100ms"
    And в audit log записана неудачная попытка аутентификации с IP и timestamp

  @use_case=uc_07_02_06b
  @critical
  Scenario: Отклоняет запросы с деактивированным API ключом a
    Given API ключ "Old Key" имеет статус "inactive"
    When пользователь делает запрос с ключом "Old Key"
    Then получен ответ с кодом "401"
    And тело ответа содержит ошибку "API_KEY_DEACTIVATED"
    And ответ содержит структуру:
      | error.code    | API_KEY_DEACTIVATED |
      | error.message | API key deactivated |
      | error.details | {...} |
      | error.request_id | uuid |
    And в логах записана попытка использования неактивного ключа
    And в audit log записана попытка с ID ключа и timestamp

  @use_case=uc_07_02_06c
  @critical
  Scenario: Отклоняет запросы с удалённым API ключом a
    Given API ключ "Deleted Key" был удалён
    When пользователь делает запрос с ключом "Deleted Key"
    Then получен ответ с кодом "401"
    And тело ответа содержит ошибку "API_KEY_NOT_FOUND"
    And ответ содержит структуру:
      | error.code    | API_KEY_NOT_FOUND |
      | error.message | API key not found |
      | error.details | {...} |
      | error.request_id | uuid |
    And в audit log записана попытка использования удалённого ключа с timestamp

  @use_case=uc_07_02_06d
  @critical
  @boundary
  Scenario: Истечение API ключа точно на границе срока действия
    Given API ключ с expires_at "2026-04-01T00:00:00Z"
    And текущее время "2026-04-01T00:00:00Z"
    When пользователь делает запрос с этим ключом
    Then получен ответ с кодом "401"
    And тело ответа содержит ошибку "API_KEY_EXPIRED"
    And ответ содержит структуру:
      | error.code    | API_KEY_EXPIRED |
      | error.message | API key expired |
      | error.details | {...} |
      | error.request_id | uuid |
    And ключ отклонён как истёкший
    And в audit log записана попытка использования истёкшего ключа с ID ключа и timestamp

  @use_case=uc_07_02_06e
  @critical
  Scenario: Валидация scope API ключа
    Given API ключ имеет scope "read_monitors"
    When пользователь пытается создать монитор с этим ключом
    Then получен ответ с кодом "403 Forbidden"
    And тело ответа содержит ошибку "INSUFFICIENT_SCOPE"
    And ответ содержит структуру:
      | error.code    | INSUFFICIENT_SCOPE |
      | error.message | Insufficient scope |
      | error.details | {...} |
      | error.request_id | uuid |
    And тело ответа содержит required_scope="write_monitors"
    And в логах записана попытка несанкционированного действия
    And в audit log записана попытка с ID ключа, запрошенным scope и timestamp

  @use_case=uc_07_02_06
  @critical
  Scenario: Белый список IP для API ключа
    Given API ключ настроен с белым списком IP ["1.2.3.4", "5.6.7.8"]
    When запрос сделан с IP "9.10.11.12"
    Then получен ответ с кодом "403 Forbidden"
    And тело ответа содержит ошибку "IP_NOT_ALLOWED"
    And ответ содержит структуру:
      | error.code    | IP_NOT_ALLOWED |
      | error.message | IP not allowed |
      | error.details | {...} |
      | error.request_id | uuid |
    And в логах записан IP адрес нарушителя
    And в audit log записана попытка с ID ключа, IP адресом и timestamp

  @use_case=uc_07_02_07a
  @critical
  Scenario: Rate limiting API запросов
    Given пользователь имеет API ключ
    When пользователь отправляет "101" запросов в минуту
    Then 101-й запрос отклонён с кодом "429"
    And ответ содержит заголовок "X-RateLimit-Limit: 100"
    And ответ содержит заголовок "X-RateLimit-Reset"
    And в audit log записано событие "rate_limit_exceeded" с ID ключа и количеством запросов

  @use_case=uc_07_02_07b
  @critical
  Scenario: Восстановление после rate limit    Given пользователь исчерпал лимит "100" запросов в минуту
    And получен ответ "429 Too Many Requests"
    And заголовок X-RateLimit-Reset содержит timestamp
    When пользователь повторяет запрос до времени reset
    Then получен ответ "429 Too Many Requests"
    When пользователь повторяет запрос после времени reset
    Then запрос выполнен успешно
    And в audit log записано событие "rate_limit_reset" с ID ключа и временем сброса

  @use_case=uc_07_02_07c
  Scenario: Предупреждение об истечении API ключа
    Given API ключ истекает через "7" дней
    When пользователь делает запрос с этим ключом
    Then заголовок ответа содержит "X-ApiKey-Expiring: true"
    And заголовок содержит "X-ApiKey-Expires-At: 2026-04-01T00:00:00Z"

  @use_case=uc_07_02_07
  Scenario: Запрос с версией API    Given текущая версия API "v1"
    When пользователь делает запрос с заголовком "API-Version: v1"
    Then получен ответ в формате API v1
    And заголовок ответа содержит "API-Version: v1"

  @use_case=uc_07_02_08a
  @critical
  Scenario: Запрос с устаревшей версией API    Given версия API "v1" устарела
    And текущая версия "v2"
    When пользователь делает запрос с заголовком "API-Version: v1"
    Then получен ответ с кодом "410 Gone"
    And тело ответа содержит ошибку "API_VERSION_DEPRECATED"
    And ответ содержит структуру:
      | error.code    | API_VERSION_DEPRECATED |
      | error.message | API version deprecated |
      | error.details | {...} |
      | error.request_id | uuid |
    And тело ответа содержит рекомендуемую версию "use API v2"
    And заголовок содержит ссылку на документацию миграции
    And в audit log записано событие "deprecated_api_version_access" с версией API и ID ключа

  @use_case=uc_07_02_08b
  Scenario: Предупреждение об устаревшем endpoint    Given endpoint /v1/monitors устарел
    And дата отключения "2026-06-01"
    When пользователь делает запрос на /v1/monitors
    Then запрос выполнен успешно
    And заголовок ответа содержит "Deprecation: true"
    And заголовок ответа содержит "Sunset: 2026-06-01"
    And заголовок ответа содержит "Link: </v2/monitors>; rel=successor-version"

  @use_case=uc_07_02_08
  Scenario: Пагинация с максимальным размером страницы
    Given пользователь имеет "1000" мониторов
    And максимальный page_size составляет "100"
    When пользователь запрашивает страницу с page_size=1000
    Then pageSize ограничен до "100"
    And возвращено только "100" мониторов
    And в ответе есть предупреждение "page_size capped at max limit"

  @use_case=uc_07_02_09
  Scenario: Пагинация при изменении данных
    Given пользователь имеет "50" мониторов
    And пользователь запрашивает первую страницу с page_size=20
    When пользователь добавляет "5" мониторов
    And пользователь запрашивает вторую страницу
    Then второй запрос возвращает мониторы 21-40 из исходного набора
    And новые мониторы не дублируют старые
    And поле page_info содержит consistent_hash для проверки

  @use_case=uc_07_02_10a
  Scenario: Стандартизированный формат ошибок
    Given происходит ошибка валидации
    When возвращается ответ об ошибке
    Then ответ содержит JSON объект "error"
    And error.code содержит строковый код "VALIDATION_ERROR"
    And error.message содержит читаемое описание
    And error.details содержит массив деталей ошибок
    And error.request_id содержит UUID для трассировки
    And Content-Type содержит "application/json"

  @use_case=uc_07_02_10b
  @critical
  @integration
  Scenario: Обработка сетевой ошибки при валидации API ключа
    Given пользователь делает API запрос
    And заголовок Authorization содержит ключ "test-key"
    And сервис валидации ключей недоступен
    When запрос отправлен
    Then получен ответ с кодом "503 Service Unavailable"
    And тело ответа содержит ошибку "SERVICE_UNAVAILABLE"
    And запрос не выполнен
    And в audit log записано событие "api_key_validation_service_unavailable" с timestamp

  @use_case=uc_07_02_10
  @critical
  @implemented
  Scenario: Отзыв API ключа
    Given пользователь имеет API ключ "test-key"
    When пользователь удаляет ключ
    Then ключ удалён
    And ключ больше не работает для аутентификации
    And в audit log записано событие "api_key_deleted" с ID ключа

  @use_case=uc_07_02_11a
  @critical
  Scenario: Ошибка при обновлении несуществующего API ключа a
    Given пользователь авторизован
    When пользователь пытается обновить API ключ с ID "nonexistent-key"
    Then получен ответ с кодом "404"
    And тело ответа содержит ошибку "API_KEY_NOT_FOUND"
    And ответ содержит структуру:
      | error.code    | API_KEY_NOT_FOUND |
      | error.message | API key not found |
      | error.details | {...} |
      | error.request_id | uuid |
    And в audit log записана попытка обновления несуществующего ключа с ID и timestamp

  @use_case=uc_07_02_11
  @critical
  Scenario: Конкурентное обновление API ключа с успехом
    Given пользователь авторизован
    And пользователь имеет API ключ "Concurrent Key"
    When пользователь отправляет "2" одновременных запроса на обновление ключа
    Then только первое обновление применено успешно
    And второе обновление отклонено с ошибкой "CONCURRENT_UPDATE"
    And ответ содержит структуру:
      | error.code       | CONCURRENT_UPDATE |
      | error.message    | Concurrent update detected |
      | error.details    | {"key_id": "uuid"} |
      | error.request_id | uuid |
    And в audit log записано событие "api_key_concurrent_update_attempt" с ID ключа и timestamp
    And параметры ключа соответствуют первому запросу

  @use_case=uc_07_02_12a
  @critical
  Scenario: Конкурентное обновление API ключа с ошибкой
    Given пользователь авторизован
    And пользователь имеет API ключ "Concurrent Key"
    And существует активное обновление того же ключа
    When пользователь отправляет запрос на обновление ключа
    Then получен ответ с кодом "409 Conflict"
    And тело ответа содержит ошибку "CONCURRENT_UPDATE"
    And ответ содержит структуру:
      | error.code       | CONCURRENT_UPDATE |
      | error.message    | Concurrent update in progress |
      | error.details    | {"key_id": "uuid", "conflicting_request_id": "uuid"} |
      | error.request_id | uuid |
    And обновление не применено
    And в audit log записано событие "api_key_concurrent_update_rejected" с ID ключа и timestamp

  @use_case=uc_07_02_12
  @implemented
  Scenario: Список API ключей
    Given пользователь имеет "3" API ключа
    When пользователь запрашивает список ключей
    Then возвращается "3" ключа
    And каждый ключ содержит только префикс

  @use_case=uc_07_02_13a
  @critical
  Scenario: Деактивация API ключа
    Given пользователь имеет активный API ключ
    When пользователь деактивирует ключ
    Then ключ имеет статус "inactive"
    And ключ не работает для аутентификации
    And в audit log записано событие "api_key_deactivated" с ID ключа

  @use_case=uc_07_02_13b
  Scenario: Повторная активация деактивированного API ключа
    Given пользователь авторизован
    And пользователь имеет деактивированный API ключ "Temporarily Disabled"
    When пользователь активирует ключ
    Then ключ имеет статус "active"
    And ключ снова работает для аутентификации
    And в audit log записано событие "api_key_activated" с ID ключа и timestamp

  @use_case=uc_07_02_13
  Scenario: Временное отключение API ключа для технического обслуживания
    Given пользователь авторизован
    And пользователь имеет API ключ "Maintenance Key"
    When пользователь устанавливает статус ключа "maintenance"
    Then ключ имеет статус "maintenance"
    And запросы с этим ключом возвращают ошибку "API_KEY_UNDER_MAINTENANCE"
    And в audit log записано событие "api_key_maintenance_mode" с ID ключа и timestamp
    When пользователь завершает техническое обслуживание
    Then ключ восстановлен в предыдущее состояние
    And в audit log записано событие "api_key_maintenance_completed" с ID ключа

  @use_case=uc_07_02_14
  @critical
  Scenario: Обновление имени API ключа
    Given пользователь авторизован
    And пользователь имеет API ключ "Old Key Name"
    When пользователь обновляет имя ключа на "New Key Name"
    Then имя ключа успешно обновлено
    And ключ остаётся активным
    And все остальные параметры ключа не изменились
    And в audit log записано событие "api_key_name_updated" с ID ключа, старым именем и новым именем

  @use_case=uc_07_02_15
  @critical
  Scenario: Обновление scopes API ключа
    Given пользователь авторизован
    And пользователь имеет API ключ с scopes "read_monitors"
    When пользователь обновляет scopes на "read_monitors,write_monitors"
    Then scopes ключа успешно обновлены
    And ключ имеет новые разрешения
    And предыдущие scopes больше не действуют
    And в audit log записано событие "api_key_scopes_updated" с ID ключа, старыми scopes и новыми scopes

  @use_case=uc_07_02_16
  @critical
  Scenario: Ротация секрета API ключа
    Given пользователь авторизован
    And пользователь имеет API ключ "Production Key"
    When пользователь запрашивает ротацию секрета ключа
    Then генерируется новый секрет для того же ключа
    And старый секрет немедленно деактивирован
    And возвращён новый секрет только один раз
    And ID ключа не изменяется
    And в audit log записано событие "api_key_secret_rotated" с ID ключа и timestamp

  @use_case=uc_07_02_17
  Scenario: Обновление белого списка IP API ключа
    Given пользователь авторизован
    And пользователь имеет API ключ с белым списком IP ["1.2.3.4"]
    When пользователь обновляет белый список на ["1.2.3.4", "5.6.7.8", "9.10.11.12"]
    Then белый список успешно обновлён
    And запросы с новых IP разрешены
    And в audit log записано событие "api_key_ip_whitelist_updated" с ID ключа, старым списком и новым списком

  @use_case=uc_07_02_18
  Scenario: Продление срока действия API ключа
    Given пользователь авторизован
    And пользователь имеет API ключ с expires_at "2026-04-01T00:00:00Z"
    When пользователь обновляет expires_at на "2026-12-31T23:59:59Z"
    Then срок действия ключа успешно продлён
    And ключ продолжает работать после "2026-04-01T00:00:00Z"
    And в audit log записано событие "api_key_expiration_extended" с ID ключа, старой датой и новой датой

  @use_case=uc_07_02_19
  @critical
  Scenario: Удаление описания из API ключа
    Given пользователь авторизован
    And пользователь имеет API ключ с описанием "Old description"
    When пользователь удаляет описание ключа
    Then описание ключа удалено
    And остальные параметры ключа не изменились
    And в audit log записано событие "api_key_description_removed" с ID ключа и старым описанием

  @use_case=uc_07_02_20a
  @critical
  @integration
  Scenario: Сбой базы данных при обновлении API ключа a
    Given пользователь авторизован
    And имеет активный API ключ
    And база данных недоступна для записи
    When пользователь обновляет параметры API ключа
    Then возвращается ошибка "DATABASE_WRITE_FAILED"
    And ответ содержит структуру:
      | error.code       | DATABASE_WRITE_FAILED |
      | error.message    | Failed to update API key |
      | error.details    | {"operation": "update_api_key"} |
      | error.request_id | <uuid> |
    And API ключ не изменён
    And предыдущие параметры сохранены

  @use_case=uc_07_02_20
  @critical
  Scenario: Импорт мониторов из UptimeRobot
    Given пользователь авторизован
    And пользователь имеет файл импорта "uptimerobot_export.json"
    When пользователь импортирует мониторы из UptimeRobot
    Then мониторы созданы успешно
    And настройки мониторов сохранены
    And история импорта записана
    And в audit log записано событие "monitors_imported" с источником "uptimerobot" и количеством

  @use_case=uc_07_02_21a
  @critical
  Scenario: Валидация при импорте мониторов
    Given пользователь авторизован
    And пользователь имеет JSON файл с невалидными данными
    When пользователь импортирует мониторы
    Then невалидные мониторы пропущены
    And возвращён список ошибок валидации
    And валидные мониторы созданы

  @use_case=uc_07_02_21b
  @critical
  Scenario: Проверка лимита мониторов при импорте
    Given пользователь имеет тариф "Free"
    And пользователь имеет "25" мониторов
    And пользователь имеет файл с "5" мониторами
    When пользователь импортирует мониторы
    Then система возвращает ошибку "MONITOR_LIMIT_EXCEEDED"
    And ответ содержит структуру:
      | error.code    | MONITOR_LIMIT_EXCEEDED |
      | error.message | Monitor limit exceeded |
      | error.details | {...} |
      | error.request_id | uuid |
    And ни один монитор не создан
    And в audit log записано событие "monitor_limit_exceeded" с тарифом, текущим количеством и запрошенным количеством

  @use_case=uc_07_02_21c
  Scenario: Импорт с конфликтом имён
    Given пользователь имеет монитор "API Service"
    And импортируемый файл содержит монитор "API Service"
    When пользователь импортирует мониторы без флага перезаписи
    Then дублирующийся монитор пропущен
    And возвращён список конфликтов
    And валидные мониторы созданы
    And ни один существующий монитор не изменён

  @use_case=uc_07_02_21
  Scenario: Импорт с перезаписью существующих данных
    Given пользователь имеет монитор "API Service"
    And импортируемый файл содержит монитор "API Service"
    When пользователь импортирует мониторы с флагом "overwrite: true"
    Then существующий монитор "API Service" обновлён
    And возвращён список обновлённых мониторов
    And в истории записана информация о перезаписи

  @use_case=uc_07_02_22a
  @critical
  @integration
  Scenario: Таймаут при импорте мониторов из UptimeRobot API a
    And UptimeRobot API не отвечает в течение "30" секунд
    When пользователь импортирует мониторы из UptimeRobot
    Then возвращается ошибка "IMPORT_TIMEOUT"
    And ответ содержит структуру:
      | error.code       | IMPORT_TIMEOUT |
      | error.message    | Import timeout |
      | error.details    | {"source": "uptimerobot", "timeout_seconds": 30} |
      | error.request_id | <uuid> |
    And ни один монитор не создан
    And пользователю предложено повторить попытку

  @use_case=uc_07_02_22
  @critical
  Scenario: Импорт мониторов из Pingdom
    Given пользователь авторизован
    And пользователь имеет файл импорта "pingmon_export.json"
    When пользователь импортирует мониторы из Pingdom
    Then мониторы созданы успешно
    And настройки мониторов сохранены
    And история импорта записана
    And в audit log записано событие "monitors_imported" с источником "pingdom" и количеством

  @use_case=uc_07_02_23
  @critical
  Scenario: Импорт мониторов из CSV
    Given пользователь авторизован
    And пользователь имеет CSV файл с мониторами
    When пользователь импортирует мониторы из CSV
    Then мониторы созданы успешно
    And валидные мониторы добавлены
    And невалидные строки пропущены с ошибкой
    And в audit log записано событие "monitors_imported" с источником "csv" и количеством

  @use_case=uc_07_02_24
  @critical
  Scenario: Импорт мониторов из JSON
    Given пользователь авторизован
    And пользователь имеет JSON файл с мониторами
    When пользователь импортирует мониторы из JSON
    Then мониторы созданы успешно
    And настройки мониторов сохранены
    And история импорта записана
    And в audit log записано событие "monitors_imported" с источником "json" и количеством

  @use_case=uc_07_02_25
  Scenario: История импорта мониторов
    Given пользователь выполнил "3" импорта
    When пользователь запрашивает историю импорта
    Then возвращается "3" записи импорта
    And каждая запись содержит дату, источник и количество

  @use_case=uc_07_02_26
  @critical
  Scenario: Отслеживание использования API ключа
    Given API ключ "Analytics Key" создан "30" дней назад
    And ключ использовался для "500" запросов
    When пользователь запрашивает статистику использования ключа
    Then возвращена статистика:
      | total_requests | 500 |
      | successful     | 485 |
      | failed         | 15 |
      | avg_latency_ms | 95 |
      | last_used      | <timestamp> |
      | most_used_endpoint | /api/v1/monitors |
    And в audit log записано событие "api_key_statistics_accessed" с ID ключа

  @use_case=uc_07_02_27
  Scenario: Экспорт истории использования API ключа
    Given пользователь имеет API ключ "Production Key"
    And ключ использовался в течение последних "90" дней
    When пользователь запрашивает экспорт истории использования
    Then сгенерирован CSV файл с историей запросов
    And файл содержит колонки: timestamp, endpoint, status_code, latency_ms, ip_address
    And файл доступен для загрузки в течение "1" часа
    And в audit log записано событие "api_key_history_exported" с ID ключа и периодом
