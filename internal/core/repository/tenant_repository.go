package repository

import (
	"context"
	"errors"
	"fmt"

	"controlplane/internal/core/domain/entity"
	core_errorx "controlplane/internal/core/errorx"
	core_model "controlplane/internal/core/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantRepository struct {
	db *pgxpool.Pool
}

func NewTenantRepository(db *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) ListTenants(ctx context.Context, filter entity.TenantListFilter) (*entity.TenantPage, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("core repo: tenant db is nil")
	}

	offset := (filter.Page - 1) * filter.Limit
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			name,
			slug,
			status,
			created_at,
			updated_at,
			COUNT(*) OVER() AS total_count
		FROM core.tenants
		WHERE ($1 = '' OR name ILIKE '%' || $1 || '%' OR slug ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR status = $2)
		ORDER BY updated_at DESC, created_at DESC, name ASC
		LIMIT $3 OFFSET $4
	`, filter.Query, filter.Status, filter.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("core repo: list tenants: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.Tenant, 0, filter.Limit)
	var total int64
	for rows.Next() {
		var (
			row        core_model.Tenant
			totalCount int64
		)
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Slug,
			&row.Status,
			&row.CreatedAt,
			&row.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, fmt.Errorf("core repo: scan tenant: %w", err)
		}
		total = totalCount
		items = append(items, core_model.TenantModelToEntity(&row))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("core repo: iterate tenants: %w", err)
	}

	return &entity.TenantPage{
		Items: items,
		Pagination: entity.Pagination{
			Page:       filter.Page,
			Limit:      filter.Limit,
			Total:      total,
			TotalPages: totalPages(total, filter.Limit),
		},
	}, nil
}

func (r *TenantRepository) GetTenant(ctx context.Context, id string) (*entity.Tenant, error) {
	if r == nil || r.db == nil {
		return nil, core_errorx.ErrTenantNotFound
	}

	var row core_model.Tenant
	err := r.db.QueryRow(ctx, `
		SELECT id, name, slug, status, created_at, updated_at
		FROM core.tenants
		WHERE id = $1
	`, id).Scan(
		&row.ID,
		&row.Name,
		&row.Slug,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core_errorx.ErrTenantNotFound
		}
		return nil, fmt.Errorf("core repo: get tenant: %w", err)
	}

	return core_model.TenantModelToEntity(&row), nil
}

func (r *TenantRepository) CreateTenant(ctx context.Context, tenant *entity.Tenant) (*entity.Tenant, error) {
	if r == nil || r.db == nil || tenant == nil {
		return nil, core_errorx.ErrTenantInvalid
	}

	var row core_model.Tenant
	err := r.db.QueryRow(ctx, `
		INSERT INTO core.tenants (id, name, slug, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, name, slug, status, created_at, updated_at
	`, tenant.ID, tenant.Name, tenant.Slug, tenant.Status).Scan(
		&row.ID,
		&row.Name,
		&row.Slug,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if tenantConflictErr(err) {
			return nil, core_errorx.ErrTenantAlreadyExists
		}
		return nil, fmt.Errorf("core repo: create tenant: %w", err)
	}

	return core_model.TenantModelToEntity(&row), nil
}

func (r *TenantRepository) UpdateTenant(ctx context.Context, id string, patch entity.TenantPatch) (*entity.Tenant, error) {
	if r == nil || r.db == nil {
		return nil, core_errorx.ErrTenantNotFound
	}

	var row core_model.Tenant
	err := r.db.QueryRow(ctx, `
		UPDATE core.tenants
		SET
			name = CASE WHEN $2::text IS NULL THEN name ELSE $2::text END,
			status = CASE WHEN $3::text IS NULL THEN status ELSE $3::text END
		WHERE id = $1
		RETURNING id, name, slug, status, created_at, updated_at
	`, id, textOrNil(patch.Name), textOrNil(patch.Status)).Scan(
		&row.ID,
		&row.Name,
		&row.Slug,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core_errorx.ErrTenantNotFound
		}
		if tenantConflictErr(err) {
			return nil, core_errorx.ErrTenantAlreadyExists
		}
		return nil, fmt.Errorf("core repo: update tenant: %w", err)
	}

	return core_model.TenantModelToEntity(&row), nil
}

func (r *TenantRepository) DeleteTenant(ctx context.Context, id string) error {
	if r == nil || r.db == nil {
		return core_errorx.ErrTenantNotFound
	}

	tag, err := r.db.Exec(ctx, `DELETE FROM core.tenants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("core repo: delete tenant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core_errorx.ErrTenantNotFound
	}

	return nil
}

func tenantConflictErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "tenants_slug_key"
}

func totalPages(total int64, limit int) int {
	if total <= 0 || limit <= 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}

func textOrNil(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
