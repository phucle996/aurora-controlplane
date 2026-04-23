package service

import (
	"context"
	"strings"

	"controlplane/internal/core/domain/entity"
	core_domainrepo "controlplane/internal/core/domain/repository"
	core_errorx "controlplane/internal/core/errorx"
	"controlplane/pkg/id"
)

type WorkspaceService struct {
	repo core_domainrepo.WorkspaceRepository
}

func NewWorkspaceService(repo core_domainrepo.WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{repo: repo}
}

func (s *WorkspaceService) ListWorkspaceOptions(ctx context.Context) ([]*entity.WorkspaceOption, error) {
	if s == nil || s.repo == nil {
		return []*entity.WorkspaceOption{}, nil
	}
	return s.repo.ListWorkspaceOptions(ctx)
}

func (s *WorkspaceService) ListWorkspaces(ctx context.Context, filter entity.WorkspaceListFilter) (*entity.WorkspacePage, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrWorkspaceNotFound
	}

	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.TenantID = strings.TrimSpace(filter.TenantID)
	if filter.Status != "" && !isAllowedStatus(filter.Status, workspaceStatuses) {
		return nil, core_errorx.ErrWorkspaceInvalid
	}

	return s.repo.ListWorkspaces(ctx, filter)
}

func (s *WorkspaceService) GetWorkspace(ctx context.Context, id string) (*entity.WorkspaceView, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrWorkspaceNotFound
	}
	return s.repo.GetWorkspace(ctx, strings.TrimSpace(id))
}

func (s *WorkspaceService) CreateWorkspace(ctx context.Context, workspace *entity.Workspace) (*entity.WorkspaceView, error) {
	if s == nil || s.repo == nil || workspace == nil {
		return nil, core_errorx.ErrWorkspaceInvalid
	}

	workspace.Name = strings.TrimSpace(workspace.Name)
	workspace.TenantID = strings.TrimSpace(workspace.TenantID)
	workspace.Status = strings.ToLower(strings.TrimSpace(workspace.Status))
	if workspace.Name == "" {
		return nil, core_errorx.ErrWorkspaceInvalid
	}
	if workspace.Status == "" {
		workspace.Status = "active"
	}
	if !isAllowedStatus(workspace.Status, workspaceStatuses) {
		return nil, core_errorx.ErrWorkspaceInvalid
	}

	workspace.Slug = normalizeGeneratedSlug(workspace.Name, 100)
	if workspace.Slug == "" {
		return nil, core_errorx.ErrWorkspaceInvalid
	}

	workspaceID, err := id.Generate()
	if err != nil {
		return nil, err
	}
	workspace.ID = workspaceID

	return s.repo.CreateWorkspace(ctx, workspace)
}

func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, id string, patch entity.WorkspacePatch) (*entity.WorkspaceView, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrWorkspaceNotFound
	}
	if patch.Name == nil && patch.Status == nil {
		return nil, core_errorx.ErrWorkspaceInvalid
	}
	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name == "" {
			return nil, core_errorx.ErrWorkspaceInvalid
		}
		patch.Name = &name
	}
	if patch.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*patch.Status))
		if !isAllowedStatus(status, workspaceStatuses) {
			return nil, core_errorx.ErrWorkspaceInvalid
		}
		patch.Status = &status
	}

	return s.repo.UpdateWorkspace(ctx, strings.TrimSpace(id), patch)
}

func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, id string) error {
	if s == nil || s.repo == nil {
		return core_errorx.ErrWorkspaceNotFound
	}
	return s.repo.DeleteWorkspace(ctx, strings.TrimSpace(id))
}
