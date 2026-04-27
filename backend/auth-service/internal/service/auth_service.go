package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"

	"github.com/raul/monitor/backend/auth-service/internal/infrastructure/oauth"
	"github.com/raul/monitor/backend/auth-service/internal/model"
	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
	"github.com/raul/monitor/backend/auth-service/pkg/errors"
)

// Локальные частично применяемые интерфейсы — только нужные методы.

type userRepo interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
	IncrementLoginAttempts(ctx context.Context, userID uuid.UUID) error
	ResetLoginAttempts(ctx context.Context, userID uuid.UUID) error
	LockAccount(ctx context.Context, userID uuid.UUID, lockedUntil any) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type oauthRepo interface {
	Create(ctx context.Context, account *model.OAuthAccount) (*model.OAuthAccount, error)
	GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*model.OAuthAccount, error)
}

type refreshTokenRepo interface {
	Create(ctx context.Context, token *model.RefreshToken) (*model.RefreshToken, error)
}

type sessionCreator interface {
	Create(ctx context.Context, session *model.Session) error
}

type auditCreator interface {
	Create(ctx context.Context, log *model.AuditLog) (*model.AuditLog, error)
}

type tokenGenerator interface {
	GenerateTokenPair(ctx context.Context, userID uuid.UUID, email, tier string) (string, string, error)
}

// teamTokenGenerator опционально расширяет tokenGenerator поддержкой team
// context (org_id + role в JWT claims). Реализуется jwt.TokenService.
type teamTokenGenerator interface {
	GenerateTokenPairWithTeam(ctx context.Context, userID uuid.UUID, email, tier string, orgID uuid.UUID, role string) (string, string, error)
}

// teamEnsurer описывает методы TeamService, нужные AuthService для
// автосоздания организации после регистрации/OAuth (uc_05_02_01).
type teamEnsurer interface {
	EnsureOrganization(ctx context.Context, ownerUserID uuid.UUID) (*model.Organization, bool, error)
	GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*model.Membership, error)
}

// OAuthProvider — публичный интерфейс для OAuth-провайдеров; позволяет инжектировать
// фейковые провайдеры из тестового пакета без нарушения инкапсуляции.
type OAuthProvider = oauth.Provider

type eventPublisher interface {
	PublishUserCreated(ctx context.Context, userID, email string) error
	PublishUserLoggedIn(ctx context.Context, userID, email, ip, userAgent string) error
	PublishUserFailedLogin(ctx context.Context, userID, email, ip, reason string) error
	PublishUserLocked(ctx context.Context, userID, email string, attempts int) error
	PublishOAuthAccountLinked(ctx context.Context, userID, email, provider, providerID string) error
	PublishSessionCreated(ctx context.Context, userID, email, sessionID string, expiresAt int64) error
}

// StateStore — интерфейс для хранения OAuth-состояния при защите от CSRF.
type StateStore interface {
	Set(ctx context.Context, key string, value any) error
	Get(ctx context.Context, key string) (any, error)
	Delete(ctx context.Context, key string) error
}

// noopPublisher реализует eventPublisher без побочных эффектов.
type noopPublisher struct{}

