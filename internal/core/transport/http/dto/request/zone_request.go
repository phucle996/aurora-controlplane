package core_reqdto

// CreateZoneRequest creates a new zone.
type CreateZoneRequest struct {
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// UpdateZoneDescriptionRequest patches only the description field.
type UpdateZoneDescriptionRequest struct {
	Description *string `json:"description"`
}
