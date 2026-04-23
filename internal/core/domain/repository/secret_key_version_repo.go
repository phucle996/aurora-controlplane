package core_domainrepo

import (
	"context"

	"controlplane/internal/core/domain/entity"
)

type SecretKeyVersionRepository interface {
	ListLatestStatePerFamily(ctx context.Context) ([]*entity.SecretFamilyState, error)
	GetFamilyVersions(ctx context.Context, family string) ([]*entity.SecretKeyVersion, error)
	CreateInitialActive(ctx context.Context, value *entity.SecretKeyVersion) error
	RotateFamilyTx(ctx context.Context, family string, nextActive *entity.SecretKeyVersion) error
}
