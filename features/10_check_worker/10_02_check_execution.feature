@epic=10_check_worker
@user_story=10_02_check_execution
# Description: Выполнение HTTP проверок воркером: запрос, классификация, обработка ошибок

Feature: Выполнение проверок воркером
  Как check-worker
  Я хочу выполнять HTTP проверки полученных мониторов
  Чтобы собирать данные о доступности сервисов из своей зоны

  # Integration: Worker получает конфигурацию монитора через MonitorService.GetMonitor
  # See: Epic 01, uc_01_01_02
  # Business rules for UP/DOWN/DEGRADED: See Epic 01, 01_02_check_execution
  @use_case=uc_10_02_01
  @critical
  Scenario: Выполнение успешной HTTP проверки
    Given воркер получает задание на проверку монитора:
      | url     | https://api.example.com |
      | timeout | 10 seconds             |
      | method  | GET                    |
    And сервис отвечает со статусом 200 OK
    And время ответа "150 ms"
    When воркер выполняет проверку
    Then результат проверки:
      | status       | UP   |
      | status_code  | 200  |
      | response_time| 150ms|
    And результат подготовлен для отправки в monitor-service
    And действие в аудит лог записано как "check_executed"
      | field         | value                    |
      | monitor_url   | https://api.example.com  |
      | status        | UP                       |
      | status_code   | 200                      |
      | response_time | 150ms                    |
      | worker_id     | <worker_id>              |
      | zone          | moscow                   |
      | timestamp     | <iso8601>                |

  # Business rules: 5xx → DOWN
  # See: Epic 01, uc_01_02_02
  @use_case=uc_10_02_02
  @critical
  Scenario: Выполнение проверки с ошибкой 5xx
    Given воркер получает задание на проверку монитора:
      | url     | https://api.example.com |
      | timeout | 10 seconds             |
    And сервис отвечает со статусом 500 Internal Server Error
    When воркер выполняет проверку
    Then результат проверки:
      | status       | DOWN                    |
      | status_code  | 500                     |
      | error        | Internal Server Error   |

  # Business rules: timeout → DOWN
  # See: Epic 01, uc_01_02_03
  @use_case=uc_10_02_03
  @critical
  Scenario: Выполнение проверки с таймаутом
    Given воркер получает задание на проверку монитора:
      | url     | https://api.example.com |
      | timeout | 10 seconds             |
    And сервис не отвечает в течение таймаута
    When воркер выполняет проверку
    Then результат проверки:
      | status       | DOWN                 |
      | error        | CONNECTION_TIMEOUT   |
      | response_time| 10000ms              |

  # Business rules: DNS failure → DOWN
  # See: Epic 01, uc_01_02_03a
  @use_case=uc_10_02_04
  @critical
  Scenario: Выполнение проверки с ошибкой DNS
    Given воркер получает задание на проверку монитора:
      | url     | https://nonexistent-domain-12345.com |
      | timeout | 10 seconds                           |
    When воркер выполняет проверку
    Then результат проверки:
      | status       | DOWN                  |
      | error        | DNS_RESOLUTION_FAILED |

  # Business rules: SSL expired → DOWN
  # See: Epic 01, uc_01_02_03b
  @use_case=uc_10_02_05
  @critical
  Scenario: Выполнение проверки с истёкшим SSL сертификатом
    Given воркер получает задание на проверку монитора:
      | url     | https://expired-cert.example.com |
      | timeout | 10 seconds                       |
    And SSL сертификат истёк
    When воркер выполняет проверку
    Then результат проверки:
      | status       | DOWN                    |
      | error        | SSL_CERTIFICATE_EXPIRED |

  @use_case=uc_10_02_06
  @critical
  Scenario: Выполнение проверки с редиректом
    Given воркер получает задание на проверку монитора:
      | url             | https://example.com |
      | timeout         | 10 seconds          |
      | follow_redirects| true                 |
      | max_redirects   | 5                    |
    And сервис возвращает редирект 301 → https://www.example.com
    And https://www.example.com отвечает 200 OK
    When воркер выполняет проверку
    Then результат проверки:
      | status       | UP                   |
      | status_code  | 200                  |
    And финальный URL = "https://www.example.com"

  # Business rules: DEGRADED при медленном ответе
  # See: Epic 01, uc_01_02_05
  @use_case=uc_10_02_07
  @critical
  Scenario: Выполнение проверки с DEGRADED статусом
    Given воркер получает задание на проверку монитора:
      | url                     | https://api.example.com |
      | timeout                 | 30 seconds              |
      | degraded_response_time  | 5000ms                  |
    And сервис отвечает со статусом 200 OK
    And время ответа "6000 ms"
    When воркер выполняет проверку
    Then результат проверки:
      | status       | DEGRADED              |
      | status_code  | 200                   |
      | response_time| 6000ms                |
      | reason       | SLOW_RESPONSE         |

  # Business rules: maintenance window → skip
  # See: Epic 08, uc_08_01_26
  @use_case=uc_10_02_08
  @critical
  Scenario: Пропуск проверки во время maintenance window
    Given воркер получает задание на проверку монитора
    And монитор имеет активное окно обслуживания
    When воркер проверяет условия выполнения
    Then проверка пропущена
    And результат содержит статус "SKIPPED"
    And причина = "MAINTENANCE_WINDOW"

  @use_case=uc_10_02_02a
  @critical
  Scenario: Выполнение проверки с ошибкой 4xx
    Given воркер получает задание на проверку монитора:
      | url     | https://api.example.com |
      | timeout | 10 seconds             |
    And настройка 4xx = "400+ as DOWN"
    And сервис отвечает со статусом 404 Not Found
    When воркер выполняет проверку
    Then результат проверки:
      | status       | DOWN             |
      | status_code  | 404              |
      | error        | Not Found        |

  @use_case=uc_10_02_09
  @critical
  Scenario: Выполнение проверки с пустым ответом
    Given воркер получает задание на проверку монитора:
      | url     | https://api.example.com |
      | timeout | 10 seconds             |
    And сервис отвечает со статусом 200
    And тело ответа пустое
    When воркер выполняет проверку
    Then результат проверки:
      | status       | UP               |
      | status_code  | 200              |
      | warning      | EMPTY_RESPONSE   |

  # Business rules: response too large → DOWN
  # See: Epic 01, uc_01_02_03c
  @use_case=uc_10_02_03a
  @critical
  Scenario: Ошибка: Превышение максимального размера ответа a
    Given воркер получает задание на проверку монитора:
      | url             | https://api.example.com |
      | timeout         | 10 seconds              |
      | max_response_size| 10485760               |
    And сервис возвращает "100 MB" данных
    When воркер выполняет проверку
    Then результат проверки:
      | status       | DOWN               |
      | error        | RESPONSE_TOO_LARGE |

  # Business rules: too many redirects → DOWN
  # See: Epic 01, uc_01_02_03d
  @use_case=uc_10_02_06a
  @critical
  Scenario: Выполнение проверки с циклическим редиректом
    Given воркер получает задание на проверку монитора:
      | url             | https://api.example.com |
      | max_redirects   | 5                       |
    And сервис возвращает бесконечный редирект
    When воркер выполняет проверку
    Then результат проверки:
      | status       | DOWN                  |
      | error        | TOO_MANY_REDIRECTS    |

  @use_case=uc_10_02_04a
  @critical
  Scenario: Выполнение проверки с connection refused
    Given воркер получает задание на проверку монитора:
      | url     | https://api.example.com |
      | timeout | 10 seconds             |
    And соединение отклонено сервером
    When воркер выполняет проверку
    Then результат проверки:
      | status       | DOWN                |
      | error        | CONNECTION_REFUSED  |

  @use_case=uc_10_02_01a
  @critical
  @validation
  Scenario: Ошибка выполнения проверки с невалидной конфигурацией монитора a
    Given воркер получает задание на проверку монитора
    And конфигурация монитора не содержит URL
    When воркер пытается выполнить проверку
    Then проверка завершена с ошибкой "INVALID_MONITOR_CONFIG"
    And результат отправлен как FAILED

  @use_case=uc_10_02_10
  @critical
  Scenario: Выполнение HEAD запроса
    Given воркер получает задание на проверку монитора:
      | url    | https://api.example.com |
      | method | HEAD                    |
    And сервис отвечает со статусом 200 OK
    When воркер выполняет проверку
    Then результат проверки:
      | status       | UP  |
      | status_code  | 200 |
    And тело ответа не загружено

  @use_case=uc_10_02_11
  @critical
  Scenario: Выполнение проверки с basic auth
    Given воркер получает задание на проверку монитора:
      | url      | https://api.example.com |
      | method   | GET                     |
      | username | admin                   |
      | password | secret                  |
    And сервис требует аутентификацию
    And аутентификация успешна
    And сервис отвечает со статусом 200 OK
    When воркер выполняет проверку
    Then результат проверки:
      | status       | UP  |
      | status_code  | 200 |

  @use_case=uc_10_02_12
  @critical
  Scenario: Выполнение проверки с кастомными заголовками
    Given воркер получает задание на проверку монитора:
      | url     | https://api.example.com |
      | method  | GET                     |
    And заголовки:
      | key           | value        |
      | Authorization | Bearer token |
      | X-Custom      | value123     |
    And сервис отвечает со статусом 200 OK
    When воркер выполняет проверку
    Then результат проверки:
      | status       | UP  |
      | status_code  | 200 |

  @use_case=uc_10_02_13
  @critical
  @performance
  Scenario: Выполнение 50 параллельных проверок
    Given воркер получает "50" заданий на проверку
    And все целевые сервисы отвечают за "1 second"
    When воркер выполняет все проверки параллельно
    Then все "50" проверок выполнены
    And общее время выполнения не превышает "10 seconds"
    And ни одна проверка не потеряна
