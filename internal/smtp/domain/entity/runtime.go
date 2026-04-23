package entity

import (
	"encoding/json"
	"time"
)

type RuntimeHeartbeat struct {
	DataPlaneID   string
	SentAt        time.Time
	LocalVersion  int64
	GatewayCount  int
	ConsumerCount int
	MemberState   string
	Capacity      int
	GRPCAddr      string
	UpdatedAt     time.Time
}

type GatewayShardAssignment struct {
	GatewayID       string
	ShardID         int
	DataPlaneID     string
	GRPCEndpoint    string
	Generation      int64
	AssignmentState string
	DesiredState    string
	LeaseExpiresAt  time.Time
	AssignedAt      time.Time
	UpdatedAt       time.Time
}

type ConsumerShardAssignment struct {
	ConsumerID      string
	ShardID         int
	DataPlaneID     string
	TargetGatewayID string
	TargetShardID   int
	TargetPlaneID   string
	TargetGRPCAddr  string
	Generation      int64
	AssignmentState string
	DesiredState    string
	LeaseExpiresAt  time.Time
	AssignedAt      time.Time
	UpdatedAt       time.Time
}

type GatewayShardStatus struct {
	GatewayID       string
	ShardID         int
	Status          string
	InflightCount   int64
	DesiredWorkers  int
	ActiveWorkers   int
	RelayQueueDepth int64
	PoolOpenConns   int
	PoolBusyConns   int
	SendRate        float64
	Backpressure    string
	LastError       string
	Generation      int64
	AssignmentState string
	RevokingDone    bool
}

type ConsumerShardStatus struct {
	ConsumerID      string
	ShardID         int
	GatewayID       string
	Status          string
	InflightCount   int64
	BrokerLag       int64
	OldestUnackedMS int64
	DesiredWorkers  int
	ActiveWorkers   int
	RelayQueueDepth int64
	LastError       string
	Generation      int64
	AssignmentState string
	RevokingDone    bool
}

type DeliveryAttempt struct {
	ID                 string
	ConsumerID         string
	TemplateID         string
	GatewayID          string
	EndpointID         string
	MessageID          string
	TransportMessageID string
	Subject            string
	Status             string
	ErrorMessage       string
	ErrorClass         string
	RetryCount         int
	TraceID            string
	Payload            json.RawMessage
	WorkspaceID        string
	CreatedAt          time.Time
}

type ActivityLog struct {
	ID         string
	EntityType string
	EntityID   string
	EntityName string
	Action     string
	ActorName  string
	Note       string
	WorkspaceID string
	CreatedAt  time.Time
}

type RuntimeDataPlane struct {
	ID           string
	ZoneID       string
	GRPCEndpoint string
	Status       string
	LastSeenAt   *time.Time
	Capacity     int
}

type RuntimeSyncRequest struct {
	DataPlaneID  string
	LocalVersion int64
	Capacity     int
	GRPCEndpoint string
}

type RuntimeSyncResponse struct {
	RuntimeVersion      int64
	Consumers           []*Consumer
	Templates           []*Template
	Gateways            []*Gateway
	Endpoints           []*Endpoint
	ConsumerAssignments []*ConsumerShardAssignment
	GatewayAssignments  []*GatewayShardAssignment
	SyncInterval        time.Duration
	FullResync          bool
}

type RuntimeReportRequest struct {
	DataPlaneID      string
	LocalVersion     int64
	Capacity         int
	GRPCEndpoint     string
	ConsumerStatuses []*ConsumerShardStatus
	GatewayStatuses  []*GatewayShardStatus
}

type RuntimeReportResponse struct {
	ReportInterval time.Duration
	ForceResync    bool
}
