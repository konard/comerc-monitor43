# TECH_DEBT.md — Технический долг

Известные компромиссы и временные решения, требующие возврата.

Запись содержит: что сделано не идеально, почему, какой риск, и что
сделать, чтобы закрыть.

---

## Открытые пункты

### TD-001 — USER_LIMIT_EXCEEDED через локальный map в auth-service

**Где:** `backend/auth-service/internal/service/team_limits.go` —
`TierUserLimits = {"Free":1, "Starter":5, "Pro":20, "Enterprise":100}`.

**Что сделано:** Лимит участников по тарифу проверяется в `InviteService`
по локальному map; org.tier хранится в `organizations.tier` (копия из
billing).

**Почему:** Billing-service не знает про organizations и не имеет RPC,
возвращающего лимит участников по подписке. Делать такой RPC и менять
billing-домен — отдельная задача, которая блокировала бы зелёный BDD
прогон 05_02 на этой итерации.

**Риск:**
1. Drift тарифов между auth-service и billing-service — меняется
   `Subscription.MaxUsers` в billing, остаётся стабильным в auth.
2. Лимит не обновляется при upgrade/downgrade подписки в реальном
   времени (поле `org.tier` обновляется отдельно — пока не реализовано).

**Что сделать:**
1. Добавить в `api/proto/billing.proto` RPC
   `GetOrganizationLimits(org_id) (UsersLimit, MonitorsLimit, ...)`.
2. Сделать billing-service знающим про `organization_id` (миграция
   billing-схемы — добавить FK в `subscriptions`).
3. В auth-service `InviteService.Invite` вызывать этот RPC вместо чтения
   локального map; кешировать ответ с TTL.
4. Удалить `TierUserLimits` из auth-service.

**Связанные сценарии:** `uc_05_02_27`, `uc_04_01_08`.

**Связано с:** TD-002.

---

### TD-002 — Email-доставка приглашений: только publish, без consumer

**Где:**
- Publisher: `backend/auth-service/internal/publisher/publisher.go` —
  `PublishInviteCreated` шлёт event `team.invite.created` в exchange
  `auth.events`.
- Consumer: отсутствует.
- Шаблон email: отсутствует в notification-template-service
  (`backend/notification-template-service/`).

**Что сделано:** При `InviteMember` событие публикуется в очередь, но
никто его не читает. BDD-шаг «на email отправлено письмо…»
(`uc_05_02_02`) проверяется по факту записи `member_invited` в
`auth_audit_log`, а не по реальной доставке.

**Почему:** Реализация SMTP-консьюмера + новый шаблон `TYPE_TEAM_INVITE`
+ wiring в notification-template-service — отдельный объём работы,
не блокирующий контракт invite flow.

**Риск:** В проде приглашения создаются, но письма не уходят →
пользователь не получает invite-ссылку → invite expires unused → silent
fail.

**Что сделать:**
1. Добавить enum `TYPE_TEAM_INVITE` в
   `api/proto/notification_template.proto:120`.
