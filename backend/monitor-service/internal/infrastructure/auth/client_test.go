package auth

import (
	"context"
	"errors"
	"testing"

	authapi "github.com/raul/monitor/api/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockAuthServiceClient является mock реализацией AuthServiceClient для тестов.
type MockAuthServiceClient struct {
	ValidateTokenFunc func(ctx context.Context, in *authapi.ValidateTokenRequest, opts ...grpc.CallOption) (*authapi.ValidateTokenResponse, error)
}

func (m *MockAuthServiceClient) ValidateToken(ctx context.Context, in *authapi.ValidateTokenRequest, opts ...grpc.CallOption) (*authapi.ValidateTokenResponse, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(ctx, in)
	}
	return &authapi.ValidateTokenResponse{Valid: false}, nil
}

// Implement other required methods from AuthServiceClient interface
func (m *MockAuthServiceClient) Register(ctx context.Context, in *authapi.RegisterRequest, opts ...grpc.CallOption) (*authapi.AuthResponse, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) PasswordLogin(ctx context.Context, in *authapi.PasswordLoginRequest, opts ...grpc.CallOption) (*authapi.AuthResponse, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) OAuthLogin(ctx context.Context, in *authapi.OAuthLoginRequest, opts ...grpc.CallOption) (*authapi.OAuthLoginResponse, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) OAuthCallback(ctx context.Context, in *authapi.OAuthCallbackRequest, opts ...grpc.CallOption) (*authapi.AuthResponse, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) RefreshToken(ctx context.Context, in *authapi.RefreshTokenRequest, opts ...grpc.CallOption) (*authapi.RefreshTokenResponse, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) Logout(ctx context.Context, in *authapi.LogoutRequest, opts ...grpc.CallOption) (*authapi.Empty, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) GetUser(ctx context.Context, in *authapi.GetUserRequest, opts ...grpc.CallOption) (*authapi.User, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) UpdateUser(ctx context.Context, in *authapi.UpdateUserRequest, opts ...grpc.CallOption) (*authapi.User, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) ListSessions(ctx context.Context, in *authapi.ListSessionsRequest, opts ...grpc.CallOption) (*authapi.ListSessionsResponse, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) RevokeSession(ctx context.Context, in *authapi.RevokeSessionRequest, opts ...grpc.CallOption) (*authapi.Empty, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) LockAccount(ctx context.Context, in *authapi.LockAccountRequest, opts ...grpc.CallOption) (*authapi.Empty, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) UnlockAccount(ctx context.Context, in *authapi.UnlockAccountRequest, opts ...grpc.CallOption) (*authapi.Empty, error) {
	return nil, nil
}

// Team management RPCs (us=05_02_team_management) — заглушки для совместимости
// с auth-service gRPC contract. Юнит-тесты monitor-service их не используют.

func (m *MockAuthServiceClient) InviteMember(ctx context.Context, in *authapi.InviteMemberRequest, opts ...grpc.CallOption) (*authapi.Invite, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) RevokeInvite(ctx context.Context, in *authapi.RevokeInviteRequest, opts ...grpc.CallOption) (*authapi.Empty, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) AcceptInvite(ctx context.Context, in *authapi.AcceptInviteRequest, opts ...grpc.CallOption) (*authapi.Membership, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) ListMembers(ctx context.Context, in *authapi.ListMembersRequest, opts ...grpc.CallOption) (*authapi.ListMembersResponse, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) ChangeMemberRole(ctx context.Context, in *authapi.ChangeMemberRoleRequest, opts ...grpc.CallOption) (*authapi.Membership, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) RemoveMember(ctx context.Context, in *authapi.RemoveMemberRequest, opts ...grpc.CallOption) (*authapi.Empty, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) TransferOwnership(ctx context.Context, in *authapi.TransferOwnershipRequest, opts ...grpc.CallOption) (*authapi.Empty, error) {
	return nil, nil
}

func (m *MockAuthServiceClient) LeaveOrganization(ctx context.Context, in *authapi.LeaveOrganizationRequest, opts ...grpc.CallOption) (*authapi.Empty, error) {
	return nil, nil
}

