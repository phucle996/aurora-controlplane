package entity

import "time"

// User represents the core IAM identity.
type User struct {
	ID        string
	Email     string
	Password  string
	Role      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
