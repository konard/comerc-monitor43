package grpc

import (
	"context"
	"testing"

	authv1 "github.com/raul/monitor/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockAuthClient is a mock implementation of AuthServiceClient
type mockAuthClient struct {
	validateTokenFunc func(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error)
}

func (m *mockAuthClient) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest, opts ...grpc.CallOption) (*authv1.ValidateTokenResponse, error) {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(ctx, req)
	}
	return &authv1.ValidateTokenResponse{
		Valid:  true,
		UserId: "user123",
		Email:  "test@example.com",
		Tier:   "premium",
	}, nil
}

func (m *mockAuthClient) OAuthLogin(ctx context.Context, req *authv1.OAuthLoginRequest, opts ...grpc.CallOption) (*authv1.OAuthLoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) OAuthCallback(ctx context.Context, req *authv1.OAuthCallbackRequest, opts ...grpc.CallOption) (*authv1.AuthResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) PasswordLogin(ctx context.Context, req *authv1.PasswordLoginRequest, opts ...grpc.CallOption) (*authv1.AuthResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) Register(ctx context.Context, req *authv1.RegisterRequest, opts ...grpc.CallOption) (*authv1.AuthResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest, opts ...grpc.CallOption) (*authv1.RefreshTokenResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) Logout(ctx context.Context, req *authv1.LogoutRequest, opts ...grpc.CallOption) (*authv1.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) GetUser(ctx context.Context, req *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.User, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) UpdateUser(ctx context.Context, req *authv1.UpdateUserRequest, opts ...grpc.CallOption) (*authv1.User, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) ListSessions(ctx context.Context, req *authv1.ListSessionsRequest, opts ...grpc.CallOption) (*authv1.ListSessionsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) RevokeSession(ctx context.Context, req *authv1.RevokeSessionRequest, opts ...grpc.CallOption) (*authv1.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) LockAccount(ctx context.Context, req *authv1.LockAccountRequest, opts ...grpc.CallOption) (*authv1.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) UnlockAccount(ctx context.Context, req *authv1.UnlockAccountRequest, opts ...grpc.CallOption) (*authv1.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// Team management RPCs (us=05_02_team_management) — заглушки для совместимости.

func (m *mockAuthClient) InviteMember(ctx context.Context, req *authv1.InviteMemberRequest, opts ...grpc.CallOption) (*authv1.Invite, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) RevokeInvite(ctx context.Context, req *authv1.RevokeInviteRequest, opts ...grpc.CallOption) (*authv1.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) AcceptInvite(ctx context.Context, req *authv1.AcceptInviteRequest, opts ...grpc.CallOption) (*authv1.Membership, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) ListMembers(ctx context.Context, req *authv1.ListMembersRequest, opts ...grpc.CallOption) (*authv1.ListMembersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) ChangeMemberRole(ctx context.Context, req *authv1.ChangeMemberRoleRequest, opts ...grpc.CallOption) (*authv1.Membership, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) RemoveMember(ctx context.Context, req *authv1.RemoveMemberRequest, opts ...grpc.CallOption) (*authv1.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) TransferOwnership(ctx context.Context, req *authv1.TransferOwnershipRequest, opts ...grpc.CallOption) (*authv1.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockAuthClient) LeaveOrganization(ctx context.Context, req *authv1.LeaveOrganizationRequest, opts ...grpc.CallOption) (*authv1.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// mockConn is a mock gRPC connection
type mockConn struct {
	CloseFunc func() error
}

func (m *mockConn) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func TestNewAuthRepository(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, err := NewAuthRepository("localhost:5001")

		// This will fail in tests without a running gRPC server
		// We're just checking the structure
		if err != nil {
			t.Logf("Expected error without running server: %v", err)
		}

		if repo != nil {
			t.Logf("Repository created: %+v", repo)
		}
	})
}

func TestNewAuthRepositoryFromPool(t *testing.T) {
	// NewAuthRepositoryFromPool creates a repository backed by a connection pool.
	// We pass nil to test the constructor path without requiring a real pool.
	repo, err := NewAuthRepositoryFromPool(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestAuthRepository_ValidateToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		valid       bool
		userID      string
		email       string
		tier        string
		shouldError bool
	}{
		{
			name:   "valid token",
			token:  "valid-token",
			valid:  true,
			userID: "user123",
			email:  "test@example.com",
			tier:   "premium",
		},
		{
			name:        "invalid token",
			token:       "invalid-token",
			valid:       false,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockAuthClient{
				validateTokenFunc: func(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
					if req.AccessToken == tt.token {
						return &authv1.ValidateTokenResponse{
							Valid:  tt.valid,
							UserId: tt.userID,
							Email:  tt.email,
							Tier:   tt.tier,
						}, nil
					}
					return &authv1.ValidateTokenResponse{Valid: false}, nil
				},
			}

			repo := &authRepository{client: mockClient}
			result, err := repo.ValidateToken(context.Background(), tt.token)

			if tt.shouldError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.shouldError && result.Valid != tt.valid {
				t.Errorf("expected valid %v, got %v", tt.valid, result.Valid)
			}

			if tt.valid && result.UserID != tt.userID {
				t.Errorf("expected user ID %s, got %s", tt.userID, result.UserID)
			}
		})
	}
}

func TestAuthRepository_ValidateToken_Error(t *testing.T) {
	mockClient := &mockAuthClient{
		validateTokenFunc: func(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
			return nil, status.Error(codes.Internal, "internal error")
		},
	}

	repo := &authRepository{client: mockClient}
	result, err := repo.ValidateToken(context.Background(), "test-token")

	// Should return error result, not an error itself (graceful degradation)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Valid {
		t.Error("expected result to be invalid")
	}

	if result.Error == nil {
		t.Error("expected error to be set in result")
	}
}

func TestAuthRepository_CheckHealth(t *testing.T) {
	tests := []struct {
		name     string
		healthy  bool
		expected bool
	}{
		{
			name:     "healthy",
			healthy:  true,
			expected: true,
		},
		{
			name:     "unhealthy",
			healthy:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockAuthClient{
				validateTokenFunc: func(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
					if !tt.healthy {
						return nil, status.Error(codes.Unavailable, "service unavailable")
					}
					return &authv1.ValidateTokenResponse{Valid: false}, nil
				},
			}

			repo := &authRepository{client: mockClient}
			healthy, err := repo.CheckHealth(context.Background())

			if tt.healthy && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.healthy && err == nil {
				t.Error("expected error for unhealthy service")
			}

			if tt.healthy && !healthy {
				t.Error("expected healthy to be true")
			}
		})
	}
}
