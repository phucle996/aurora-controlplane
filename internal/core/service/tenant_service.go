package service

import (
	"context"
	"strings"

	"controlplane/internal/core/domain/entity"
	core_domainrepo "controlplane/internal/core/domain/repository"
	core_errorx "controlplane/internal/core/errorx"
	"controlplane/pkg/id"
)

type TenantService struct {
	repo core_domainrepo.TenantRepository
}

func NewTenantService(repo core_domainrepo.TenantRepository) *TenantService {
	return &TenantService{repo: repo}
}

func (s *TenantService) ListTenants(ctx context.Context, filter entity.TenantListFilter) (*entity.TenantPage, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrTenantNotFound
	}

	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	if filter.Status != "" && !isAllowedStatus(filter.Status, tenantStatuses) {
		return nil, core_errorx.ErrTenantInvalid
	}

	return s.repo.ListTenants(ctx, filter)
}

func (s *TenantService) GetTenant(ctx context.Context, id string) (*entity.Tenant, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrTenantNotFound
	}
	return s.repo.GetTenant(ctx, strings.TrimSpace(id))
}

func (s *TenantService) CreateTenant(ctx context.Context, tenant *entity.Tenant) (*entity.Tenant, error) {
	if s == nil || s.repo == nil || tenant == nil {
		return nil, core_errorx.ErrTenantInvalid
	}

	tenant.Name = strings.TrimSpace(tenant.Name)
	tenant.Status = strings.ToLower(strings.TrimSpace(tenant.Status))
	if tenant.Name == "" {
		return nil, core_errorx.ErrTenantInvalid
	}
	if tenant.Status == "" {
		tenant.Status = "active"
	}
	if !isAllowedStatus(tenant.Status, tenantStatuses) {
		return nil, core_errorx.ErrTenantInvalid
	}

	tenant.Slug = normalizeGeneratedSlug(tenant.Name, 100)
	if tenant.Slug == "" {
		return nil, core_errorx.ErrTenantInvalid
	}

	tenantID, err := id.Generate()
	if err != nil {
		return nil, err
	}
	tenant.ID = tenantID

	return s.repo.CreateTenant(ctx, tenant)
}

func (s *TenantService) UpdateTenant(ctx context.Context, id string, patch entity.TenantPatch) (*entity.Tenant, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrTenantNotFound
	}
	if patch.Name == nil && patch.Status == nil {
		return nil, core_errorx.ErrTenantInvalid
	}
	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name == "" {
			return nil, core_errorx.ErrTenantInvalid
		}
		patch.Name = &name
	}
	if patch.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*patch.Status))
		if !isAllowedStatus(status, tenantStatuses) {
			return nil, core_errorx.ErrTenantInvalid
		}
		patch.Status = &status
	}

	return s.repo.UpdateTenant(ctx, strings.TrimSpace(id), patch)
}

func (s *TenantService) DeleteTenant(ctx context.Context, id string) error {
	if s == nil || s.repo == nil {
		return core_errorx.ErrTenantNotFound
	}
	return s.repo.DeleteTenant(ctx, strings.TrimSpace(id))
}
