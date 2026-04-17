package iam_domainrepo

import (
	"context"

	"controlplane/internal/iam/domain/entity"
)

// UserRepository defines data access methods for User.
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByID(ctx context.Context, id string) (*entity.User, error)
}
