package service

import (
	"context"
	"testing"

	"controlplane/internal/core/domain/entity"
	core_errorx "controlplane/internal/core/errorx"
)

type tenantRepoStub struct {
	listFilter   entity.TenantListFilter
	createTenant *entity.Tenant
	updateID     string
	updatePatch  entity.TenantPatch
}

func (s *tenantRepoStub) ListTenants(_ context.Context, filter entity.TenantListFilter) (*entity.TenantPage, error) {
	s.listFilter = filter
	return &entity.TenantPage{
		Items: []*entity.Tenant{},
		Pagination: entity.Pagination{
			Page:       filter.Page,
			Limit:      filter.Limit,
			Total:      0,
			TotalPages: 0,
		},
	}, nil
}

func (s *tenantRepoStub) GetTenant(context.Context, string) (*entity.Tenant, error) {
	return nil, nil
}

func (s *tenantRepoStub) CreateTenant(_ context.Context, tenant *entity.Tenant) (*entity.Tenant, error) {
	s.createTenant = tenant
	return tenant, nil
}

func (s *tenantRepoStub) UpdateTenant(_ context.Context, id string, patch entity.TenantPatch) (*entity.Tenant, error) {
	s.updateID = id
	s.updatePatch = patch
	return &entity.Tenant{ID: id}, nil
}

func (s *tenantRepoStub) DeleteTenant(context.Context, string) error {
	return nil
}

func TestTenantServiceCreateTenantGeneratesSlugAndDefaultStatus(t *testing.T) {
	repo := &tenantRepoStub{}
	svc := NewTenantService(repo)

	item, err := svc.CreateTenant(context.Background(), &entity.Tenant{
		Name: "  Aurora Tenant Alpha  ",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if item == nil || item.ID == "" {
		t.Fatalf("expected generated tenant id, got %#v", item)
	}
	if repo.createTenant == nil {
		t.Fatalf("expected repo create to be called")
	}
	if repo.createTenant.Slug != "aurora-tenant-alpha" {
		t.Fatalf("expected slug aurora-tenant-alpha, got %q", repo.createTenant.Slug)
	}
	if repo.createTenant.Status != "active" {
		t.Fatalf("expected default status active, got %q", repo.createTenant.Status)
	}
}

func TestTenantServiceListTenantsRejectsInvalidStatus(t *testing.T) {
	svc := NewTenantService(&tenantRepoStub{})

	_, err := svc.ListTenants(context.Background(), entity.TenantListFilter{
		Status: "disabled",
	})
	if err == nil {
		t.Fatalf("expected invalid status error")
	}
	if err != core_errorx.ErrTenantInvalid {
		t.Fatalf("expected ErrTenantInvalid, got %v", err)
	}
}

func TestTenantServiceListTenantsNormalizesPagination(t *testing.T) {
	repo := &tenantRepoStub{}
	svc := NewTenantService(repo)

	if _, err := svc.ListTenants(context.Background(), entity.TenantListFilter{}); err != nil {
		t.Fatalf("list tenants: %v", err)
	}
	if repo.listFilter.Page != 1 {
		t.Fatalf("expected page 1, got %d", repo.listFilter.Page)
	}
	if repo.listFilter.Limit != 20 {
		t.Fatalf("expected limit 20, got %d", repo.listFilter.Limit)
	}
}
