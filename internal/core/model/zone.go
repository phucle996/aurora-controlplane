package core_model

import (
	"controlplane/internal/core/domain/entity"
	"time"
)

// Zone is a row in core.zones.
type Zone struct {
	ID          string    `db:"id"`
	Slug        string    `db:"slug"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

func ZoneEntityToModel(v *entity.Zone) *Zone {
	if v == nil {
		return nil
	}
	return &Zone{
		ID:          v.ID,
		Slug:        v.Slug,
		Name:        v.Name,
		Description: v.Description,
		CreatedAt:   v.CreatedAt,
	}
}

func ZoneModelToEntity(v *Zone) *entity.Zone {
	if v == nil {
		return nil
	}
	return &entity.Zone{
		ID:          v.ID,
		Slug:        v.Slug,
		Name:        v.Name,
		Description: v.Description,
		CreatedAt:   v.CreatedAt,
	}
}
