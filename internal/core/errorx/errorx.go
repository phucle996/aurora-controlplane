package core_errorx

import "errors"

var (
	ErrTenantNotFound      = errors.New("core: tenant not found")
	ErrTenantAlreadyExists = errors.New("core: tenant already exists")
	ErrTenantInvalid       = errors.New("core: tenant invalid")

	ErrWorkspaceNotFound      = errors.New("core: workspace not found")
	ErrWorkspaceAlreadyExists = errors.New("core: workspace already exists")
	ErrWorkspaceInvalid       = errors.New("core: workspace invalid")

	ErrDataPlaneNotFound      = errors.New("core: data plane not found")
	ErrDataPlaneAlreadyExists = errors.New("core: data plane already exists")
	ErrDataPlaneInvalid       = errors.New("core: data plane invalid")
	ErrDataPlaneEnrollDenied  = errors.New("core: data plane enroll denied")
	ErrDataPlaneCSRInvalid    = errors.New("core: data plane csr invalid")
	ErrDataPlanePeerInvalid   = errors.New("core: data plane peer invalid")
	ErrDataPlaneUnavailable   = errors.New("core: data plane unavailable")
	ErrZoneNotFound           = errors.New("core: zone not found")
	ErrZoneInUse              = errors.New("core: zone is in use")
	ErrSecretFamilyNotFound   = errors.New("core: secret family not found")

	ErrTenantMemberAlreadyExists    = errors.New("core: tenant member already exists")
	ErrWorkspaceMemberAlreadyExists = errors.New("core: workspace member already exists")
)
