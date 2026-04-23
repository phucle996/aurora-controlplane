package request

type ListHostsRequest struct {
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
	Query    string `form:"q"`
	Status   string `form:"status"`
	ZoneSlug string `form:"zone_slug"`
}