2. Создать дефолтный шаблон в notification-template-service.
3. Поднять консьюмер `team.invite.created` в notification-template-service
   (или отдельном email-worker'е), чтобы он формировал email из шаблона
   и отправлял через SMTP-адаптер.
4. В BDD-стеке заменить fake-проверку на реальную через `fake_smtp.go`
   (он уже есть в `features/suite/`).

**Связанные сценарии:** `uc_05_02_02`, `uc_05_02_06`.

---

### TD-003 — Cross-org RBAC enforcement в monitor-service

**Где:** `features/suite/steps_05_security_team.go` — шаг
`userTriesOperation` для `create_monitor` имеет TODO и тег
`@requires_monitor_service` на сценарии.

**Что сделано:** Локальная фабрикация ошибки `INSUFFICIENT_PERMISSIONS`
для `VIEWER + create_monitor`. Реальный gRPC к monitor-service не
выполняется, потому что monitor-service не поднят в стеке security и
не парсит `Role`/`OrgID` из JWT auth-service'а.

**Почему:** Для зелёного RBAC matrix uc_05_02_25 (24 комбинации) нужно
было бы поднимать monitor-service в стеке + расширять monitor-service
RBAC middleware на `Role` claim из JWT. Это объём отдельной задачи.

**Риск:** В проде VIEWER может создать монитор, потому что
monitor-service не валидирует `role` из JWT — только `user_id`.

**Что сделать:**
1. Расширить monitor-service JWT middleware: парсить `role` claim,
   валидировать что не VIEWER при write-операциях.
2. Добавить monitor-service в `features/suite/stack.go` для эпика
   `05_security`.
3. Заменить локальную фабрикацию на реальный gRPC-вызов.
4. Снять тег `@requires_monitor_service`.

**Связанные сценарии:** `uc_05_02_25` (строки matrix с
`create_monitor`).

---

### TD-004 — Параллельный `GenerateTokenPairWithTeam`

**Где:** `backend/auth-service/internal/infrastructure/jwt/jwt.go` —
два метода: старый `GenerateTokenPair(userID, email, tier)` и новый
`GenerateTokenPairWithTeam(userID, email, tier, orgID, role)`.

**Что сделано:** Новый метод добавлен параллельно, чтобы не ломать
существующие 4 mock-файла, `jwt_test.go` и `token_service_test.go`.
В callers (`auth_service.go`) выбор между ними по наличию team-context.

**Почему:** Расширение единой сигнатуры с `orgID, role` потребовало бы
обновления всех callers и регенерации mocks; на быстрой итерации green
выбран наименее инвазивный вариант.

**Риск:** API double-surface — два способа сгенерировать пару токенов
для одного и того же кейса. Будущие callers могут случайно использовать
старый и потерять team-context в JWT.

**Что сделать:**
1. Унифицировать в один `GenerateTokenPair(userID, email, tier, orgID, role)`.
2. Удалить `GenerateTokenPairWithTeam`.
3. Обновить mocks (`mockery v3` regen).
4. Обновить тесты: добавить `uuid.Nil, ""` в кейсы, где team-context
   неактуален.

**Связано с:** TD-005.

---

### TD-005 — Permissions cache не реализован

**Где:** Концепт упомянут в `uc_05_02_29` («permissions cache участника
инвалидирован»).

**Что сделано:** Сценарий проходит через `RefreshToken` — после смены
роли участник вызывает refresh, получает новый JWT с актуальным `role`
claim.

**Почему:** Реализация Redis-кеша permissions с pub/sub
инвалидацией — отдельная инфраструктурная история. На данном этапе нет
доказанной необходимости (проверка role укладывается в JWT validation
без отдельного запроса).

**Риск:** Если access token TTL длинный (по умолчанию 15 мин), участник
с понижением роли продолжает иметь старые права до истечения токена.
В UI это компенсируется частыми refresh, в backend-to-backend вызовах —
требует более короткого TTL или отдельного кеша.

**Что сделать (если/когда понадобится):**
1. Добавить Redis-кеш `permissions:{user_id}:{org_id}` со списком
   разрешённых операций.
2. Инвалидировать ключ при `ChangeMemberRole`/`RemoveMember`/
   `TransferOwnership`.
3. JWT middleware проверяет кеш в дополнение к claim.

**Решение:** оставить в backlog до появления требования. Сейчас
RefreshToken покрывает кейс.

---

### TD-006 — Pre-existing lint в root (не мой scope)

**Где:**
- `test/migration_roundtrip_test.go:64,88,96` — 3× errcheck
  (`adminDB.Close`, `adminDB.ExecContext`, `db.Close` в
  `t.Cleanup(func(){ _ = ... })`).
- `features/suite/state.go:29` — gci (форматирование импортов).

**Что сделано:** Не трогал — pre-existing на момент входа в задачу
team management.

**Риск:** Низкий. Lint-нойз в общем прогоне `bash scripts/lint.sh` из
корня. Lint в auth-service чист.

**Что сделать:** Поправить владельцу соответствующих файлов отдельным
PR.

---

## Закрытые пункты

(пусто — добавлять сюда после фактического закрытия с указанием PR/коммита)

---

*Версия: 1.0 | Создан: 2026-04-25 | Контекст: Team Management
(epic=05_security, us=02_team_management)*
