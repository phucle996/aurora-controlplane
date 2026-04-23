package entity

import "time"

// Tenant is the top-level organization boundary.
type Tenant struct {
	ID        string
	Name      string
	Slug      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TenantMember maps a user into a tenant with a role from IAM.
type TenantMember struct {
	TenantID string
	UserID   string
	RoleID   string
	JoinedAt time.Time
}
