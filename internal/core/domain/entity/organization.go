package entity

import "time"

// Pagination captures normalized paging metadata.
type Pagination struct {
	Page       int
	Limit      int
	Total      int64
	TotalPages int
}

// TenantListFilter captures tenant list query options.
type TenantListFilter struct {
	Page   int
	Limit  int
	Query  string
	Status string
}

// WorkspaceListFilter captures workspace list query options.
type WorkspaceListFilter struct {
	Page     int
	Limit    int
	Query    string
	Status   string
	TenantID string
}

// TenantPatch contains mutable tenant fields.
type TenantPatch struct {
	Name   *string
	Status *string
}

// WorkspacePatch contains mutable workspace fields.
type WorkspacePatch struct {
	Name   *string
	Status *string
}

// TenantPage is the paginated tenant result set.
type TenantPage struct {
	Items      []*Tenant
	Pagination Pagination
}

// WorkspaceView is the organization-facing workspace projection.
type WorkspaceView struct {
	ID         string
	TenantID   string
	TenantName string
	Name       string
	Slug       string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// WorkspacePage is the paginated workspace result set.
type WorkspacePage struct {
	Items      []*WorkspaceView
	Pagination Pagination
}
