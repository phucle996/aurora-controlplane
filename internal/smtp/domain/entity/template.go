package entity

import "time"

type Template struct {
	ID                  string
	WorkspaceID         string
	OwnerUserID         string
	Name                string
	Category            string
	TrafficClass        string
	Subject             string
	FromEmail           string
	ToEmail             string
	Status              string
	Variables           []string
	ConsumerID          string
	ActiveVersion       int
	RetryMaxAttempts    int
	RetryBackoffSeconds int
	TextBody            string
	HTMLBody            string
	RuntimeVersion      int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
