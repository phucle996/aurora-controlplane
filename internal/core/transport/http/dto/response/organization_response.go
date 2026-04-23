package core_resdto

import (
	"time"

	"controlplane/internal/core/domain/entity"
)

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TenantPage struct {
	Items      []*Tenant  `json:"items"`
	Pagination Pagination `json:"pagination"`
}

type Workspace struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	TenantName string    `json:"tenant_name"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WorkspacePage struct {
	Items      []*Workspace `json:"items"`
	Pagination Pagination   `json:"pagination"`
}

func TenantFromEntity(item *entity.Tenant) *Tenant {
	if item == nil {
		return nil
	}
	return &Tenant{
		ID:        item.ID,
		Name:      item.Name,
		Slug:      item.Slug,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func TenantPageFromEntity(page *entity.TenantPage) *TenantPage {
	if page == nil {
		return &TenantPage{
			Items:      []*Tenant{},
			Pagination: Pagination{},
		}
	}

	items := make([]*Tenant, 0, len(page.Items))
	for _, item := range page.Items {
		if mapped := TenantFromEntity(item); mapped != nil {
			items = append(items, mapped)
		}
	}

	return &TenantPage{
		Items: items,
		Pagination: Pagination{
			Page:       page.Pagination.Page,
			Limit:      page.Pagination.Limit,
			Total:      page.Pagination.Total,
			TotalPages: page.Pagination.TotalPages,
		},
	}
}

func WorkspaceFromEntity(item *entity.WorkspaceView) *Workspace {
	if item == nil {
		return nil
	}
	return &Workspace{
		ID:         item.ID,
		TenantID:   item.TenantID,
		TenantName: item.TenantName,
		Name:       item.Name,
		Slug:       item.Slug,
		Status:     item.Status,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
}

func WorkspacePageFromEntity(page *entity.WorkspacePage) *WorkspacePage {
	if page == nil {
		return &WorkspacePage{
			Items:      []*Workspace{},
			Pagination: Pagination{},
		}
	}

	items := make([]*Workspace, 0, len(page.Items))
	for _, item := range page.Items {
		if mapped := WorkspaceFromEntity(item); mapped != nil {
			items = append(items, mapped)
		}
	}

	return &WorkspacePage{
		Items: items,
		Pagination: Pagination{
			Page:       page.Pagination.Page,
			Limit:      page.Pagination.Limit,
			Total:      page.Pagination.Total,
			TotalPages: page.Pagination.TotalPages,
		},
	}
}
