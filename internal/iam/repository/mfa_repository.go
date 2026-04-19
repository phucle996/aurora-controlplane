package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"controlplane/internal/iam/domain/entity"
	iam_errorx "controlplane/internal/iam/errorx"
	iam_model "controlplane/internal/iam/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MfaRepository handles MFA settings and recovery code persistence.
type MfaRepository struct {
	db *pgxpool.Pool
}

func NewMfaRepository(db *pgxpool.Pool) *MfaRepository {
	return &MfaRepository{db: db}
}

// ── Settings ──────────────────────────────────────────────────────────────────

func (r *MfaRepository) ListEnabled(ctx context.Context, userID string) ([]*entity.MfaSetting, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, mfa_type, device_name, is_primary,
		       secret_encrypted, is_enabled, created_at, updated_at
		FROM iam.mfa_settings
		WHERE user_id = $1 AND is_enabled = true
		ORDER BY is_primary DESC, created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("mfa repo: list enabled: %w", err)
	}
	defer rows.Close()

	var result []*entity.MfaSetting
	for rows.Next() {
		var m iam_model.MfaSetting
		var deviceName *string
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.MfaType, &deviceName,
			&m.IsPrimary, &m.SecretEncrypted, &m.IsEnabled,
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("mfa repo: scan setting: %w", err)
		}
		if deviceName != nil {
			m.DeviceName = deviceName
		} else {
			empty := ""
			m.DeviceName = &empty
		}
		result = append(result, iam_model.MfaSettingModelToEntity(&m))
	}

	return result, rows.Err()
}

func (r *MfaRepository) GetByID(ctx context.Context, id string) (*entity.MfaSetting, error) {
	if r == nil || r.db == nil {
		return nil, iam_errorx.ErrMfaSettingNotFound
	}

	var m iam_model.MfaSetting
	var deviceName *string

	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, mfa_type, device_name, is_primary,
		       secret_encrypted, is_enabled, created_at, updated_at
		FROM iam.mfa_settings WHERE id = $1`, id,
	).Scan(
		&m.ID, &m.UserID, &m.MfaType, &deviceName,
		&m.IsPrimary, &m.SecretEncrypted, &m.IsEnabled,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, iam_errorx.ErrMfaSettingNotFound
		}
		return nil, fmt.Errorf("mfa repo: get by id: %w", err)
	}

	if deviceName != nil {
		m.DeviceName = deviceName
	} else {
		empty := ""
		m.DeviceName = &empty
	}

	return iam_model.MfaSettingModelToEntity(&m), nil
}

func (r *MfaRepository) GetByUserAndType(ctx context.Context, userID, mfaType string) (*entity.MfaSetting, error) {
	if r == nil || r.db == nil {
		return nil, iam_errorx.ErrMfaSettingNotFound
	}

	var m iam_model.MfaSetting
	var deviceName *string

	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, mfa_type, device_name, is_primary,
		       secret_encrypted, is_enabled, created_at, updated_at
		FROM iam.mfa_settings
		WHERE user_id = $1 AND mfa_type = $2 AND is_enabled = true
		LIMIT 1`, userID, mfaType,
	).Scan(
		&m.ID, &m.UserID, &m.MfaType, &deviceName,
		&m.IsPrimary, &m.SecretEncrypted, &m.IsEnabled,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, iam_errorx.ErrMfaSettingNotFound
		}
		return nil, fmt.Errorf("mfa repo: get by user+type: %w", err)
	}

	if deviceName != nil {
		m.DeviceName = deviceName
	} else {
		empty := ""
		m.DeviceName = &empty
	}

	return iam_model.MfaSettingModelToEntity(&m), nil
}

