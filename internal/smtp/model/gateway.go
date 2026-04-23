package smtp_model

import (
	"time"

	"controlplane/internal/smtp/domain/entity"
)

type Gateway struct {
	ID                string    `db:"id"`
	WorkspaceID       *string   `db:"workspace_id"`
	OwnerUserID       *string   `db:"owner_user_id"`
	ZoneID            *string   `db:"zone_id"`
	Name              string    `db:"name"`
	TrafficClass      string    `db:"traffic_class"`
	Status            string    `db:"status"`
	RoutingMode       string    `db:"routing_mode"`
	Priority          int       `db:"priority"`
	FallbackGatewayID *string   `db:"fallback_gateway_id"`
	RuntimeVersion    int64     `db:"runtime_version"`
	DesiredShardCount int       `db:"desired_shard_count"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

func GatewayEntityToModel(v *entity.Gateway) *Gateway {
	if v == nil {
		return nil
	}

	return &Gateway{
		ID:                v.ID,
		WorkspaceID:       stringPtr(v.WorkspaceID),
		OwnerUserID:       stringPtr(v.OwnerUserID),
		ZoneID:            stringPtr(v.ZoneID),
		Name:              v.Name,
		TrafficClass:      v.TrafficClass,
		Status:            v.Status,
		RoutingMode:       v.RoutingMode,
		Priority:          v.Priority,
		FallbackGatewayID: stringPtr(v.FallbackGatewayID),
		RuntimeVersion:    v.RuntimeVersion,
		DesiredShardCount: v.DesiredShardCount,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
	}
}

func GatewayModelToEntity(v *Gateway) *entity.Gateway {
	if v == nil {
		return nil
	}

	return &entity.Gateway{
		ID:                v.ID,
		WorkspaceID:       stringValue(v.WorkspaceID),
		OwnerUserID:       stringValue(v.OwnerUserID),
		ZoneID:            stringValue(v.ZoneID),
		Name:              v.Name,
		TrafficClass:      v.TrafficClass,
		Status:            v.Status,
		RoutingMode:       v.RoutingMode,
		Priority:          v.Priority,
		FallbackGatewayID: stringValue(v.FallbackGatewayID),
		RuntimeVersion:    v.RuntimeVersion,
		DesiredShardCount: v.DesiredShardCount,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
	}
}
