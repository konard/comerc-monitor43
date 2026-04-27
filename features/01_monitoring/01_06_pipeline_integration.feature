@epic=01_monitoring
@user_story=01_06_pipeline_integration
# Description: Межсервисная интеграция monitor-service с dashboard и scheduler через события и gRPC

Feature: Межсервисная интеграция мониторов
  Как платформа
  Я хочу чтобы изменения мониторов распространялись между сервисами
  Чтобы dashboard и scheduler видели актуальное состояние

  @use_case=uc_01_06_01
  @implemented
  @integration
  Scenario: Создание монитора отражается в dashboard через событие monitor.created
    Given пользователь аутентифицирован с тиром "Pro"
    When пользователь создаёт монитор с параметрами:
      | name       | E2E Pipeline Monitor         |
      | url        | https://pipeline.example.com |
      | check_type | http                         |
      | interval   | 60                           |
      | timeout    | 30                           |
    Then монитор создан успешно
    When публикуется событие monitor.created для созданного монитора
    Then монитор появляется в dashboard с тем же URL

  @use_case=uc_01_06_02
  @implemented
  @integration
  Scenario: Удаление монитора через событие monitor.deleted убирает запись из dashboard
    Given в dashboard опубликовано событие monitor.created для нового монитора
    And запись о мониторе присутствует в dashboard
    When публикуется событие monitor.deleted для этого монитора
    Then запись о мониторе удалена из dashboard

  @use_case=uc_01_06_03
  @implemented
  @integration
  Scenario: Scheduler видит созданные мониторы и пропускает приостановленные
    Given пользователь аутентифицирован с тиром "Pro"
    When пользователь создаёт мониторы для scheduler:
      | name          | url                       |
      | Service Alpha | https://alpha.example.com |
      | Service Beta  | https://beta.example.com  |
      | Service Gamma | https://gamma.example.com |
    Then список мониторов в статусе "PENDING" содержит "3" записи
    When пользователь приостанавливает первый созданный монитор
    Then приостановленный монитор имеет статус "PAUSED"
    And список мониторов в статусе "PENDING" содержит "2" записи

  # === Сценарии scheduler ↔ monitor-service ===

  @use_case=uc_01_06_04
  @implemented
  @integration
  Scenario: Scheduler получает список активных мониторов через ListMonitors
    Given пользователь аутентифицирован с тиром "Pro"
    When пользователь создаёт мониторы для scheduler:
      | name          | url                       |
      | Service Alpha | https://alpha.example.com |
      | Service Beta  | https://beta.example.com  |
    When scheduler запрашивает список мониторов через ListMonitors
    Then получен список из "2" мониторов
    And каждый монитор в списке имеет непустой идентификатор и положительный интервал

  @use_case=uc_01_06_05
  @implemented
  @integration
  Scenario: Scheduler получает детали монитора через GetMonitor
    Given пользователь аутентифицирован с тиром "Pro"
    When пользователь создаёт монитор с параметрами:
      | name       | Scheduler Detail Test      |
      | url        | https://detail.example.com |
      | check_type | http                       |
      | interval   | 120                        |
      | timeout    | 15                         |
    Then монитор создан успешно
    When scheduler запрашивает детали созданного монитора через GetMonitor
    Then детали монитора совпадают с параметрами:
      | name     | Scheduler Detail Test |
      | interval | 120                   |
      | timeout  | 15                    |

  @use_case=uc_01_06_06
  @implemented
  @integration
  Scenario: Приостановленный монитор имеет статус PAUSED для scheduler
    Given пользователь аутентифицирован с тиром "Pro"
    When пользователь создаёт монитор с параметрами:
      | name       | Paused Monitor             |
      | url        | https://paused.example.com |
      | check_type | http                       |
      | interval   | 60                         |
      | timeout    | 30                         |
    Then монитор создан успешно
    When пользователь приостанавливает монитор
    And scheduler запрашивает детали созданного монитора через GetMonitor
    Then статус полученного монитора равен "PAUSED"

  @use_case=uc_01_06_07
  @implemented
  @integration
  Scenario: Удалённый монитор возвращает NotFound для scheduler
    Given пользователь аутентифицирован с тиром "Pro"
    When пользователь создаёт монитор с параметрами:
      | name       | Soon Deleted                |
      | url        | https://deleted.example.com |
      | check_type | http                        |
      | interval   | 60                          |
      | timeout    | 30                          |
    Then монитор создан успешно
    When пользователь удаляет монитор
    And scheduler запрашивает детали созданного монитора через GetMonitor
    Then получена ошибка NotFound
