package smtp_model

import (
	"encoding/json"
	"time"

	"controlplane/internal/smtp/domain/entity"
)

type RuntimeHeartbeat struct {
	DataPlaneID   string    `db:"data_plane_id"`
	SentAt        time.Time `db:"sent_at"`
	LocalVersion  int64     `db:"local_version"`
	GatewayCount  int       `db:"gateway_count"`
	ConsumerCount int       `db:"consumer_count"`
	MemberState   string    `db:"member_state"`
	Capacity      int       `db:"capacity"`
	GRPCAddr      string    `db:"grpc_addr"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type GatewayShardAssignment struct {
	GatewayID       string    `db:"gateway_id"`
	ShardID         int       `db:"shard_id"`
	DataPlaneID     string    `db:"data_plane_id"`
	GRPCEndpoint    string    `db:"grpc_endpoint"`
	Generation      int64     `db:"generation"`
	AssignmentState string    `db:"assignment_state"`
	DesiredState    string    `db:"desired_state"`
	LeaseExpiresAt  time.Time `db:"lease_expires_at"`
	AssignedAt      time.Time `db:"assigned_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type ConsumerShardAssignment struct {
	ConsumerID      string    `db:"consumer_id"`
	ShardID         int       `db:"shard_id"`
	DataPlaneID     string    `db:"data_plane_id"`
	TargetGatewayID *string   `db:"target_gateway_id"`
	TargetShardID   *int      `db:"target_gateway_shard_id"`
	TargetPlaneID   *string   `db:"target_gateway_data_plane_id"`
	TargetGRPCAddr  string    `db:"target_gateway_grpc_endpoint"`
	Generation      int64     `db:"generation"`
	AssignmentState string    `db:"assignment_state"`
	DesiredState    string    `db:"desired_state"`
	LeaseExpiresAt  time.Time `db:"lease_expires_at"`
	AssignedAt      time.Time `db:"assigned_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type DeliveryAttempt struct {
	ID                 string    `db:"id"`
	ConsumerID         *string   `db:"consumer_id"`
	TemplateID         *string   `db:"template_id"`
	GatewayID          *string   `db:"gateway_id"`
	EndpointID         *string   `db:"endpoint_id"`
	MessageID          string    `db:"message_id"`
	TransportMessageID string    `db:"transport_message_id"`
	Subject            string    `db:"subject"`
	Status             string    `db:"status"`
	ErrorMessage       string    `db:"error_message"`
	ErrorClass         string    `db:"error_class"`
	RetryCount         int       `db:"retry_count"`
	TraceID            string    `db:"trace_id"`
	Payload            []byte    `db:"payload"`
	WorkspaceID        string    `db:"workspace_id"`
	CreatedAt          time.Time `db:"created_at"`
}

type ActivityLog struct {
	ID         string    `db:"id"`
	EntityType string    `db:"entity_type"`
	EntityID   string    `db:"entity_id"`
	EntityName string    `db:"entity_name"`
	Action     string    `db:"action"`
	ActorName  string    `db:"actor_name"`
	Note       string    `db:"note"`
	WorkspaceID string    `db:"workspace_id"`
	CreatedAt  time.Time `db:"created_at"`
}

type RuntimeDataPlane struct {
	ID           string     `db:"id"`
	ZoneID       *string    `db:"zone_id"`
	GRPCEndpoint string     `db:"grpc_endpoint"`
	Status       string     `db:"status"`
	LastSeenAt   *time.Time `db:"last_seen_at"`
	Capacity     int        `db:"capacity"`
}

func RuntimeHeartbeatModelToEntity(v *RuntimeHeartbeat) *entity.RuntimeHeartbeat {
	if v == nil {
		return nil
	}
	return &entity.RuntimeHeartbeat{
		DataPlaneID:   v.DataPlaneID,
		SentAt:        v.SentAt,
		LocalVersion:  v.LocalVersion,
		GatewayCount:  v.GatewayCount,
		ConsumerCount: v.ConsumerCount,
		MemberState:   v.MemberState,
		Capacity:      v.Capacity,
		GRPCAddr:      v.GRPCAddr,
		UpdatedAt:     v.UpdatedAt,
	}
}

func GatewayShardAssignmentModelToEntity(v *GatewayShardAssignment) *entity.GatewayShardAssignment {
	if v == nil {
		return nil
	}
	return &entity.GatewayShardAssignment{
		GatewayID:       v.GatewayID,
		ShardID:         v.ShardID,
		DataPlaneID:     v.DataPlaneID,
		GRPCEndpoint:    v.GRPCEndpoint,
		Generation:      v.Generation,
		AssignmentState: v.AssignmentState,
		DesiredState:    v.DesiredState,
		LeaseExpiresAt:  v.LeaseExpiresAt,
		AssignedAt:      v.AssignedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}

func ConsumerShardAssignmentModelToEntity(v *ConsumerShardAssignment) *entity.ConsumerShardAssignment {
	if v == nil {
		return nil
	}
	return &entity.ConsumerShardAssignment{
		ConsumerID:      v.ConsumerID,
		ShardID:         v.ShardID,
		DataPlaneID:     v.DataPlaneID,
		TargetGatewayID: stringValue(v.TargetGatewayID),
		TargetShardID:   intValue(v.TargetShardID),
		TargetPlaneID:   stringValue(v.TargetPlaneID),
		TargetGRPCAddr:  v.TargetGRPCAddr,
		Generation:      v.Generation,
		AssignmentState: v.AssignmentState,
		DesiredState:    v.DesiredState,
		LeaseExpiresAt:  v.LeaseExpiresAt,
		AssignedAt:      v.AssignedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}

func DeliveryAttemptModelToEntity(v *DeliveryAttempt) *entity.DeliveryAttempt {
	if v == nil {
		return nil
	}
	return &entity.DeliveryAttempt{
		ID:                 v.ID,
		ConsumerID:         stringValue(v.ConsumerID),
		TemplateID:         stringValue(v.TemplateID),
		GatewayID:          stringValue(v.GatewayID),
		EndpointID:         stringValue(v.EndpointID),
		MessageID:          v.MessageID,
		TransportMessageID: v.TransportMessageID,
		Subject:            v.Subject,
		Status:             v.Status,
		ErrorMessage:       v.ErrorMessage,
		ErrorClass:         v.ErrorClass,
		RetryCount:         v.RetryCount,
		TraceID:            v.TraceID,
		Payload:            json.RawMessage(v.Payload),
		WorkspaceID:        v.WorkspaceID,
		CreatedAt:          v.CreatedAt,
	}
}

func ActivityLogModelToEntity(v *ActivityLog) *entity.ActivityLog {
	if v == nil {
		return nil
	}
	return &entity.ActivityLog{
		ID:         v.ID,
		EntityType: v.EntityType,
		EntityID:   v.EntityID,
		EntityName: v.EntityName,
		Action:     v.Action,
		ActorName:  v.ActorName,
		Note:       v.Note,
		WorkspaceID: v.WorkspaceID,
		CreatedAt:  v.CreatedAt,
	}
}

func RuntimeDataPlaneModelToEntity(v *RuntimeDataPlane) *entity.RuntimeDataPlane {
	if v == nil {
		return nil
	}
	return &entity.RuntimeDataPlane{
		ID:           v.ID,
		ZoneID:       stringValue(v.ZoneID),
		GRPCEndpoint: v.GRPCEndpoint,
		Status:       v.Status,
		LastSeenAt:   v.LastSeenAt,
		Capacity:     v.Capacity,
	}
}

func intValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