func TestValidateToken_Success(t *testing.T) {
	t.Parallel()
	// Arrange
	mockClient := &MockAuthServiceClient{
		ValidateTokenFunc: func(ctx context.Context, in *authapi.ValidateTokenRequest, opts ...grpc.CallOption) (*authapi.ValidateTokenResponse, error) {
			return &authapi.ValidateTokenResponse{
				Valid:  true,
				UserId: "user-123",
				Email:  "test@example.com",
				Tier:   "free",
			}, nil
		},
	}

	authClient := &AuthClient{
		client: mockClient,
	}

	ctx := context.Background()
	token := "valid-jwt-token" //nolint:gosec // G101: тестовый токен

	// Act
	userID, err := authClient.ValidateToken(ctx, token)

	// Assert
	if err != nil {
		t.Errorf("ValidateToken() error = %v", err)
		return
	}

	if userID != "user-123" {
		t.Errorf("ValidateToken() userID = %v, want %v", userID, "user-123")
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	t.Parallel()
	// Arrange
	mockClient := &MockAuthServiceClient{
		ValidateTokenFunc: func(ctx context.Context, in *authapi.ValidateTokenRequest, opts ...grpc.CallOption) (*authapi.ValidateTokenResponse, error) {
			return &authapi.ValidateTokenResponse{
				Valid: false,
			}, nil
		},
	}

	authClient := &AuthClient{
		client: mockClient,
	}

	ctx := context.Background()
	token := "invalid-token"

	// Act
	userID, err := authClient.ValidateToken(ctx, token)

	// Assert
	if err == nil {
		t.Error("ValidateToken() expected error for invalid token, got nil")
		return
	}

	if userID != "" {
		t.Errorf("ValidateToken() userID = %v, want empty string", userID)
	}
}

func TestValidateToken_UnauthenticatedError(t *testing.T) {
	t.Parallel()
	// Arrange
	mockClient := &MockAuthServiceClient{
		ValidateTokenFunc: func(ctx context.Context, in *authapi.ValidateTokenRequest, opts ...grpc.CallOption) (*authapi.ValidateTokenResponse, error) {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		},
	}

	authClient := &AuthClient{
		client: mockClient,
	}

	ctx := context.Background()
	token := "expired-token"

	// Act
	userID, err := authClient.ValidateToken(ctx, token)

	// Assert
	if err == nil {
		t.Error("ValidateToken() expected error for unauthenticated token, got nil")
		return
	}

	if userID != "" {
		t.Errorf("ValidateToken() userID = %v, want empty string", userID)
	}
}

func TestValidateToken_EmptyUserID(t *testing.T) {
	t.Parallel()
	// Arrange
	mockClient := &MockAuthServiceClient{
		ValidateTokenFunc: func(ctx context.Context, in *authapi.ValidateTokenRequest, opts ...grpc.CallOption) (*authapi.ValidateTokenResponse, error) {
			return &authapi.ValidateTokenResponse{
				Valid:  true,
				UserId: "", // Empty user_id
				Email:  "test@example.com",
			}, nil
		},
	}

	authClient := &AuthClient{
		client: mockClient,
	}

	ctx := context.Background()
	token := "valid-token-but-no-userid"

	// Act
	userID, err := authClient.ValidateToken(ctx, token)

	// Assert
	if err == nil {
		t.Error("ValidateToken() expected error for empty user_id, got nil")
		return
	}

	if userID != "" {
		t.Errorf("ValidateToken() userID = %v, want empty string", userID)
	}
}

func TestValidateTokenWithTier_Success(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		token          string
		mockResponse   *authapi.ValidateTokenResponse
		mockError      error
		expectedUserID string
		expectedTier   string
		expectError    bool
		errorContains  string
	}{
		{ //nolint:gosec // G101: тестовые токены
			name:  "valid token with tier",
			token: "valid-jwt-token",
			mockResponse: &authapi.ValidateTokenResponse{
				Valid:  true,
				UserId: "user-123",
				Email:  "test@example.com",
				Tier:   "premium",
			},
			expectedUserID: "user-123",
			expectedTier:   "premium",
			expectError:    false,
		},
		{ //nolint:gosec // G101: тестовые токены
			name:  "valid token with empty tier defaults to Free",
			token: "valid-jwt-token",
			mockResponse: &authapi.ValidateTokenResponse{
				Valid:  true,
				UserId: "user-456",
				Email:  "test@example.com",
				Tier:   "",
			},
			expectedUserID: "user-456",
			expectedTier:   "Free",
			expectError:    false,
		},
		{ //nolint:gosec // G101: тестовые токены
			name:  "valid token with Pro tier",
			token: "valid-jwt-token",
			mockResponse: &authapi.ValidateTokenResponse{
				Valid:  true,
				UserId: "user-789",
				Email:  "pro@example.com",
				Tier:   "Pro",
			},
			expectedUserID: "user-789",
			expectedTier:   "Pro",
			expectError:    false,
		},
		{
			name:          "invalid token",
			token:         "invalid-token",
			mockResponse:  &authapi.ValidateTokenResponse{Valid: false},
			expectError:   true,
			errorContains: "invalid token",
		},
		{
			name:  "valid token but empty user_id",
			token: "token-without-userid",
			mockResponse: &authapi.ValidateTokenResponse{
				Valid:  true,
				UserId: "",
				Email:  "test@example.com",
			},
			expectError:   true,
			errorContains: "user_id is empty",
		},
		{
			name:          "unauthenticated error",
			token:         "expired-token",
			mockResponse:  nil,
			mockError:     status.Error(codes.Unauthenticated, "token expired"),
			expectError:   true,
			errorContains: "token validation failed",
		},
		{
			name:          "internal error from auth service",
			token:         "valid-token",
			mockResponse:  nil,
			mockError:     status.Error(codes.Internal, "database connection failed"),
			expectError:   true,
			errorContains: "auth service error",
		},
		{
			name:          "unavailable error from auth service",
			token:         "valid-token",
			mockResponse:  nil,
			mockError:     status.Error(codes.Unavailable, "auth service unavailable"),
			expectError:   true,
			errorContains: "auth service error",
		},
		{
			name:          "network error without status code",
			token:         "valid-token",
			mockResponse:  nil,
			mockError:     status.Error(codes.Unknown, "connection refused"),
			expectError:   true,
			errorContains: "auth service error",
		},
		{
			name:          "generic error without status",
			token:         "valid-token",
			mockResponse:  nil,
			mockError:     errors.New("some generic error"),
			expectError:   true,
			errorContains: "failed to validate token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockAuthServiceClient{
				ValidateTokenFunc: func(ctx context.Context, in *authapi.ValidateTokenRequest, opts ...grpc.CallOption) (*authapi.ValidateTokenResponse, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			authClient := &AuthClient{
				client: mockClient,
			}

			ctx := context.Background()

			// Act
			userID, tier, err := authClient.ValidateTokenWithTier(ctx, tt.token)

			// Assert
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateTokenWithTier() expected error containing '%s', got nil", tt.errorContains)
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("ValidateTokenWithTier() error = %v, want error containing '%s'", err, tt.errorContains)
				}
				if userID != "" || tier != "" {
					t.Errorf("ValidateTokenWithTier() userID = %v, tier = %v, want both empty on error", userID, tier)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTokenWithTier() unexpected error = %v", err)
					return
				}
				if userID != tt.expectedUserID {
					t.Errorf("ValidateTokenWithTier() userID = %v, want %v", userID, tt.expectedUserID)
				}
				if tier != tt.expectedTier {
					t.Errorf("ValidateTokenWithTier() tier = %v, want %v", tier, tt.expectedTier)
				}
			}
		})
	}
}

