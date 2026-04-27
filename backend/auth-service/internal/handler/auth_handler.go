package handler

import (
	"context"
	"net"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/raul/monitor/backend/auth-service/internal/service/dto"
	"github.com/raul/monitor/backend/auth-service/pkg/errors"
)

// AuthHandler implements gRPC AuthService
type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	authService  authServicer
	tokenService tokenServicer
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService authServicer, tokenService tokenServicer) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		tokenService: tokenService,
	}
}

// GoogleOAuthLogin implements Google OAuth login (uc_05_01_01)
func (h *AuthHandler) GoogleOAuthLogin(ctx context.Context, req *authv1.OAuthLoginRequest) (*authv1.OAuthLoginResponse, error) {
	oauthURL, state, err := h.authService.OAuthLogin(ctx, "google", req.RedirectUri)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.OAuthLoginResponse{
		AuthUrl: oauthURL,
		State:   state,
	}, nil
}

// GoogleOAuthCallback implements Google OAuth callback (uc_05_01_02)
func (h *AuthHandler) GoogleOAuthCallback(ctx context.Context, req *authv1.OAuthCallbackRequest) (*authv1.AuthResponse, error) {
	dtoReq := &dto.OAuthCallbackRequest{
		Code:        req.Code,
		State:       req.State,
		RedirectURI: req.RedirectUri,
	}

	ipAddress := getIPAddress(ctx)
	userAgent := getUserAgent(ctx)

	resp, err := h.authService.OAuthCallback(ctx, dtoReq, ipAddress, userAgent)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
		User: &authv1.User{
			Id:          resp.User.ID.String(),
			Email:       resp.User.Email,
			FullName:    resp.User.FullName,
			Tier:        resp.User.Tier,
			CreatedAt:   timestampProto(resp.User.CreatedAt),
			LastLoginAt: timestampProtoPtr(resp.User.LastLoginAt),
		},
	}, nil
}

// PasswordLogin implements password login (uc_05_01_03)
func (h *AuthHandler) PasswordLogin(ctx context.Context, req *authv1.PasswordLoginRequest) (*authv1.AuthResponse, error) {
	dtoReq := &dto.PasswordLoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	ipAddress := getIPAddress(ctx)
	userAgent := getUserAgent(ctx)

	resp, err := h.authService.PasswordLogin(ctx, dtoReq, ipAddress, userAgent)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
		User: &authv1.User{
			Id:          resp.User.ID.String(),
			Email:       resp.User.Email,
			FullName:    resp.User.FullName,
			Tier:        resp.User.Tier,
			CreatedAt:   timestampProto(resp.User.CreatedAt),
			LastLoginAt: timestampProtoPtr(resp.User.LastLoginAt),
		},
	}, nil
}

// Register implements user registration (uc_05_01_04)
func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.AuthResponse, error) {
	dtoReq := &dto.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	}

	ipAddress := getIPAddress(ctx)
	userAgent := getUserAgent(ctx)

	resp, err := h.authService.Register(ctx, dtoReq, ipAddress, userAgent)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
		TokenType:    resp.TokenType,
		User: &authv1.User{
			Id:        resp.User.ID.String(),
			Email:     resp.User.Email,
			FullName:  resp.User.FullName,
			Tier:      resp.User.Tier,
			CreatedAt: timestampProto(resp.User.CreatedAt),
		},
	}, nil
}

// RefreshToken implements token refresh (uc_05_01_05)
func (h *AuthHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	dtoReq := &dto.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	}

	ipAddress := getIPAddress(ctx)
	userAgent := getUserAgent(ctx)

	resp, err := h.tokenService.RefreshToken(ctx, dtoReq, ipAddress, userAgent)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
		User: &authv1.User{
			Id:        resp.User.ID.String(),
			Email:     resp.User.Email,
			FullName:  resp.User.FullName,
			Tier:      resp.User.Tier,
			CreatedAt: timestampProto(resp.User.CreatedAt),
		},
	}, nil
}

// Logout implements logout (uc_05_01_06)
func (h *AuthHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.Empty, error) {
	ipAddress := getIPAddress(ctx)
	userAgent := getUserAgent(ctx)

	// Pass uuid.Nil for userID - the TokenService.Logout will extract the user ID
	// from the refresh token for proper audit logging
	err := h.tokenService.Logout(ctx, uuid.Nil, req.RefreshToken, req.SessionId, ipAddress, userAgent)
	if err != nil {
		return nil, errorToStatus(err)
	}

	return &authv1.Empty{}, nil
}

// ValidateToken implements token validation (uc_05_01_07)
func (h *AuthHandler) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	resp, err := h.tokenService.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return &authv1.ValidateTokenResponse{
			Valid: false,
		}, nil
	}

	return &authv1.ValidateTokenResponse{
		Valid:  resp.Valid,
		UserId: resp.UserID,
		Email:  resp.Email,
		Tier:   resp.Tier,
	}, nil
}

// Вспомогательные функции

func getIPAddress(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		// Пробуем общие заголовки для IP-адреса
		if ips := md.Get("x-forwarded-for"); len(ips) > 0 && ips[0] != "" {
			return ips[0]
		}
		if ips := md.Get("x-real-ip"); len(ips) > 0 && ips[0] != "" {
			return ips[0]
		}
	}
	// Fallback: извлекаем адрес из gRPC-соединения
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		addr := p.Addr.String()
		if host, _, err := net.SplitHostPort(addr); err == nil && host != "" {
			return host
		}
	}
	return "127.0.0.1"
}

func getUserAgent(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	if agents := md.Get("user-agent"); len(agents) > 0 && agents[0] != "" {
		return agents[0]
	}
	return ""
}

func errorToStatus(err error) error {
	if appErr, ok := err.(*errors.AppError); ok {
		code := mapErrorCode(appErr.Code)
		return status.Error(code, appErr.Message)
	}
	return status.Error(codes.Internal, "internal error")
}

func mapErrorCode(code string) codes.Code {
	switch code {
	case errors.ErrorCodeInvalidCredentials:
		return codes.Unauthenticated
	case errors.ErrorCodeInvalidToken:
		return codes.Unauthenticated
	case errors.ErrorCodeExpiredToken:
		return codes.Unauthenticated
	case errors.ErrorCodeAccountLocked:
		return codes.PermissionDenied
	case errors.ErrorCodeUserExists:
		return codes.AlreadyExists
	case errors.ErrorCodeInvalidRequest:
		return codes.InvalidArgument
	default:
		return codes.Internal
	}
}

func timestampProto(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

func timestampProtoPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
