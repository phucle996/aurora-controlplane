package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"controlplane/internal/iam/domain/entity"
	iam_errorx "controlplane/internal/iam/errorx"
	iam_model "controlplane/internal/iam/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	activationRoleName     = "user"
	activationStatusReason = "activated account"
)

// UserRepository persists IAM identity data.
type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// CreatePendingAccount inserts the user and profile rows in one transaction.
func (r *UserRepository) CreatePendingAccount(ctx context.Context, user *entity.User, profile *entity.UserProfile) error {
	if r == nil || r.db == nil {
		return iam_errorx.ErrRegistrationFailed
	}

	dbUser := iam_model.UserEntityToModel(user)
	dbProfile := iam_model.UserProfileEntityToModel(profile)
	if dbUser == nil || dbProfile == nil {
		return iam_errorx.ErrRegistrationFailed
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("iam repo: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(
		ctx,
		`INSERT INTO iam.users (
			id, username, email, phone, password_hash, security_level, status, status_reason, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`,
		dbUser.ID,
		dbUser.Username,
		dbUser.Email,
		dbUser.Phone,
		dbUser.PasswordHash,
		dbUser.SecurityLevel,
		dbUser.Status,
		dbUser.StatusReason,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_username_key":
				return iam_errorx.ErrUsernameAlreadyExists
			case "users_email_key":
				return iam_errorx.ErrEmailAlreadyExists
			case "users_phone_key":
				return iam_errorx.ErrPhoneAlreadyExists
			default:
				return iam_errorx.ErrRegistrationFailed
			}
		}
		return err
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO iam.user_profiles (
			id, user_id, fullname, avatar_url, bio, timezone, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
		dbProfile.ID,
		dbProfile.UserID,
		dbProfile.Fullname,
		dbProfile.AvatarURL,
		dbProfile.Bio,
		dbProfile.Timezone,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "user_profiles_user_id_key":
				return iam_errorx.ErrRegistrationFailed
			case "users_username_key":
				return iam_errorx.ErrUsernameAlreadyExists
			case "users_email_key":
				return iam_errorx.ErrEmailAlreadyExists
			case "users_phone_key":
				return iam_errorx.ErrPhoneAlreadyExists
			default:
				return iam_errorx.ErrRegistrationFailed
			}
		}
		return fmt.Errorf("iam repo: registration db error: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("iam repo: commit tx: %w", err)
	}

	return nil
}

// GetByEmail returns a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	if r == nil || r.db == nil {
		return nil, iam_errorx.ErrUserNotFound
	}

	var (
		row    iam_model.User
		phone  sql.NullString
		reason sql.NullString
		role   string
	)

	err := r.db.QueryRow(ctx, `SELECT
		id, username, email, phone, password_hash, security_level, status, status_reason,
		COALESCE((SELECT rl.name
			FROM iam.user_roles ur
			JOIN iam.roles rl ON rl.id = ur.role_id
			WHERE ur.user_id = u.id
			ORDER BY rl.name
			LIMIT 1), ''),
		created_at, updated_at
		FROM iam.users u
		WHERE email = $1`, email).Scan(
		&row.ID,
		&row.Username,
		&row.Email,
		&phone,
		&row.PasswordHash,
		&row.SecurityLevel,
		&row.Status,
		&reason,
		&role,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, iam_errorx.ErrUserNotFound
		}
		return nil, fmt.Errorf("iam repo: get user by email: %w", err)
	}

	phoneValue := ""
	if phone.Valid {
		phoneValue = phone.String
	}
	reasonValue := ""
	if reason.Valid {
		reasonValue = reason.String
	}
	roleValue := role

	row.Phone = &phoneValue
	row.StatusReason = &reasonValue
	row.Role = &roleValue

	return iam_model.UserModelToEntity(&row), nil
}

