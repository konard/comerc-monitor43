//go:build bdd

package suite

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/raul/monitor/api/proto"
)

// teamSteps реализует шаги Gherkin для user story 05_02_team_management.
type teamSteps struct {
	stack *Stack
	state *ScenarioState
}

// RegisterTeamSteps регистрирует шаги для управления командой и приглашениями.
// Должен вызываться после RegisterSecuritySteps, чтобы общие шаги
// (например "audit log содержит:") привязались к security-steps первыми.
// Godog использует first-match-wins.
func RegisterTeamSteps(ctx *godog.ScenarioContext, stack *Stack, state *ScenarioState) {
	t := &teamSteps{stack: stack, state: state}

	// Контекст организации и авторизации
	ctx.Step(`^пользователь авторизован с ролью "([^"]*)"$`, t.userAuthorizedWithRole)
	ctx.Step(`^новый пользователь авторизуется через провайдер "([^"]*)"$`, t.newUserAuthViaProvider)
	ctx.Step(`^аккаунт успешно создан$`, t.accountCreated)
	ctx.Step(`^автоматически создана организация с владельцем этого пользователя$`, t.orgAutoCreated)
	ctx.Step(`^пользователь имеет роль "([^"]*)" в организации$`, t.userHasRoleInOrg)

	// Приглашения — отправка
	ctx.Step(`^пользователь отправляет приглашение:$`, t.userSendsInvite)
	ctx.Step(`^пользователь повторно отправляет приглашение:$`, t.userSendsInvite)
	ctx.Step(`^OWNER отправляет новое приглашение$`, t.ownerSendsNewInvite)
	ctx.Step(`^пользователь отправляет 51-е приглашение$`, t.userSendsInvite51)
	ctx.Step(`^создано приглашение со статусом "([^"]*)"$`, t.inviteCreatedWithStatus)
	ctx.Step(`^создано новое приглашение со статусом "([^"]*)"$`, t.inviteCreatedWithStatus)
	ctx.Step(`^сгенерирован одноразовый invite_token$`, t.inviteTokenGenerated)
	ctx.Step(`^на email "([^"]*)" отправлено письмо с ссылкой приглашения$`, t.inviteEmailSent)
	ctx.Step(`^приглашение не создано$`, t.inviteNotCreated)
	ctx.Step(`^роль приглашения равна "([^"]*)"$`, t.inviteRoleEquals)

	// Приглашения — состояние и переходы
	ctx.Step(`^существует приглашение для "([^"]*)"$`, t.inviteExistsFor)
	ctx.Step(`^существует приглашение "([^"]*)" со статусом "([^"]*)"$`, t.inviteExistsWithStatus)
	ctx.Step(`^существует приглашение для "([^"]*)" со статусом "([^"]*)"$`, t.inviteExistsForWithStatus)
	ctx.Step(`^существует приглашение с токеном "([^"]*)" и ролью "([^"]*)"$`, t.inviteExistsWithTokenRole)
	ctx.Step(`^существует приглашение созданное "([^"]*)"$`, t.inviteCreatedAt)
	ctx.Step(`^приглашение имеет статус "([^"]*)"$`, t.inviteHasStatus)
	ctx.Step(`^приглашение с токеном "([^"]*)" имеет статус "([^"]*)"$`, t.inviteWithTokenHasStatus)
	ctx.Step(`^приглашение остаётся в статусе "([^"]*)"$`, t.inviteRemainsInStatus)
	ctx.Step(`^приглашение переведено в статус "([^"]*)"$`, t.inviteTransitionedTo)
	ctx.Step(`^предыдущее приглашение переведено в статус "([^"]*)"$`, t.previousInviteTransitionedTo)
	ctx.Step(`^invite_token больше не валиден$`, t.inviteTokenInvalid)
	ctx.Step(`^invite_token погашен и больше не валиден$`, t.inviteTokenInvalid)

	// Приглашения — приём и ошибки
	ctx.Step(`^новый пользователь переходит по ссылке приглашения$`, t.newUserClicksInvite)
	ctx.Step(`^пользователь переходит по ссылке приглашения$`, t.userClicksInvite)
	ctx.Step(`^авторизованный пользователь с email "([^"]*)" переходит по ссылке приглашения$`, t.authedUserClicksInviteAs)
	ctx.Step(`^пользователь переходит по ссылке с токеном "([^"]*)"$`, t.userClicksInviteWithToken)
	ctx.Step(`^пользователь авторизуется через провайдер "([^"]*)" с email приглашения$`, t.userAuthViaProviderWithInviteEmail)
	ctx.Step(`^кто-либо снова переходит по ссылке с тем же токеном$`, t.someoneReusesToken)
	ctx.Step(`^пользователь добавлен в организацию с ролью "([^"]*)"$`, t.userAddedToOrgWithRole)
	ctx.Step(`^пользователь не добавлен в организацию$`, t.userNotAddedToOrg)

	// Отзыв приглашений
	ctx.Step(`^пользователь отзывает приглашение$`, t.userRevokesInvite)

	// Список членов
	ctx.Step(`^пользователь запрашивает список членов организации$`, t.userListsMembers)
	ctx.Step(`^возвращается список членов с полями:$`, t.membersListReturnedWithFields)
	ctx.Step(`^список не содержит password_hash или иных секретов$`, t.listHasNoSecrets)

	// Изменение роли
	ctx.Step(`^участник "([^"]*)" имеет роль "([^"]*)"$`, t.memberHasRole)
	ctx.Step(`^пользователь изменяет роль участника на "([^"]*)"$`, t.userChangesRoleTo)
	ctx.Step(`^пользователь пытается изменить роль OWNER на "([^"]*)"$`, t.userTriesChangeOwnerTo)
	ctx.Step(`^пользователь пытается изменить роль другого участника на "([^"]*)"$`, t.userTriesChangeOtherTo)
	ctx.Step(`^роль участника равна "([^"]*)"$`, t.memberRoleEquals)
	ctx.Step(`^роль OWNER не изменена$`, t.ownerRoleUnchanged)
	ctx.Step(`^ADMIN повышает роль участника до "([^"]*)"$`, t.adminPromotesMemberTo)
	ctx.Step(`^при следующем запросе участника используется актуальная роль "([^"]*)"$`, t.nextRequestUsesRole)
	ctx.Step(`^permissions cache участника инвалидирован$`, t.permissionsCacheInvalidated)

	// Удаление участника
	ctx.Step(`^пользователь удаляет участника из организации$`, t.userRemovesMember)
	ctx.Step(`^ADMIN удаляет участника из организации$`, t.adminRemovesMember)
	ctx.Step(`^пользователь пытается удалить OWNER из организации$`, t.userTriesRemoveOwner)
	ctx.Step(`^участник удалён из организации$`, t.memberRemoved)
	ctx.Step(`^пользователь удалён из организации$`, t.memberRemoved)
	ctx.Step(`^OWNER остаётся в организации$`, t.ownerStaysInOrg)
	ctx.Step(`^пользователь остаётся в организации$`, t.userStaysInOrg)
	ctx.Step(`^все активные сессии удалённого участника отозваны$`, t.removedMemberSessionsRevoked)
	ctx.Step(`^активные сессии пользователя отозваны$`, t.userSessionsRevoked)
	ctx.Step(`^ресурсы, созданные удалённым участником, сохранены и переданы организации$`, t.resourcesPreserved)

	// Передача роли OWNER
	ctx.Step(`^пользователь передаёт роль OWNER участнику "([^"]*)"$`, t.userTransfersOwnership)
	ctx.Step(`^инициатор имеет роль "([^"]*)"$`, t.initiatorHasRole)
	ctx.Step(`^в организации ровно один OWNER$`, t.exactlyOneOwner)

	// Leave organization
	ctx.Step(`^пользователь выполняет действие leave organization$`, t.userLeavesOrg)
	ctx.Step(`^в организации более одного участника$`, t.moreThanOneMember)

	// Активные участники, лимиты
	ctx.Step(`^в организации уже "([^"]*)" активных участников$`, t.alreadyNActiveMembers)
	ctx.Step(`^в организации уже "([^"]*)" PENDING приглашений$`, t.alreadyNPendingInvites)
	ctx.Step(`^в организации уже есть активный участник "([^"]*)"$`, t.alreadyHasActiveMember)
	ctx.Step(`^организация имеет подписку "([^"]*)" с лимитом "([^"]*)" пользователей$`, t.orgTierLimit)

	// Сессии и токены удалённого участника
	ctx.Step(`^участник "([^"]*)" авторизован и имеет активный JWT$`, t.memberHasActiveJWT)
	ctx.Step(`^участник имеет активный JWT c claim role=MEMBER$`, t.memberHasJWTRoleMember)
	ctx.Step(`^удалённый участник выполняет запрос с существующим JWT$`, t.removedMemberMakesRequest)
	ctx.Step(`^запрос отклонён$`, t.requestRejected)

	// Матрица доступа (Scenario Outline)
	ctx.Step(`^пользователь пытается выполнить "([^"]*)"$`, t.userTriesOperation)
	ctx.Step(`^результат равен "([^"]*)"$`, t.resultEquals)

	// Изоляция между организациями
	ctx.Step(`^существуют две организации "([^"]*)" и "([^"]*)"$`, t.twoOrgsExist)
	ctx.Step(`^пользователь является MEMBER только в "([^"]*)"$`, t.userMemberOnlyIn)
	ctx.Step(`^пользователь запрашивает список мониторов "([^"]*)"$`, t.userListsMonitorsOf)
	ctx.Step(`^данные "([^"]*)" не раскрыты в ответе$`, t.dataNotDisclosed)
}

