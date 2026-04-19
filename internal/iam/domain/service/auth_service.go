package iam_domainsvc

import (
	"context"

	"controlplane/internal/iam/domain/entity"
)

// AuthService defines primary authentication actions.
type AuthService interface {
	Login(ctx context.Context, username, password string) (*entity.LoginResult, error)
	Register(ctx context.Context, user *entity.User, profile *entity.UserProfile, rawPassword string) error
	Activate(ctx context.Context, token string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	Logout(ctx context.Context, jti string, rawRefreshToken string) error
}
