package core_domainsvc

import (
	"context"

	"controlplane/internal/core/domain/entity"
)

type TenantService interface {
	ListTenants(ctx context.Context, filter entity.TenantListFilter) (*entity.TenantPage, error)
	GetTenant(ctx context.Context, id string) (*entity.Tenant, error)
	CreateTenant(ctx context.Context, tenant *entity.Tenant) (*entity.Tenant, error)
	UpdateTenant(ctx context.Context, id string, patch entity.TenantPatch) (*entity.Tenant, error)
	DeleteTenant(ctx context.Context, id string) error
}
