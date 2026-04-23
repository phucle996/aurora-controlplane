package entity

import "time"

// Pagination captures normalized list metadata.
type Pagination struct {
	Page       int
	Limit      int
	Total      int64
	TotalPages int
}

// HostListFilter captures host list query options.
type HostListFilter struct {
	Page     int
	Limit    int
	Query    string
	Status   string
	ZoneSlug string
}

// Host represents a KVM host enrolled through a dataplane.
type Host struct {
	HostID           string
	AgentID          string
	ZoneID           string
	ZoneSlug         string
	DataPlaneID      string
	Hostname         string
	PrivateIP        string
	HypervisorType   string
	AgentVersion     string
	CapabilitiesJSON string
	CPUCores         int32
	MemoryBytes      int64
	DiskBytes        int64
	Status           string
	LastSeenAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// HostBinding describes the authoritative host/agent ownership state.
type HostBinding struct {
	HostID           string
	RequestedAgentID string
	BoundAgentID     string
	Allowed          bool
	Current          *Host
}

// HostPage is the paginated host result set.
type HostPage struct {
	Items      []*Host
	Pagination Pagination
}

// HostOption is the minimal projection used by UI pickers.
type HostOption struct {
	ID          string
	Label       string
	ZoneSlug    string
	Status      string
	DataPlaneID string
}
