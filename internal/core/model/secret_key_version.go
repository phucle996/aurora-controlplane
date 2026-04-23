package core_model

import (
	"time"

	"controlplane/internal/core/domain/entity"
)

type SecretKeyVersion struct {
	ID               string    `db:"id"`
	Family           string    `db:"family"`
	Version          int64     `db:"version"`
	State            string    `db:"state"`
	SecretCiphertext string    `db:"secret_ciphertext"`
	ExpiresAt        time.Time `db:"expires_at"`
	RotatedAt        time.Time `db:"rotated_at"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

func SecretKeyVersionEntityToModel(v *entity.SecretKeyVersion) *SecretKeyVersion {
	if v == nil {
		return nil
	}

	return &SecretKeyVersion{
		ID:               v.ID,
		Family:           v.Family,
		Version:          v.Version,
		State:            v.State,
		SecretCiphertext: v.SecretCiphertext,
		ExpiresAt:        v.ExpiresAt,
		RotatedAt:        v.RotatedAt,
		CreatedAt:        v.CreatedAt,
		UpdatedAt:        v.UpdatedAt,
	}
}

func SecretKeyVersionModelToEntity(v *SecretKeyVersion) *entity.SecretKeyVersion {
	if v == nil {
		return nil
	}

	return &entity.SecretKeyVersion{
		ID:               v.ID,
		Family:           v.Family,
		Version:          v.Version,
		State:            v.State,
		SecretCiphertext: v.SecretCiphertext,
		ExpiresAt:        v.ExpiresAt,
		RotatedAt:        v.RotatedAt,
		CreatedAt:        v.CreatedAt,
		UpdatedAt:        v.UpdatedAt,
	}
}
