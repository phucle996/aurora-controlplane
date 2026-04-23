package entity

import "time"

// DataPlane represents an execution plane registered to the controlplane.
type DataPlane struct {
	ID           string
	NodeKey      string
	Name         string
	ZoneSlug     string
	ZoneID       string
	GRPCEndpoint string
	Version      string
	CertSerial   string
	CertNotAfter *time.Time
	Status       string
	LastSeenAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DataPlaneEnrollResult is returned after a successful bootstrap enrollment.
type DataPlaneEnrollResult struct {
	DataPlaneID       string
	ClientCertPEM     string
	CACertPEM         string
	CertNotAfter      time.Time
	HeartbeatInterval time.Duration
}

// DataPlaneHeartbeatResult is returned after a successful heartbeat.
type DataPlaneHeartbeatResult struct {
	HeartbeatInterval time.Duration
}