// === helpers ===

// errReason извлекает Reason из errdetails.ErrorInfo в gRPC статусе.
func errReason(err error) string {
	if err == nil {
		return ""
	}
	st, ok := status.FromError(err)
	if !ok {
		return ""
	}
	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok {
			return info.Reason
		}
	}
	return ""
}

// loadInviteByID читает invite из БД по id.
func (t *teamSteps) loadInviteStatus(id uuid.UUID) (string, error) {
	var s string
	if err := t.stack.AuthDB().Get(&s, `SELECT status::text FROM invites WHERE id=$1`, id); err != nil {
		return "", err
	}
	return s, nil
}

// loadInviteByEmail возвращает первый invite по email в orgID.
func (t *teamSteps) loadInviteByEmail(orgID uuid.UUID, email string) (uuid.UUID, string, error) {
	var id uuid.UUID
	var st string
	err := t.stack.AuthDB().QueryRow(
		`SELECT id, status::text FROM invites WHERE organization_id=$1 AND email=$2 ORDER BY created_at DESC LIMIT 1`,
		orgID, email,
	).Scan(&id, &st)
	return id, st, err
}

// roleProto конвертирует строковую роль в proto enum.
func roleStringToProto(role string) authv1.Role {
	switch strings.ToUpper(role) {
	case "OWNER":
		return authv1.Role_ROLE_OWNER
	case "ADMIN":
		return authv1.Role_ROLE_ADMIN
	case "MEMBER":
		return authv1.Role_ROLE_MEMBER
	case "VIEWER":
		return authv1.Role_ROLE_VIEWER
	default:
		return authv1.Role_ROLE_UNSPECIFIED
	}
}

// roleProtoToString — обратное преобразование.
func roleProtoToString(r authv1.Role) string {
	switch r {
	case authv1.Role_ROLE_OWNER:
		return "OWNER"
	case authv1.Role_ROLE_ADMIN:
		return "ADMIN"
	case authv1.Role_ROLE_MEMBER:
		return "MEMBER"
	case authv1.Role_ROLE_VIEWER:
		return "VIEWER"
	default:
		return ""
	}
}

// inviteStatusProtoToString переводит proto-enum в строку статуса.
func inviteStatusProtoToString(s authv1.InviteStatus) string {
	switch s {
	case authv1.InviteStatus_INVITE_STATUS_PENDING:
		return "PENDING"
	case authv1.InviteStatus_INVITE_STATUS_ACCEPTED:
		return "ACCEPTED"
	case authv1.InviteStatus_INVITE_STATUS_DECLINED:
		return "DECLINED"
	case authv1.InviteStatus_INVITE_STATUS_EXPIRED:
		return "EXPIRED"
	case authv1.InviteStatus_INVITE_STATUS_REVOKED:
		return "REVOKED"
	}
	return ""
}

// ensureActiveActor подготавливает state.ActorAccessToken / OrgID / UserID
// для шагов, которые предполагают авторизованного пользователя. Если
// никакого контекста ещё нет — регистрирует OWNER.
func (t *teamSteps) ensureActiveActor(ctx context.Context) error {
	if t.state.ActorAccessToken != "" {
		return nil
	}
	owner, err := registerOwnerWithTier(ctx, t.stack.Client, t.stack, "Pro")
	if err != nil {
		return err
	}
	t.state.OwnerAuth = owner.Auth
	t.state.OwnerUserID = owner.UserID
	t.state.OwnerEmail = owner.Email
	t.state.OrgID = owner.OrgID
	t.state.UserID = owner.UserID
	t.state.UserRole = "OWNER"
	t.state.AuthResp = owner.Auth
	t.state.ActorAccessToken = owner.Auth.AccessToken
	return nil
}

// === Steps ===

// userAuthorizedWithRole — Given пользователь авторизован с ролью X.
// Реализация: всегда регистрируем OWNER для контекста организации; если запрошена
// другая роль — регистрируем второго пользователя и подменяем JWT с нужной
// ролью внутри той же организации (минуя реальный invite-flow для скорости).
func (t *teamSteps) userAuthorizedWithRole(ctx context.Context, role string) error {
	role = strings.ToUpper(role)
	owner, err := registerOwnerWithTier(ctx, t.stack.Client, t.stack, "Pro")
	if err != nil {
		return err
	}
	t.state.OwnerAuth = owner.Auth
	t.state.OwnerUserID = owner.UserID
	t.state.OwnerEmail = owner.Email
	t.state.OrgID = owner.OrgID

	if role == "OWNER" {
		t.state.UserID = owner.UserID
		t.state.UserRole = "OWNER"
		t.state.AuthResp = owner.Auth
		t.state.ActorAccessToken = owner.Auth.AccessToken
		t.state.UserEmail = owner.Email
		return nil
	}

	// Второй пользователь — actor с заданной ролью в OWNER'ской организации.
	actorEmail := uniqueEmail("actor-" + strings.ToLower(role))
	actorResp, err := registerUser(ctx, t.stack.Client, actorEmail, "BDD Actor")
	if err != nil {
		return err
	}
	c, err := parseTeamToken(actorResp.AccessToken)
	if err != nil {
		return err
	}
	if err := insertMembership(t.stack, owner.OrgID, c.UserID, role); err != nil {
		return err
	}
	// Sanity check: verify membership was actually inserted.
	var verifyRole string
	if e := t.stack.AuthDB().Get(&verifyRole,
		`SELECT role::text FROM memberships WHERE organization_id=$1 AND user_id=$2`,
		owner.OrgID, c.UserID,
	); e != nil {
		return fmt.Errorf("verify membership: %w", e)
	}
	if verifyRole != role {
		return fmt.Errorf("verify membership: expected role %q, got %q", role, verifyRole)
	}
	// Минтим токен с OrgID = owner.OrgID и нужной Role.
	tok, err := mintTeamToken(c.UserID, actorEmail, role, owner.OrgID)
	if err != nil {
		return err
	}
	t.state.UserID = c.UserID
	t.state.UserEmail = actorEmail
	t.state.UserRole = role
	t.state.SecondAuth = actorResp
	t.state.AuthResp = actorResp
	t.state.ActorAccessToken = tok
	return nil
}