func (r *MfaRepository) Create(ctx context.Context, setting *entity.MfaSetting) error {
	if r == nil || r.db == nil {
		return iam_errorx.ErrMfaEnrollFailed
	}

	m := iam_model.MfaSettingEntityToModel(setting)
	if m == nil {
		return iam_errorx.ErrMfaEnrollFailed
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO iam.mfa_settings (
			id, user_id, mfa_type, device_name, is_primary,
			secret_encrypted, is_enabled, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,NOW(),NOW())`,
		m.ID, m.UserID, m.MfaType, m.DeviceName,
		m.IsPrimary, m.SecretEncrypted, m.IsEnabled,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return iam_errorx.ErrMfaEnrollFailed
		}
		return fmt.Errorf("mfa repo: create: %w", err)
	}

	return nil
}

func (r *MfaRepository) UpdateEnabled(ctx context.Context, id string, enabled bool) error {
	if r == nil || r.db == nil {
		return iam_errorx.ErrMfaSettingNotFound
	}

	tag, err := r.db.Exec(ctx,
		`UPDATE iam.mfa_settings SET is_enabled = $2, updated_at = NOW() WHERE id = $1`,
		id, enabled,
	)
	if err != nil {
		return fmt.Errorf("mfa repo: update enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return iam_errorx.ErrMfaSettingNotFound
	}

	return nil
}

func (r *MfaRepository) SetPrimary(ctx context.Context, userID, settingID string) error {
	if r == nil || r.db == nil {
		return iam_errorx.ErrMfaSettingNotFound
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mfa repo: begin set-primary tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Clear all primaries first.
	if _, err := tx.Exec(ctx,
		`UPDATE iam.mfa_settings SET is_primary = false WHERE user_id = $1`, userID,
	); err != nil {
		return fmt.Errorf("mfa repo: clear primary: %w", err)
	}

	tag, err := tx.Exec(ctx,
		`UPDATE iam.mfa_settings SET is_primary = true, updated_at = NOW() WHERE id = $1 AND user_id = $2`,
		settingID, userID,
	)
	if err != nil {
		return fmt.Errorf("mfa repo: set primary: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return iam_errorx.ErrMfaSettingNotFound
	}

	return tx.Commit(ctx)
}

func (r *MfaRepository) Delete(ctx context.Context, id, userID string) error {
	if r == nil || r.db == nil {
		return iam_errorx.ErrMfaSettingNotFound
	}

	tag, err := r.db.Exec(ctx,
		`DELETE FROM iam.mfa_settings WHERE id = $1 AND user_id = $2`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("mfa repo: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return iam_errorx.ErrMfaSettingNotFound
	}

	return nil
}

// ── Recovery Codes ────────────────────────────────────────────────────────────

func (r *MfaRepository) ReplaceRecoveryCodes(ctx context.Context, codes []*entity.RecoveryCode) error {
	if r == nil || r.db == nil || len(codes) == 0 {
		return iam_errorx.ErrMfaEnrollFailed
	}

	userID := codes[0].UserID

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mfa repo: begin recovery-codes tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Delete old codes.
	if _, err := tx.Exec(ctx,
		`DELETE FROM iam.recovery_codes WHERE user_id = $1`, userID,
	); err != nil {
		return fmt.Errorf("mfa repo: delete old recovery codes: %w", err)
	}

	// Insert new codes.
	for _, c := range codes {
		m := iam_model.RecoveryCodeEntityToModel(c)
		if _, err := tx.Exec(ctx, `
			INSERT INTO iam.recovery_codes (id, user_id, code_hash, is_used, created_at)
			VALUES ($1,$2,$3,false,NOW())`,
			m.ID, m.UserID, m.CodeHash,
		); err != nil {
			return fmt.Errorf("mfa repo: insert recovery code: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *MfaRepository) GetUnusedRecoveryCode(ctx context.Context, userID, codeHash string) (*entity.RecoveryCode, error) {
	if r == nil || r.db == nil {
		return nil, iam_errorx.ErrMfaCodeInvalid
	}

	var m iam_model.RecoveryCode
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, code_hash, is_used, used_at, created_at
		FROM iam.recovery_codes
		WHERE user_id = $1 AND code_hash = $2 AND is_used = false
		LIMIT 1`, userID, codeHash,
	).Scan(&m.ID, &m.UserID, &m.CodeHash, &m.IsUsed, &m.UsedAt, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, iam_errorx.ErrMfaCodeInvalid
		}
		return nil, fmt.Errorf("mfa repo: get recovery code: %w", err)
	}

	return iam_model.RecoveryCodeModelToEntity(&m), nil
}

func (r *MfaRepository) MarkRecoveryCodeUsed(ctx context.Context, id string) error {
	if r == nil || r.db == nil {
		return iam_errorx.ErrMfaCodeInvalid
	}

	usedAt := time.Now().UTC()
	tag, err := r.db.Exec(ctx,
		`UPDATE iam.recovery_codes SET is_used = true, used_at = $2 WHERE id = $1`,
		id, usedAt,
	)
	if err != nil {
		return fmt.Errorf("mfa repo: mark recovery code used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return iam_errorx.ErrMfaCodeInvalid
	}

	return nil
}
