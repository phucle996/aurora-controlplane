package iam_domainsvc

import (
	"context"
)

// AuthService defines primary authentication actions.
type AuthService interface {
	Login(ctx context.Context, email, password string) (token string, err error)
}
