package entity

import "time"

// Workspace is the tenant-scoped working boundary scheduled onto a data plane.
type Workspace struct {
	ID          string
	TenantID    string
	DataPlaneID string
	Name        string
	Slug        string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkspaceMember maps a user into a workspace with an IAM role.
type WorkspaceMember struct {
	WorkspaceID string
	UserID      string
	RoleID      string
	JoinedAt    time.Time
}
