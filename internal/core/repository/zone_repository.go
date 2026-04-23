package repository

import (
	"context"
	"errors"
	"fmt"

	"controlplane/internal/core/domain/entity"
	core_errorx "controlplane/internal/core/errorx"
	core_model "controlplane/internal/core/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ZoneRepository struct {
	db *pgxpool.Pool
}

func NewZoneRepository(db *pgxpool.Pool) *ZoneRepository {
	return &ZoneRepository{db: db}
}

func (r *ZoneRepository) ListZones(ctx context.Context) ([]*entity.Zone, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("core repo: zone db is nil")
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, slug, name, description, created_at
		FROM core.zones
		ORDER BY name ASC, created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("core repo: list zones: %w", err)
	}
	defer rows.Close()

	zones := make([]*entity.Zone, 0)
	for rows.Next() {
		var row core_model.Zone
		if err := rows.Scan(&row.ID, &row.Slug, &row.Name, &row.Description, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("core repo: scan zone: %w", err)
		}
		zones = append(zones, core_model.ZoneModelToEntity(&row))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("core repo: iterate zones: %w", err)
	}

	return zones, nil
}

func (r *ZoneRepository) GetZone(ctx context.Context, id string) (*entity.Zone, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("core repo: zone db is nil")
	}

	var row core_model.Zone
	err := r.db.QueryRow(ctx, `
		SELECT id, slug, name, description, created_at
		FROM core.zones
		WHERE id = $1
	`, id).Scan(&row.ID, &row.Slug, &row.Name, &row.Description, &row.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core_errorx.ErrZoneNotFound
		}
		return nil, fmt.Errorf("core repo: get zone: %w", err)
	}

	return core_model.ZoneModelToEntity(&row), nil
}

func (r *ZoneRepository) GetZoneBySlug(ctx context.Context, slug string) (*entity.Zone, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("core repo: zone db is nil")
	}

	var row core_model.Zone
	err := r.db.QueryRow(ctx, `
		SELECT id, slug, name, description, created_at
		FROM core.zones
		WHERE slug = $1
	`, slug).Scan(&row.ID, &row.Slug, &row.Name, &row.Description, &row.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core_errorx.ErrZoneNotFound
		}
		return nil, fmt.Errorf("core repo: get zone by slug: %w", err)
	}

	return core_model.ZoneModelToEntity(&row), nil
}

func (r *ZoneRepository) CreateZone(ctx context.Context, zone *entity.Zone) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("core repo: zone db is nil")
	}

	dbZone := core_model.ZoneEntityToModel(zone)
	if dbZone == nil {
		return fmt.Errorf("core repo: zone model is nil")
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO core.zones (id, slug, name, description, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, dbZone.ID, dbZone.Slug, dbZone.Name, dbZone.Description)
	if err != nil {
		return fmt.Errorf("core repo: create zone: %w", err)
	}

	return nil
}

func (r *ZoneRepository) UpdateZoneDescription(ctx context.Context, id, description string) (*entity.Zone, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("core repo: zone db is nil")
	}

	var row core_model.Zone
	err := r.db.QueryRow(ctx, `
		UPDATE core.zones
		SET description = $2
		WHERE id = $1
		RETURNING id, slug, name, description, created_at
	`, id, description).Scan(&row.ID, &row.Slug, &row.Name, &row.Description, &row.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core_errorx.ErrZoneNotFound
		}
		return nil, fmt.Errorf("core repo: update zone description: %w", err)
	}

	return core_model.ZoneModelToEntity(&row), nil
}

func (r *ZoneRepository) DeleteZone(ctx context.Context, id string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("core repo: zone db is nil")
	}

	tag, err := r.db.Exec(ctx, `DELETE FROM core.zones WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("core repo: delete zone: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core_errorx.ErrZoneNotFound
	}

	return nil
}

func (r *ZoneRepository) CountDataPlanesByZoneID(ctx context.Context, zoneID string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("core repo: zone db is nil")
	}

	var count int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(1)
		FROM core.data_planes
		WHERE zone_id = $1
	`, zoneID).Scan(&count); err != nil {
		return 0, fmt.Errorf("core repo: count dataplanes by zone: %w", err)
	}

	return count, nil
}
