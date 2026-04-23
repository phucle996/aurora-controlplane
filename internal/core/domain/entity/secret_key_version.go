package entity

import "time"

const (
	SecretStateActive   = "active"
	SecretStatePrevious = "previous"
)

type SecretKeyVersion struct {
	ID               string
	Family           string
	Version          int64
	State            string
	SecretCiphertext string
	ExpiresAt        time.Time
	RotatedAt        time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type SecretFamilyState struct {
	Family    string
	RotatedAt time.Time
	UpdatedAt time.Time
}
