package entity

// WorkspaceOption is the minimal workspace projection used by the SMTP UI.
type WorkspaceOption struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Status          string `json:"status"`
	DefaultZoneID   string `json:"default_zone_id"`
	DefaultZoneName string `json:"default_zone_name"`
}
