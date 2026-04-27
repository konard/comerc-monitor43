@epic=05_security
@user_story=05_02_team_management
# Description: Управление командой, ролями и приглашениями (RBAC + invite flow)

Feature: Управление членами команды и ролями
  Как владелец организации
  Я хочу приглашать участников и управлять их ролями
  Чтобы разграничить доступ к мониторам, алертам и настройкам

  # Роли:
  # OWNER  - полный доступ, единственный владелец организации, не может быть удалён
  # ADMIN  - управление членами, мониторами, алертами, интеграциями; не может удалить OWNER
  # MEMBER - создание/редактирование мониторов и алертов, без управления членами
  # VIEWER - read-only доступ ко всем ресурсам организации
  #
  # Invite flow:
  # invite_token - одноразовый UUID, TTL 7 days
  # Статусы invite: PENDING, ACCEPTED, DECLINED, EXPIRED, REVOKED
  #

  @critical
  @use_case=uc_05_02_01
  Scenario: Автоматическое создание организации при регистрации первого пользователя
    Given новый пользователь авторизуется через провайдер "Google"
    When аккаунт успешно создан
    Then автоматически создана организация с владельцем этого пользователя
    And пользователь имеет роль "OWNER" в организации
    And действие в аудит лог записано как "organization_created"
    And audit log содержит:
      | action          | organization_created |
      | user_id         | <user_id> |
      | organization_id | <org_id> |
      | role            | OWNER |
      | timestamp       | <iso8601> |

  @critical
  @use_case=uc_05_02_02
  Scenario: Приглашение нового участника с ролью MEMBER
    Given пользователь авторизован с ролью "OWNER"
    When пользователь отправляет приглашение:
      | email | new.member@example.com |
      | role  | MEMBER                  |
    Then создано приглашение со статусом "PENDING"
    And сгенерирован одноразовый invite_token
    And токен действителен "7 days"
    And на email "new.member@example.com" отправлено письмо с ссылкой приглашения
    And действие в аудит лог записано как "member_invited"
    And audit log содержит:
      | action          | member_invited |
      | inviter_id      | <user_id> |
      | invitee_email   | new.member@example.com |
      | role            | MEMBER |
      | organization_id | <org_id> |
      | invite_id       | <invite_id> |
      | expires_at      | <iso8601> |
      | timestamp       | <iso8601> |

  @critical
  @validation
  @use_case=uc_05_02_03
  Scenario Outline: Приглашение с разными ролями от OWNER
    Given пользователь авторизован с ролью "OWNER"
    When пользователь отправляет приглашение:
      | email | invitee@example.com |
      | role  | <role>              |
    Then создано приглашение со статусом "PENDING"
    And роль приглашения равна "<role>"

    Examples:
      | role   |
      | ADMIN  |
      | MEMBER |
      | VIEWER |

  @critical
  @validation
  @use_case=uc_05_02_04
  Scenario: Невалидный email при приглашении
    Given пользователь авторизован с ролью "OWNER"
    When пользователь отправляет приглашение:
      | email | not-an-email |
      | role  | MEMBER       |
    Then возвращается ошибка с кодом "INVALID_EMAIL"
    And ответ содержит структуру:
      | error.code              | INVALID_EMAIL |
      | error.details.field     | email         |
      | error.request_id        | <uuid>        |
    And приглашение не создано

  @critical
  @validation
  @use_case=uc_05_02_05
  Scenario: Попытка повторного приглашения уже активного участника
    Given пользователь авторизован с ролью "OWNER"
    And в организации уже есть активный участник "existing@example.com"
    When пользователь отправляет приглашение:
      | email | existing@example.com |
      | role  | MEMBER               |
    Then возвращается ошибка с кодом "USER_ALREADY_MEMBER"
    And приглашение не создано

  @critical
  @validation
  @use_case=uc_05_02_06
  Scenario: Повторное приглашение при наличии PENDING invite отзывает старое
    Given пользователь авторизован с ролью "OWNER"
    And существует приглашение "pending@example.com" со статусом "PENDING"
    When пользователь повторно отправляет приглашение:
      | email | pending@example.com |
      | role  | MEMBER              |
    Then предыдущее приглашение переведено в статус "REVOKED"
    And создано новое приглашение со статусом "PENDING"
    And действие в аудит лог записано как "invite_resent"
    And audit log содержит:
      | action           | invite_resent |
      | inviter_id       | <user_id> |
      | invitee_email    | pending@example.com |
      | old_invite_id    | <old_invite_id> |
      | new_invite_id    | <new_invite_id> |
      | timestamp        | <iso8601> |

  @critical
  @security
  @use_case=uc_05_02_07
  Scenario: MEMBER не имеет права приглашать участников
    Given пользователь авторизован с ролью "MEMBER"
    When пользователь отправляет приглашение:
      | email | new@example.com |
      | role  | MEMBER          |
    Then возвращается ошибка с кодом "INSUFFICIENT_PERMISSIONS"
    And ответ содержит структуру:
      | error.code              | INSUFFICIENT_PERMISSIONS |
      | error.details.required_role | ADMIN                |
      | error.request_id        | <uuid>                   |
    And приглашение не создано
    And действие в аудит лог записано как "permission_denied"
    And audit log содержит:
      | action          | permission_denied |
      | user_id         | <user_id> |
      | actual_role     | MEMBER |
      | required_role   | ADMIN |
      | resource        | invite |
      | timestamp       | <iso8601> |

  @critical
  @security
  @use_case=uc_05_02_08
  Scenario: VIEWER не имеет права приглашать участников
    Given пользователь авторизован с ролью "VIEWER"
    When пользователь отправляет приглашение:
      | email | new@example.com |
      | role  | MEMBER          |
    Then возвращается ошибка с кодом "INSUFFICIENT_PERMISSIONS"
    And приглашение не создано

  @critical
  @security
  @use_case=uc_05_02_09
  Scenario: ADMIN не может пригласить нового OWNER
    Given пользователь авторизован с ролью "ADMIN"
    When пользователь отправляет приглашение:
      | email | new@example.com |
      | role  | OWNER           |
    Then возвращается ошибка с кодом "CANNOT_ASSIGN_OWNER_ROLE"
    And приглашение не создано

  @critical
  @state_transition
  @use_case=uc_05_02_10
  Scenario: Принятие приглашения новым пользователем
    Given существует приглашение с токеном "<invite_token>" и ролью "MEMBER"
    And приглашение имеет статус "PENDING"
    When новый пользователь переходит по ссылке приглашения
    And пользователь авторизуется через провайдер "Google" с email приглашения
    Then приглашение переведено в статус "ACCEPTED"
    And пользователь добавлен в организацию с ролью "MEMBER"
    And invite_token погашен и больше не валиден
    And действие в аудит лог записано как "invite_accepted"
    And audit log содержит:
      | action          | invite_accepted |
      | invite_id       | <invite_id> |
      | user_id         | <user_id> |
      | organization_id | <org_id> |
      | role            | MEMBER |
      | timestamp       | <iso8601> |

  @critical
  @security
  @use_case=uc_05_02_11
  Scenario: Принятие приглашения под чужим email запрещено
    Given существует приглашение для "invited@example.com"
    When авторизованный пользователь с email "other@example.com" переходит по ссылке приглашения
    Then возвращается ошибка с кодом "INVITE_EMAIL_MISMATCH"
    And приглашение остаётся в статусе "PENDING"
    And пользователь не добавлен в организацию
    And действие в аудит лог записано как "invite_email_mismatch"
    And audit log содержит:
      | action          | invite_email_mismatch |
      | invite_id       | <invite_id> |
      | expected_email  | invited@example.com |
      | actual_email    | other@example.com |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

  @critical
  @state_transition
  @boundary
  @use_case=uc_05_02_12
  Scenario: Приглашение истекает ровно через 7 дней
    Given существует приглашение созданное "2026-03-02T10:00:00Z"
    And текущее время "2026-03-09T10:00:00Z"
    When пользователь переходит по ссылке приглашения
    Then возвращается ошибка с кодом "INVITE_EXPIRED"
    And приглашение переведено в статус "EXPIRED"
    And действие в аудит лог записано как "invite_expired"

  @critical
  @security
  @use_case=uc_05_02_13
  Scenario: Повторное использование уже принятого invite_token
    Given приглашение с токеном "<invite_token>" имеет статус "ACCEPTED"
    When кто-либо снова переходит по ссылке с тем же токеном
    Then возвращается ошибка с кодом "INVITE_ALREADY_USED"
    And действие в аудит лог записано как "invite_token_reuse_attempt"
    And audit log содержит:
      | action          | invite_token_reuse_attempt |
      | invite_id       | <invite_id> |
      | ip_address      | <ip> |
      | timestamp       | <iso8601> |

  @critical
  @security
  @use_case=uc_05_02_14
  Scenario: Невалидный invite_token
    When пользователь переходит по ссылке с токеном "00000000-0000-0000-0000-000000000000"
    Then возвращается ошибка с кодом "INVITE_NOT_FOUND"

  @critical
  @state_transition
  @use_case=uc_05_02_15
  Scenario: Отзыв приглашения администратором
    Given пользователь авторизован с ролью "ADMIN"
    And существует приглашение для "pending@example.com" со статусом "PENDING"
    When пользователь отзывает приглашение
    Then приглашение переведено в статус "REVOKED"
    And invite_token больше не валиден
    And действие в аудит лог записано как "invite_revoked"
    And audit log содержит:
      | action          | invite_revoked |
      | invite_id       | <invite_id> |
      | revoked_by      | <user_id> |
      | timestamp       | <iso8601> |

  @critical
  @validation
  @use_case=uc_05_02_16
  Scenario: Список членов организации
    Given пользователь авторизован с ролью "VIEWER"
    When пользователь запрашивает список членов организации
    Then возвращается список членов с полями:
      | field           |
      | user_id         |
      | email           |
      | full_name       |
      | role            |
      | joined_at       |
      | last_active_at  |
    And список не содержит password_hash или иных секретов

  @critical
  @state_transition
  @use_case=uc_05_02_17
  Scenario: Изменение роли участника OWNER'ом
    Given пользователь авторизован с ролью "OWNER"
    And участник "member@example.com" имеет роль "MEMBER"
    When пользователь изменяет роль участника на "ADMIN"
    Then роль участника равна "ADMIN"
    And действие в аудит лог записано как "member_role_changed"
    And audit log содержит:
      | action          | member_role_changed |
      | actor_id        | <user_id> |
      | target_user_id  | <member_id> |
      | old_role        | MEMBER |
      | new_role        | ADMIN |
      | timestamp       | <iso8601> |

  @critical
  @security
  @use_case=uc_05_02_18
  Scenario: ADMIN не может изменить роль OWNER
    Given пользователь авторизован с ролью "ADMIN"
    And участник "owner@example.com" имеет роль "OWNER"
    When пользователь пытается изменить роль OWNER на "MEMBER"
    Then возвращается ошибка с кодом "CANNOT_MODIFY_OWNER"
    And роль OWNER не изменена

  @critical
  @security
  @use_case=uc_05_02_19
  Scenario: MEMBER не может изменить роль другого участника
    Given пользователь авторизован с ролью "MEMBER"
    When пользователь пытается изменить роль другого участника на "ADMIN"
    Then возвращается ошибка с кодом "INSUFFICIENT_PERMISSIONS"

  @critical
  @state_transition
  @use_case=uc_05_02_20
  Scenario: Удаление участника администратором
    Given пользователь авторизован с ролью "ADMIN"
    And участник "member@example.com" имеет роль "MEMBER"
    When пользователь удаляет участника из организации
    Then участник удалён из организации
    And все активные сессии удалённого участника отозваны
    And ресурсы, созданные удалённым участником, сохранены и переданы организации
    And действие в аудит лог записано как "member_removed"
    And audit log содержит:
      | action          | member_removed |
      | actor_id        | <user_id> |
      | target_user_id  | <member_id> |
      | target_role     | MEMBER |
      | timestamp       | <iso8601> |

  @critical
  @security
  @use_case=uc_05_02_21
  Scenario: Удаление OWNER запрещено
    Given пользователь авторизован с ролью "ADMIN"
    And участник "owner@example.com" имеет роль "OWNER"
    When пользователь пытается удалить OWNER из организации
    Then возвращается ошибка с кодом "CANNOT_REMOVE_OWNER"
    And OWNER остаётся в организации

  @critical
  @state_transition
  @use_case=uc_05_02_22
  Scenario: Передача роли OWNER другому участнику
    Given пользователь авторизован с ролью "OWNER"
    And участник "admin@example.com" имеет роль "ADMIN"
    When пользователь передаёт роль OWNER участнику "admin@example.com"
    Then участник "admin@example.com" имеет роль "OWNER"
    And инициатор имеет роль "ADMIN"
    And в организации ровно один OWNER
    And действие в аудит лог записано как "ownership_transferred"
    And audit log содержит:
      | action          | ownership_transferred |
      | previous_owner  | <user_id> |
      | new_owner       | <target_user_id> |
      | timestamp       | <iso8601> |

  @critical
  @state_transition
  @use_case=uc_05_02_23
  Scenario: Участник покидает организацию
    Given пользователь авторизован с ролью "MEMBER"
    When пользователь выполняет действие leave organization
    Then пользователь удалён из организации
    And активные сессии пользователя отозваны
    And действие в аудит лог записано как "member_left"

  @critical
  @security
  @use_case=uc_05_02_24
  Scenario: OWNER не может покинуть организацию без передачи роли
    Given пользователь авторизован с ролью "OWNER"
    And в организации более одного участника
    When пользователь выполняет действие leave organization
    Then возвращается ошибка с кодом "OWNER_CANNOT_LEAVE"
    And ответ содержит структуру:
      | error.code                     | OWNER_CANNOT_LEAVE |
      | error.details.required_action  | transfer_ownership |
      | error.request_id               | <uuid>             |
    And пользователь остаётся в организации

  @critical
  @security
  @use_case=uc_05_02_25
  Scenario Outline: Матрица доступа для операции <operation>
    Given пользователь авторизован с ролью "<role>"
    When пользователь пытается выполнить "<operation>"
    Then результат равен "<result>"

    Examples:
      | role   | operation               | result                      |
      | OWNER  | invite_member           | ALLOWED                     |
      | ADMIN  | invite_member           | ALLOWED                     |
      | MEMBER | invite_member           | INSUFFICIENT_PERMISSIONS    |
      | VIEWER | invite_member           | INSUFFICIENT_PERMISSIONS    |
      | OWNER  | remove_member           | ALLOWED                     |
      | ADMIN  | remove_member           | ALLOWED                     |
      | MEMBER | remove_member           | INSUFFICIENT_PERMISSIONS    |
      | VIEWER | remove_member           | INSUFFICIENT_PERMISSIONS    |
      | OWNER  | change_role             | ALLOWED                     |
      | ADMIN  | change_role             | ALLOWED                     |
      | MEMBER | change_role             | INSUFFICIENT_PERMISSIONS    |
      | VIEWER | change_role             | INSUFFICIENT_PERMISSIONS    |
      | OWNER  | transfer_ownership      | ALLOWED                     |
      | ADMIN  | transfer_ownership      | INSUFFICIENT_PERMISSIONS    |
      | MEMBER | transfer_ownership      | INSUFFICIENT_PERMISSIONS    |
      | VIEWER | transfer_ownership      | INSUFFICIENT_PERMISSIONS    |
      | OWNER  | create_monitor          | ALLOWED                     |
      | ADMIN  | create_monitor          | ALLOWED                     |
      | MEMBER | create_monitor          | ALLOWED                     |
      | VIEWER | create_monitor          | INSUFFICIENT_PERMISSIONS    |
      | OWNER  | list_members            | ALLOWED                     |
      | ADMIN  | list_members            | ALLOWED                     |
      | MEMBER | list_members            | ALLOWED                     |
      | VIEWER | list_members            | ALLOWED                     |

  @critical
  @boundary
  @use_case=uc_05_02_26
  Scenario: Лимит одновременно активных приглашений
    Given пользователь авторизован с ролью "OWNER"
    And в организации уже "50" PENDING приглашений
    When пользователь отправляет 51-е приглашение
    Then возвращается ошибка с кодом "INVITE_LIMIT_REACHED"
    And ответ содержит структуру:
      | error.code              | INVITE_LIMIT_REACHED |
      | error.details.limit     | 50                    |
      | error.request_id        | <uuid>                |
    And приглашение не создано

  @critical
  @boundary
  @use_case=uc_05_02_27
  Scenario: Лимит участников организации по тарифу
    # Согласовано с epic=04_billing, uc_04_01_08 — используется общий код USER_LIMIT_EXCEEDED
    Given организация имеет подписку "Starter" с лимитом "5" пользователей
    And в организации уже "5" активных участников
    When OWNER отправляет новое приглашение
    Then возвращается ошибка с кодом "USER_LIMIT_EXCEEDED"
    And ответ содержит структуру:
      | error.code              | USER_LIMIT_EXCEEDED |
      | error.details.limit     | 5                    |
      | error.details.current   | 5                    |
      | error.details.tier      | Starter              |
      | error.request_id        | <uuid>               |
    And приглашение не создано

  @critical
  @security
  @use_case=uc_05_02_28
  Scenario: Удалённый участник теряет доступ немедленно
    Given участник "member@example.com" авторизован и имеет активный JWT
    When ADMIN удаляет участника из организации
    And удалённый участник выполняет запрос с существующим JWT
    Then возвращается ошибка с кодом "SESSION_REVOKED"
    And запрос отклонён

  @critical
  @security
  @use_case=uc_05_02_29
  Scenario: Изменение роли инвалидирует кэш permissions
    Given участник "member@example.com" имеет роль "MEMBER"
    And участник имеет активный JWT c claim role=MEMBER
    When ADMIN повышает роль участника до "ADMIN"
    Then при следующем запросе участника используется актуальная роль "ADMIN"
    And permissions cache участника инвалидирован

  @critical
  @validation
  @use_case=uc_05_02_30
  Scenario: Изоляция данных между организациями
    Given существуют две организации "Org A" и "Org B"
    And пользователь является MEMBER только в "Org A"
    When пользователь запрашивает список мониторов "Org B"
    Then возвращается ошибка с кодом "ORGANIZATION_ACCESS_DENIED"
    And данные "Org B" не раскрыты в ответе