func (p *noopPublisher) PublishUserCreated(_ context.Context, _, _ string) error        { return nil }
func (p *noopPublisher) PublishUserLoggedIn(_ context.Context, _, _, _, _ string) error { return nil }
func (p *noopPublisher) PublishUserFailedLogin(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (p *noopPublisher) PublishUserLocked(_ context.Context, _, _ string, _ int) error { return nil }
func (p *noopPublisher) PublishOAuthAccountLinked(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (p *noopPublisher) PublishSessionCreated(_ context.Context, _, _, _ string, _ int64) error {
	return nil
}

// AuthService обрабатывает операции аутентификации.
type AuthService struct {
	userRepo         userRepo
	oauthRepo        oauthRepo
	refreshTokenRepo refreshTokenRepo
	sessionRepo      sessionCreator
	auditRepo        auditCreator
	tokenService     tokenGenerator
	teamTokens       teamTokenGenerator
	teamSvc          teamEnsurer
	providers        map[string]OAuthProvider
	stateStore       StateStore
	publisher        eventPublisher
	maxAttempts      int
	lockDuration     time.Duration
	tracer           trace.Tracer
	logger           *slog.Logger
}

// NewAuthService создаёт новый AuthService.
func NewAuthService(
	userRepo userRepo,
	oauthRepo oauthRepo,
	refreshTokenRepo refreshTokenRepo,
	sessionRepo sessionCreator,
	auditRepo auditCreator,
	tokenService tokenGenerator,
	providers map[string]OAuthProvider,
	stateStore StateStore,
	maxAttempts int,
	lockDuration time.Duration,
	tracer trace.Tracer,
	opts ...AuthServiceOption,
) *AuthService {
	s := &AuthService{
		userRepo:         userRepo,
		oauthRepo:        oauthRepo,
		refreshTokenRepo: refreshTokenRepo,
		sessionRepo:      sessionRepo,
		auditRepo:        auditRepo,
		tokenService:     tokenService,
		providers:        providers,
		stateStore:       stateStore,
		publisher:        &noopPublisher{}, // по умолчанию no-op
		maxAttempts:      maxAttempts,
		lockDuration:     lockDuration,
		tracer:           tracer,
		logger:           slog.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// AuthServiceOption задаёт опциональные параметры для AuthService.
type AuthServiceOption func(*AuthService)

// WithPublisher устанавливает публикатор событий.
func WithPublisher(p eventPublisher) AuthServiceOption {
	return func(s *AuthService) {
		s.publisher = p
	}
}

// WithTeamEnsurer подключает TeamService для авто-создания организации
// после регистрации/OAuth (uc_05_02_01) и обогащения JWT team-context'ом.
func WithTeamEnsurer(t teamEnsurer) AuthServiceOption {
	return func(s *AuthService) {
		s.teamSvc = t
	}
}

// WithTeamTokenGenerator подключает реализацию JWT, поддерживающую
// team-context в claims.
func WithTeamTokenGenerator(g teamTokenGenerator) AuthServiceOption {
	return func(s *AuthService) {
		s.teamTokens = g
	}
}

// WithAuthLogger подключает logger в AuthService.
func WithAuthLogger(l *slog.Logger) AuthServiceOption {
	return func(s *AuthService) {
		s.logger = l
	}
}

// ensureTeamContext идемпотентно создаёт организацию пользователя при
// первом успешном auth и возвращает orgID + роль для прокидывания в JWT.
// Возвращает (uuid.Nil, "") если teamSvc не подключён или операция упала —
// в этом случае токен будет выпущен без team-context (обратно совместимо).
func (s *AuthService) ensureTeamContext(ctx context.Context, userID uuid.UUID) (uuid.UUID, string) {
	if s.teamSvc == nil {
		return uuid.Nil, ""
	}
	org, created, err := s.teamSvc.EnsureOrganization(ctx, userID)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to ensure organization", "user_id", userID, "error", err)
		return uuid.Nil, ""
	}
	if org == nil {
		return uuid.Nil, ""
	}
	if created {
		s.logger.InfoContext(ctx, "Auto-created organization on registration",
			"user_id", userID, "org_id", org.ID)
		// Аудит-запись organization_created (uc_05_02_01).
		uid := userID
		log := model.NewAuditLog(&uid, model.EventTypeOrganizationCreated, "", true, "", "", "")
		if _, err := s.auditRepo.Create(ctx, log); err != nil {
			s.logger.WarnContext(ctx, "Failed to write organization_created audit log",
				"user_id", userID, "error", err)
		}
	}
	role := string(model.RoleOwner)
	if m, err := s.teamSvc.GetMembership(ctx, org.ID, userID); err == nil && m != nil {
		role = string(m.Role)
	}
	return org.ID, role
}

// generateTokens вызывает team-aware генератор, если он подключён и есть
// orgID; иначе fallback на базовый GenerateTokenPair без team-context.
func (s *AuthService) generateTokens(ctx context.Context, userID uuid.UUID, email, tier string, orgID uuid.UUID, role string) (string, string, error) {
	if s.teamTokens != nil && orgID != uuid.Nil {
		return s.teamTokens.GenerateTokenPairWithTeam(ctx, userID, email, tier, orgID, role)
	}
	return s.tokenService.GenerateTokenPair(ctx, userID, email, tier)
}

// PasswordLogin аутентифицирует пользователя по email и паролю.
func (s *AuthService) PasswordLogin(ctx context.Context, req *dto.PasswordLoginRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.PasswordLogin")
	defer span.End()

	span.SetAttributes(
		attribute.String("auth.ip_address", ipAddress),
		attribute.String("auth.user_agent", userAgent),
	)

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		s.logAuditEvent(ctx, nil, model.EventTypeLogin, "", false, ipAddress, userAgent, "user not found")
		span.AddEvent("user_not_found")
		return nil, errors.InvalidCredentials()
	}

	span.SetAttributes(attribute.String("user.id", user.ID.String()))

	// Check if account is locked
	if user.IsLocked() {
		s.logAuditEvent(ctx, &user.ID, model.EventTypeLogin, "", false, ipAddress, userAgent, "account locked")
		span.AddEvent("account_locked", trace.WithAttributes(attribute.String("user_id", user.ID.String())))
		return nil, errors.AccountLocked()
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Increment failed login attempts
		if incrementErr := s.userRepo.IncrementLoginAttempts(ctx, user.ID); incrementErr != nil {
			span.RecordError(incrementErr)
		}

		// Check if should lock account
		if user.ShouldLock(s.maxAttempts) {
			lockedUntil := time.Now().Add(s.lockDuration)
			if lockErr := s.userRepo.LockAccount(ctx, user.ID, lockedUntil); lockErr != nil {
				span.RecordError(lockErr)
			}
			s.logAuditEvent(ctx, &user.ID, model.EventTypeLogin, "", false, ipAddress, userAgent, "account locked due to too many failed attempts")
			if publishErr := s.publisher.PublishUserLocked(ctx, user.ID.String(), user.Email, s.maxAttempts); publishErr != nil {
				span.RecordError(publishErr)
			}
			span.AddEvent("account_locked_too_many_attempts", trace.WithAttributes(attribute.String("user_id", user.ID.String())))
			return nil, errors.AccountLocked()
		}

		s.logAuditEvent(ctx, &user.ID, model.EventTypeLogin, "", false, ipAddress, userAgent, "invalid password")
		if publishErr := s.publisher.PublishUserFailedLogin(ctx, user.ID.String(), user.Email, ipAddress, "invalid password"); publishErr != nil {
			span.RecordError(publishErr)
		}
		span.AddEvent("invalid_password", trace.WithAttributes(attribute.String("user_id", user.ID.String())))
		return nil, errors.InvalidCredentials()
	}

	// Password correct, reset login attempts
	if err := s.userRepo.ResetLoginAttempts(ctx, user.ID); err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, &user.ID, model.EventTypeLogin, "", true, ipAddress, userAgent, "failed to reset login attempts")
		return nil, errors.InternalError(err)
	}

	// Update last login
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, &user.ID, model.EventTypeLogin, "", true, ipAddress, userAgent, "failed to update last login")
		return nil, errors.InternalError(err)
	}

	// Ensure organization (uc_05_02_01) and gather team context for JWT.
	orgID, role := s.ensureTeamContext(ctx, user.ID)

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokens(ctx, user.ID, user.Email, user.Tier, orgID, role)
	if err != nil {
		span.RecordError(err)
		err := errors.InternalError(err)
		s.logAuditEvent(ctx, &user.ID, model.EventTypeLogin, "", false, ipAddress, userAgent, err.Error())
		return nil, err
	}

	// Store refresh token
	refreshTokenHash := model.HashToken(refreshToken)
	refreshTokenEntity := model.NewRefreshToken(user.ID, refreshTokenHash, "password", ipAddress, 7*24*time.Hour)
	if _, err := s.refreshTokenRepo.Create(ctx, refreshTokenEntity); err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, &user.ID, model.EventTypeLogin, "", true, ipAddress, userAgent, "failed to store refresh token")
		return nil, errors.InternalError(err)
	}

	// Create session
	session := model.NewSession(user.ID, "password", ipAddress, 24*time.Hour)
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, &user.ID, model.EventTypeLogin, "", true, ipAddress, userAgent, "failed to create session")
		return nil, errors.InternalError(err)
	}

	// Log audit event
	s.logAuditEvent(ctx, &user.ID, model.EventTypeLogin, "", true, ipAddress, userAgent, "")

	span.AddEvent("login_success", trace.WithAttributes(attribute.String("session_id", session.ID)))

	// Publish events
	if publishErr := s.publisher.PublishUserLoggedIn(ctx, user.ID.String(), user.Email, ipAddress, userAgent); publishErr != nil {
		span.RecordError(publishErr)
	}
	if publishErr := s.publisher.PublishSessionCreated(ctx, user.ID.String(), user.Email, session.ID, time.Now().Add(24*time.Hour).Unix()); publishErr != nil {
		span.RecordError(publishErr)
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes
		User:         dto.ToUserDTO(user),
	}, nil
}

