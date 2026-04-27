@epic=08_maintenance
@user_story=08_01_maintenance_windows
# Description: Управление окнами технического обслуживания

Feature: Окна обслуживания
  Как пользователь
  Я хочу создавать окна технического обслуживания
  Чтобы предотвратить ложные алерты во время работ

  # Integration: Pauses monitoring in Epic 01 (Monitoring) during maintenance
  # Integration: Suppresses alerts in Epic 02 (Alerting) when suppress_alerts=true
  # Integration: Excludes maintenance time from Epic 06 (Reporting) SLA calculations
  # See: Epic 01, uc_01_02_09; Epic 02, uc_08_01_26; Epic 06, uc_08_01_06
  @use_case=uc_08_01_01
  @critical
  @implemented
  Scenario: Создание maintenance window
    Given пользователь авторизован
    And пользователь имеет монитор "API Service"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Weekly DB Maintenance |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-10T04:00:00Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
      | pause_monitoring | true                   |
    Then окно обслуживания создано успешно
    And окно имеет статус "SCHEDULED"
    And событие создания записано в audit log с полями:
      | action        | maintenance_window.created |
      | user_id       | <current_user> |
      | window_name   | Weekly DB Maintenance |
      | window_id     | <uuid> |
      | timestamp     | <iso_timestamp> |
      | details       | {"recurrence": "ONCE", "monitor_count": 1} |

  @use_case=uc_08_01_02
  @critical
  @implemented
  Scenario: Создание maintenance window - Создание окна с максимальной продолжительностью
    Given пользователь авторизован
    And максимальная продолжительность окна настроена как "24" часа
    And пользователь имеет монитор "API Service"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Long Maintenance Window |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-11T02:00:00Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then окно обслуживания создано успешно
    And продолжительность окна составляет "24" часа
    And в audit log записано событие "maintenance_window_created_max_duration" с ID окна и продолжительностью

  @use_case=uc_08_01_03a
  @critical
  @implemented
  Scenario: Создание maintenance window - Отказ в создании окна превышающего максимальную продолжительность a
    Given пользователь авторизован
    And максимальная продолжительность окна настроена как "24" часа
    And пользователь имеет монитор "API Service"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Too Long Maintenance |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-12T02:00:00Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then возвращается ошибка с кодом "DURATION_EXCEEDS_MAXIMUM"
    And ответ содержит структуру:
      | error.code       | DURATION_EXCEEDS_MAXIMUM |
      | error.details    | {"maximum": 24, "unit": "hours", "requested": 48} |
      | error.request_id | <uuid> |
    And окно обслуживания не создано
    And действие в аудит лог записано как "maintenance_window_duration_exceeded"
      | field       | value                     |
      | timestamp   | <iso8601>                 |
      | user_id     | <user_id>                 |
      | maximum     | 24                        |
      | requested   | 48                        |
      | unit        | hours                     |

  @use_case=uc_08_01_03
  @critical
  @implemented
  Scenario: Создание maintenance window - Создание еженедельного повторяющегося окна
    Given пользователь авторизован
    And пользователь имеет монитор "Database"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Weekly Backup |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-10T04:00:00Z |
      | recurrence      | WEEKLY                 |
      | monitor_ids     | Database              |
    Then окно обслуживания создано успешно
    And окно настроено как повторяющееся еженедельно
    And событие создания записано в audit log с полями:
      | action        | maintenance_window.created |
      | user_id       | <current_user> |
      | window_name   | Weekly Backup |
      | window_id     | <uuid> |
      | recurrence    | WEEKLY |
      | timestamp     | <iso_timestamp> |

  @use_case=uc_08_01_04
  @critical
  @implemented
  Scenario: Создание maintenance window - Создание ежемесячного повторяющегося окна
    Given пользователь авторизован
    And пользователь имеет монитор "Main Application"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Monthly Updates |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-10T06:00:00Z |
      | recurrence      | MONTHLY                |
      | monitor_ids     | Main Application      |
    Then окно обслуживания создано успешно
    And окно настроено как повторяющееся ежемесячно
    And событие создания записано в audit log с полями:
      | action        | maintenance_window.created |
      | user_id       | <current_user> |
      | window_name   | Monthly Updates |
      | window_id     | <uuid> |
      | recurrence    | MONTHLY |
      | timestamp     | <iso_timestamp> |

  @use_case=uc_08_01_05
  @critical
  @implemented
  Scenario: Создание maintenance window - Создание глобального окна обслуживания администратором
    Given пользователь авторизован как "ADMIN"
    And пользователь имеет "3" монитора
    When пользователь создаёт глобальное окно обслуживания с параметрами:
      | name            | Global Maintenance |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-10T04:00:00Z |
      | recurrence      | ONCE                   |
      | is_global       | true                   |
    Then окно обслуживания создано успешно
    And окно применяется ко всем мониторам аккаунта
    And в audit log записано событие "global_maintenance_window_created" с ID окна, ролью администратора и количеством затронутых мониторов

  @use_case=uc_08_01_06a
  @critical
  @implemented
  Scenario: Создание maintenance window - Отказ в создании глобального окна обычным пользователем a
    Given пользователь авторизован как "USER"
    And пользователь имеет "2" монитора
    When пользователь создаёт глобальное окно обслуживания с параметрами:
      | name            | Global Maintenance |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-10T04:00:00Z |
      | recurrence      | ONCE                   |
      | is_global       | true                   |
    Then возвращается ошибка с кодом "INSUFFICIENT_PERMISSIONS"
    And ответ содержит структуру:
      | error.code       | INSUFFICIENT_PERMISSIONS |
      | error.details    | {"required_role": "ADMIN", "current_role": "USER"} |
      | error.request_id | <uuid> |
    And окно обслуживания не создано
    And действие в аудит лог записано как "global_maintenance_window_creation_denied"
      | field       | value              |
      | timestamp   | <iso8601>          |
      | user_id     | <user_id>          |
      | required_role | ADMIN            |
      | current_role | USER              |

  @use_case=uc_08_01_06
  Scenario: Создание maintenance window - Исключение периода обслуживания из расчета SLA
    Given монитор имеет SLA "99.9%"
    And активно окно обслуживания для монитора
    And период обслуживания длится "2" часа
    When рассчитывается SLA за период включающий обслуживание
    Then время обслуживания исключается из расчета downtime
    And SLA рассчитывается без учёта периода обслуживания

  @use_case=uc_08_01_07a
  Scenario: Создание maintenance window - Пересечение окон для разных мониторов разрешено
    Given пользователь авторизован
    And пользователь имеет монитор "API Service"
    And пользователь имеет монитор "Database"
    And существует активное окно для "API Service"
    When пользователь создаёт окно обслуживания для "Database" с пересекающимся временем
    Then окно обслуживания создано успешно
    And оба окна активны одновременно

  @use_case=uc_08_01_07b
  @critical
  Scenario: Создание maintenance window - Отказ при пересечении окон для одного монитора b
    Given пользователь авторизован
    And пользователь имеет монитор "API Service"
    And существует активное окно для "API Service" с периодом "2026-03-10T02:00:00Z" to "2026-03-10T06:00:00Z"
    When пользователь создаёт окно обслуживания для "API Service" с периодом "2026-03-10T04:00:00Z" to "2026-03-10T08:00:00Z"
    Then возвращается ошибка с кодом "OVERLAPPING_MAINTENANCE_WINDOWS"
    And ответ содержит структуру:
      | error.code       | OVERLAPPING_MAINTENANCE_WINDOWS |
      | error.details    | {"existing_window_id": <uuid>, "conflicting_period": {"start": "2026-03-10T02:00:00Z", "end": "2026-03-10T06:00:00Z"}} |
      | error.request_id | <uuid> |
    And окно обслуживания не создано
    And действие в аудит лог записано как "overlapping_maintenance_windows_rejected"
      | field             | value                                |
      | timestamp         | <iso8601>                            |
      | user_id           | <user_id>                            |
      | existing_window_id| <uuid>                               |
      | conflicting_period| {"start": "2026-03-10T02:00:00Z", "end": "2026-03-10T06:00:00Z"} |

  @use_case=uc_08_01_07
  Scenario: Создание maintenance window - Уведомление о предстоящем обслуживании
    Given пользователь создаёт окно обслуживания
    And начало окна запланировано через "1" час
    And пользователь настроил уведомления за "1" час
    When текущее время достигает времени отправки уведомления
    Then пользователю отправлено уведомление о предстоящем обслуживании
    And уведомление содержит имя окна и время начала

  @use_case=uc_08_01_08a
  @critical
  @implemented
  Scenario: Создание maintenance window - Отказ при создании окна с датой окончания раньше начала a
    Given пользователь авторизован
    And пользователь имеет монитор "API Service"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Invalid Time Range |
      | start_time      | 2026-03-10T04:00:00Z |
      | end_time        | 2026-03-10T02:00:00Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then возвращается ошибка с кодом "END_TIME_BEFORE_START_TIME"
    And ответ содержит структуру:
      | error.code       | END_TIME_BEFORE_START_TIME |
      | error.details    | {"start_time": "2026-03-10T04:00:00Z", "end_time": "2026-03-10T02:00:00Z"} |
      | error.request_id | <uuid> |
    And окно обслуживания не создано
    And действие в аудит лог записано как "maintenance_window_invalid_time_range"
      | field       | value                       |
      | timestamp   | <iso8601>                   |
      | user_id     | <user_id>                   |
      | start_time  | 2026-03-10T04:00:00Z        |
      | end_time    | 2026-03-10T02:00:00Z        |

  @use_case=uc_08_01_08b
  @critical
  Scenario: Создание maintenance window - Отказ при создании окна с датой в прошлом b
    Given пользователь авторизован
    And пользователь имеет монитор "API Service"
    And текущее время "2026-03-10T10:00:00Z"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Past Maintenance |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-10T04:00:00Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then возвращается ошибка с кодом "CANNOT_CREATE_MAINTENANCE_WINDOW_IN_PAST"
    And ответ содержит структуру:
      | error.code       | CANNOT_CREATE_MAINTENANCE_WINDOW_IN_PAST |
      | error.details    | {"current_time": "2026-03-10T10:00:00Z", "requested_start": "2026-03-10T02:00:00Z"} |
      | error.request_id | <uuid> |
    And окно обслуживания не создано
    And действие в аудит лог записано как "maintenance_window_creation_in_past_denied"
      | field           | value                       |
      | timestamp       | <iso8601>                   |
      | user_id         | <user_id>                   |
      | current_time    | 2026-03-10T10:00:00Z        |
      | requested_start | 2026-03-10T02:00:00Z        |

  @use_case=uc_08_01_08c
  @critical
  @integration
  Scenario: Создание maintenance window - Сбой базы данных при создании окна обслуживания c
    Given пользователь авторизован
    And пользователь имеет монитор "API Service"
    And база данных недоступна для записи
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Weekly Maintenance |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-10T04:00:00Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then возвращается ошибка "DATABASE_WRITE_FAILED"
    And ответ содержит структуру:
      | error.code       | DATABASE_WRITE_FAILED |
      | error.message    | Failed to create maintenance window |
      | error.details    | {"operation": "create_window"} |
      | error.request_id | <uuid> |
    And окно обслуживания не создано
    And действие в аудит лог записано как "database_write_failed"
      | field       | value                   |
      | timestamp   | <iso8601>               |
      | user_id     | <user_id>               |
      | operation   | create_window           |

  @use_case=uc_08_01_08d
  @critical
  @integration
  Scenario: Создание maintenance window - Создание окна при недоступности audit log service
    Given пользователь авторизован
    And пользователь имеет монитор "API Service"
    And audit log service недоступен
    When пользователь создаёт окно обслуживания
    Then окно создано успешно
    And статус окна "SCHEDULED"
    And событие не записано в audit log
    And создана задача на асинхронную запись в audit log
    And в системных логах записано предупреждение "audit_log_unavailable"

  @use_case=uc_08_01_08
  @critical
  @boundary
  @implemented
  Scenario: Создание maintenance window - Создание окна с максимальной продолжительностью (24 часа)
    Given пользователь авторизован
    And максимальная продолжительность окна настроена как "24" часа
    And пользователь имеет монитор "API Service"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Maximum Duration Window |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-11T02:00:00Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then окно обслуживания создано успешно
    And продолжительность окна составляет ровно "24" часа
    And статус окна "SCHEDULED"

  @use_case=uc_08_01_09
  @critical
  @boundary
  @implemented
  Scenario: Создание maintenance window - Попытка создания окна превышающего максимальную продолжительность на 1 секунду a
    Given пользователь авторизован
    And максимальная продолжительность окна настроена как "24" часа
    And пользователь имеет монитор "API Service"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Too Long Window |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-11T02:00:01Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then возвращается ошибка "DURATION_EXCEEDS_MAXIMUM"
    And ответ содержит структуру:
      | error.code       | DURATION_EXCEEDS_MAXIMUM |
      | error.details    | {"maximum_hours": 24, "requested_hours": 24.0003} |
      | error.request_id | <uuid> |
    And окно обслуживания не создано
    And действие в аудит лог записано как "maintenance_window_duration_exceeded"
      | field       | value                   |
      | timestamp   | <iso8601>               |
      | user_id     | <user_id>               |
      | maximum     | 24                      |
      | requested   | 24.0003                 |

  @use_case=uc_08_01_10
  @critical
  @boundary
  Scenario: Создание maintenance window - Достижение максимума окон для аккаунта
    Given пользователь имеет тариф "Free"
    And тариф ограничивает "5" окон обслуживания
    And пользователь создал "5" окон
    When пользователь пытается создать шестое окно
    Then возвращается ошибка "MAINTENANCE_WINDOW_LIMIT_REACHED"
    And ответ содержит структуру:
      | error.code       | MAINTENANCE_WINDOW_LIMIT_REACHED |
      | error.details    | {"limit": 5, "current": 5} |
      | error.request_id | <uuid> |
    And окно не создано
    And действие в аудит лог записано как "maintenance_window_limit_reached"
      | field       | value        |
      | timestamp   | <iso8601>    |
      | user_id     | <user_id>    |
      | limit       | 5            |
      | current     | 5            |

  @use_case=uc_08_01_11
  @critical
  @boundary
  Scenario: Создание maintenance window - Достижение максимума мониторов в окне
    Given система ограничивает "50" мониторов в одном окне
    And пользователь имеет "50" мониторов
    When пользователь создаёт окно обслуживания для всех "50" мониторов
    Then окно создано успешно
    And все "50" мониторов добавлены в окно

  @use_case=uc_08_01_12
  @critical
  @boundary
  Scenario: Создание maintenance window - Попытка добавления 51-го монитора в окно a
    Given система ограничивает "50" мониторов в одном окне
    And пользователь имеет окно с "50" мониторами
    When пользователь пытается добавить 51-й монитор
    Then возвращается ошибка "MONITOR_LIMIT_PER_WINDOW_REACHED"
    And ответ содержит структуру:
      | error.code       | MONITOR_LIMIT_PER_WINDOW_REACHED |
      | error.details    | {"limit": 50, "current": 50} |
      | error.request_id | <uuid> |
    And монитор не добавлен
    And действие в аудит лог записано как "monitor_limit_per_window_reached"
      | field       | value        |
      | timestamp   | <iso8601>    |
      | user_id     | <user_id>    |
      | window_id   | <window_id>  |
      | limit       | 50           |

  @use_case=uc_08_01_13
  @critical
  @boundary
  @implemented
  Scenario: Создание maintenance window - Создание окна с минимальной продолжительностью (1 минута)
    Given пользователь авторизован
    And минимальная продолжительность окна настроена как "1" минута
    And пользователь имеет монитор "API Service"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Minimal Window |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-10T02:01:00Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then окно обслуживания создано успешно
    And продолжительность окна составляет ровно "1" минуту

  @use_case=uc_08_01_14
  @critical
  @boundary
  @implemented
  Scenario: Создание maintenance window - Попытка создания окна короче минимальной продолжительности a
    Given пользователь авторизован
    And минимальная продолжительность окна настроена как "1" минута
    And пользователь имеет монитор "API Service"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Too Short Window |
      | start_time      | 2026-03-10T02:00:00Z |
      | end_time        | 2026-03-10T02:00:59Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then возвращается ошибка "DURATION_BELOW_MINIMUM"
    And ответ содержит структуру:
      | error.code       | DURATION_BELOW_MINIMUM |
      | error.details    | {"minimum_minutes": 1, "requested_seconds": 59} |
      | error.request_id | <uuid> |
    And окно не создано
    And действие в аудит лог записано как "duration_below_minimum"
      | field             | value              |
      | timestamp         | <iso8601>          |
      | user_id           | <user_id>          |
      | minimum_minutes   | 1                  |
      | requested_seconds | 59                 |

  @use_case=uc_08_01_15
  @critical
  @boundary
  Scenario: Создание maintenance window - Создание окна с временем начала точно сейчас
    Given пользователь авторизован
    And текущее время "2026-03-10T10:00:00Z"
    And пользователь имеет монитор "API Service"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Immediate Window |
      | start_time      | 2026-03-10T10:00:00Z |
      | end_time        | 2026-03-10T11:00:00Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then окно обслуживания создано успешно
    And окно немедленно активируется
    And статус окна "ACTIVE"

  @use_case=uc_08_01_16
  @critical
  @boundary
  Scenario: Создание maintenance window - Попытка создания окна с временем начала на 1 секунду в прошлом a
    Given пользователь авторизован
    And текущее время "2026-03-10T10:00:00Z"
    And пользователь имеет монитор "API Service"
    When пользователь создаёт окно обслуживания с параметрами:
      | name            | Past Window |
      | start_time      | 2026-03-10T09:59:59Z |
      | end_time        | 2026-03-10T10:59:59Z |
      | recurrence      | ONCE                   |
      | monitor_ids     | API Service            |
    Then возвращается ошибка "CANNOT_CREATE_MAINTENANCE_WINDOW_IN_PAST"
    And ответ содержит структуру:
      | error.code       | CANNOT_CREATE_MAINTENANCE_WINDOW_IN_PAST |
      | error.details    | {"current_time": "2026-03-10T10:00:00Z", "requested_start": "2026-03-10T09:59:59Z"} |
      | error.request_id | <uuid> |
    And окно не создано
    And действие в аудит лог записано как "maintenance_window_creation_in_past_denied"
      | field           | value                       |
      | timestamp       | <iso8601>                   |
      | user_id         | <user_id>                   |
      | current_time    | 2026-03-10T10:00:00Z        |
      | requested_start | 2026-03-10T09:59:59Z        |

  @use_case=uc_08_01_17
  @critical
  @boundary
  @implemented
  Scenario: Создание maintenance window - Окно ровно с 0 мониторами
    Given пользователь авторизован как "ADMIN"
    And пользователь не имеет мониторов
    When пользователь создаёт глобальное окно обслуживания
    Then окно создано успешно
    And окно не содержит мониторов
    And применяется к будущим мониторам

  @use_case=uc_08_01_18
  @performance
  @implemented
  Scenario: Создание maintenance window - Массовое создание 10+ окон обслуживания
    Given пользователь авторизован
    And пользователь имеет "10" мониторов
    When пользователь создаёт "10" отдельных окон для каждого монитора одновременно
    Then все "10" окон созданы успешно
    And время создания не превышает "5" секунд
    And все окна имеют статус "SCHEDULED"
    And в audit log записано "10" событий создания

  @use_case=uc_08_01_19
  @critical
  Scenario: Обновление maintenance window
    Given окно обслуживания со статусом "SCHEDULED"
    And окно назначено на "2026-03-10T02:00:00Z"
    When пользователь изменяет параметры окна:
      | start_time | 2026-03-11T02:00:00Z |
      | end_time   | 2026-03-11T04:00:00Z |
    Then параметры окна обновлены успешно
    And статус окна остаётся "SCHEDULED"
    And уведомление об изменении отправляется пользователю
    And событие обновления записано в audit log с полями:
      | action          | maintenance_window.updated |
      | user_id         | <current_user> |
      | window_id       | <uuid> |
      | timestamp       | <iso_timestamp> |
      | changed_fields  | ["start_time", "end_time"] |
      | old_values      | {"start_time": "2026-03-10T02:00:00Z", "end_time": "2026-03-10T04:00:00Z"} |
      | new_values      | {"start_time": "2026-03-11T02:00:00Z", "end_time": "2026-03-11T04:00:00Z"} |

  @use_case=uc_08_01_20
  @critical
  @implemented
  Scenario: Обновление maintenance window - Список окон обслуживания
    Given пользователь имеет "3" окна обслуживания
    When пользователь запрашивает список окон
    Then возвращается "3" окна
    And каждое окно содержит имя, статус, время начала и окончания
    And в audit log записано событие "maintenance_windows_listed" с ID пользователя и количеством окон

  @use_case=uc_08_01_21a
  @critical
  @integration
  Scenario: Обновление maintenance window - Повторная активация окна при недоступности scheduler
    Given окно обслуживания со статусом "SCHEDULED"
    And текущее время достигло start_time
    And scheduler service недоступен
    When система пытается активировать окно
    Then окно остаётся в статусе "SCHEDULED"
    And создана задача на повторную активацию через "1" minute
    And действие в аудит лог записано как "scheduler_unavailable"
      | field       | value               |
      | timestamp   | <iso8601>           |
      | window_id   | <window_id>         |
      | error       | scheduler_unavailable |
    And администратору отправлено уведомление о проблеме

  @use_case=uc_08_01_21b
  Scenario: Обновление maintenance window - Поведение при изменении системного времени во время обслуживания
    Given активно окно обслуживания для монитора "API Service"
    And окно должно завершиться через "2" часа
    When системное время изменяется (DST или NTP коррекция)
    Then время окончания окна пересчитывается корректно
    And фактическая продолжительность окна остаётся "2" часа
    And статус окна остаётся "ACTIVE"

  @use_case=uc_08_01_21
  Scenario: Обновление maintenance window - Просмотр истории окон обслуживания
    Given пользователь имеет "5" завершённых окон обслуживания
    When пользователь запрашивает историю окон
    Then отображается список всех завершённых окон
    And каждое окно содержит: имя, период, статус, создателя
    And доступна фильтрация по датам и мониторам
    And можно экспортировать историю в CSV

  @use_case=uc_08_01_22
  @performance
  Scenario: Обновление maintenance window - Высокочастотные обновления окон обслуживания
    Given пользователь имеет активное окно обслуживания
    And пользователь обновляет параметры окна
    When пользователь выполняет "10" обновлений в течение "1" секунды
    Then все обновления обработаны
    And финальное состояние корректно
    And время ответа не превышает "100ms" на обновление
    And нет потери данных

  @use_case=uc_08_01_23
  @performance
  Scenario: Обновление maintenance window - Конкурентная модификация окна обслуживания
    Given окно обслуживания со статусом "SCHEDULED"
    And пользователь "user1@example.com" начинает редактирование
    And пользователь "user2@example.com" начинает редактирование одновременно
    When оба пользователя сохраняют изменения
    Then первое сохранение применяется успешно
    And второе сохранение возвращает ошибку "CONFLICT"
    And ответ содержит структуру:
      | error.code       | CONFLICT |
      | error.message    | Window was modified by another user |
      | error.details    | {"current_version": <version>} |
      | error.request_id | <uuid> |
    And второму пользователю предложено перезагрузить данные

  @use_case=uc_08_01_24
  @critical
  @implemented
  Scenario: Удаление maintenance window
    Given пользователь имеет окно со статусом "SCHEDULED"
    When пользователь удаляет окно
    Then окно удалено
    And окно не появляется в списке
    And событие удаления записано в audit log с полями:
      | action        | maintenance_window.deleted |
      | user_id       | <current_user> |
      | window_id     | <uuid> |
      | window_status | SCHEDULED |
      | timestamp     | <iso_timestamp> |

  @use_case=uc_08_01_25a
  @critical
  Scenario: Удаление maintenance window - Отмена активного окна обслуживания
    Given окно обслуживания со статусом "ACTIVE"
    And монитор "API Service" подавлен
    When пользователь отменяет окно обслуживания
    Then статус окна изменяется на "CANCELLED"
    And мониторинг монитора возобновляется немедленно
    And выполняется проверка монитора
    And сохраняется запись о досрочном завершении
    And действие в аудит лог записано как "maintenance_window.cancelled"
      | field              | value            |
      | timestamp          | <iso8601>        |
      | initiator          | USER             |
      | user_id            | <user_id>        |
      | window_id          | <window_id>      |
      | old_status         | ACTIVE           |
      | new_status         | CANCELLED        |
      | cancellation_reason| MANUAL           |

  @use_case=uc_08_01_25b
  @critical
  @implemented
  Scenario: Удаление maintenance window - Отмена запланированного окна обслуживания
    Given окно обслуживания со статусом "SCHEDULED"
    And до начала окна осталось "2" часа
    When пользователь отменяет окно обслуживания
    Then статус окна изменяется на "CANCELLED"
    And окно удаляется из расписания
    And уведомление об отмене отправляется пользователю
    And действие в аудит лог записано как "maintenance_window_cancelled_scheduled"
      | field       | value               |
      | timestamp   | <iso8601>           |
      | user_id     | <user_id>           |
      | window_id   | <window_id>         |
      | start_time  | <start_time>        |

  @use_case=uc_08_01_25
  Scenario: Удаление maintenance window - Удаление монитора во время активного обслуживания
    Given активно окно обслуживания для монитора "API Service"
    When пользователь удаляет монитор "API Service"
    Then окно обслуживания помечается как "ORPHANED"
    And статус окна изменяется на "CANCELLED"
    And сохраняется запись об удалении монитора

  @use_case=uc_08_01_26a
  @critical
  @integration
  Scenario: Удаление maintenance window - Частичный сбой системы при обновлении статуса окна
    Given окно обслуживания со статусом "ACTIVE"
    And база данных доступна только для чтения
    And текущее время достигло end_time
    When система пытается завершить окно
    Then статус окна изменяется на "COMPLETED" в памяти
    And обновление в базе данных отложено
    And мониторинг возобновляется
    And задача на обновление базы данных поставлена в очередь
    And действие в аудит лог записано как "database_update_deferred"
      | field       | value                   |
      | timestamp   | <iso8601>               |
      | window_id   | <window_id>             |
      | old_status  | ACTIVE                  |
      | new_status  | COMPLETED               |

  @use_case=uc_08_01_26
  @critical
  Scenario: Подавление алертов во время maintenance
    Given активно окно обслуживания
    And параметр suppress_alerts = true
    When проверка монитора завершается с ошибкой
    Then алерт не создаётся
    And статус монитора не изменяется
    And в audit log записано событие "alert_suppressed_by_maintenance" с ID окна, ID монитора и типом проверки

  @use_case=uc_08_01_27
  @critical
  Scenario: Подавление алертов во время maintenance - Автоматический статус окна при наступлении времени
    Given окно обслуживания со статусом "SCHEDULED"
    And текущее время достигло start_time
    When система обрабатывает начало окна
    Then статус окна изменяется на "ACTIVE"
    And алерты для указанных мониторов подавляются
    And событие изменения статуса записано в audit log с полями:
      | action          | maintenance_window.status_changed |
      | initiator       | SYSTEM |
      | window_id       | <uuid> |
      | old_status      | SCHEDULED |
      | new_status      | ACTIVE |
      | timestamp       | <iso_timestamp> |
      | trigger_type    | AUTOMATIC |

  @use_case=uc_08_01_28
  @critical
  Scenario: Подавление алертов во время maintenance - Автоматическое завершение окна по времени
    Given окно обслуживания со статусом "ACTIVE"
    And текущее время достигло end_time
    When система обрабатывает окончание окна
    Then статус окна изменяется на "COMPLETED"
    And мониторинг мониторов возобновляется
    And сохраняется запись о периоде обслуживания в истории
    And событие завершения записано в audit log с полями:
      | action          | maintenance_window.status_changed |
      | initiator       | SYSTEM |
      | window_id       | <uuid> |
      | old_status      | ACTIVE |
      | new_status      | COMPLETED |
      | timestamp       | <iso_timestamp> |
      | trigger_type    | AUTOMATIC |
      | duration_minutes| <duration> |

  @use_case=uc_08_01_29a
  Scenario: Подавление алертов во время maintenance - Поведение при активной проверке в момент начала обслуживания
    Given выполняется проверка монитора "API Service"
    And окно обслуживания для "API Service" начинается в "2026-03-10T02:00:00Z"
    When текущее время достигает "2026-03-10T02:00:00Z"
    Then статус окна изменяется на "ACTIVE"
    And текущая проверка завершается
    And результат проверки не создаёт алерт
    And статус монитора не изменяется

  @use_case=uc_08_01_29b
  Scenario: Подавление алертов во время maintenance - Поведение активного алерта при начале обслуживания
    Given монитор "API Service" имеет активный алерт "DOWN"
    And окно обслуживания для "API Service" начинается в "2026-03-10T02:00:00Z"
    When текущее время достигает "2026-03-10T02:00:00Z"
    Then статус окна изменяется на "ACTIVE"
    And активный алерт сохраняется в истории
    And статус алерта изменяется на "SUPPRESSED_BY_MAINTENANCE"
    And алерт продолжает отображаться в дашборде с пометкой "подавлен обслуживанием"

  @use_case=uc_08_01_29c
  Scenario: Подавление алертов во время maintenance - Автоматическое закрытие алерта после восстановления при окончании обслуживания
    Given монитор "API Service" был в состоянии "DOWN"
    And активно окно обслуживания для "API Service"
    When текущее время достигает end_time окна
    And следующая проверка монитора завершается успешно
    Then статус окна изменяется на "COMPLETED"
    And активный алерт автоматически закрывается
    And статус монитора изменяется на "UP"
    And создаётся запись о восстановлении в истории

  @use_case=uc_08_01_29d
  Scenario: Подавление алертов во время maintenance - Группировка алертов после окончания обслуживания
    Given активно окно обслуживания для монитора "API Service"
    And во время обслуживания произошло "5" неудачных проверок
    When текущее время достигает end_time окна
    Then статус окна изменяется на "COMPLETED"
    And создаётся один агрегированный алерт
    And алерт содержит информацию о количестве неудачных проверок
    And уведомление отправляется один раз

  @use_case=uc_08_01_29e
  Scenario: Подавление алертов во время maintenance - Поведение grace period при начале обслуживания
    Given монитор "API Service" имеет настройку grace_period "5" минут
    And происходит сбой проверки монитора
    And окно обслуживания начинается через "2" минуты
    When начинается окно обслуживания
    Then grace period прерывается
    And алерт не создаётся
    And счётчик неудачных проверок сбрасывается

  @use_case=uc_08_01_29f
  Scenario: Подавление алертов во время maintenance - Восстановление состояния окон после перезапуска системы
    Given активно окно обслуживания для монитора "API Service"
    And система перезапускается
    When система завершает перезапуск
    Then активное окно обслуживания восстанавливается
    And статус окна остаётся "ACTIVE"
    And подавление алертов продолжается
    And время окончания окна сохраняется корректно

  @use_case=uc_08_01_29g
  Scenario: Подавление алертов во время maintenance - Выявление инцидента во время обслуживания при активном safe_mode
    Given активно окно обслуживания для монитора "API Service"
    And параметр safe_mode = true
    And безопасные проверки разрешены
    When безопасная проверка обнаруживает критический сбой
    Then создаётся алерт с приоритетом "CRITICAL"
    And алерт помечается как "detected during maintenance"
    And уведомление отправляется немедленно
    And в дашборде отображается предупреждение о проблеме

  @use_case=uc_08_01_29h
  @critical
  @integration
  Scenario: Подавление алертов во время maintenance - Повторная отправка уведомления при недоступности notification service
    Given пользователь создаёт окно обслуживания
    And начало окна запланировано через "1" час
    And notification service недоступен
    When наступает время отправки уведомления
    Then уведомление не отправлено
    And создана задача на повторную отправку через "10" минут
    And в логах записана ошибка "notification_failed"
    And статус окна остаётся "SCHEDULED"

  @use_case=uc_08_01_29i
  @critical
  @integration
  Scenario: Подавление алертов во время maintenance - Постановка подавления алертов в очередь
    Given активно окно обслуживания для монитора "API Service"
    And monitor service недоступен
    When система пытается подавить алерты
    Then статус окна изменяется на "ACTIVE"
    And подавление алертов поставлено в очередь
    And мониторинг продолжается
    And в логах записано "monitor_service_unavailable"

  @use_case=uc_08_01_29
  @critical
  @state_transition
  Scenario: Подавление алертов во время maintenance - Сохранение состояния монитора при активации окна
    Given монитор "API Service" в статусе "DOWN"
    And активно окно обслуживания для "API Service"
    And окно переходит в статус "ACTIVE"
    When начинается окно обслуживания
    Then предыдущий статус монитора сохранён
    And текущий алерт помечен как "SUPPRESSED_BY_MAINTENANCE"
    And статус монитора остаётся "DOWN"
    And алерт не удаляется

  @use_case=uc_08_01_30a
  @critical
  @state_transition
  Scenario: Подавление алертов во время maintenance - Восстановление состояния монитора после завершения окна
    Given монитор "API Service" был в статусе "DOWN"
    And активно окно обслуживания для "API Service"
    And монитор находился в состоянии "DOWN" до начала окна
    When окно обслуживания завершается
    And следующая проверка завершается успешно
    Then статус монитора обновляется до "UP"
    And предыдущий статус "DOWN" сохранён в истории
    And создаётся запись о восстановлении

  @use_case=uc_08_01_30b
  @critical
  @state_transition
  Scenario: Подавление алертов во время maintenance - Переход алерта в suppressed при активации окна
    Given монитор "API Service" имеет активный алерт
    And статус алерта "TRIGGERED"
    When окно обслуживания для монитора становится активным
    Then статус алерта изменяется на "SUPPRESSED_BY_MAINTENANCE"
    And алерт остаётся в истории
    And уведомления об алерте не отправляются
    And действие в аудит лог записано как "alert_status_changed"
      | field       | value                        |
      | timestamp   | <iso8601>                    |
      | alert_id    | <alert_id>                   |
      | old_status  | TRIGGERED                    |
      | new_status  | SUPPRESSED_BY_MAINTENANCE    |

  @use_case=uc_08_01_30c
  @critical
  @state_transition
  Scenario: Подавление алертов во время maintenance - Переход окна из SCHEDULED в ACTIVE автоматически
    Given окно обслуживания со статусом "SCHEDULED"
    And текущее время "2026-03-10T01:59:59Z"
    And start_time окна "2026-03-10T02:00:00Z"
    When текущее время достигает "2026-03-10T02:00:00Z"
    Then статус окна изменяется на "ACTIVE"
    And алерты подавлены
    And действие в аудит лог записано как "maintenance_window.status_changed"
      | field       | value                        |
      | timestamp   | <iso8601>                    |
      | window_id   | <window_id>                  |
      | old_status  | SCHEDULED                    |
      | new_status  | ACTIVE                       |
      | trigger_type| AUTOMATIC                    |

  @use_case=uc_08_01_30d
  @critical
  @state_transition
  Scenario: Подавление алертов во время maintenance - Переход окна из ACTIVE в COMPLETED автоматически
    Given окно обслуживания со статусом "ACTIVE"
    And текущее время "2026-03-10T03:59:59Z"
    And end_time окна "2026-03-10T04:00:00Z"
    When текущее время достигает "2026-03-10T04:00:00Z"
    Then статус окна изменяется на "COMPLETED"
    And мониторинг возобновлён
    And действие в аудит лог записано как "maintenance_window.status_changed"
      | field       | value                        |
      | timestamp   | <iso8601>                    |
      | window_id   | <window_id>                  |
      | old_status  | ACTIVE                       |
      | new_status  | COMPLETED                    |
      | trigger_type| AUTOMATIC                    |
      | duration_minutes| <duration>               |

  @use_case=uc_08_01_30e
  @critical
  @state_transition
  Scenario: Подавление алертов во время maintenance - Переход окна из SCHEDULED в CANCELLED вручную
    Given окно обслуживания со статусом "SCHEDULED"
    And до начала осталось "2" часа
    When пользователь отменяет окно
    Then статус окна изменяется на "CANCELLED"
    And окно удалено из расписания
    And действие в аудит лог записано как "maintenance_window.cancelled"
      | field       | value                        |
      | timestamp   | <iso8601>                    |
      | user_id     | <user_id>                    |
      | window_id   | <window_id>                  |
      | old_status  | SCHEDULED                    |
      | new_status  | CANCELLED                    |
      | trigger_type| MANUAL                       |

  @use_case=uc_08_01_30f
  @performance
  Scenario: Подавление алертов во время maintenance - Масштабное подавление алертов для большого количества мониторов
    Given активно глобальное окно обслуживания
    And аккаунт имеет "100" мониторов
    And все мониторы имеют активные алерты
    When окно обслуживания активируется
    Then все "100" алертов помечены как "SUPPRESSED_BY_MAINTENANCE"
    And время подавления не превышает "5" секунд
    And все алерты корректно обновлены в базе данных
    And действие в аудит лог записано как "alerts_suppressed_bulk"
      | field       | value                        |
      | timestamp   | <iso8601>                    |
      | window_id   | <window_id>                  |
      | count       | 100                          |
