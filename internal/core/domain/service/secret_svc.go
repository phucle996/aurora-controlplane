package core_domainsvc

import (
	"context"

	"controlplane/internal/security"
)

type SecretService interface {
	security.SecretProvider
	Bootstrap(ctx context.Context) error
	RotateDue(ctx context.Context) error
	RefreshChangedFamilies(ctx context.Context) error
}
