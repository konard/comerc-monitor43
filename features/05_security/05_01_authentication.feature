@epic=05_security
@user_story=05_01_authentication
# Description: Аутентификация и управление пользователями

Feature: Аутентификация
  Как пользователь
  Я хочу безопасно авторизовываться в системе
  Чтобы получить доступ к своим мониторам

  # Authentication Methods:
  # OAuth (Google, GitHub, Yandex, VK, mail.ru, Telegram) - For external users
  # Password-based - For internal/service accounts only
  #
  @critical
  @integration
  @use_case=uc_05_01_01
  Scenario Outline: Вход через OAuth провайдер
    Given пользователь переходит на страницу входа
    When пользователь выбирает провайдер "<provider>"
    And пользователь авторизуется в "<provider>"
    Then пользователь перенаправлен в приложение
    And создана сессия действительная "7 days"
    And получен JWT токен
    And пользователь авторизован
    And действие в аудит лог записано как "oauth_login_success"
    And audit log содержит:
      | action          | oauth_login_success |
      | user_id         | <user_id> |
      | provider        | <provider> |
      | session_id      | <session_id> |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

    Examples:
      | provider |
      | Google   |
      | GitHub   |
      | Yandex   |
      | VK       |
      | mail.ru  |
      | Telegram |

  @critical
  @integration
  @state_transition
  @use_case=uc_05_01_07
  Scenario: Создание аккаунта при первом входе через OAuth
    Given новый пользователь выбирает провайдер "Google"
    And пользователь авторизуется в Google
    Then автоматически создан новый аккаунт
    And пользователь имеет роль "USER"
    And создана сессия действительная "7 days"
    And создана запись audit log об auto account creation
    And audit log содержит:
      | action          | account_auto_created |
      | user_id         | <user_id> |
      | oauth_provider  | Google |
      | role            | USER |
      | timestamp       | <iso8601> |
      | ip_address      | <ip> |

  @critical
  @state_transition
  @boundary
  @use_case=uc_05_01_08
  Scenario: Продление сессии при активности
    Given пользователь авторизован
    And сессия создана "6 days" назад
    When пользователь выполняет действие в системе
    Then сессия продлена на "7 days"
    And пользователь остается авторизованным
    And создана запись audit log о session renewal
    And audit log содержит:
      | action          | session_renewed |
      | user_id         | <user_id> |
      | session_age_days | 6 |
      | new_expiry      | <iso8601> |
      | timestamp       | <iso8601> |

  @critical
  @state_transition
  @boundary
  @use_case=uc_05_01_09
  Scenario: Завершение сессии по неактивности
    Given пользователь авторизован
    And сессия неактивна "7 days"
    When пользователь выполняет действие в системе
    Then возвращается ошибка "SESSION_EXPIRED"
    And пользователь перенаправлен на страницу входа
    And действие в аудит лог записано как "session_expired"

  @critical
  @state_transition
  @use_case=uc_05_01_10
  Scenario: Выход из системы
    Given пользователь авторизован
    When пользователь нажимает кнопку выхода
    Then токен отозван
    And сессия завершена
    And пользователь перенаправлен на страницу входа
    And действие в аудит лог записано как "logout"
    And audit log содержит:
      | action          | logout |
      | user_id         | <user_id> |
      | session_duration | <duration> |
      | timestamp       | <iso8601> |
      | ip_address      | <ip> |

  @critical
  @security
  @boundary
  @use_case=uc_05_01_11
  Scenario: Блокировка после 10 неудачных попыток OAuth a
    Given пользователь существует
    And было "9" неудачных попыток входа
    When пользователь делает "10" попытку входа с неверными данными
    Then аккаунт заблокирован на "1 hour"
    And возвращается ошибка с кодом "ACCOUNT_LOCKED"
    And ответ содержит структуру:
      | error.code              | ACCOUNT_LOCKED |
      | error.details.retry_after | 1 hour        |
      | error.request_id        | <uuid>         |
    And действие в аудит лог записано как "account_locked"
    And audit log содержит:
      | action          | account_locked |
      | user_id         | <user_id> |
      | failed_attempts | 10 |
      | locked_until    | <iso8601> |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

  @critical
  @security
  @validation
  @use_case=uc_05_01_12
  Scenario: Попытка входа во время блокировки a
    Given аккаунт заблокирован до "2026-03-07T12:00:00Z"
    When пользователь пытается войти
    Then возвращается ошибка с кодом "ACCOUNT_LOCKED"
    And ответ содержит структуру:
      | error.code              | ACCOUNT_LOCKED |
      | error.details.locked_until | 2026-03-07T12:00:00Z |
      | error.request_id        | <uuid>         |
    And вход не выполнен
    And действие в аудит лог записано как "login_attempt_during_lockout"
    And audit log содержит:
      | action          | login_attempt_during_lockout |
      | user_id         | <user_id> |
      | locked_until    | 2026-03-07T12:00:00Z |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

  @critical
  @state_transition
  @use_case=uc_05_01_13
  Scenario: Разблокировка после истечения времени
    Given аккаунт был заблокирован
    And время блокировки истекло
    When пользователь пытается войти с валидными OAuth данными
    Then вход выполнен успешно
    And счетчик неудачных попыток сброшен
    And действие в аудит лог записано как "account_auto_unlocked"
    And audit log содержит:
      | action          | account_auto_unlocked |
      | user_id         | <user_id> |
      | previous_locked_until | <iso8601> |
      | failed_attempts_reset | 0 |
      | timestamp       | <iso8601> |

  @critical
  @validation
  @use_case=uc_05_01_14
  Scenario: Запрос информации о пользователе
    Given пользователь авторизован
    When пользователь запрашивает свою информацию
    Then получены email, full_name и created_at
    And OAuth провайдер не включен в ответ
    And действие в аудит лог записано как "user_info_accessed"
    And audit log содержит:
      | action          | user_info_accessed |
      | user_id         | <user_id> |
      | accessed_fields | email,full_name,created_at |
      | timestamp       | <iso8601> |

  @critical
  @validation
  @use_case=uc_05_01_15
  Scenario: Обновление пароля (для внутренних сервисов)
    Given пользователь авторизован
    When пользователь обновляет пароль на новый:
      | password        | NewSecure123 |
    Then пароль обновлен
    And новый пароль содержит минимум "8" символов
    And новый пароль содержит заглавные буквы
    And новый пароль содержит цифры
    And сессии сохранены

  @critical
  @validation
  @use_case=uc_05_01_16
  Scenario: Валидация требований к паролю (для внутренних сервисов)
    Given пользователь авторизован
    When пользователь пытается установить пароль "weak"
    Then возвращается ошибка с кодом "WEAK_PASSWORD"
    And ответ содержит структуру:
      | error.code              | WEAK_PASSWORD |
      | error.details.min_length | 8             |
      | error.details.requires_uppercase | true |
      | error.details.requires_digit | true |
      | error.request_id        | <uuid>         |
    And пароль не обновлен
    And действие в аудит лог записано как "weak_password_attempt"
    And audit log содержит:
      | action          | weak_password_attempt |
      | user_id         | <user_id> |
      | reason          | password_too_weak |
      | timestamp       | <iso8601> |

  @critical
  @integration
  @use_case=uc_05_01_17
  Scenario: Обработка ошибки OAuth провайдера
    Given пользователь выбирает провайдер "Google"
    And провайдер Google возвращает ошибку "provider_unavailable"
    When система обрабатывает ответ
    Then возвращается ошибка с кодом "OAUTH_PROVIDER_ERROR"
    And ответ содержит структуру:
      | error.code              | OAUTH_PROVIDER_ERROR |
      | error.details.provider  | Google               |
      | error.details.error_type | provider_unavailable |
      | error.request_id        | <uuid>               |
    And действие в аудит лог записано как "oauth_provider_failure"
    And audit log содержит:
      | action          | oauth_provider_failure |
      | provider        | Google |
      | error_type      | provider_unavailable |
      | timestamp       | <iso8601> |

  @critical
  @validation
  @use_case=uc_05_01_18
  Scenario: Пользователь отклоняет OAuth авторизацию
    Given пользователь выбирает провайдер "Google"
    And пользователь отклоняет доступ в Google
    When пользователь перенаправлен обратно
    Then возвращается ошибка "OAUTH_DENIED"
    And пользователь не авторизован
    And действие в аудит лог записано как "oauth_denied"
    And audit log содержит:
      | action          | oauth_denied |
      | provider        | Google |
      | user_id         | <user_id> |
      | timestamp       | <iso8601> |

  @critical
  @validation
  @integration
  @use_case=uc_05_01_19
  Scenario: Один email через разные OAuth провайдеры
    Given пользователь существует с email "user@example.com" через провайдер "Google"
    When пользователь пытается войти с тем же email через провайдер "GitHub"
    Then система предлагает выбрать:
      | option                     | description           |
      | Войти в существующий аккаунт | Объединить с текущим  |
      | Создать новый аккаунт        | Отдельная регистрация |
    And действие в аудит лог записано как "multiple_oauth_providers_detected"
    And audit log содержит:
      | action          | multiple_oauth_providers_detected |
      | email           | user@example.com |
      | existing_provider | Google |
      | new_provider    | GitHub |
      | timestamp       | <iso8601> |

  @critical
  @state_transition
  @boundary
  @use_case=uc_05_01_20
  Scenario: Истечение сессии во время операции
    Given пользователь авторизован
    And сессия истекает через "1 minute"
    When пользователь начинает создание монитора
    And сессия истекает во время операции
    Then возвращается ошибка "SESSION_EXPIRED"
    And незавершенная операция сохранена как черновик
    And пользователю предложено войти повторно
    And действие в аудит лог записано как "session_expired_during_operation"
    And audit log содержит:
      | action          | session_expired_during_operation |
      | user_id         | <user_id> |
      | operation       | monitor_creation |
      | draft_saved     | true |
      | timestamp       | <iso8601> |

  @critical
  @security
  @use_case=uc_05_01_21
  Scenario: Одновременные сессии из разных локаций
    Given пользователь авторизован с IP "192.168.1.100" в "Москва"
    When пользователь создает новую сессию с IP "203.0.113.50" в "Сан-Паулу"
    Then предыдущая сессия завершена
    And отправлено уведомление о новой сессии на email
    And действие в аудит лог записано как "concurrent_session_detected"
    And audit log содержит:
      | action          | concurrent_session_detected |
      | user_id         | <user_id> |
      | old_ip          | 192.168.1.100 |
      | old_location    | Москва |
      | new_ip          | 203.0.113.50 |
      | new_location    | Сан-Паулу |
      | user_agent      | <user_agent> |
      | timestamp       | <iso8601> |

  @critical
  @security
  @use_case=uc_05_01_22
  Scenario: Предотвращение фиксации сессии
    Given злоумышленник создает сессию с идентификатором "compromised-session-id"
    When пользователь авторизуется с существующей сессией
    Then идентификатор сессии изменен на новый
    And старый идентификатор недействителен
    And действие в аудит лог записано как "session_fixation_prevented"
    And audit log содержит:
      | action          | session_fixation_prevented |
      | user_id         | <user_id> |
      | old_session_id  | compromised-session-id |
      | new_session_id  | <new_id> |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

  @critical
  @integration
  @use_case=uc_05_01_23
  Scenario: Ошибка обновления токена a
    Given пользователь авторизован
    And токен истекает через "1 minute"
    When система пытается обновить токен
    And сервис обновления токенов недоступен
    Then возвращается ошибка "TOKEN_REFRESH_FAILED"
    And пользователю предложено войти повторно
    And действие в аудит лог записано как "token_refresh_failed"
    And audit log содержит:
      | action          | token_refresh_failed |
      | user_id         | <user_id> |
      | error_reason    | refresh_service_unavailable |
      | timestamp       | <iso8601> |

  @critical
  @state_transition
  @use_case=uc_05_01_24
  Scenario: Разблокировка администратором
    Given аккаунт заблокирован из-за неудачных попыток
    And пользователь подтверждает свою личность через поддержку
    When администратор разблокирует аккаунт
    Then счетчик неудачных попыток сброшен
    And пользователю отправлено уведомление о разблокировке
    And действие в аудит лог записано как "account_unlocked_by_admin"
    And audit log содержит:
      | action          | account_unlocked_by_admin |
      | user_id         | <user_id> |
      | admin_id        | <admin_id> |
      | reason          | <reason> |
      | timestamp       | <iso8601> |

  @critical
  @state_transition
  @validation
  @use_case=uc_05_01_25
  Scenario: Сброс пароля для внутренних сервисов
    Given пользователь авторизован
    And пользователь имеет роль "SERVICE_ACCOUNT"
    When пользователь запрашивает сброс пароля
    Then отправлен токен сброса на email
    And токен действителен "1 hour"
    And действие в аудит лог записано как "password_reset_requested"
    And audit log содержит:
      | action          | password_reset_requested |
      | user_id         | <user_id> |
      | token_expiry    | 1 hour |
      | timestamp       | <iso8601> |
    When пользователь устанавливает новый пароль по токену
    Then пароль изменен
    And все сессии кроме текущей отозваны
    And действие в аудит лог записано как "password_reset_completed"
    And audit log содержит:
      | action          | password_reset_completed |
      | user_id         | <user_id> |
      | sessions_revoked | <count> |
      | timestamp       | <iso8601> |

  @critical
  @security
  @use_case=uc_05_01_26
  Scenario: Обнаружение подозрительной активности
    Given пользователь обычно авторизуется из "Москва"
    And пользователь имеет "5" успешных входов из "Москва"
    When пользователь авторизуется из "Нигерия" в течение "1 hour"
    Then требуется дополнительная верификация
    And отправлено уведомление о подозрительном входе
    And сессия ограничена "read-only" до верификации
    And действие в аудит лог записано как "suspicious_login_detected"
    And audit log содержит:
      | action          | suspicious_login_detected |
      | user_id         | <user_id> |
      | login_location  | Нигерия |
      | typical_location | Москва |
      | ip_address      | <ip> |
      | anomaly_score   | <score> |
      | session_restricted_to | read-only |
      | timestamp       | <iso8601> |

  @critical
  @security
  @boundary
  @use_case=uc_05_01_27
  Scenario: Ограничение частоты запросов к OAuth
    Given IP адрес "192.168.1.100" сделал "100" запросов к OAuth за "1 minute"
    When делается "101" запрос с того же IP
    Then возвращается ошибка с кодом "RATE_LIMIT_EXCEEDED"
    And ответ содержит структуру:
      | error.code              | RATE_LIMIT_EXCEEDED |
      | error.details.retry_after | 5 minutes         |
      | error.details.limit     | 100                 |
      | error.details.window    | 1 minute            |
      | error.request_id        | <uuid>              |
    And действие в аудит лог записано как "oauth_rate_limit_exceeded"
    And audit log содержит:
      | action          | oauth_rate_limit_exceeded |
      | ip_address      | 192.168.1.100 |
      | request_count   | 101 |
      | limit           | 100 |
      | window          | 1 minute |
      | blocked_until   | <iso8601> |
      | timestamp       | <iso8601> |
    And IP временно заблокирован на "5 minutes"

  @critical
  @validation
  @integration
  @use_case=uc_05_01_28
  Scenario: Ошибка обработки невалидного формата OAuth токена a
    Given пользователь авторизуется через провайдер "Google"
    And провайдер возвращает токен в невалидном формате
    When система обрабатывает OAuth callback
    Then возвращается ошибка с кодом "INVALID_OAUTH_TOKEN"
    And ответ содержит структуру:
      | error.code              | INVALID_OAUTH_TOKEN |
      | error.details.provider  | Google              |
      | error.details.reason    | invalid_token_format |
      | error.request_id        | <uuid>              |
    And пользователь не авторизован
    And действие в аудит лог записано как "invalid_oauth_token"
    And audit log содержит:
      | action          | invalid_oauth_token |
      | provider        | Google |
      | token_format    | invalid |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

  @critical
  @validation
  @integration
  @use_case=uc_05_01_29
  Scenario: Обработка просроченного OAuth токена
    Given пользователь авторизуется через провайдер "GitHub"
    And провайдер возвращает токен с истекшим сроком действия
    When система обрабатывает OAuth callback
    Then возвращается ошибка с кодом "EXPIRED_OAUTH_TOKEN"
    And ответ содержит структуру:
      | error.code              | EXPIRED_OAUTH_TOKEN |
      | error.details.provider  | GitHub              |
      | error.details.token_expiry | <iso8601>        |
      | error.request_id        | <uuid>              |
    And пользователь не авторизован
    And действие в аудит лог записано как "expired_oauth_token"
    And audit log содержит:
      | action          | expired_oauth_token |
      | provider        | GitHub |
      | token_expiry    | <iso8601> |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

  @critical
  @integration
  @use_case=uc_05_01_30
  Scenario: Ошибка подключения к базе данных при аутентификации a
    Given пользователь авторизуется через провайдер "Google"
    And база данных недоступна
    When система пытается создать или обновить пользователя
    Then возвращается ошибка с кодом "DATABASE_UNAVAILABLE"
    And ответ содержит структуру:
      | error.code              | DATABASE_UNAVAILABLE |
      | error.details.reason    | connection_failed     |
      | error.request_id        | <uuid>                |
    And пользователь не авторизован
    And действие в аудит лог записано как "authentication_database_error"
    And audit log содержит:
      | action          | authentication_database_error |
      | provider        | Google |
      | error_type      | connection_failed |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

  @critical
  @integration
  @use_case=uc_05_01_31
  Scenario: Недоступность Redis кеша при создании сессии
    Given пользователь успешно авторизуется через провайдер "Yandex"
    And Redis кеш недоступен
    When система пытается создать сессию в кеше
    Then сессия создана в базе данных как fallback
    And JWT токен выдан пользователю
    And пользователь авторизован
    And действие в аудит лог записано как "session_cache_fallback"
    And audit log содержит:
      | action          | session_cache_fallback |
      | provider        | Yandex |
      | fallback_storage | database |
      | user_id         | <user_id> |
      | session_id      | <session_id> |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

  @critical
  @boundary
  @state_transition
  @use_case=uc_05_01_32
  Scenario: Истечение сессии ровно через 7 дней
    Given пользователь авторизован
    And сессия создана "7 days" назад в "2026-03-02T10:00:00Z"
    And текущее время "2026-03-09T10:00:00Z"
    When пользователь выполняет действие в системе
    Then возвращается ошибка "SESSION_EXPIRED"
    And пользователь перенаправлен на страницу входа
    And действие в аудит лог записано как "session_expired_exact_boundary"
    And audit log содержит:
      | action          | session_expired_exact_boundary |
      | user_id         | <user_id> |
      | session_duration | 7 days |
      | expiry_timestamp | 2026-03-09T10:00:00Z |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

  @critical
  @boundary
  @use_case=uc_05_01_33
  Scenario: Rate limit достигает ровно 100 запросов
    Given IP адрес "192.168.1.100" сделал "99" запросов к OAuth за "1 minute"
    When делается "100" запрос с того же IP
    Then запрос обработан успешно
    And действие в аудит лог записано как "oauth_rate_limit_boundary_reached"
    And audit log содержит:
      | action          | oauth_rate_limit_boundary_reached |
      | ip_address      | 192.168.1.100 |
      | request_count   | 100 |
      | limit           | 100 |
      | window          | 1 minute |
      | at_limit        | true |
      | timestamp       | <iso8601> |

  # Password-based flow (для внутренних/сервисных аккаунтов)

  @critical
  @use_case=uc_05_01_34
  Scenario: Регистрация, вход по паролю и валидация токена
    Given уникальный email подготовлен для сценария
    When пользователь регистрируется с паролем "SecurePassword123!"
    Then регистрация выполнена успешно
    And email пользователя совпадает с переданным
    When пользователь выполняет вход по паролю "SecurePassword123!"
    Then получены access token и refresh token
    When токен валидируется
    Then токен является действительным
    And email в ответе валидации совпадает с переданным

  @critical
  @use_case=uc_05_01_35
  Scenario: Невалидный токен отклоняется
    Given уникальный email подготовлен для сценария
    When невалидный токен "this.is.not.a.valid.jwt.token" валидируется
    Then токен является недействительным

  @critical
  @use_case=uc_05_01_36
  Scenario: Повторная регистрация с тем же email возвращает ошибку
    Given уникальный email подготовлен для сценария
    When пользователь регистрируется с паролем "Password123!"
    Then регистрация выполнена успешно
    When пользователь регистрируется повторно с тем же email и паролем "Password123!"
    Then регистрация отклонена с кодом AlreadyExists

  @critical
  @use_case=uc_05_01_37
  Scenario: Вход с неверным паролем отклоняется
    Given уникальный email подготовлен для сценария
    When пользователь регистрируется с паролем "CorrectPassword123!"
    Then регистрация выполнена успешно
    When пользователь выполняет вход по паролю "WrongPassword456!"
    Then вход отклонён с кодом Unauthenticated

  @critical
  @use_case=uc_05_01_38
  Scenario: Обновление access token через refresh token
    Given уникальный email подготовлен для сценария
    When пользователь регистрируется с паролем "Password123!"
    Then регистрация выполнена успешно
    When пользователь выполняет вход по паролю "Password123!"
    Then получены access token и refresh token
    When выполняется обновление токена через refresh token
    Then получен новый access token
