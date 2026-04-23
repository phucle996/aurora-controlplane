package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"controlplane/internal/core/domain/entity"
	core_errorx "controlplane/internal/core/errorx"
	core_model "controlplane/internal/core/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkspaceRepository struct {
	db *pgxpool.Pool
}

func NewWorkspaceRepository(db *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

func (r *WorkspaceRepository) ListWorkspaceOptions(ctx context.Context) ([]*entity.WorkspaceOption, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("core repo: workspace db is nil")
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			w.id,
			w.name,
			w.slug,
			w.status,
			COALESCE(z.id, '') AS default_zone_id,
			COALESCE(z.name, '') AS default_zone_name
		FROM core.workspaces w
		LEFT JOIN core.data_planes dp ON dp.id = w.data_plane_id
		LEFT JOIN core.zones z ON z.id = dp.zone_id
		ORDER BY
			CASE WHEN w.status = 'active' THEN 0 ELSE 1 END,
			w.name ASC,
			w.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("core repo: list workspace options: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.WorkspaceOption, 0)
	for rows.Next() {
		var row core_model.WorkspaceOption
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Slug,
			&row.Status,
			&row.DefaultZoneID,
			&row.DefaultZoneName,
		); err != nil {
			return nil, fmt.Errorf("core repo: scan workspace option: %w", err)
		}
		items = append(items, core_model.WorkspaceOptionModelToEntity(&row))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("core repo: iterate workspace options: %w", err)
	}

	return items, nil
}

func (r *WorkspaceRepository) ListWorkspaces(ctx context.Context, filter entity.WorkspaceListFilter) (*entity.WorkspacePage, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("core repo: workspace db is nil")
	}

	offset := (filter.Page - 1) * filter.Limit
	rows, err := r.db.Query(ctx, `
		SELECT
			w.id,
			COALESCE(w.tenant_id, '') AS tenant_id,
			COALESCE(t.name, '') AS tenant_name,
			w.name,
			w.slug,
			w.status,
			w.created_at,
			w.updated_at,
			COUNT(*) OVER() AS total_count
		FROM core.workspaces w
		LEFT JOIN core.tenants t ON t.id = w.tenant_id
		WHERE ($1 = '' OR w.name ILIKE '%' || $1 || '%' OR w.slug ILIKE '%' || $1 || '%' OR t.name ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR w.status = $2)
		  AND ($3 = '' OR w.tenant_id = $3)
		ORDER BY w.updated_at DESC, w.created_at DESC, w.name ASC
		LIMIT $4 OFFSET $5
	`, filter.Query, filter.Status, filter.TenantID, filter.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("core repo: list workspaces: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.WorkspaceView, 0, filter.Limit)
	var total int64
	for rows.Next() {
		var (
			item       entity.WorkspaceView
			totalCount int64
		)
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.TenantName,
			&item.Name,
			&item.Slug,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, fmt.Errorf("core repo: scan workspace view: %w", err)
		}
		total = totalCount
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("core repo: iterate workspaces: %w", err)
	}

	return &entity.WorkspacePage{
		Items: items,
		Pagination: entity.Pagination{
			Page:       filter.Page,
			Limit:      filter.Limit,
			Total:      total,
			TotalPages: totalPages(total, filter.Limit),
		},
	}, nil
}

func (r *WorkspaceRepository) GetWorkspace(ctx context.Context, id string) (*entity.WorkspaceView, error) {
	if r == nil || r.db == nil {
		return nil, core_errorx.ErrWorkspaceNotFound
	}

	return r.getWorkspaceView(ctx, `
		SELECT
			w.id,
			COALESCE(w.tenant_id, '') AS tenant_id,
			COALESCE(t.name, '') AS tenant_name,
			w.name,
			w.slug,
			w.status,
			w.created_at,
			w.updated_at
		FROM core.workspaces w
		LEFT JOIN core.tenants t ON t.id = w.tenant_id
		WHERE w.id = $1
	`, id)
}

func (r *WorkspaceRepository) CreateWorkspace(ctx context.Context, workspace *entity.Workspace) (*entity.WorkspaceView, error) {
	if r == nil || r.db == nil || workspace == nil {
		return nil, core_errorx.ErrWorkspaceInvalid
	}

	tenantID := nullableText(workspace.TenantID)
	item, err := r.getWorkspaceView(ctx, `
		WITH inserted AS (
			INSERT INTO core.workspaces (
				id,
				tenant_id,
				data_plane_id,
				name,
				slug,
				status,
				created_at,
				updated_at
			)
			VALUES ($1, $2, NULL, $3, $4, $5, NOW(), NOW())
			RETURNING id, tenant_id, name, slug, status, created_at, updated_at
		)
		SELECT
			i.id,
			COALESCE(i.tenant_id, '') AS tenant_id,
			COALESCE(t.name, '') AS tenant_name,
			i.name,
			i.slug,
			i.status,
			i.created_at,
			i.updated_at
		FROM inserted i
		LEFT JOIN core.tenants t ON t.id = i.tenant_id
	`, workspace.ID, tenantID, workspace.Name, workspace.Slug, workspace.Status)
	if err != nil {
		if workspaceConflictErr(err) {
			return nil, core_errorx.ErrWorkspaceAlreadyExists
		}
		if workspaceTenantFKErr(err) {
			return nil, core_errorx.ErrTenantNotFound
		}
		return nil, err
	}

	return item, nil
}

func (r *WorkspaceRepository) UpdateWorkspace(ctx context.Context, id string, patch entity.WorkspacePatch) (*entity.WorkspaceView, error) {
	if r == nil || r.db == nil {
		return nil, core_errorx.ErrWorkspaceNotFound
	}

	item, err := r.getWorkspaceView(ctx, `
		WITH updated AS (
			UPDATE core.workspaces
			SET
				name = CASE WHEN $2::text IS NULL THEN name ELSE $2::text END,
				status = CASE WHEN $3::text IS NULL THEN status ELSE $3::text END
			WHERE id = $1
			RETURNING id, tenant_id, name, slug, status, created_at, updated_at
		)
		SELECT
			u.id,
			COALESCE(u.tenant_id, '') AS tenant_id,
			COALESCE(t.name, '') AS tenant_name,
			u.name,
			u.slug,
			u.status,
			u.created_at,
			u.updated_at
		FROM updated u
		LEFT JOIN core.tenants t ON t.id = u.tenant_id
	`, id, textOrNil(patch.Name), textOrNil(patch.Status))
	if err != nil {
		if errors.Is(err, core_errorx.ErrWorkspaceNotFound) {
			return nil, err
		}
		if workspaceConflictErr(err) {
			return nil, core_errorx.ErrWorkspaceAlreadyExists
		}
		return nil, err
	}

	return item, nil
}

func (r *WorkspaceRepository) DeleteWorkspace(ctx context.Context, id string) error {
	if r == nil || r.db == nil {
		return core_errorx.ErrWorkspaceNotFound
	}

	tag, err := r.db.Exec(ctx, `DELETE FROM core.workspaces WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("core repo: delete workspace: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core_errorx.ErrWorkspaceNotFound
	}

	return nil
}

func (r *WorkspaceRepository) getWorkspaceView(ctx context.Context, query string, args ...any) (*entity.WorkspaceView, error) {
	var (
		item       entity.WorkspaceView
		tenantID   sql.NullString
		tenantName sql.NullString
	)

	err := r.db.QueryRow(ctx, query, args...).Scan(
		&item.ID,
		&tenantID,
		&tenantName,
		&item.Name,
		&item.Slug,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core_errorx.ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("core repo: get workspace view: %w", err)
	}

	if tenantID.Valid {
		item.TenantID = tenantID.String
	}
	if tenantName.Valid {
		item.TenantName = tenantName.String
	}

	return &item, nil
}

func workspaceConflictErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "workspaces_tenant_id_slug_key"
}

func workspaceTenantFKErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "workspaces_tenant_id_fkey"
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
