package core_reqdto

type ListWorkspacesRequest struct {
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
	Query    string `form:"q"`
	Status   string `form:"status"`
	TenantID string `form:"tenant_id"`
}

type CreateWorkspaceRequest struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	TenantID string `json:"tenant_id"`
}

type UpdateWorkspaceRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
}