// OAuthLogin инициирует OAuth-авторизацию.
func (s *AuthService) OAuthLogin(ctx context.Context, provider, redirectURI string) (string, string, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.OAuthLogin")
	defer span.End()

	span.SetAttributes(attribute.String("auth.provider", provider))

	p, ok := s.providers[provider]
	if !ok {
		span.AddEvent("unsupported_provider", trace.WithAttributes(attribute.String("provider", provider)))
		return "", "", fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	// Generate state for CSRF protection
	state := uuid.New().String()
	stateKey := fmt.Sprintf("oauth_state:%s", state)

	// Store state with user info (redirectURI)
	stateData := map[string]string{
		"provider":     provider,
		"redirect_uri": redirectURI,
	}

	if err := s.stateStore.Set(ctx, stateKey, stateData); err != nil {
		span.RecordError(err)
		return "", "", fmt.Errorf("failed to store OAuth state: %v", err)
	}

	// Generate OAuth URL
	oauthURL := p.GetAuthURL(state, redirectURI)

	span.AddEvent("oauth_url_generated", trace.WithAttributes(attribute.String("state", state)))

	return oauthURL, state, nil
}

// OAuthCallback обрабатывает OAuth-callback и возвращает ответ с токенами.
func (s *AuthService) OAuthCallback(ctx context.Context, req *dto.OAuthCallbackRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.OAuthCallback")
	defer span.End()

	span.SetAttributes(
		attribute.String("auth.ip_address", ipAddress),
		attribute.String("auth.user_agent", userAgent),
	)

	// Verify OAuth state
	provider, redirectURI, err := s.verifyState(ctx, req.State)
	if err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, nil, model.EventTypeOAuthCallback, provider, false, ipAddress, userAgent, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.String("auth.provider", provider))

	// Get OAuth provider
	p, ok := s.providers[provider]
	if !ok {
		err := errors.OAuthFailed(fmt.Errorf("unsupported provider: %s", provider))
		span.AddEvent("unsupported_provider", trace.WithAttributes(attribute.String("provider", provider)))
		s.logAuditEvent(ctx, nil, model.EventTypeOAuthCallback, provider, false, ipAddress, userAgent, err.Error())
		return nil, err
	}

	// Exchange code for user info
	userInfo, err := p.Exchange(ctx, req.Code, redirectURI)
	if err != nil {
		span.RecordError(err)
		err := errors.OAuthFailed(err)
		s.logAuditEvent(ctx, nil, model.EventTypeOAuthCallback, provider, false, ipAddress, userAgent, err.Error())
		return nil, err
	}

	// Find existing OAuth account
	providerUserID := userInfo.ProviderUserID
	oauthAccount, err := s.oauthRepo.GetByProviderUserID(ctx, provider, providerUserID)

	var user *model.User

	if err != nil {
		// OAuth account not found, check if user with email exists
		existingUser, err := s.userRepo.GetByEmail(ctx, userInfo.Email)
		if err == nil {
			// User exists, link OAuth account
			user = existingUser
			newAccount := model.NewOAuthAccount(user.ID, provider, providerUserID, userInfo.Email, map[string]any{
				"picture": userInfo.Picture,
				"name":    userInfo.Name,
			})
			if _, err := s.oauthRepo.Create(ctx, newAccount); err != nil {
				span.RecordError(err)
				err := errors.InternalError(err)
				s.logAuditEvent(ctx, &user.ID, model.EventTypeOAuthCallback, provider, false, ipAddress, userAgent, err.Error())
				return nil, err
			}
			span.AddEvent("oauth_account_linked", trace.WithAttributes(attribute.String("user_id", user.ID.String())))
			if publishErr := s.publisher.PublishOAuthAccountLinked(ctx, user.ID.String(), user.Email, provider, providerUserID); publishErr != nil {
				span.RecordError(publishErr)
			}
		} else {
			// Create new user
			user, err = s.createUser(ctx, userInfo, provider, providerUserID)
			if err != nil {
				span.RecordError(err)
				err := errors.InternalError(err)
				s.logAuditEvent(ctx, nil, model.EventTypeOAuthCallback, provider, false, ipAddress, userAgent, err.Error())
				return nil, err
			}
			span.AddEvent("user_created_via_oauth")
			if publishErr := s.publisher.PublishUserCreated(ctx, user.ID.String(), user.Email); publishErr != nil {
				span.RecordError(publishErr)
			}
		}
	} else {
		// OAuth account exists, get user
		user, err = s.userRepo.GetByID(ctx, oauthAccount.UserID)
		if err != nil {
			span.RecordError(err)
			err := errors.InternalError(err)
			s.logAuditEvent(ctx, nil, model.EventTypeOAuthCallback, provider, false, ipAddress, userAgent, err.Error())
			return nil, err
		}
	}

	span.SetAttributes(attribute.String("user.id", user.ID.String()))

	// Check if account is locked
	if user.IsLocked() {
		err := errors.AccountLocked()
		span.AddEvent("account_locked", trace.WithAttributes(attribute.String("user_id", user.ID.String())))
		s.logAuditEvent(ctx, &user.ID, model.EventTypeOAuthCallback, provider, false, ipAddress, userAgent, err.Error())
		return nil, err
	}

	// Update last login
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, &user.ID, model.EventTypeOAuthCallback, provider, true, ipAddress, userAgent, "failed to update last login")
		return nil, errors.InternalError(err)
	}

	// Ensure organization (uc_05_02_01) — idempotent for existing users.
	orgID, role := s.ensureTeamContext(ctx, user.ID)

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokens(ctx, user.ID, user.Email, user.Tier, orgID, role)
	if err != nil {
		span.RecordError(err)
		err := errors.InternalError(err)
		s.logAuditEvent(ctx, &user.ID, model.EventTypeOAuthCallback, provider, false, ipAddress, userAgent, err.Error())
		return nil, err
	}

	// Store refresh token
	refreshTokenHash := model.HashToken(refreshToken)
	refreshTokenEntity := model.NewRefreshToken(user.ID, refreshTokenHash, "oauth", ipAddress, 7*24*time.Hour)
	if _, err := s.refreshTokenRepo.Create(ctx, refreshTokenEntity); err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, &user.ID, model.EventTypeOAuthCallback, provider, true, ipAddress, userAgent, "failed to store refresh token")
		return nil, errors.InternalError(err)
	}

	// Create session
	session := model.NewSession(user.ID, "oauth", ipAddress, 24*time.Hour)
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		span.RecordError(err)
		s.logAuditEvent(ctx, &user.ID, model.EventTypeOAuthCallback, provider, true, ipAddress, userAgent, "failed to create session")
		return nil, errors.InternalError(err)
	}

	// Log audit event
	s.logAuditEvent(ctx, &user.ID, model.EventTypeOAuthCallback, provider, true, ipAddress, userAgent, "")

	span.AddEvent("oauth_login_success", trace.WithAttributes(attribute.String("session_id", session.ID)))

	// Publish session created event
	if publishErr := s.publisher.PublishSessionCreated(ctx, user.ID.String(), user.Email, session.ID, time.Now().Add(24*time.Hour).Unix()); publishErr != nil {
		span.RecordError(publishErr)
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes
		User:         dto.ToUserDTO(user),
	}, nil
}

