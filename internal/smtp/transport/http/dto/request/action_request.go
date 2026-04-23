package smtp_reqdto

type GatewayTemplateBindingsRequest struct {
	TemplateIDs []string `json:"template_ids"`
}

type GatewayEndpointBindingsRequest struct {
	EndpointIDs []string `json:"endpoint_ids"`
}
