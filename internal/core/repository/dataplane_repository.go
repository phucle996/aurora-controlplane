package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"controlplane/internal/core/domain/entity"
	core_errorx "controlplane/internal/core/errorx"
	core_model "controlplane/internal/core/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DataPlaneRepository struct {
	db *pgxpool.Pool
}

func NewDataPlaneRepository(db *pgxpool.Pool) *DataPlaneRepository {
	return &DataPlaneRepository{db: db}
}

func (r *DataPlaneRepository) SaveEnrollment(ctx context.Context, dataPlane *entity.DataPlane) (*entity.DataPlane, error) {
	if r == nil || r.db == nil {
		return nil, core_errorx.ErrDataPlaneUnavailable
	}

	dbDataPlane := core_model.DataPlaneEntityToModel(dataPlane)
	if dbDataPlane == nil {
		return nil, core_errorx.ErrDataPlaneInvalid
	}

	now := time.Now().UTC()
	dbDataPlane.LastSeenAt = &now
	if dbDataPlane.Status == "" {
		dbDataPlane.Status = "healthy"
	}

	var saved core_model.DataPlane
	err := r.db.QueryRow(ctx, `
		INSERT INTO core.data_planes (
			id,
			node_key,
			name,
			zone_id,
			grpc_endpoint,
			version,
			cert_serial,
			cert_not_after,
			status,
			last_seen_at,
			created_at,
			updated_at
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			NOW(),
			NOW()
		)
		ON CONFLICT (node_key) DO UPDATE SET
			name = EXCLUDED.name,
			zone_id = EXCLUDED.zone_id,
			grpc_endpoint = EXCLUDED.grpc_endpoint,
			version = EXCLUDED.version,
			cert_serial = EXCLUDED.cert_serial,
			cert_not_after = EXCLUDED.cert_not_after,
			status = EXCLUDED.status,
			last_seen_at = EXCLUDED.last_seen_at,
			updated_at = NOW()
		RETURNING
			id,
			node_key,
			name,
			zone_id,
			grpc_endpoint,
			version,
			cert_serial,
			cert_not_after,
			status,
			last_seen_at,
			created_at,
			updated_at
	`,
		dbDataPlane.ID,
		dbDataPlane.NodeKey,
		dbDataPlane.Name,
		dbDataPlane.ZoneID,
		dbDataPlane.GRPCEndpoint,
		dbDataPlane.Version,
		dbDataPlane.CertSerial,
		dbDataPlane.CertNotAfter,
		dbDataPlane.Status,
		dbDataPlane.LastSeenAt,
	).Scan(
		&saved.ID,
		&saved.NodeKey,
		&saved.Name,
		&saved.ZoneID,
		&saved.GRPCEndpoint,
		&saved.Version,
		&saved.CertSerial,
		&saved.CertNotAfter,
		&saved.Status,
		&saved.LastSeenAt,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("core repo: save enrollment: %w", err)
	}

	return core_model.DataPlaneModelToEntity(&saved), nil
}

func (r *DataPlaneRepository) GetByID(ctx context.Context, id string) (*entity.DataPlane, error) {
	if r == nil || r.db == nil {
		return nil, core_errorx.ErrDataPlaneUnavailable
	}

	var row core_model.DataPlane
	err := r.db.QueryRow(ctx, `
		SELECT
			id,
			node_key,
			name,
			zone_id,
			grpc_endpoint,
			version,
			cert_serial,
			cert_not_after,
			status,
			last_seen_at,
			created_at,
			updated_at
		FROM core.data_planes
		WHERE id = $1
	`, id).Scan(
		&row.ID,
		&row.NodeKey,
		&row.Name,
		&row.ZoneID,
		&row.GRPCEndpoint,
		&row.Version,
		&row.CertSerial,
		&row.CertNotAfter,
		&row.Status,
		&row.LastSeenAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core_errorx.ErrDataPlaneNotFound
		}
		return nil, fmt.Errorf("core repo: get data plane by id: %w", err)
	}

	return core_model.DataPlaneModelToEntity(&row), nil
}

