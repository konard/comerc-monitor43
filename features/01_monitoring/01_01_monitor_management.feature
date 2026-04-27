@epic=01_monitoring
@user_story=01_01_monitor_management
# Description: CRUD операции для HTTP/HTTPS мониторов

Feature: Управление мониторами
  Как пользователь
  Я хочу управлять своими мониторами
  Чтобы отслеживать доступность моих сервисов

  @use_case=uc_01_01_01
  @critical
  Scenario: Создание нового HTTP монитора
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name     | API Service             |
      | url      | https://api.example.com |
      | interval | 5 minutes               |
    Then монитор должен быть создан успешно
    And монитор должен иметь статус "PENDING"
    And действие в аудит лог записано как "monitor_created"
      | field    | value                   |
      | name     | API Service             |
      | url      | https://api.example.com |
      | interval | 5 minutes               |

  @use_case=uc_01_01_02
  @critical
  Scenario: Получение деталей монитора
    Given пользователь имеет монитор с именем "API Service"
    When пользователь запрашивает детали монитора
    Then пользователь получает конфигурацию монитора
    And ответ содержит ID монитора
    And действие в аудит лог записано как "monitor_viewed"
      | field     | value       |
      | name      | API Service |
      | user_id   | <user_id>   |
      | timestamp | <iso8601>   |

  @use_case=uc_01_01_03
  @critical
  Scenario: Список всех мониторов
    Given пользователь имеет следующие мониторы:
      | name        | url                     |
      | API Service | https://api.example.com |
      | Web Site    | https://example.com     |
      | Database    | https://db.example.com  |
    When пользователь запрашивает список мониторов
    Then пользователь получает список из "3" мониторов
    And действие в аудит лог записано как "monitors_listed"
      | field     | value     |
      | count     | 3         |
      | user_id   | <user_id> |
      | timestamp | <iso8601> |

  @use_case=uc_01_01_04
  @critical
  Scenario: Обновление монитора
    Given пользователь имеет монитор с именем "API Service"
    When пользователь обновляет монитор с параметрами:
      | interval | 1 minute |
    Then конфигурация монитора обновлена
    And интервал проверки равен "1 minute"
    And действие в аудит лог записано как "monitor_updated"
      | field    | value    |
      | interval | 1 minute |

  @use_case=uc_01_01_05
  @critical
  Scenario: Удаление монитора
    Given пользователь имеет монитор с именем "API Service"
    When пользователь удаляет монитор
    Then монитор удалён
    And монитор больше не появляется в списке
    And действие в аудит лог записано как "monitor_deleted"
      | field   | value       |
      | name    | API Service |
      | deleted | true        |

  @use_case=uc_01_01_06
  @critical
  Scenario: Приостановка монитора
    Given пользователь имеет активный монитор с именем "API Service"
    When пользователь приостанавливает монитор
    Then статус монитора должен быть "PAUSED"
    And новые проверки не должны планироваться
    And действие в аудит лог записано как "monitor_paused"
      | field  | value       |
      | name   | API Service |
      | status | PAUSED      |

  @use_case=uc_01_01_07
  @critical
  Scenario: Возобновление монитора
    Given пользователь имеет приостановленный монитор с именем "API Service"
    When пользователь возобновляет монитор
    Then статус монитора должен быть "UP"
    And проверки должны планироваться
    And действие в аудит лог записано как "monitor_resumed"
      | field  | value       |
      | name   | API Service |
      | status | UP          |

  @use_case=uc_01_01_08
  @critical
  Scenario: Создание монитора с интервалом 15 секунд (платный тариф)
    Given пользователь имеет подписку "Starter"
    When пользователь создаёт монитор с параметрами:
      | name     | Critical Service        |
      | url      | https://api.example.com |
      | interval | 15 seconds              |
    Then монитор должен быть создан успешно
    And интервал проверки равен "15 seconds"
    And действие в аудит лог записано как "monitor_created"
      | field     | value            |
      | name      | Critical Service |
      | interval  | 15 seconds       |
      | tier      | Starter          |
      | user_id   | <user_id>        |
      | timestamp | <iso8601>        |

  @use_case=uc_01_01_09
  @critical
  Scenario: Создание монитора с таймаутом
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name     | API Service             |
      | url      | https://api.example.com |
      | interval | 5 minutes               |
      | timeout  | 10 seconds              |
    Then монитор должен быть создан успешно
    And таймаут проверки равен "10 seconds"

  @use_case=uc_01_01_10
  @critical
  Scenario: Создание монитора с рабочими часами
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name          | Business Hours API      |
      | url           | https://api.example.com |
      | interval      | 5 minutes               |
      | working_hours | 09:00-18:00             |
      | working_days  | Monday-Friday           |
    Then монитор должен быть создан успешно
    And рабочие часы настроены

  @use_case=uc_01_01_11
  @critical
  Scenario: Создание монитора с DEGRADED порогами
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name                   | API Service             |
      | url                    | https://api.example.com |
      | interval               | 5 minutes               |
      | degraded_response_time | 5 seconds               |
      | degraded_failure_rate  | 10%                     |
    Then монитор должен быть создан успешно
    And пороги для DEGRADED статуса настроены

  @use_case=uc_01_01_01a
  @critical
  @validation
  Scenario: Попытка создания монитора с дублирующимся именем a
    Given пользователь авторизован
    And пользователь имеет монитор с именем "API Service"
    When пользователь создаёт монитор с параметрами:
      | name     | API Service             |
      | url      | https://api.example.com |
      | interval | 5 minutes               |
    Then система возвращает ошибку "MONITOR_NAME_EXISTS"
    And монитор не создан

  @use_case=uc_01_01_01b
  @critical
  @business_rule
  Scenario: Превышение лимита мониторов на Free тарифе b
    Given пользователь имеет подписку "Free"
    And лимит мониторов для Free = "25"
    And пользователь имеет "25" активных мониторов
    When пользователь создаёт ещё один монитор
    Then система возвращает ошибку "MONITOR_LIMIT_REACHED"
    And монитор не создан
    And система предлагает апгрейд тарифа

  @use_case=uc_01_01_01c
  @critical
  @business_rule
  Scenario: Попытка создания монитора с интервалом 15 секунд на Free c
    Given пользователь имеет подписку "Free"
    When пользователь создаёт монитор с параметрами:
      | name     | Critical Service        |
      | url      | https://api.example.com |
      | interval | 15 seconds              |
    Then система возвращает ошибку "INTERVAL_REQUIRES_PAID_PLAN"
    And монитор не создан

  @use_case=uc_01_01_01d
  @critical
  @validation
  Scenario: Попытка создания монитора с невалидным URL d
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name | Invalid Monitor |
      | url  | not-a-valid-url |
    Then система возвращает ошибку "INVALID_URL"
    And монитор не создан

  @use_case=uc_01_01_01e
  @critical
  @validation
  Scenario: Попытка создания монитора с пустым именем e
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name |                         |
      | url  | https://api.example.com |
    Then система возвращает ошибку "NAME_REQUIRED"
    And монитор не создан

  @use_case=uc_01_01_01f
  @critical
  @validation
  Scenario: Попытка создания монитора с именем длиннее 255 символов f
    Given пользователь авторизован
    When пользователь создаёт монитор с именем из "300" символов
    Then система возвращает ошибку "NAME_TOO_LONG"
    And монитор не создан

  @use_case=uc_01_01_01g
  @critical
  @validation
  Scenario: Попытка создания монитора с нулевым интервалом g
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name     | API Service             |
      | url      | https://api.example.com |
      | interval | 0 seconds               |
    Then система возвращает ошибку "INVALID_INTERVAL"

  @use_case=uc_01_01_01h
  @critical
  @validation
  Scenario: Попытка создания монитора с таймаутом больше интервала h
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | interval | 1 minute  |
      | timeout  | 2 minutes |
    Then система возвращает ошибку "TIMEOUT_EXCEEDS_INTERVAL"

  @use_case=uc_01_01_01i
  @critical
  @validation
  Scenario: Попытка создания монитора с невалидными рабочими часами i
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | working_hours | 25:00-30:00 |
    Then система возвращает ошибку "INVALID_WORKING_HOURS"

  @use_case=uc_01_01_04a
  @critical
  @validation
  Scenario: Попытка обновления монитора на дублирующееся имя a
    Given пользователь имеет мониторы "API Service" и "Web Site"
    When пользователь обновляет "API Service" с именем "Web Site"
    Then система возвращает ошибку "MONITOR_NAME_EXISTS"
    And имя не изменено

  @use_case=uc_01_01_01j
  @critical
  @security
  Scenario: Создание монитора с специальными символами в имени
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name | <script>alert('xss')</script> |
      | url  | https://api.example.com       |
    Then специальные символы экранированы или удалены
    And монитор создан с безопасным именем

  @use_case=uc_01_01_01k
  @critical
  @validation
  Scenario: Попытка создания монитора с отрицательным порогом DEGRADED k
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | degraded_failure_rate | -5% |
    Then система возвращает ошибку "INVALID_THRESHOLD"

  @use_case=uc_01_01_01l
  @critical
  @security
  Scenario: Попытка создания монитора неавторизованным пользователем l
    Given пользователь не авторизован
    When пользователь создаёт монитор с параметрами:
      | name     | API Service             |
      | url      | https://api.example.com |
      | interval | 5 minutes               |
    Then система возвращает ошибку "UNAUTHORIZED"
    And монитор не создан
    And действие в аудит лог записано как "unauthorized_monitor_create_attempt"
      | field      | value     |
      | ip_address | <ip>      |
      | timestamp  | <iso8601> |

  @use_case=uc_01_01_02a
  @critical
  @security
  Scenario: Попытка просмотра чужого монитора a
    Given пользователь "UserA" имеет монитор "API Service"
    And пользователь "UserB" авторизован
    When пользователь "UserB" запрашивает детали монитора "API Service"
    Then система возвращает ошибку "FORBIDDEN"
    And детали монитора не показаны
    And действие в аудит лог записано как "unauthorized_monitor_view_attempt"
      | field       | value       |
      | attacker_id | UserB       |
      | target_id   | API Service |
      | ip_address  | <ip>        |
      | timestamp   | <iso8601>   |

  @use_case=uc_01_01_04b
  @critical
  @security
  Scenario: Попытка обновления чужого монитора b
    Given пользователь "UserA" имеет монитор "API Service"
    And пользователь "UserB" авторизован
    When пользователь "UserB" обновляет монитор "API Service" с параметрами:
      | interval | 1 minute |
    Then система возвращает ошибку "FORBIDDEN"
    And монитор не обновлен
    And действие в аудит лог записано как "unauthorized_monitor_update_attempt"
      | field       | value       |
      | attacker_id | UserB       |
      | target_id   | API Service |
      | ip_address  | <ip>        |
      | timestamp   | <iso8601>   |

  @use_case=uc_01_01_05a
  @critical
  @security
  Scenario: Попытка удаления чужого монитора a
    Given пользователь "UserA" имеет монитор "API Service"
    And пользователь "UserB" авторизован
    When пользователь "UserB" удаляет монитор "API Service"
    Then система возвращает ошибку "FORBIDDEN"
    And монитор не удален
    And действие в аудит лог записано как "unauthorized_monitor_delete_attempt"
      | field       | value       |
      | attacker_id | UserB       |
      | target_id   | API Service |
      | ip_address  | <ip>        |
      | timestamp   | <iso8601>   |

  @use_case=uc_01_01_01m
  @critical
  @security
  Scenario: Попытка SQL инъекции в поле URL монитора m
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name | API Service                                      |
      | url  | https://api.example.com'; DROP TABLE monitors;-- |
    Then система возвращает ошибку "INVALID_URL"
    And вредоносный код не выполнен
    And действие в аудит лог записано как "sql_injection_attempt"
      | field     | value                                   |
      | user_id   | <user_id>                               |
      | payload   | https://api.example.com'; DROP TABLE... |
      | timestamp | <iso8601>                               |

  @use_case=uc_01_01_01n
  @critical
  @security
  Scenario: Попытка XSS атаки в поле name монитора n
    Given пользователь авторизован
    When пользователь создаёт монитор с параметрами:
      | name | <img src=x onerror=alert('XSS')> |
      | url  | https://api.example.com          |
    Then специальные символы экранированы
    And монитор создан с безопасным именем
    And XSS атака предотвращена
    And действие в аудит лог записано как "xss_attack_attempt"
      | field     | value                            |
      | user_id   | <user_id>                        |
      | payload   | <img src=x onerror=alert('XSS')> |
      | timestamp | <iso8601>                        |

  @use_case=uc_01_01_12
  @critical
  @boundary
  Scenario: Создание монитора с именем ровно 255 символов
    Given пользователь авторизован
    When пользователь создаёт монитор с именем из "255" символов
    Then монитор создан успешно
    And имя сохранено полностью

  @use_case=uc_01_01_13
  @critical
  @boundary
  Scenario: Создание монитора с интервалом ровно 30 секунд
    Given пользователь имеет подписку "Pro"
    When пользователь создаёт монитор с параметрами:
      | name     | Critical Service        |
      | url      | https://api.example.com |
      | interval | 30 seconds              |
    Then монитор создан успешно
    And интервал проверки равен "30 seconds"

  @use_case=uc_01_01_14
  @critical
  @boundary
  Scenario: Создание 25-го монитора на Free тарифе (граница лимита)
    Given пользователь имеет подписку "Free"
    And лимит мониторов для Free = "25"
    And пользователь имеет "24" активных мониторов
    When пользователь создаёт 25-й монитор
    Then монитор создан успешно
    And пользователь имеет "25" мониторов

  # === Реализованный срез (legacy e2e) ===
  # Следующие сценарии перенесены из backend/monitor-service/test/e2e/monitors.feature
  # и покрыты step definitions в features/suite/steps_01_monitoring.go.

  @use_case=uc_01_01_01
  @implemented
  Scenario: Создание монитора через gRPC (legacy)
    Given пользователь аутентифицирован с тиром "free"
    When пользователь создаёт монитор с параметрами:
      | name       | Test Monitor        |
      | url        | https://example.com |
      | check_type | http                |
      | interval   | 60                  |
      | timeout    | 10                  |
    Then монитор создан успешно
    And монитор имеет статус "PENDING"
    And монитор имеет имя "Test Monitor"
    And монитор имеет URL "https://example.com"

  @use_case=uc_01_01_02
  @implemented
  Scenario: Получение информации о мониторе через gRPC (legacy)
    Given пользователь аутентифицирован с тиром "free"
    And монитор существует с параметрами:
      | name       | Existing Monitor |
      | url        | https://test.com |
      | check_type | http             |
      | interval   | 60               |
      | timeout    | 10               |
    When пользователь запрашивает информацию о мониторе
    Then информация о мониторе получена успешно
    And монитор имеет имя "Existing Monitor"
    And монитор имеет URL "https://test.com"
    And монитор имеет статус "PENDING"

  @use_case=uc_01_01_05
  @implemented
  Scenario: Удаление монитора через gRPC (legacy)
    Given пользователь аутентифицирован с тиром "free"
    And монитор существует с параметрами:
      | name       | To Delete Monitor  |
      | url        | https://delete.com |
      | check_type | http               |
      | interval   | 60                 |
      | timeout    | 10                 |
    When пользователь удаляет монитор
    Then монитор удалён успешно
    When пользователь запрашивает информацию о мониторе
    Then получена ошибка "not found"

  @use_case=uc_01_01_06
  @implemented
  Scenario: Полный CRUD цикл монитора через gRPC (legacy)
    Given пользователь аутентифицирован с тиром "free"
    And монитор существует с параметрами:
      | name       | Integration Test Monitor |
      | url        | https://example.com      |
      | check_type | http                     |
      | interval   | 60                       |
      | timeout    | 30                       |
    When пользователь запрашивает информацию о мониторе
    Then информация о мониторе получена успешно
    And монитор имеет имя "Integration Test Monitor"
    When пользователь обновляет монитор с параметрами:
      | name     | Updated Monitor Name        |
      | url      | https://example.com/updated |
      | interval | 120                         |
      | timeout  | 30                          |
    Then монитор обновлён успешно
    And монитор имеет имя "Updated Monitor Name"
    When пользователь запрашивает информацию о мониторе
    Then монитор имеет имя "Updated Monitor Name"
    When пользователь удаляет монитор
    Then монитор удалён успешно
    When пользователь запрашивает информацию о мониторе
    Then получена ошибка "not found"

  @use_case=uc_01_01_07
  @implemented
  Scenario: Запрос несуществующего монитора возвращает NotFound (legacy)
    Given пользователь аутентифицирован с тиром "free"
    When пользователь запрашивает монитор с ID "00000000-0000-0000-0000-000000000000"
    Then получена ошибка "not found"

  @use_case=uc_01_01_08
  @implemented
  Scenario: Список мониторов на чистой БД пуст (legacy)
    Given пользователь аутентифицирован с тиром "free"
    When пользователь запрашивает список мониторов
    Then список мониторов пуст

  @use_case=uc_01_01_09
  @implemented
  Scenario: Приостановка и возобновление монитора через gRPC (legacy)
    Given пользователь аутентифицирован с тиром "free"
    And монитор существует с параметрами:
      | name       | Pause Test Monitor        |
      | url        | https://pause.example.com |
      | check_type | http                      |
      | interval   | 60                        |
      | timeout    | 30                        |
    When пользователь приостанавливает монитор
    Then монитор приостановлен успешно
    When пользователь запрашивает информацию о мониторе
    Then монитор имеет статус "PAUSED"
    When пользователь возобновляет монитор
    Then монитор возобновлён успешно
    When пользователь запрашивает информацию о мониторе
    Then статус монитора отличается от "PAUSED"

  @use_case=uc_01_01_10
  @implemented
  Scenario: Создание нескольких мониторов и их отображение в списке (legacy)
    Given пользователь аутентифицирован с тиром "free"
    When пользователь создаёт мониторы с именами:
      | Monitor Alpha |
      | Monitor Beta  |
      | Monitor Gamma |
    Then все мониторы созданы успешно
    When пользователь запрашивает список мониторов
    Then список мониторов содержит "3" монитора
    And список мониторов содержит все созданные мониторы
