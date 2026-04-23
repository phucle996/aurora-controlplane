package iam_domainsvc

import "context"

type AdminAPITokenService interface {
	EnsureBootstrapToken(ctx context.Context) (token string, created bool, err error)
	Validate(ctx context.Context, token string) (bool, error)
}
