package iam_domainrepo

import (
	"context"

	"controlplane/internal/iam/domain/entity"
)

// TokenRepository defines persistence operations for refresh tokens.
// It is intentionally separate from DeviceRepository (device lifecycle)
// and UserRepository (user identity).
type TokenRepository interface {
	// Create persists a new hashed refresh token record.
	Create(ctx context.Context, token *entity.RefreshToken) error

	// GetByHash looks up a non-revoked, non-expired token by its HMAC digest.
	GetByHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)

	// Revoke marks a single token as revoked (soft-delete; keeps audit trail).
	Revoke(ctx context.Context, tokenID string) error

	// RevokeAllByDevice revokes every token bound to a device.
	RevokeAllByDevice(ctx context.Context, deviceID string) error

	// RevokeAllByUser revokes every token belonging to a user (e.g. password change).
	RevokeAllByUser(ctx context.Context, userID string) error

	// DeleteExpired hard-deletes tokens past their expiry (admin cleanup).
	DeleteExpired(ctx context.Context) (int64, error)
}
