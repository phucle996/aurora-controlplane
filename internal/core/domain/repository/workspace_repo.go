package core_domainrepo

import (
	"context"

	"controlplane/internal/core/domain/entity"
)

type WorkspaceRepository interface {
	ListWorkspaceOptions(ctx context.Context) ([]*entity.WorkspaceOption, error)
	ListWorkspaces(ctx context.Context, filter entity.WorkspaceListFilter) (*entity.WorkspacePage, error)
	GetWorkspace(ctx context.Context, id string) (*entity.WorkspaceView, error)
	CreateWorkspace(ctx context.Context, workspace *entity.Workspace) (*entity.WorkspaceView, error)
	UpdateWorkspace(ctx context.Context, id string, patch entity.WorkspacePatch) (*entity.WorkspaceView, error)
	DeleteWorkspace(ctx context.Context, id string) error
}
