package core_reqdto

type ListTenantsRequest struct {
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
	Query  string `form:"q"`
	Status string `form:"status"`
}

type CreateTenantRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type UpdateTenantRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
}