// GetByUsername returns a user by username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	if r == nil || r.db == nil {
		return nil, iam_errorx.ErrUserNotFound
	}

	var (
		row    iam_model.User
		phone  sql.NullString
		reason sql.NullString
		role   string
	)

	err := r.db.QueryRow(ctx, `SELECT
		id, username, email, phone, password_hash, security_level, status, status_reason,
		COALESCE((SELECT rl.name
			FROM iam.user_roles ur
			JOIN iam.roles rl ON rl.id = ur.role_id
			WHERE ur.user_id = u.id
			ORDER BY rl.name
			LIMIT 1), ''),
		created_at, updated_at
		FROM iam.users u
		WHERE username = $1`, username).Scan(
		&row.ID,
		&row.Username,
		&row.Email,
		&phone,
		&row.PasswordHash,
		&row.SecurityLevel,
		&row.Status,
		&reason,
		&role,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, iam_errorx.ErrUserNotFound
		}
		return nil, fmt.Errorf("iam repo: get user by username: %w", err)
	}

	phoneValue := ""
	if phone.Valid {
		phoneValue = phone.String
	}
	reasonValue := ""
	if reason.Valid {
		reasonValue = reason.String
	}
	roleValue := role

	row.Phone = &phoneValue
	row.StatusReason = &reasonValue
	row.Role = &roleValue

	return iam_model.UserModelToEntity(&row), nil
}

// GetByID returns a user by id.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	if r == nil || r.db == nil {
		return nil, iam_errorx.ErrUserNotFound
	}

	var (
		row    iam_model.User
		phone  sql.NullString
		reason sql.NullString
		role   string
	)

	err := r.db.QueryRow(ctx, `SELECT
		id, username, email, phone, password_hash, security_level, status, status_reason,
		COALESCE((SELECT rl.name
			FROM iam.user_roles ur
			JOIN iam.roles rl ON rl.id = ur.role_id
			WHERE ur.user_id = u.id
			ORDER BY rl.name
			LIMIT 1), ''),
		created_at, updated_at
		FROM iam.users u
		WHERE id = $1`, id).Scan(
		&row.ID,
		&row.Username,
		&row.Email,
		&phone,
		&row.PasswordHash,
		&row.SecurityLevel,
		&row.Status,
		&reason,
		&role,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, iam_errorx.ErrUserNotFound
		}
		return nil, fmt.Errorf("iam repo: get user by id: %w", err)
	}

	phoneValue := ""
	if phone.Valid {
		phoneValue = phone.String
	}
	reasonValue := ""
	if reason.Valid {
		reasonValue = reason.String
	}
	roleValue := role

	row.Phone = &phoneValue
	row.StatusReason = &reasonValue
	row.Role = &roleValue

	return iam_model.UserModelToEntity(&row), nil
}

// GetProfileByUserID returns a user profile for resend/login flows.
func (r *UserRepository) GetProfileByUserID(ctx context.Context, userID string) (*entity.UserProfile, error) {
	if r == nil || r.db == nil {
		return nil, iam_errorx.ErrProfileNotFound
	}

	var (
		row       iam_model.UserProfile
		bio       sql.NullString
		avatarURL sql.NullString
	)

	err := r.db.QueryRow(ctx, `SELECT
		id, user_id, fullname, bio, avatar_url, timezone, created_at, updated_at
		FROM iam.user_profiles
		WHERE user_id = $1`, userID).Scan(
		&row.ID,
		&row.UserID,
		&row.Fullname,
		&bio,
		&avatarURL,
		&row.Timezone,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, iam_errorx.ErrProfileNotFound
		}
		return nil, fmt.Errorf("iam repo: get profile by user id: %w", err)
	}

	bioValue := ""
	if bio.Valid {
		bioValue = bio.String
	}
	avatarValue := ""
	if avatarURL.Valid {
		avatarValue = avatarURL.String
	}

	row.Bio = &bioValue
	row.AvatarURL = &avatarValue

	return iam_model.UserProfileModelToEntity(&row), nil
}