// Register создаёт новую учётную запись пользователя.
func (s *AuthService) Register(ctx context.Context, req *dto.RegisterRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.Register")
	defer span.End()

	span.SetAttributes(
		attribute.String("auth.ip_address", ipAddress),
		attribute.String("auth.user_agent", userAgent),
	)

	// Check if user already exists
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		span.RecordError(err)
		return nil, errors.InternalError(err)
	}
	if exists {
		span.AddEvent("user_already_exists")
		return nil, errors.UserExists()
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		span.RecordError(err)
		return nil, errors.InternalError(fmt.Errorf("failed to hash password: %v", err))
	}

	// Create user
	user := model.NewUser(req.Email, string(passwordHash), req.FullName, "Free")
	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		span.RecordError(err)
		return nil, errors.InternalError(fmt.Errorf("failed to create user: %v", err))
	}

	span.SetAttributes(attribute.String("user.id", createdUser.ID.String()))
	span.AddEvent("user_registered")

	// Log audit event
	log := model.NewAuditLog(&createdUser.ID, model.EventTypeRegister, "", true, ipAddress, userAgent, "")
	if _, err := s.auditRepo.Create(ctx, log); err != nil {
		return nil, errors.InternalError(err)
	}

	// Publish user created event
	if publishErr := s.publisher.PublishUserCreated(ctx, createdUser.ID.String(), createdUser.Email); publishErr != nil {
		span.RecordError(publishErr)
	}

	// Auto-create organization (uc_05_02_01).
	s.ensureTeamContext(ctx, createdUser.ID)

	return &dto.AuthResponse{
		User: dto.ToUserDTO(createdUser),
	}, nil
}