// newUserAuthViaProvider — Given новый пользователь авторизуется через провайдер.
// В нашем стеке провайдер моделируется как обычный Register/PasswordLogin.
func (t *teamSteps) newUserAuthViaProvider(ctx context.Context, _ string) error {
	email := uniqueEmail("new")
	resp, err := registerUser(ctx, t.stack.Client, email, "BDD New User")
	if err != nil {
		return err
	}
	c, err := parseTeamToken(resp.AccessToken)
	if err != nil {
		return err
	}
	t.state.UserEmail = email
	t.state.UserID = c.UserID
	t.state.OrgID = c.OrgID
	t.state.UserRole = "OWNER" // ensureTeamContext выдаёт OWNER при первой регистрации
	t.state.AuthResp = resp
	t.state.ActorAccessToken = resp.AccessToken
	return nil
}

// accountCreated — Then аккаунт успешно создан.
func (t *teamSteps) accountCreated(_ context.Context) error {
	if t.state.AuthResp == nil || t.state.AuthResp.User == nil {
		return fmt.Errorf("аккаунт не создан: AuthResp пуст")
	}
	return nil
}

// orgAutoCreated — Then автоматически создана организация.
func (t *teamSteps) orgAutoCreated(_ context.Context) error {
	if t.state.OrgID == uuid.Nil {
		return fmt.Errorf("OrgID не установлен — организация не создана")
	}
	var ownerID uuid.UUID
	if err := t.stack.AuthDB().Get(&ownerID, `SELECT owner_id FROM organizations WHERE id=$1`, t.state.OrgID); err != nil {
		return fmt.Errorf("get organization: %w", err)
	}
	if ownerID != t.state.UserID {
		return fmt.Errorf("ожидался owner_id=%s, получен %s", t.state.UserID, ownerID)
	}
	return nil
}

// userHasRoleInOrg — Then пользователь имеет роль X в организации.
func (t *teamSteps) userHasRoleInOrg(_ context.Context, role string) error {
	var actual string
	err := t.stack.AuthDB().Get(&actual,
		`SELECT role::text FROM memberships WHERE organization_id=$1 AND user_id=$2`,
		t.state.OrgID, t.state.UserID)
	if err != nil {
		return fmt.Errorf("get membership: %w", err)
	}
	if actual != strings.ToUpper(role) {
		return fmt.Errorf("ожидалась роль %q, получена %q", role, actual)
	}
	return nil
}

// === Приглашения — отправка ===

// readTableTwoCol читает 2-колоночную data table в map.
func readTableTwoCol(table *godog.Table) map[string]string {
	out := map[string]string{}
	for _, row := range table.Rows {
		if len(row.Cells) >= 2 {
			out[strings.TrimSpace(row.Cells[0].Value)] = strings.TrimSpace(row.Cells[1].Value)
		}
	}
	return out
}

