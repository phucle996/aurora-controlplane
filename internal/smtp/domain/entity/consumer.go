package entity

import (
	"encoding/json"
	"time"
)

type Consumer struct {
	ID                string
	WorkspaceID       string
	OwnerUserID       string
	ZoneID            string
	Name              string
	TransportType     string
	Source            string
	ConsumerGroup     string
	WorkerConcurrency int
	AckTimeoutSeconds int
	BatchSize         int
	Status            string
	Note              string
	ConnectionConfig  json.RawMessage
	RuntimeVersion    int64
	DesiredShardCount int
	SecretConfig      json.RawMessage
	SecretRef         string
	SecretVersion     int64
	SecretProvider    string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
