package entity

import "time"

type Gateway struct {
	ID                string
	WorkspaceID       string
	OwnerUserID       string
	ZoneID            string
	Name              string
	TrafficClass      string
	Status            string
	RoutingMode       string
	Priority          int
	FallbackGatewayID string
	RuntimeVersion    int64
	DesiredShardCount int
	TemplateIDs       []string
	EndpointIDs       []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