func (r *DataPlaneRepository) GetByNodeKey(ctx context.Context, nodeKey string) (*entity.DataPlane, error) {
	if r == nil || r.db == nil {
		return nil, core_errorx.ErrDataPlaneUnavailable
	}

	var row core_model.DataPlane
	err := r.db.QueryRow(ctx, `
		SELECT
			id,
			node_key,
			name,
			zone_id,
			grpc_endpoint,
			version,
			cert_serial,
			cert_not_after,
			status,
			last_seen_at,
			created_at,
			updated_at
		FROM core.data_planes
		WHERE node_key = $1
	`, nodeKey).Scan(
		&row.ID,
		&row.NodeKey,
		&row.Name,
		&row.ZoneID,
		&row.GRPCEndpoint,
		&row.Version,
		&row.CertSerial,
		&row.CertNotAfter,
		&row.Status,
		&row.LastSeenAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core_errorx.ErrDataPlaneNotFound
		}
		return nil, fmt.Errorf("core repo: get data plane by node key: %w", err)
	}

	return core_model.DataPlaneModelToEntity(&row), nil
}

func (r *DataPlaneRepository) UpdateHeartbeat(ctx context.Context, id, grpcEndpoint, version, status string, seenAt time.Time) error {
	if r == nil || r.db == nil {
		return core_errorx.ErrDataPlaneUnavailable
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE core.data_planes
		SET
			grpc_endpoint = $2,
			version = $3,
			status = $4,
			last_seen_at = $5,
			updated_at = NOW()
		WHERE id = $1
	`, id, grpcEndpoint, version, status, seenAt)
	if err != nil {
		return fmt.Errorf("core repo: update heartbeat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core_errorx.ErrDataPlaneNotFound
	}
	return nil
}

func (r *DataPlaneRepository) ListHealthyByZoneID(ctx context.Context, zoneID string) ([]*entity.DataPlane, error) {
	if r == nil || r.db == nil {
		return nil, core_errorx.ErrDataPlaneUnavailable
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			node_key,
			name,
			zone_id,
			grpc_endpoint,
			version,
			cert_serial,
			cert_not_after,
			status,
			last_seen_at,
			created_at,
			updated_at
		FROM core.data_planes
		WHERE zone_id = $1
		  AND status = 'healthy'
		ORDER BY last_seen_at DESC NULLS LAST, created_at ASC
	`, zoneID)
	if err != nil {
		return nil, fmt.Errorf("core repo: list healthy dataplanes by zone: %w", err)
	}
	defer rows.Close()

	dataPlanes := make([]*entity.DataPlane, 0)
	for rows.Next() {
		var row core_model.DataPlane
		if err := rows.Scan(
			&row.ID,
			&row.NodeKey,
			&row.Name,
			&row.ZoneID,
			&row.GRPCEndpoint,
			&row.Version,
			&row.CertSerial,
			&row.CertNotAfter,
			&row.Status,
			&row.LastSeenAt,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("core repo: scan healthy dataplane by zone: %w", err)
		}
		dataPlanes = append(dataPlanes, core_model.DataPlaneModelToEntity(&row))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("core repo: iterate healthy dataplanes by zone: %w", err)
	}

	return dataPlanes, nil
}

func (r *DataPlaneRepository) MarkStaleBefore(ctx context.Context, staleBefore time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, core_errorx.ErrDataPlaneUnavailable
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE core.data_planes
		SET
			status = 'stale',
			updated_at = NOW()
		WHERE
			last_seen_at IS NOT NULL
			AND last_seen_at < $1
			AND status <> 'stale'
	`, staleBefore)
	if err != nil {
		return 0, fmt.Errorf("core repo: mark stale: %w", err)
	}

	return tag.RowsAffected(), nil
}
