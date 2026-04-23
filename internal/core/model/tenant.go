package core_model

import (
	"time"

	"controlplane/internal/core/domain/entity"
)

// Tenant mirrors core.tenants.
type Tenant struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Slug      string    `db:"slug"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// TenantMember mirrors core.tenant_members.
type TenantMember struct {
	TenantID string    `db:"tenant_id"`
	UserID   string    `db:"user_id"`
	RoleID   string    `db:"role_id"`
	JoinedAt time.Time `db:"joined_at"`
}

func TenantEntityToModel(v *entity.Tenant) *Tenant {
	if v == nil {
		return nil
	}
	return &Tenant{
		ID:        v.ID,
		Name:      v.Name,
		Slug:      v.Slug,
		Status:    v.Status,
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
	}
}

func TenantModelToEntity(v *Tenant) *entity.Tenant {
	if v == nil {
		return nil
	}
	return &entity.Tenant{
		ID:        v.ID,
		Name:      v.Name,
		Slug:      v.Slug,
		Status:    v.Status,
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
	}
}

func TenantMemberEntityToModel(v *entity.TenantMember) *TenantMember {
	if v == nil {
		return nil
	}
	return &TenantMember{
		TenantID: v.TenantID,
		UserID:   v.UserID,
		RoleID:   v.RoleID,
		JoinedAt: v.JoinedAt,
	}
}

func TenantMemberModelToEntity(v *TenantMember) *entity.TenantMember {
	if v == nil {
		return nil
	}
	return &entity.TenantMember{
		TenantID: v.TenantID,
		UserID:   v.UserID,
		RoleID:   v.RoleID,
		JoinedAt: v.JoinedAt,
	}
}