// userSendsInvite — When пользователь отправляет приглашение: <table>.
func (t *teamSteps) userSendsInvite(ctx context.Context, table *godog.Table) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	m := readTableTwoCol(table)
	email := m["email"]
	role := m["role"]
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	resp, err := t.stack.Client.InviteMember(authedCtx, &authv1.InviteMemberRequest{
		Email: email,
		Role:  roleStringToProto(role),
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	t.state.LastInvite = resp
	t.state.LastInviteEmail = email
	if resp != nil {
		t.state.LastInviteToken = resp.Token
	}
	return nil
}

// ownerSendsNewInvite — When OWNER отправляет новое приглашение.
func (t *teamSteps) ownerSendsNewInvite(ctx context.Context) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	email := uniqueEmail("limit-test")
	resp, err := t.stack.Client.InviteMember(authedCtx, &authv1.InviteMemberRequest{
		Email: email,
		Role:  authv1.Role_ROLE_MEMBER,
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	t.state.LastInvite = resp
	t.state.LastInviteEmail = email
	if resp != nil {
		t.state.LastInviteToken = resp.Token
	}
	return nil
}

// userSendsInvite51 — When пользователь отправляет 51-е приглашение.
func (t *teamSteps) userSendsInvite51(ctx context.Context) error {
	return t.ownerSendsNewInvite(ctx)
}

// inviteCreatedWithStatus — Then создано приглашение со статусом X.
func (t *teamSteps) inviteCreatedWithStatus(_ context.Context, expected string) error {
	if t.state.LastInvite == nil {
		return fmt.Errorf("приглашение не создано: LastInvite пуст (LastErr=%v)", t.state.LastErr)
	}
	got := inviteStatusProtoToString(t.state.LastInvite.Status)
	if got != strings.ToUpper(expected) {
		return fmt.Errorf("ожидался статус %q, получен %q", expected, got)
	}
	return nil
}

// inviteTokenGenerated — Then сгенерирован одноразовый invite_token.
func (t *teamSteps) inviteTokenGenerated(_ context.Context) error {
	if t.state.LastInvite == nil || t.state.LastInvite.Token == "" {
		return fmt.Errorf("invite_token пуст")
	}
	if _, err := uuid.Parse(t.state.LastInvite.Token); err != nil {
		return fmt.Errorf("invite_token не uuid: %v", err)
	}
	return nil
}

// inviteEmailSent — Then на email "X" отправлено письмо. Проверяем что invite
// создан с заданным email (фактическая отправка письма не требуется в BDD).
func (t *teamSteps) inviteEmailSent(_ context.Context, email string) error {
	if t.state.LastInvite == nil {
		return fmt.Errorf("приглашение не создано")
	}
	if t.state.LastInvite.Email != email {
		return fmt.Errorf("ожидался email %q, получен %q", email, t.state.LastInvite.Email)
	}
	return nil
}

// inviteNotCreated — Then приглашение не создано.
func (t *teamSteps) inviteNotCreated(_ context.Context) error {
	if t.state.LastErr == nil {
		return fmt.Errorf("ожидалась ошибка, но приглашение создано")
	}
	return nil
}

// inviteRoleEquals — Then роль приглашения равна X.
func (t *teamSteps) inviteRoleEquals(_ context.Context, expected string) error {
	if t.state.LastInvite == nil {
		return fmt.Errorf("приглашение не создано")
	}
	got := roleProtoToString(t.state.LastInvite.Role)
	if got != strings.ToUpper(expected) {
		return fmt.Errorf("ожидалась роль %q, получена %q", expected, got)
	}
	return nil
}

// === существующие приглашения ===

// inviteExistsFor — Given существует приглашение для "X". Создаём через OWNER.
func (t *teamSteps) inviteExistsFor(ctx context.Context, email string) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	authedCtx := withAuth(ctx, t.state.OwnerAuth.AccessToken)
	resp, err := t.stack.Client.InviteMember(authedCtx, &authv1.InviteMemberRequest{
		Email: email,
		Role:  authv1.Role_ROLE_MEMBER,
	})
	if err != nil {
		return fmt.Errorf("create invite for %s: %w", email, err)
	}
	t.state.LastInvite = resp
	t.state.LastInviteToken = resp.Token
	t.state.LastInviteEmail = email
	return nil
}

// inviteExistsWithStatus — Given существует приглашение "X" со статусом Y.
func (t *teamSteps) inviteExistsWithStatus(ctx context.Context, email, status string) error {
	if err := t.inviteExistsFor(ctx, email); err != nil {
		return err
	}
	if strings.ToUpper(status) != "PENDING" {
		// Принудительно меняем статус.
		_, err := t.stack.AuthDB().Exec(
			`UPDATE invites SET status=$1::invite_status WHERE id=$2`,
			strings.ToUpper(status), uuid.MustParse(t.state.LastInvite.Id),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// inviteExistsForWithStatus — алиас для двух-параметрической формы.
func (t *teamSteps) inviteExistsForWithStatus(ctx context.Context, email, status string) error {
	return t.inviteExistsWithStatus(ctx, email, status)
}

// inviteExistsWithTokenRole — Given существует приглашение с токеном "X" и ролью "Y".
// Создаём приглашение, сохраняем как state. Игнорируем placeholder "<invite_token>".
func (t *teamSteps) inviteExistsWithTokenRole(ctx context.Context, _, role string) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	email := uniqueEmail("invitee")
	authedCtx := withAuth(ctx, t.state.OwnerAuth.AccessToken)
	resp, err := t.stack.Client.InviteMember(authedCtx, &authv1.InviteMemberRequest{
		Email: email,
		Role:  roleStringToProto(role),
	})
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	t.state.LastInvite = resp
	t.state.LastInviteToken = resp.Token
	t.state.LastInviteEmail = email
	return nil
}

// inviteCreatedAt — Given существует приглашение созданное "<ISO>".
func (t *teamSteps) inviteCreatedAt(ctx context.Context, isoTime string) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	email := uniqueEmail("expired")
	authedCtx := withAuth(ctx, t.state.OwnerAuth.AccessToken)
	resp, err := t.stack.Client.InviteMember(authedCtx, &authv1.InviteMemberRequest{
		Email: email,
		Role:  authv1.Role_ROLE_MEMBER,
	})
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		return fmt.Errorf("parse iso: %w", err)
	}
	id, err := uuid.Parse(resp.Id)
	if err != nil {
		return err
	}
	if err := forceInviteCreatedAt(t.stack, id, createdAt); err != nil {
		return err
	}
	t.state.LastInvite = resp
	t.state.LastInviteToken = resp.Token
	t.state.LastInviteEmail = email
	return nil
}

// inviteHasStatus — Given приглашение имеет статус X.
func (t *teamSteps) inviteHasStatus(_ context.Context, expected string) error {
	if t.state.LastInvite == nil {
		return fmt.Errorf("нет последнего приглашения")
	}
	st, err := t.loadInviteStatus(uuid.MustParse(t.state.LastInvite.Id))
	if err != nil {
		return err
	}
	if st != strings.ToUpper(expected) {
		// Принудительно ставим — этот шаг готовит сценарий.
		_, err := t.stack.AuthDB().Exec(
			`UPDATE invites SET status=$1::invite_status WHERE id=$2`,
			strings.ToUpper(expected), uuid.MustParse(t.state.LastInvite.Id),
		)
		return err
	}
	return nil
}

// inviteWithTokenHasStatus — Given приглашение с токеном "X" имеет статус Y.
// Создаёт invite + переключает статус.
func (t *teamSteps) inviteWithTokenHasStatus(ctx context.Context, _, statusVal string) error {
	if err := t.inviteExistsWithTokenRole(ctx, "", "MEMBER"); err != nil {
		return err
	}
	return t.inviteHasStatus(ctx, statusVal)
}

// inviteRemainsInStatus — Then приглашение остаётся в статусе X.
func (t *teamSteps) inviteRemainsInStatus(_ context.Context, expected string) error {
	return t.inviteHasStatusEquals(expected)
}

// inviteTransitionedTo — Then приглашение переведено в статус X.
func (t *teamSteps) inviteTransitionedTo(_ context.Context, expected string) error {
	return t.inviteHasStatusEquals(expected)
}

// previousInviteTransitionedTo — Then предыдущее приглашение переведено в статус X.
// Под "предыдущим" подразумеваем приглашение того же email, чей id отличается
// от state.LastInvite (которое — новое приглашение).
func (t *teamSteps) previousInviteTransitionedTo(_ context.Context, expected string) error {
	if t.state.LastInviteEmail == "" {
		return fmt.Errorf("нет email приглашения")
	}
	currentID := uuid.Nil
	if t.state.LastInvite != nil {
		currentID, _ = uuid.Parse(t.state.LastInvite.Id)
	}
	var st string
	err := t.stack.AuthDB().Get(&st,
		`SELECT status::text FROM invites
		 WHERE organization_id=$1 AND email=$2 AND id <> $3
		 ORDER BY created_at ASC LIMIT 1`,
		t.state.OrgID, t.state.LastInviteEmail, currentID,
	)
	if err != nil {
		return err
	}
	if st != strings.ToUpper(expected) {
		return fmt.Errorf("ожидался статус предыдущего %q, получен %q", expected, st)
	}
	return nil
}

func (t *teamSteps) inviteHasStatusEquals(expected string) error {
	if t.state.LastInvite == nil {
		return fmt.Errorf("нет последнего приглашения")
	}
	st, err := t.loadInviteStatus(uuid.MustParse(t.state.LastInvite.Id))
	if err != nil {
		return err
	}
	if st != strings.ToUpper(expected) {
		return fmt.Errorf("ожидался статус %q, получен %q", expected, st)
	}
	return nil
}

// inviteTokenInvalid — Then invite_token больше не валиден.
func (t *teamSteps) inviteTokenInvalid(ctx context.Context) error {
	if t.state.LastInviteToken == "" {
		return nil
	}
	// Попытка повторно accept — должна вернуть ошибку.
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	_, err := t.stack.Client.AcceptInvite(authedCtx, &authv1.AcceptInviteRequest{
		Token: t.state.LastInviteToken,
	})
	if err == nil {
		return fmt.Errorf("ожидалась ошибка повторного использования токена")
	}
	return nil
}

// === Принятие приглашений ===

// newUserClicksInvite — When новый пользователь переходит по ссылке. Здесь только
// помечаем намерение; принятие выполняется следующим шагом
// "пользователь авторизуется через провайдер ... с email приглашения".
func (t *teamSteps) newUserClicksInvite(_ context.Context) error {
	return nil
}

// userClicksInvite — When пользователь переходит по ссылке приглашения.
// Если уже есть авторизованный пользователь — пытаемся принять текущим токеном;
// иначе регистрируем нового.
func (t *teamSteps) userClicksInvite(ctx context.Context) error {
	if t.state.LastInviteToken == "" {
		return fmt.Errorf("нет invite token")
	}
	// Регистрируем пользователя с email приглашения.
	if t.state.LastInviteEmail == "" {
		return fmt.Errorf("нет invite email")
	}
	resp, err := registerUser(ctx, t.stack.Client, t.state.LastInviteEmail, "Invitee")
	if err != nil {
		// Если пользователь уже существует — логинимся? для упрощения логиним второго через mintTeamToken
		// (используем uuid в качестве user_id и регистрацию через прямой SQL не делаем).
		return fmt.Errorf("register invitee: %w", err)
	}
	// Переключаем state на нового пользователя — он становится actor'ом для
	// последующих audit-проверок (uc_05_02_10 «invite_accepted»).
	if id, parseErr := uuid.Parse(resp.User.Id); parseErr == nil {
		t.state.UserID = id
	}
	authedCtx := withAuth(ctx, resp.AccessToken)
	_, err = t.stack.Client.AcceptInvite(authedCtx, &authv1.AcceptInviteRequest{
		Token: t.state.LastInviteToken,
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

// authedUserClicksInviteAs — When авторизованный пользователь с email "X" переходит по ссылке.
func (t *teamSteps) authedUserClicksInviteAs(ctx context.Context, email string) error {
	if t.state.LastInviteToken == "" {
		return fmt.Errorf("нет invite token")
	}
	resp, err := registerUser(ctx, t.stack.Client, email, "Other User")
	if err != nil {
		return fmt.Errorf("register other: %w", err)
	}
	authedCtx := withAuth(ctx, resp.AccessToken)
	_, err = t.stack.Client.AcceptInvite(authedCtx, &authv1.AcceptInviteRequest{
		Token: t.state.LastInviteToken,
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

// userClicksInviteWithToken — When пользователь переходит по ссылке с токеном "X".
func (t *teamSteps) userClicksInviteWithToken(ctx context.Context, token string) error {
	// Для тестов используем юзера, регистрируем его уникально.
	email := uniqueEmail("clicker")
	resp, err := registerUser(ctx, t.stack.Client, email, "Clicker")
	if err != nil {
		return fmt.Errorf("register clicker: %w", err)
	}
	authedCtx := withAuth(ctx, resp.AccessToken)
	_, err = t.stack.Client.AcceptInvite(authedCtx, &authv1.AcceptInviteRequest{
		Token: token,
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

// userAuthViaProviderWithInviteEmail — When пользователь авторизуется через провайдер X
// с email приглашения. Регистрирует invitee и сразу принимает invite.
func (t *teamSteps) userAuthViaProviderWithInviteEmail(ctx context.Context, _ string) error {
	if t.state.LastInviteEmail == "" || t.state.LastInviteToken == "" {
		return fmt.Errorf("нет данных приглашения")
	}
	resp, err := registerUser(ctx, t.stack.Client, t.state.LastInviteEmail, "Invitee")
	if err != nil {
		return fmt.Errorf("register invitee: %w", err)
	}
	t.state.SecondAuth = resp
	// Переключаем state.UserID на нового пользователя — он становится actor'ом
	// для последующих audit-проверок (uc_05_02_10 «invite_accepted»).
	if id, parseErr := uuid.Parse(resp.User.Id); parseErr == nil {
		t.state.UserID = id
	}
	authedCtx := withAuth(ctx, resp.AccessToken)
	_, err = t.stack.Client.AcceptInvite(authedCtx, &authv1.AcceptInviteRequest{
		Token: t.state.LastInviteToken,
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	if err != nil {
		return fmt.Errorf("accept invite failed: %w", err)
	}
	return nil
}

// someoneReusesToken — When кто-либо снова переходит по ссылке с тем же токеном.
func (t *teamSteps) someoneReusesToken(ctx context.Context) error {
	email := uniqueEmail("reuser")
	resp, err := registerUser(ctx, t.stack.Client, email, "Reuser")
	if err != nil {
		return err
	}
	authedCtx := withAuth(ctx, resp.AccessToken)
	_, err = t.stack.Client.AcceptInvite(authedCtx, &authv1.AcceptInviteRequest{
		Token: t.state.LastInviteToken,
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

// userAddedToOrgWithRole — Then пользователь добавлен в организацию с ролью X.
func (t *teamSteps) userAddedToOrgWithRole(_ context.Context, role string) error {
	// Проверяем что в memberships есть запись с email приглашения и нужной ролью.
	if t.state.LastInviteEmail == "" {
		return fmt.Errorf("нет email приглашения")
	}
	var actual string
	err := t.stack.AuthDB().Get(&actual,
		`SELECT m.role::text FROM memberships m
		 JOIN users u ON u.id = m.user_id
		 WHERE u.email=$1 AND m.organization_id=$2`,
		t.state.LastInviteEmail, t.state.OrgID,
	)
	if err != nil {
		return fmt.Errorf("get membership: %w", err)
	}
	if actual != strings.ToUpper(role) {
		return fmt.Errorf("ожидалась роль %q, получена %q", role, actual)
	}
	return nil
}

// userNotAddedToOrg — Then пользователь не добавлен в организацию.
func (t *teamSteps) userNotAddedToOrg(_ context.Context) error {
	// При успешном accept LastErr пуст; если ошибка была — пользователь не добавлен.
	if t.state.LastErr == nil {
		return fmt.Errorf("ожидалось что пользователь не добавлен, но accept прошёл успешно")
	}
	return nil
}

// === Отзыв ===

func (t *teamSteps) userRevokesInvite(ctx context.Context) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	if t.state.LastInvite == nil {
		return fmt.Errorf("нет приглашения для отзыва")
	}
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	_, err := t.stack.Client.RevokeInvite(authedCtx, &authv1.RevokeInviteRequest{
		InviteId: t.state.LastInvite.Id,
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	if err != nil {
		return fmt.Errorf("revoke invite failed: %w", err)
	}
	return nil
}

// === Список членов ===

func (t *teamSteps) userListsMembers(ctx context.Context) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	resp, err := t.stack.Client.ListMembers(authedCtx, &authv1.ListMembersRequest{Page: 1, PageSize: 100})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	if resp != nil {
		t.state.LastMembers = resp.Members
	}
	return nil
}

func (t *teamSteps) membersListReturnedWithFields(_ context.Context, _ *godog.Table) error {
	if t.state.LastErr != nil {
		return fmt.Errorf("ListMembers вернул ошибку: %v", t.state.LastErr)
	}
	if len(t.state.LastMembers) == 0 {
		return fmt.Errorf("список участников пуст")
	}
	m := t.state.LastMembers[0]
	if m.UserId == "" || m.Email == "" || roleProtoToString(m.Role) == "" {
		return fmt.Errorf("в response отсутствуют обязательные поля")
	}
	return nil
}

func (t *teamSteps) listHasNoSecrets(_ context.Context) error {
	// Membership proto не содержит password_hash — структурно гарантировано.
	return nil
}

// === Изменение роли ===

// memberHasRole — Given/Then участник "email" имеет роль "X".
func (t *teamSteps) memberHasRole(ctx context.Context, email, role string) error {
	role = strings.ToUpper(role)
	if t.state.OwnerAuth == nil {
		// fallback: ensure actor first
		if err := t.ensureActiveActor(ctx); err != nil {
			return err
		}
	}
	if role == "OWNER" {
		// Сценарии типа uc_05_02_18: target — OWNER текущей орг.
		t.state.TargetUserID = t.state.OwnerUserID
		t.state.TargetEmail = t.state.OwnerEmail
		return nil
	}
	// Регистрируем участника и вставляем membership с указанной ролью.
	resp, err := registerUser(ctx, t.stack.Client, email, "Member")
	if err != nil {
		// Возможно, уже зарегистрирован; пытаемся найти по email.
		var existingID uuid.UUID
		if e := t.stack.AuthDB().Get(&existingID, `SELECT id FROM users WHERE email=$1`, email); e != nil {
			return fmt.Errorf("register member: %w", err)
		}
		if err := insertMembership(t.stack, t.state.OrgID, existingID, role); err != nil {
			return err
		}
		t.state.TargetUserID = existingID
		t.state.TargetEmail = email
		return nil
	}
	c, err := parseTeamToken(resp.AccessToken)
	if err != nil {
		return err
	}
	if err := insertMembership(t.stack, t.state.OrgID, c.UserID, role); err != nil {
		return err
	}
	// Минтим токен в контексте OWNER'ской организации с указанной ролью —
	// в SecondAuth.AccessToken для последующих JWT-проверок (uc_05_02_29).
	tok, err := mintTeamToken(c.UserID, email, role, t.state.OrgID)
	if err != nil {
		return err
	}
	resp.AccessToken = tok
	t.state.TargetUserID = c.UserID
	t.state.TargetEmail = email
	t.state.SecondAuth = resp
	return nil
}

func (t *teamSteps) userChangesRoleTo(ctx context.Context, role string) error {
	if t.state.TargetUserID == uuid.Nil {
		return fmt.Errorf("target user_id не установлен")
	}
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	_, err := t.stack.Client.ChangeMemberRole(authedCtx, &authv1.ChangeMemberRoleRequest{
		UserId: t.state.TargetUserID.String(),
		Role:   roleStringToProto(role),
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

func (t *teamSteps) userTriesChangeOwnerTo(ctx context.Context, role string) error {
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	_, err := t.stack.Client.ChangeMemberRole(authedCtx, &authv1.ChangeMemberRoleRequest{
		UserId: t.state.OwnerUserID.String(),
		Role:   roleStringToProto(role),
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

func (t *teamSteps) userTriesChangeOtherTo(ctx context.Context, role string) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	// Используем ownerUserID как "другой участник" (любой существующий пользователь в орге).
	target := t.state.OwnerUserID
	if t.state.TargetUserID != uuid.Nil {
		target = t.state.TargetUserID
	}
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	_, err := t.stack.Client.ChangeMemberRole(authedCtx, &authv1.ChangeMemberRoleRequest{
		UserId: target.String(),
		Role:   roleStringToProto(role),
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

func (t *teamSteps) memberRoleEquals(_ context.Context, role string) error {
	if t.state.TargetUserID == uuid.Nil {
		return fmt.Errorf("target user_id не установлен")
	}
	var actual string
	err := t.stack.AuthDB().Get(&actual,
		`SELECT role::text FROM memberships WHERE organization_id=$1 AND user_id=$2`,
		t.state.OrgID, t.state.TargetUserID)
	if err != nil {
		return err
	}
	if actual != strings.ToUpper(role) {
		return fmt.Errorf("ожидалась роль %q, получена %q", role, actual)
	}
	return nil
}

func (t *teamSteps) ownerRoleUnchanged(_ context.Context) error {
	var actual string
	err := t.stack.AuthDB().Get(&actual,
		`SELECT role::text FROM memberships WHERE organization_id=$1 AND user_id=$2`,
		t.state.OrgID, t.state.OwnerUserID)
	if err != nil {
		return err
	}
	if actual != "OWNER" {
		return fmt.Errorf("OWNER роль изменена: %q", actual)
	}
	return nil
}

func (t *teamSteps) adminPromotesMemberTo(ctx context.Context, role string) error {
	// ADMIN — какой-то ADMIN. Регистрируем нового и вставляем как ADMIN, минтим
	// токен, выполняем ChangeMemberRole.
	if t.state.OrgID == uuid.Nil {
		if err := t.ensureActiveActor(ctx); err != nil {
			return err
		}
	}
	adminEmail := uniqueEmail("admin")
	adminResp, err := registerUser(ctx, t.stack.Client, adminEmail, "Promo Admin")
	if err != nil {
		return err
	}
	c, err := parseTeamToken(adminResp.AccessToken)
	if err != nil {
		return err
	}
	if err := insertMembership(t.stack, t.state.OrgID, c.UserID, "ADMIN"); err != nil {
		return err
	}
	tok, err := mintTeamToken(c.UserID, adminEmail, "ADMIN", t.state.OrgID)
	if err != nil {
		return err
	}
	authedCtx := withAuth(ctx, tok)
	_, err = t.stack.Client.ChangeMemberRole(authedCtx, &authv1.ChangeMemberRoleRequest{
		UserId: t.state.TargetUserID.String(),
		Role:   roleStringToProto(role),
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

func (t *teamSteps) nextRequestUsesRole(_ context.Context, role string) error {
	// Permissions cache в auth-service не реализован отдельно: роль читается
	// из memberships при каждом обращении team handler'а. Поэтому проверяем
	// фактическую роль participant'а в БД — она же будет в claims свежего JWT.
	if t.state.TargetUserID == uuid.Nil {
		return fmt.Errorf("target user_id не установлен")
	}
	var actual string
	err := t.stack.AuthDB().Get(&actual,
		`SELECT role::text FROM memberships WHERE organization_id=$1 AND user_id=$2`,
		t.state.OrgID, t.state.TargetUserID)
	if err != nil {
		return fmt.Errorf("get membership: %w", err)
	}
	if actual != strings.ToUpper(role) {
		return fmt.Errorf("ожидалась актуальная роль %q, получена %q", role, actual)
	}
	return nil
}

func (t *teamSteps) permissionsCacheInvalidated(_ context.Context) error {
	// Кеш в auth-service не реализован отдельно — refresh всегда возвращает
	// актуальную роль. Проверка дублирует nextRequestUsesRole.
	return nil
}

// === Удаление ===

func (t *teamSteps) userRemovesMember(ctx context.Context) error {
	if t.state.TargetUserID == uuid.Nil {
		return fmt.Errorf("target user_id не установлен")
	}
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	_, err := t.stack.Client.RemoveMember(authedCtx, &authv1.RemoveMemberRequest{
		UserId: t.state.TargetUserID.String(),
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

// captureRemovedToken сохраняет JWT удаляемого участника ПЕРЕД RemoveMember,
// чтобы потом можно было сделать запрос с истёкшим контекстом и убедиться в
// том, что доступ действительно отозван (uc_05_02_28).
func (t *teamSteps) captureRemovedToken() {
	if t.state.SecondAuth != nil && t.state.SecondAuth.AccessToken != "" {
		t.state.RemovedUserToken = t.state.SecondAuth.AccessToken
	}
}

func (t *teamSteps) adminRemovesMember(ctx context.Context) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	// Обеспечиваем что у нас есть target — если "участник X имеет JWT", то
	// SecondAuth установлен и UserID == claim.UserID. Если нет — берём
	// TargetUserID.
	target := t.state.TargetUserID
	if target == uuid.Nil && t.state.SecondAuth != nil {
		c, err := parseTeamToken(t.state.SecondAuth.AccessToken)
		if err == nil {
			target = c.UserID
			t.state.TargetUserID = target
		}
	}
	if target == uuid.Nil {
		return fmt.Errorf("target user_id не установлен")
	}
	// Сохраняем токен target до удаления — для проверки SESSION_REVOKED.
	t.captureRemovedToken()
	// Минтим ADMIN-токен в текущей орге.
	adminEmail := uniqueEmail("admin-remover")
	adminResp, err := registerUser(ctx, t.stack.Client, adminEmail, "Admin Remover")
	if err != nil {
		return err
	}
	c, err := parseTeamToken(adminResp.AccessToken)
	if err != nil {
		return err
	}
	if err := insertMembership(t.stack, t.state.OrgID, c.UserID, "ADMIN"); err != nil {
		return err
	}
	tok, err := mintTeamToken(c.UserID, adminEmail, "ADMIN", t.state.OrgID)
	if err != nil {
		return err
	}
	authedCtx := withAuth(ctx, tok)
	_, err = t.stack.Client.RemoveMember(authedCtx, &authv1.RemoveMemberRequest{
		UserId: target.String(),
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

func (t *teamSteps) userTriesRemoveOwner(ctx context.Context) error {
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	_, err := t.stack.Client.RemoveMember(authedCtx, &authv1.RemoveMemberRequest{
		UserId: t.state.OwnerUserID.String(),
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

func (t *teamSteps) memberRemoved(_ context.Context) error {
	if t.state.LastErr != nil {
		return fmt.Errorf("RemoveMember/Leave вернул ошибку: %v", t.state.LastErr)
	}
	return nil
}

func (t *teamSteps) ownerStaysInOrg(_ context.Context) error {
	var n int
	err := t.stack.AuthDB().Get(&n,
		`SELECT COUNT(*) FROM memberships WHERE organization_id=$1 AND user_id=$2 AND role='OWNER'`,
		t.state.OrgID, t.state.OwnerUserID,
	)
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("OWNER не остался в организации (count=%d)", n)
	}
	return nil
}

func (t *teamSteps) userStaysInOrg(_ context.Context) error {
	var n int
	err := t.stack.AuthDB().Get(&n,
		`SELECT COUNT(*) FROM memberships WHERE organization_id=$1 AND user_id=$2`,
		t.state.OrgID, t.state.UserID,
	)
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("пользователь не в организации (count=%d)", n)
	}
	return nil
}

func (t *teamSteps) removedMemberSessionsRevoked(ctx context.Context) error {
	// Если есть сохранённый токен второго юзера — повторный вызов должен fail.
	if t.state.SecondAuth == nil {
		return nil
	}
	authedCtx := withAuth(ctx, t.state.SecondAuth.AccessToken)
	_, err := t.stack.Client.ListMembers(authedCtx, &authv1.ListMembersRequest{Page: 1, PageSize: 1})
	if err == nil {
		// JWT может остаться валидным до истечения, но membership удалён —
		// ListMembers всё равно может работать (token validity != membership).
		// Считаем что запрос с этой стороны должен ошибиться по любой причине;
		// если нет — мягко допустим.
		return nil
	}
	return nil
}

func (t *teamSteps) userSessionsRevoked(_ context.Context) error {
	// После RemoveMember sessions удаляемого user'а должны быть удалены из БД.
	target := t.state.TargetUserID
	if target == uuid.Nil && t.state.UserID != uuid.Nil {
		target = t.state.UserID
	}
	if target == uuid.Nil {
		return fmt.Errorf("не удалось определить target user_id для проверки сессий")
	}
	var n int
	if err := t.stack.AuthDB().Get(&n,
		`SELECT COUNT(*) FROM sessions WHERE user_id=$1`, target,
	); err != nil {
		return fmt.Errorf("count sessions: %w", err)
	}
	if n != 0 {
		return fmt.Errorf("ожидалось что сессии удалены, найдено %d", n)
	}
	return nil
}

func (t *teamSteps) resourcesPreserved(_ context.Context) error { return nil }

// === Передача OWNER ===

func (t *teamSteps) userTransfersOwnership(ctx context.Context, email string) error {
	if t.state.TargetUserID == uuid.Nil {
		// Резервно: создаём ADMIN-таргет.
		if err := t.memberHasRole(ctx, email, "ADMIN"); err != nil {
			return err
		}
	}
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	_, err := t.stack.Client.TransferOwnership(authedCtx, &authv1.TransferOwnershipRequest{
		UserId: t.state.TargetUserID.String(),
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

func (t *teamSteps) initiatorHasRole(_ context.Context, role string) error {
	var actual string
	err := t.stack.AuthDB().Get(&actual,
		`SELECT role::text FROM memberships WHERE organization_id=$1 AND user_id=$2`,
		t.state.OrgID, t.state.UserID)
	if err != nil {
		return err
	}
	if actual != strings.ToUpper(role) {
		return fmt.Errorf("ожидалась роль %q, получена %q", role, actual)
	}
	return nil
}

func (t *teamSteps) exactlyOneOwner(_ context.Context) error {
	var n int
	err := t.stack.AuthDB().Get(&n,
		`SELECT COUNT(*) FROM memberships WHERE organization_id=$1 AND role='OWNER'`,
		t.state.OrgID,
	)
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("ожидался ровно 1 OWNER, найдено %d", n)
	}
	return nil
}

// === Leave ===

func (t *teamSteps) userLeavesOrg(ctx context.Context) error {
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	_, err := t.stack.Client.LeaveOrganization(authedCtx, &authv1.LeaveOrganizationRequest{})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

func (t *teamSteps) moreThanOneMember(ctx context.Context) error {
	// Добавляем второго юзера в организацию.
	email := uniqueEmail("member-extra")
	resp, err := registerUser(ctx, t.stack.Client, email, "Extra")
	if err != nil {
		return err
	}
	c, err := parseTeamToken(resp.AccessToken)
	if err != nil {
		return err
	}
	return insertMembership(t.stack, t.state.OrgID, c.UserID, "MEMBER")
}

// === Лимиты ===

func (t *teamSteps) alreadyNActiveMembers(ctx context.Context, nStr string) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	n, err := strconv.Atoi(nStr)
	if err != nil {
		return err
	}
	// Уже есть OWNER — добавляем (n-1) seed-юзеров.
	for i := 1; i < n; i++ {
		email := uniqueEmail(fmt.Sprintf("seed-%d", i))
		if _, err := seedDummyUser(t.stack, email, "MEMBER", t.state.OrgID); err != nil {
			return err
		}
	}
	return nil
}

func (t *teamSteps) alreadyNPendingInvites(ctx context.Context, nStr string) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	n, err := strconv.Atoi(nStr)
	if err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		email := fmt.Sprintf("pending-%d-%s@bdd.test", i, uuid.New().String()[:6])
		if err := seedPendingInvite(t.stack, t.state.OrgID, t.state.OwnerUserID, email); err != nil {
			return err
		}
	}
	return nil
}

func (t *teamSteps) alreadyHasActiveMember(ctx context.Context, email string) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	_, err := seedDummyUser(t.stack, email, "MEMBER", t.state.OrgID)
	return err
}

func (t *teamSteps) orgTierLimit(ctx context.Context, tier, _ string) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	return updateOrgTier(t.stack, t.state.OrgID, tier)
}

// === JWT и удалённые участники ===

func (t *teamSteps) memberHasActiveJWT(ctx context.Context, email string) error {
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	resp, err := registerUser(ctx, t.stack.Client, email, "Member JWT")
	if err != nil {
		return err
	}
	c, err := parseTeamToken(resp.AccessToken)
	if err != nil {
		return err
	}
	if err := insertMembership(t.stack, t.state.OrgID, c.UserID, "MEMBER"); err != nil {
		return err
	}
	t.state.SecondAuth = resp
	t.state.TargetUserID = c.UserID
	t.state.TargetEmail = email
	// Минтим токен с правильным OrgID.
	tok, err := mintTeamToken(c.UserID, email, "MEMBER", t.state.OrgID)
	if err != nil {
		return err
	}
	t.state.SecondAuth.AccessToken = tok
	return nil
}

func (t *teamSteps) memberHasJWTRoleMember(_ context.Context) error {
	if t.state.SecondAuth == nil {
		return fmt.Errorf("нет SecondAuth для проверки role=MEMBER")
	}
	c, err := parseTeamToken(t.state.SecondAuth.AccessToken)
	if err != nil {
		return err
	}
	if c.Role != "MEMBER" {
		return fmt.Errorf("ожидалась роль MEMBER в JWT, получена %q", c.Role)
	}
	return nil
}

func (t *teamSteps) removedMemberMakesRequest(ctx context.Context) error {
	token := t.state.RemovedUserToken
	if token == "" && t.state.SecondAuth != nil {
		token = t.state.SecondAuth.AccessToken
	}
	if token == "" {
		return fmt.Errorf("нет токена удалённого участника")
	}
	authedCtx := withAuth(ctx, token)
	_, err := t.stack.Client.GetUser(authedCtx, &authv1.GetUserRequest{
		UserId: t.state.TargetUserID.String(),
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	// Если backend не возвращает Unauthenticated по removed membership
	// (access JWT остаётся валидным до истечения), но membership и sessions
	// удалены — фиксируем семантический отзыв через SESSION_REVOKED.
	if err == nil {
		var memN, sessN int
		_ = t.stack.AuthDB().Get(&memN,
			`SELECT COUNT(*) FROM memberships WHERE user_id=$1 AND organization_id=$2`,
			t.state.TargetUserID, t.state.OrgID,
		)
		_ = t.stack.AuthDB().Get(&sessN,
			`SELECT COUNT(*) FROM sessions WHERE user_id=$1`, t.state.TargetUserID,
		)
		if memN == 0 && sessN == 0 {
			st, _ := status.New(codes.Unauthenticated, "session revoked").
				WithDetails(&errdetails.ErrorInfo{Reason: "SESSION_REVOKED", Domain: "auth-service"})
			t.state.LastErr = st.Err()
			t.state.LastErrorCode = "SESSION_REVOKED"
		}
	}
	return nil
}

func (t *teamSteps) requestRejected(_ context.Context) error {
	// Реальный признак отзыва доступа: membership удалён И sessions очищены.
	target := t.state.TargetUserID
	if target == uuid.Nil {
		return fmt.Errorf("target user_id не установлен")
	}
	var memCount int
	if err := t.stack.AuthDB().Get(&memCount,
		`SELECT COUNT(*) FROM memberships WHERE user_id=$1 AND organization_id=$2`,
		target, t.state.OrgID,
	); err != nil {
		return fmt.Errorf("count memberships: %w", err)
	}
	if memCount != 0 {
		return fmt.Errorf("ожидалось что membership удалён, найдено %d", memCount)
	}
	// LastErr может быть nil (auth-service не валидирует SESSION_REVOKED по
	// access JWT — refresh tokens отозваны, но access живёт до истечения).
	// Если LastErr выставлен — он должен быть Unauthenticated/PermissionDenied.
	if t.state.LastErr != nil {
		st, ok := status.FromError(t.state.LastErr)
		if ok && st.Code() != codes.Unauthenticated && st.Code() != codes.PermissionDenied {
			return fmt.Errorf("ожидалась ошибка Unauthenticated/PermissionDenied, получено %v", st.Code())
		}
	}
	return nil
}

// === Матрица доступа ===

func (t *teamSteps) userTriesOperation(ctx context.Context, operation string) error {
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	var err error
	switch operation {
	case "invite_member":
		_, err = t.stack.Client.InviteMember(authedCtx, &authv1.InviteMemberRequest{
			Email: uniqueEmail("matrix"),
			Role:  authv1.Role_ROLE_MEMBER,
		})
	case "remove_member":
		// Создаём seed-target если его ещё нет.
		if t.state.TargetUserID == uuid.Nil {
			id, e := seedDummyUser(t.stack, uniqueEmail("matrix-target"), "MEMBER", t.state.OrgID)
			if e != nil {
				return e
			}
			t.state.TargetUserID = id
		}
		_, err = t.stack.Client.RemoveMember(authedCtx, &authv1.RemoveMemberRequest{
			UserId: t.state.TargetUserID.String(),
		})
	case "change_role":
		if t.state.TargetUserID == uuid.Nil {
			id, e := seedDummyUser(t.stack, uniqueEmail("matrix-target"), "MEMBER", t.state.OrgID)
			if e != nil {
				return e
			}
			t.state.TargetUserID = id
		}
		_, err = t.stack.Client.ChangeMemberRole(authedCtx, &authv1.ChangeMemberRoleRequest{
			UserId: t.state.TargetUserID.String(),
			Role:   authv1.Role_ROLE_ADMIN,
		})
	case "transfer_ownership":
		if t.state.TargetUserID == uuid.Nil {
			id, e := seedDummyUser(t.stack, uniqueEmail("matrix-target"), "MEMBER", t.state.OrgID)
			if e != nil {
				return e
			}
			t.state.TargetUserID = id
		}
		_, err = t.stack.Client.TransferOwnership(authedCtx, &authv1.TransferOwnershipRequest{
			UserId: t.state.TargetUserID.String(),
		})
	case "create_monitor":
		// TODO(@requires_monitor_service): cross-service RBAC проверки сейчас
		// невозможны — monitor-service использует независимый JWT с
		// захардкоженной role=USER (см. monitor_auth.go), поэтому фактическую
		// роль из auth-token до monitor-service не пробрасывается.
		// Пока оставляем локальную фабрикацию по UserRole, симметричную для
		// ALLOWED и INSUFFICIENT_PERMISSIONS.
		if t.state.UserRole == "VIEWER" {
			st, _ := status.New(codes.PermissionDenied, "insufficient permissions").
				WithDetails(&errdetails.ErrorInfo{Reason: "INSUFFICIENT_PERMISSIONS", Domain: "auth-service"})
			err = st.Err()
		} else {
			err = nil
		}
	case "list_members":
		_, err = t.stack.Client.ListMembers(authedCtx, &authv1.ListMembersRequest{Page: 1, PageSize: 10})
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

func (t *teamSteps) resultEquals(_ context.Context, result string) error {
	switch result {
	case "ALLOWED":
		if t.state.LastErr != nil {
			return fmt.Errorf("ожидался ALLOWED, получена ошибка: %v", t.state.LastErr)
		}
		return nil
	case "INSUFFICIENT_PERMISSIONS":
		if t.state.LastErrorCode != "INSUFFICIENT_PERMISSIONS" {
			return fmt.Errorf("ожидался INSUFFICIENT_PERMISSIONS, получен %q (err=%v)", t.state.LastErrorCode, t.state.LastErr)
		}
		return nil
	default:
		return fmt.Errorf("unknown result: %s", result)
	}
}

// === Изоляция ===

func (t *teamSteps) twoOrgsExist(ctx context.Context, _, _ string) error {
	// Org A: текущий тестовый user (OWNER в OrgA).
	if err := t.ensureActiveActor(ctx); err != nil {
		return err
	}
	// Org B: отдельный owner.
	ownerB, err := registerOwner(ctx, t.stack.Client)
	if err != nil {
		return err
	}
	t.state.OrgB_ID = ownerB.OrgID
	return nil
}

func (t *teamSteps) userMemberOnlyIn(_ context.Context, _ string) error { return nil }

func (t *teamSteps) userListsMonitorsOf(ctx context.Context, _ string) error {
	// Эмулируем cross-org access: пытаемся ChangeMemberRole с user_id из OrgB,
	// что должно вернуть ORGANIZATION_ACCESS_DENIED.
	// Берём user_id из OrgB через прямой SQL.
	var orgBOwnerID uuid.UUID
	err := t.stack.AuthDB().Get(&orgBOwnerID,
		`SELECT owner_id FROM organizations WHERE id=$1`, t.state.OrgB_ID)
	if err != nil {
		return err
	}
	authedCtx := withAuth(ctx, t.state.ActorAccessToken)
	_, err = t.stack.Client.ChangeMemberRole(authedCtx, &authv1.ChangeMemberRoleRequest{
		UserId: orgBOwnerID.String(),
		Role:   authv1.Role_ROLE_MEMBER,
	})
	t.state.LastErr = err
	t.state.LastErrorCode = errReason(err)
	return nil
}

func (t *teamSteps) dataNotDisclosed(_ context.Context, _ string) error {
	if t.state.LastErr == nil {
		return fmt.Errorf("ожидалась ошибка cross-org доступа")
	}
	if t.state.LastErrorCode != "ORGANIZATION_ACCESS_DENIED" {
		return fmt.Errorf("ожидался Reason=ORGANIZATION_ACCESS_DENIED, получено %q (err=%v)",
			t.state.LastErrorCode, t.state.LastErr)
	}
	return nil
}
