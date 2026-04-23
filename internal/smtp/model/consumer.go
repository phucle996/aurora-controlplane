package smtp_model

import (
	"encoding/json"
	"time"

	"controlplane/internal/smtp/domain/entity"
)

type Consumer struct {
	ID                string    `db:"id"`
	WorkspaceID       *string   `db:"workspace_id"`
	OwnerUserID       *string   `db:"owner_user_id"`
	ZoneID            *string   `db:"zone_id"`
	Name              string    `db:"name"`
	TransportType     string    `db:"transport_type"`
	Source            string    `db:"source"`
	ConsumerGroup     string    `db:"consumer_group"`
	WorkerConcurrency int       `db:"worker_concurrency"`
	AckTimeoutSeconds int       `db:"ack_timeout_seconds"`
	BatchSize         int       `db:"batch_size"`
	Status            string    `db:"status"`
	Note              string    `db:"note"`
	ConnectionConfig  []byte    `db:"connection_config"`
	RuntimeVersion    int64     `db:"runtime_version"`
	DesiredShardCount int       `db:"desired_shard_count"`
	SecretConfig      []byte    `db:"secret_config"`
	SecretRef         string    `db:"secret_ref"`
	SecretVersion     int64     `db:"secret_version"`
	SecretProvider    string    `db:"provider"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

func ConsumerEntityToModel(v *entity.Consumer) *Consumer {
	if v == nil {
		return nil
	}

	return &Consumer{
		ID:                v.ID,
		WorkspaceID:       stringPtr(v.WorkspaceID),
		OwnerUserID:       stringPtr(v.OwnerUserID),
		ZoneID:            stringPtr(v.ZoneID),
		Name:              v.Name,
		TransportType:     v.TransportType,
		Source:            v.Source,
		ConsumerGroup:     v.ConsumerGroup,
		WorkerConcurrency: v.WorkerConcurrency,
		AckTimeoutSeconds: v.AckTimeoutSeconds,
		BatchSize:         v.BatchSize,
		Status:            v.Status,
		Note:              v.Note,
		ConnectionConfig:  []byte(v.ConnectionConfig),
		RuntimeVersion:    v.RuntimeVersion,
		DesiredShardCount: v.DesiredShardCount,
		SecretConfig:      []byte(v.SecretConfig),
		SecretRef:         v.SecretRef,
		SecretVersion:     v.SecretVersion,
		SecretProvider:    v.SecretProvider,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
	}
}

func ConsumerModelToEntity(v *Consumer) *entity.Consumer {
	if v == nil {
		return nil
	}

	return &entity.Consumer{
		ID:                v.ID,
		WorkspaceID:       stringValue(v.WorkspaceID),
		OwnerUserID:       stringValue(v.OwnerUserID),
		ZoneID:            stringValue(v.ZoneID),
		Name:              v.Name,
		TransportType:     v.TransportType,
		Source:            v.Source,
		ConsumerGroup:     v.ConsumerGroup,
		WorkerConcurrency: v.WorkerConcurrency,
		AckTimeoutSeconds: v.AckTimeoutSeconds,
		BatchSize:         v.BatchSize,
		Status:            v.Status,
		Note:              v.Note,
		ConnectionConfig:  rawJSON(v.ConnectionConfig),
		RuntimeVersion:    v.RuntimeVersion,
		DesiredShardCount: v.DesiredShardCount,
		SecretConfig:      rawJSON(v.SecretConfig),
		SecretRef:         v.SecretRef,
		SecretVersion:     v.SecretVersion,
		SecretProvider:    v.SecretProvider,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
	}
}

func rawJSON(v []byte) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(v)
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
