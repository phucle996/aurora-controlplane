package leaseassign

import "time"

const (
	StateActive   = "active"
	StatePending  = "pending"
	StateRevoking = "revoking"
)

type WorkShard struct {
	WorkID       string
	ZoneID       string
	GroupKey     string
	Weight       int
	DesiredState string
	Metadata     map[string]string
}

type HealthyNode struct {
	NodeID         string
	ZoneID         string
	GRPCEndpoint   string
	Capacity       int
	LeaseExpiresAt time.Time
}

type Assignment struct {
	WorkID           string
	OwnerNodeID      string
	AssignmentState  string
	DesiredState     string
	Generation       int64
	LeaseExpiresAt   time.Time
	LastTransitionAt time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Metadata         map[string]string
}

type PublishedAssignment struct {
	WorkID           string
	OwnerNodeID      string
	AssignmentState  string
	DesiredState     string
	Generation       int64
	LeaseExpiresAt   time.Time
	LastTransitionAt time.Time
	Metadata         map[string]string
}
