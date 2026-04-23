package entity

import "time"

// Zone represents a geographic zone.
type Zone struct {
	ID          string
	Slug        string
	Name        string
	Description string
	CreatedAt   time.Time
}
