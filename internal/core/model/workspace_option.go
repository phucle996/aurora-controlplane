package core_model

import "controlplane/internal/core/domain/entity"

type WorkspaceOption struct {
	ID              string `db:"id"`
	Name            string `db:"name"`
	Slug            string `db:"slug"`
	Status          string `db:"status"`
	DefaultZoneID   string `db:"default_zone_id"`
	DefaultZoneName string `db:"default_zone_name"`
}

func WorkspaceOptionModelToEntity(v *WorkspaceOption) *entity.WorkspaceOption {
	if v == nil {
		return nil
	}

	return &entity.WorkspaceOption{
		ID:              v.ID,
		Name:            v.Name,
		Slug:            v.Slug,
		Status:          v.Status,
		DefaultZoneID:   v.DefaultZoneID,
		DefaultZoneName: v.DefaultZoneName,
	}
}
