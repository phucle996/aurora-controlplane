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

type SecretKeyVersionRepository struct {
	db *pgxpool.Pool
}

func NewSecretKeyVersionRepository(db *pgxpool.Pool) *SecretKeyVersionRepository {
	return &SecretKeyVersionRepository{db: db}
}

func (r *SecretKeyVersionRepository) ListLatestStatePerFamily(ctx context.Context) ([]*entity.SecretFamilyState, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("core repo: secret key version db is nil")
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			family,
			MAX(rotated_at) AS rotated_at,
			MAX(updated_at) AS updated_at
		FROM core.secret_key_versions
		GROUP BY family
	`)
	if err != nil {
		return nil, fmt.Errorf("core repo: list secret family state: %w", err)
	}
	defer rows.Close()

	values := make([]*entity.SecretFamilyState, 0, 4)
	for rows.Next() {
		var item entity.SecretFamilyState
		if err := rows.Scan(&item.Family, &item.RotatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("core repo: scan secret family state: %w", err)
		}
		values = append(values, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("core repo: iterate secret family state: %w", err)
	}

	return values, nil
}

func (r *SecretKeyVersionRepository) GetFamilyVersions(ctx context.Context, family string) ([]*entity.SecretKeyVersion, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("core repo: secret key version db is nil")
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, family, version, state, secret_ciphertext, expires_at, rotated_at, created_at, updated_at
		FROM core.secret_key_versions
		WHERE family = $1
		ORDER BY CASE state WHEN 'active' THEN 0 ELSE 1 END, version DESC
	`, family)
	if err != nil {
		return nil, fmt.Errorf("core repo: get family versions: %w", err)
	}
	defer rows.Close()

	values := make([]*entity.SecretKeyVersion, 0, 2)
	for rows.Next() {
		var row core_model.SecretKeyVersion
		if err := rows.Scan(
			&row.ID,
			&row.Family,
			&row.Version,
			&row.State,
			&row.SecretCiphertext,
			&row.ExpiresAt,
			&row.RotatedAt,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("core repo: scan family version: %w", err)
		}
		values = append(values, core_model.SecretKeyVersionModelToEntity(&row))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("core repo: iterate family versions: %w", err)
	}

	return values, nil
}

func (r *SecretKeyVersionRepository) CreateInitialActive(ctx context.Context, value *entity.SecretKeyVersion) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("core repo: secret key version db is nil")
	}

	modelValue := core_model.SecretKeyVersionEntityToModel(value)
	if modelValue == nil {
		return fmt.Errorf("core repo: secret key version model is nil")
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO core.secret_key_versions (
			id,
			family,
			version,
			state,
			secret_ciphertext,
			expires_at,
			rotated_at,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (family, state) DO NOTHING
	`, modelValue.ID, modelValue.Family, modelValue.Version, modelValue.State, modelValue.SecretCiphertext, modelValue.ExpiresAt, modelValue.RotatedAt)
	if err != nil {
		return fmt.Errorf("core repo: create initial active secret: %w", err)
	}

	return nil
}

func (r *SecretKeyVersionRepository) RotateFamilyTx(ctx context.Context, family string, nextActive *entity.SecretKeyVersion) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("core repo: secret key version db is nil")
	}
	modelValue := core_model.SecretKeyVersionEntityToModel(nextActive)
	if modelValue == nil {
		return fmt.Errorf("core repo: next active secret model is nil")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("core repo: begin rotate family tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		currentID      string
		currentVersion int64
	)
	err = tx.QueryRow(ctx, `
		SELECT id, version
		FROM core.secret_key_versions
		WHERE family = $1 AND state = 'active'
		FOR UPDATE
	`, family).Scan(&currentID, &currentVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core_errorx.ErrSecretFamilyNotFound
		}
		return fmt.Errorf("core repo: lock active family secret: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM core.secret_key_versions
		WHERE family = $1 AND state = 'previous'
	`, family); err != nil {
		return fmt.Errorf("core repo: delete previous family secret: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE core.secret_key_versions
		SET state = 'previous',
			updated_at = NOW()
		WHERE id = $1
	`, currentID); err != nil {
		return fmt.Errorf("core repo: demote active family secret: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO core.secret_key_versions (
			id,
			family,
			version,
			state,
			secret_ciphertext,
			expires_at,
			rotated_at,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, 'active', $4, $5, $6, NOW(), NOW())
	`, modelValue.ID, family, currentVersion+1, modelValue.SecretCiphertext, modelValue.ExpiresAt, modelValue.RotatedAt); err != nil {
		return fmt.Errorf("core repo: insert new active family secret: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("core repo: commit rotate family tx: %w", err)
	}

	return nil
}