// verifyState verifies the OAuth state
func (s *AuthService) verifyState(ctx context.Context, state string) (string, string, error) {
	stateKey := fmt.Sprintf("oauth_state:%s", state)

	data, err := s.stateStore.Get(ctx, stateKey)
	if err != nil {
		return "", "", errors.OAuthStateMismatch()
	}

	// Delete state after verification
	if err := s.stateStore.Delete(ctx, stateKey); err != nil {
		return "", "", errors.OAuthStateMismatch()
	}

	stateData, ok := data.(map[string]string)
	if !ok {
		return "", "", errors.OAuthStateMismatch()
	}

	return stateData["provider"], stateData["redirect_uri"], nil
}

// createUser creates a new user from OAuth info
func (s *AuthService) createUser(ctx context.Context, userInfo *oauth.UserInfo, provider, providerUserID string) (*model.User, error) {
	// Generate random password for OAuth users (user won't use it)
	password := uuid.New().String()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password hash: %v", err)
	}

	user := model.NewUser(userInfo.Email, string(passwordHash), userInfo.Name, "Free")

	// Create user
	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	// Create OAuth account
	oauthAccount := model.NewOAuthAccount(user.ID, provider, providerUserID, userInfo.Email, map[string]any{
		"picture": userInfo.Picture,
		"name":    userInfo.Name,
	})
	if _, err := s.oauthRepo.Create(ctx, oauthAccount); err != nil {
		return nil, fmt.Errorf("failed to create OAuth account: %v", err)
	}

	return createdUser, nil
}

// logAuditEvent logs an audit event
func (s *AuthService) logAuditEvent(ctx context.Context, userID *uuid.UUID, eventType, provider string, success bool, ipAddress, userAgent, errorMessage string) {
	log := model.NewAuditLog(userID, eventType, provider, success, ipAddress, userAgent, errorMessage)
	if _, err := s.auditRepo.Create(ctx, log); err != nil {
		return
	}
}
