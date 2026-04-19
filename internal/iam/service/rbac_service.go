package service

import (
	"context"
	"fmt"

	"controlplane/internal/http/middleware"
	"controlplane/internal/iam/domain/entity"
	iam_domainrepo "controlplane/internal/iam/domain/repository"
	iam_errorx "controlplane/internal/iam/errorx"
)

// RbacService implements iam_domainsvc.RbacService.
//
// Cache-aside: every GetRoleEntry checks RoleRegistry first; on miss it fetches
// from Postgres, populates the registry (15-min TTL by default), then returns.
// Every mutation that changes a role's name/level/permissions calls
// InvalidateRole so the next request sees fresh data.
type RbacService struct {
	repo     iam_domainrepo.RbacRepository
	registry *middleware.RoleRegistry
}

func NewRbacService(repo iam_domainrepo.RbacRepository, registry *middleware.RoleRegistry) *RbacService {
	return &RbacService{repo: repo, registry: registry}
}

// ── RoleResolver ──────────────────────────────────────────────────────────────

// GetRoleEntry resolves role name → RoleEntry, implementing middleware.RoleResolver.
func (s *RbacService) GetRoleEntry(ctx context.Context, role string) (middleware.RoleEntry, error) {
	// 1. In-memory hit.
	if entry, ok := s.registry.Get(role); ok {
		return entry, nil
	}

	// 2. Miss — load from DB.
	rp, err := s.repo.GetRoleByName(ctx, role)
	if err != nil {
		return middleware.RoleEntry{}, fmt.Errorf("rbac svc: resolve %q: %w", role, err)
	}

	// 3. Cache with default TTL (15 min).
	entry := middleware.RoleEntry{
		Level:       rp.Role.Level,
		Permissions: rp.Permissions,
	}
	s.registry.Set(role, entry)
	return entry, nil
}

// InvalidateRole evicts one role so the next request refetches.
func (s *RbacService) InvalidateRole(role string) { s.registry.Invalidate(role) }

// InvalidateAll clears the entire cache.
func (s *RbacService) InvalidateAll() { s.registry.InvalidateAll() }

// WarmUp preloads all roles at startup to avoid cold cache on first requests.
func (s *RbacService) WarmUp(ctx context.Context) error {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return fmt.Errorf("rbac svc: warm-up: %w", err)
	}
	for _, r := range roles {
		rp, err := s.repo.GetRoleByName(ctx, r.Name)
		if err != nil {
			return fmt.Errorf("rbac svc: warm-up %q: %w", r.Name, err)
		}
		s.registry.Set(r.Name, middleware.RoleEntry{
			Level:       rp.Role.Level,
			Permissions: rp.Permissions,
		})
	}
	return nil
}

// ── Role admin ────────────────────────────────────────────────────────────────

func (s *RbacService) ListRoles(ctx context.Context) ([]*entity.Role, error) {
	return s.repo.ListRoles(ctx)
}

func (s *RbacService) GetRole(ctx context.Context, id string) (*entity.RoleWithPermissions, error) {
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetRoleByName(ctx, role.Name)
}

func (s *RbacService) CreateRole(ctx context.Context, role *entity.Role) error {
	return s.repo.CreateRole(ctx, role)
}

// UpdateRole persists changes and invalidates the old + new role name from cache.
func (s *RbacService) UpdateRole(ctx context.Context, role *entity.Role) error {
	old, err := s.repo.GetRoleByID(ctx, role.ID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return err
	}
	s.registry.Invalidate(old.Name)
	if role.Name != old.Name {
		s.registry.Invalidate(role.Name)
	}
	return nil
}

// DeleteRole removes the role from DB and evicts it from cache.
func (s *RbacService) DeleteRole(ctx context.Context, id string) error {
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteRole(ctx, id); err != nil {
		return err
	}
	s.registry.Invalidate(role.Name)
	return nil
}

// ── Permission admin ──────────────────────────────────────────────────────────

func (s *RbacService) ListPermissions(ctx context.Context) ([]*entity.Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *RbacService) CreatePermission(ctx context.Context, perm *entity.Permission) error {
	return s.repo.CreatePermission(ctx, perm)
}

// AssignPermission adds a permission and invalidates the affected role's cache.
func (s *RbacService) AssignPermission(ctx context.Context, roleID, permID string) error {
	if err := s.repo.AssignPermission(ctx, roleID, permID); err != nil {
		return err
	}
	if role, err := s.repo.GetRoleByID(ctx, roleID); err == nil {
		s.registry.Invalidate(role.Name)
	}
	return nil
}

// RevokePermission removes a permission and invalidates the affected role's cache.
func (s *RbacService) RevokePermission(ctx context.Context, roleID, permID string) error {
	if err := s.repo.RevokePermission(ctx, roleID, permID); err != nil {
		return err
	}
	if role, err := s.repo.GetRoleByID(ctx, roleID); err == nil {
		s.registry.Invalidate(role.Name)
	}
	return nil
}

// keep import used
var _ = iam_errorx.ErrRoleNotFound