// GetWhoAmI returns the flattened authenticated session view for UI bootstrap.
func (r *UserRepository) GetWhoAmI(ctx context.Context, userID string) (*entity.WhoAmI, error) {
	if r == nil || r.db == nil {
		return nil, iam_errorx.ErrUserNotFound
	}

	var (
		result entity.WhoAmI
		roles  []string
		perms  []string
	)

	err := r.db.QueryRow(ctx, `
		SELECT
			u.id AS user_id,
			u.username,
			u.email,
			COALESCE(u.phone, '') AS phone,
			u.security_level,
			u.status,
			COALESCE(p.fullname, '') AS fullname,
			COALESCE(p.avatar_url, '') AS avatar_url,
			COALESCE(p.bio, '') AS bio,
			COALESCE(
				ARRAY_AGG(DISTINCT r.name ORDER BY r.name)
				FILTER (WHERE r.name IS NOT NULL),
				ARRAY[]::text[]
			) AS role_names,
			COALESCE(
				ARRAY_AGG(
					DISTINCT COALESCE(NULLIF(perm.name, ''), NULLIF(perm.slug, ''))
					ORDER BY COALESCE(NULLIF(perm.name, ''), NULLIF(perm.slug, ''))
				)
				FILTER (WHERE COALESCE(NULLIF(perm.name, ''), NULLIF(perm.slug, '')) IS NOT NULL),
				ARRAY[]::text[]
			) AS permission_names
		FROM iam.users u
		JOIN iam.user_profiles p ON p.user_id = u.id
		LEFT JOIN iam.user_roles ur ON ur.user_id = u.id
		LEFT JOIN iam.roles r ON r.id = ur.role_id
		LEFT JOIN iam.role_permissions rp ON rp.role_id = r.id
		LEFT JOIN iam.permissions perm ON perm.id = rp.permission_id
		WHERE u.id = $1
		GROUP BY
			u.id, u.username, u.email, u.phone, u.security_level, u.status,
			p.fullname, p.avatar_url, p.bio`, userID).Scan(
		&result.UserID,
		&result.Username,
		&result.Email,
		&result.Phone,
		&result.Level,
		&result.Status,
		&result.FullName,
		&result.AvatarURL,
		&result.Bio,
		&roles,
		&perms,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, iam_errorx.ErrUserNotFound
		}
		return nil, fmt.Errorf("iam repo: get whoami: %w", err)
	}

	result.OnBoarding = false
	result.AuthType = "password"
	result.Roles = append([]string(nil), roles...)
	result.Permissions = append([]string(nil), perms...)

	return &result, nil
}

// CreateRefreshToken persists a hashed refresh token.
func (r *UserRepository) CreateRefreshToken(ctx context.Context, token *entity.RefreshToken) error {
	if r == nil || r.db == nil {
		return iam_errorx.ErrRegistrationFailed
	}

	dbToken := iam_model.RefreshTokenEntityToModel(token)
	if dbToken == nil {
		return iam_errorx.ErrRegistrationFailed
	}

	_, err := r.db.Exec(
		ctx,
		`INSERT INTO iam.refresh_tokens (
			id, device_id, user_id, token_hash, expires_at, is_revoked, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		dbToken.ID,
		dbToken.DeviceID,
		dbToken.UserID,
		dbToken.TokenHash,
		dbToken.ExpiresAt,
		dbToken.IsRevoked,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "refresh_tokens_token_hash_key":
				return iam_errorx.ErrRegistrationFailed
			default:
				return iam_errorx.ErrRegistrationFailed
			}
		}
		return fmt.Errorf("iam repo: create refresh token: %w", err)
	}

	return nil
}

// Activate marks a user as active.
func (r *UserRepository) Activate(ctx context.Context, userID string) error {
	if r == nil || r.db == nil {
		return iam_errorx.ErrActivationFailed
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("iam repo: begin activation tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(
		ctx,
		`UPDATE iam.users
		 SET status = 'active', status_reason = $2, updated_at = NOW()
		 WHERE id = $1`,
		userID,
		activationStatusReason,
	)
	if err != nil {
		return fmt.Errorf("iam repo: activate user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return iam_errorx.ErrUserNotFound
	}

	var (
		roleRow iam_model.Role
		desc    sql.NullString
	)

	err = tx.QueryRow(ctx, `SELECT id, name, description, created_at
		FROM iam.roles
		WHERE name = $1`, activationRoleName).Scan(
		&roleRow.ID,
		&roleRow.Name,
		&desc,
		&roleRow.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return iam_errorx.ErrActivationRoleMissing
		}
		return fmt.Errorf("iam repo: lookup activation role: %w", err)
	}
	if desc.Valid {
		roleRow.Description = &desc.String
	}

	userRole := iam_model.UserRole{
		UserID: userID,
		RoleID: roleRow.ID,
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO iam.user_roles (user_id, role_id)
		 VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		userRole.UserID,
		userRole.RoleID,
	)
	if err != nil {
		return fmt.Errorf("iam repo: grant activation role: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("iam repo: commit activation tx: %w", err)
	}

	return nil
}

// UpdatePassword replaces the password hash for a user.
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, newPasswordHash string) error {
	if r == nil || r.db == nil {
		return iam_errorx.ErrResetFailed
	}

	tag, err := r.db.Exec(ctx,
		`UPDATE iam.users SET password_hash = $2, updated_at = NOW() WHERE id = $1`,
		userID, newPasswordHash,
	)
	if err != nil {
		return fmt.Errorf("iam repo: update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return iam_errorx.ErrUserNotFound
	}

	return nil
}
