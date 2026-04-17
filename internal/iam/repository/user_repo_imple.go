package iam_repoImple

import (
	"context"

	"controlplane/internal/iam/domain/entity"
	iam_domainrepo "controlplane/internal/iam/domain/repository"
	iam_errorx "controlplane/internal/iam/errorx"
	iam_model "controlplane/internal/iam/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepoImple struct {
	db *pgxpool.Pool
}

func NewUserRepoImple(db *pgxpool.Pool) iam_domainrepo.UserRepository {
	return &userRepoImple{db: db}
}

func (r *userRepoImple) Create(ctx context.Context, u *entity.User) error {
	m := iam_model.UserEntityToModel(u)

	query := `
		INSERT INTO iam.users (id, email, password, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query, m.ID, m.Email, m.Password, m.Role, m.Status, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return err // TODO: map to specific error if duplicate
	}
	return nil
}

func (r *userRepoImple) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `SELECT id, email, password, role, status, created_at, updated_at FROM iam.users WHERE email = $1`
	row := r.db.QueryRow(ctx, query, email)

	var m iam_model.User
	err := row.Scan(&m.ID, &m.Email, &m.Password, &m.Role, &m.Status, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, iam_errorx.ErrUserNotFound
		}
		return nil, err
	}

	return iam_model.UserModelToEntity(&m), nil
}

func (r *userRepoImple) GetByID(ctx context.Context, id string) (*entity.User, error) {
	query := `SELECT id, email, password, role, status, created_at, updated_at FROM iam.users WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)

	var m iam_model.User
	err := row.Scan(&m.ID, &m.Email, &m.Password, &m.Role, &m.Status, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, iam_errorx.ErrUserNotFound
		}
		return nil, err
	}

	return iam_model.UserModelToEntity(&m), nil
}
