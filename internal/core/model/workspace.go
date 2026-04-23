package core_model

import (
	"time"

	"controlplane/internal/core/domain/entity"
)

// Workspace mirrors core.workspaces.
type Workspace struct {
	ID          string    `db:"id"`
	TenantID    string    `db:"tenant_id"`
	DataPlaneID string    `db:"data_plane_id"`
	Name        string    `db:"name"`
	Slug        string    `db:"slug"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// WorkspaceMember mirrors core.workspace_members.
type WorkspaceMember struct {
	WorkspaceID string    `db:"workspace_id"`
	UserID      string    `db:"user_id"`
	RoleID      string    `db:"role_id"`
	JoinedAt    time.Time `db:"joined_at"`
}

func WorkspaceEntityToModel(v *entity.Workspace) *Workspace {
	if v == nil {
		return nil
	}
	return &Workspace{
		ID:          v.ID,
		TenantID:    v.TenantID,
		DataPlaneID: v.DataPlaneID,
		Name:        v.Name,
		Slug:        v.Slug,
		Status:      v.Status,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

func WorkspaceModelToEntity(v *Workspace) *entity.Workspace {
	if v == nil {
		return nil
	}
	return &entity.Workspace{
		ID:          v.ID,
		TenantID:    v.TenantID,
		DataPlaneID: v.DataPlaneID,
		Name:        v.Name,
		Slug:        v.Slug,
		Status:      v.Status,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

func WorkspaceMemberEntityToModel(v *entity.WorkspaceMember) *WorkspaceMember {
	if v == nil {
		return nil
	}
	return &WorkspaceMember{
		WorkspaceID: v.WorkspaceID,
		UserID:      v.UserID,
		RoleID:      v.RoleID,
		JoinedAt:    v.JoinedAt,
	}
}

func WorkspaceMemberModelToEntity(v *WorkspaceMember) *entity.WorkspaceMember {
	if v == nil {
		return nil
	}
	return &entity.WorkspaceMember{
		WorkspaceID: v.WorkspaceID,
		UserID:      v.UserID,
		RoleID:      v.RoleID,
		JoinedAt:    v.JoinedAt,
	}
}