func TestNewAuthClient(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		address     string
		expectError bool
	}{
		{
			name:        "valid address format (connection will fail in real usage)",
			address:     "localhost:50051",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			client, err := NewAuthClient(tt.address)

			// Assert
			if tt.expectError {
				if err == nil {
					t.Error("NewAuthClient() expected error, got nil")
					return
				}
				if client != nil {
					t.Error("NewAuthClient() expected nil client on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewAuthClient() unexpected error = %v", err)
					return
				}
				if client == nil {
					t.Error("NewAuthClient() expected non-nil client")
				}
				// Clean up
				assert.NoError(t, client.Close())
			}
		})
	}
}

func TestNewAuthClientWithConn(t *testing.T) {
	t.Parallel()
	// This test verifies NewAuthClientWithConn creates a client with the provided connection
	// Note: We can't easily create a real gRPC connection in unit tests, so we just test nil behavior
	t.Run("nil connection", func(t *testing.T) {
		// Arrange & Act
		client := NewAuthClientWithConn(nil)

		// Assert
		if client == nil {
			t.Error("NewAuthClientWithConn() expected non-nil client")
		}
	})
}

func TestAuthClient_Close(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		setupClient func() *AuthClient
		expectError bool
	}{
		{
			name: "close client with nil connection",
			setupClient: func() *AuthClient {
				return &AuthClient{
					conn:   nil,
					client: &MockAuthServiceClient{},
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			client := tt.setupClient()

			// Act
			err := client.Close()

			// Assert
			if tt.expectError && err == nil {
				t.Error("Close() expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Close() unexpected error = %v", err)
			}
		})
	}
}

func TestValidateToken_VariousErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		mockError     error
		errorContains string
	}{
		{
			name:          "permission denied",
			mockError:     status.Error(codes.PermissionDenied, "access denied"),
			errorContains: "auth service error",
		},
		{
			name:          "deadline exceeded",
			mockError:     status.Error(codes.DeadlineExceeded, "request timeout"),
			errorContains: "auth service error",
		},
		{
			name:          "not found",
			mockError:     status.Error(codes.NotFound, "user not found"),
			errorContains: "auth service error",
		},
		{
			name:          "already exists",
			mockError:     status.Error(codes.AlreadyExists, "token already used"),
			errorContains: "auth service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockAuthServiceClient{
				ValidateTokenFunc: func(ctx context.Context, in *authapi.ValidateTokenRequest, opts ...grpc.CallOption) (*authapi.ValidateTokenResponse, error) {
					return nil, tt.mockError
				},
			}

			authClient := &AuthClient{
				client: mockClient,
			}

			ctx := context.Background()
			token := "test-token"

			// Act
			userID, err := authClient.ValidateToken(ctx, token)

			// Assert
			if err == nil {
				t.Error("ValidateToken() expected error, got nil")
				return
			}
			if !containsString(err.Error(), tt.errorContains) {
				t.Errorf("ValidateToken() error = %v, want containing '%s'", err, tt.errorContains)
			}
			if userID != "" {
				t.Errorf("ValidateToken() userID = %v, want empty string", userID)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
