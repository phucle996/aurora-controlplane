package iam_svcImple

import (
	"context"

	"controlplane/internal/iam/domain/repository"
	"controlplane/internal/iam/domain/service"
	"controlplane/internal/iam/errorx"
)

type authSvcImple struct {
	userRepo iam_domainrepo.UserRepository
}

func NewAuthSvcImple(userRepo iam_domainrepo.UserRepository) iam_domainsvc.AuthService {
	return &authSvcImple{userRepo: userRepo}
}

func (s *authSvcImple) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	if user.Status != "active" {
		return "", iam_errorx.ErrUserInactive
	}

	// TODO: Replace with proper bcrypt password comparison.
	if user.Password != password {
		return "", iam_errorx.ErrInvalidCredentials
	}

	// TODO: Replace with proper JWT generation.
	token := "dummy-token-for-" + user.ID

	return token, nil
}
