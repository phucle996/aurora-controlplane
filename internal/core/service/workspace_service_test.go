package service

import (
	"context"
	"testing"

	"controlplane/internal/core/domain/entity"
	core_errorx "controlplane/internal/core/errorx"
)

type workspaceRepoStub struct {
	listFilter      entity.WorkspaceListFilter
	createWorkspace *entity.Workspace
	updateID        string
	updatePatch     entity.WorkspacePatch
}

func (s *workspaceRepoStub) ListWorkspaceOptions(context.Context) ([]*entity.WorkspaceOption, error) {
	return []*entity.WorkspaceOption{}, nil
}

func (s *workspaceRepoStub) ListWorkspaces(_ context.Context, filter entity.WorkspaceListFilter) (*entity.WorkspacePage, error) {
	s.listFilter = filter
	return &entity.WorkspacePage{
		Items: []*entity.WorkspaceView{},
		Pagination: entity.Pagination{
			Page:       filter.Page,
			Limit:      filter.Limit,
			Total:      0,
			TotalPages: 0,
		},
	}, nil
}

func (s *workspaceRepoStub) GetWorkspace(context.Context, string) (*entity.WorkspaceView, error) {
	return nil, nil
}

func (s *workspaceRepoStub) CreateWorkspace(_ context.Context, workspace *entity.Workspace) (*entity.WorkspaceView, error) {
	s.createWorkspace = workspace
	return &entity.WorkspaceView{
		ID:       workspace.ID,
		TenantID: workspace.TenantID,
		Name:     workspace.Name,
		Slug:     workspace.Slug,
		Status:   workspace.Status,
	}, nil
}

func (s *workspaceRepoStub) UpdateWorkspace(_ context.Context, id string, patch entity.WorkspacePatch) (*entity.WorkspaceView, error) {
	s.updateID = id
	s.updatePatch = patch
	return &entity.WorkspaceView{ID: id}, nil
}

func (s *workspaceRepoStub) DeleteWorkspace(context.Context, string) error {
	return nil
}

func TestWorkspaceServiceCreateWorkspaceGeneratesSlug(t *testing.T) {
	repo := &workspaceRepoStub{}
	svc := NewWorkspaceService(repo)

	item, err := svc.CreateWorkspace(context.Background(), &entity.Workspace{
		Name:     "  Workspace Prime  ",
		Status:   "",
		TenantID: "tenant_01",
	})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if item == nil || item.ID == "" {
		t.Fatalf("expected generated workspace id, got %#v", item)
	}
	if repo.createWorkspace == nil {
		t.Fatalf("expected repo create to be called")
	}
	if repo.createWorkspace.Slug != "workspace-prime" {
		t.Fatalf("expected slug workspace-prime, got %q", repo.createWorkspace.Slug)
	}
	if repo.createWorkspace.Status != "active" {
		t.Fatalf("expected default status active, got %q", repo.createWorkspace.Status)
	}
	if repo.createWorkspace.TenantID != "tenant_01" {
		t.Fatalf("expected tenant id tenant_01, got %q", repo.createWorkspace.TenantID)
	}
}

func TestWorkspaceServiceUpdateWorkspaceRejectsInvalidStatus(t *testing.T) {
	repo := &workspaceRepoStub{}
	svc := NewWorkspaceService(repo)
	status := "suspended"

	_, err := svc.UpdateWorkspace(context.Background(), "ws_01", entity.WorkspacePatch{
		Status: &status,
	})
	if err == nil {
		t.Fatalf("expected invalid status error")
	}
	if err != core_errorx.ErrWorkspaceInvalid {
		t.Fatalf("expected ErrWorkspaceInvalid, got %v", err)
	}
}

func TestWorkspaceServiceListWorkspacesNormalizesPagination(t *testing.T) {
	repo := &workspaceRepoStub{}
	svc := NewWorkspaceService(repo)

	if _, err := svc.ListWorkspaces(context.Background(), entity.WorkspaceListFilter{
		Page:  0,
		Limit: 999,
	}); err != nil {
		t.Fatalf("list workspaces: %v", err)
	}
	if repo.listFilter.Page != 1 {
		t.Fatalf("expected page 1, got %d", repo.listFilter.Page)
	}
	if repo.listFilter.Limit != 100 {
		t.Fatalf("expected limit 100, got %d", repo.listFilter.Limit)
	}
}
