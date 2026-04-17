package iam_model

import (
	"time"

	"controlplane/internal/iam/domain/entity"
)

// User represents the database schema map for IAM users
type User struct {
	ID        string    `db:"id"`
	Email     string    `db:"email"`
	Password  string    `db:"password"`
	Role      string    `db:"role"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func UserEntityToModel(u *entity.User) *User {
	if u == nil {
		return nil
	}
	return &User{
		ID:        u.ID,
		Email:     u.Email,
		Password:  u.Password,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func UserModelToEntity(m *User) *entity.User {
	if m == nil {
		return nil
	}
	return &entity.User{
		ID:        m.ID,
		Email:     m.Email,
		Password:  m.Password,
		Role:      m.Role,
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
