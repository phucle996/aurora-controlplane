package model

import (
	"time"

	"controlplane/internal/virtualmachine/domain/entity"
)

// Host mirrors virtual_machine.hosts.
type Host struct {
	HostID           string     `db:"host_id"`
	AgentID          string     `db:"agent_id"`
	ZoneID           string     `db:"zone_id"`
	ZoneSlug         string     `db:"zone_slug"`
	DataPlaneID      string     `db:"data_plane_id"`
	Hostname         string     `db:"hostname"`
	PrivateIP        string     `db:"private_ip"`
	HypervisorType   string     `db:"hypervisor_type"`
	AgentVersion     string     `db:"agent_version"`
	CapabilitiesJSON string     `db:"capabilities_json"`
	CPUCores         int32      `db:"cpu_cores"`
	MemoryBytes      int64      `db:"memory_bytes"`
	DiskBytes        int64      `db:"disk_bytes"`
	Status           string     `db:"status"`
	LastSeenAt       *time.Time `db:"last_seen_at"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
}

// HostOption mirrors the minimal host picker projection.
type HostOption struct {
	HostID      string `db:"host_id"`
	Label       string `db:"label"`
	ZoneSlug    string `db:"zone_slug"`
	Status      string `db:"status"`
	DataPlaneID string `db:"data_plane_id"`
}

func HostEntityToModel(v *entity.Host) *Host {
	if v == nil {
		return nil
	}
	return &Host{
		HostID:           v.HostID,
		AgentID:          v.AgentID,
		ZoneID:           v.ZoneID,
		ZoneSlug:         v.ZoneSlug,
		DataPlaneID:      v.DataPlaneID,
		Hostname:         v.Hostname,
		PrivateIP:        v.PrivateIP,
		HypervisorType:   v.HypervisorType,
		AgentVersion:     v.AgentVersion,
		CapabilitiesJSON: v.CapabilitiesJSON,
		CPUCores:         v.CPUCores,
		MemoryBytes:      v.MemoryBytes,
		DiskBytes:        v.DiskBytes,
		Status:           v.Status,
		LastSeenAt:       v.LastSeenAt,
		CreatedAt:        v.CreatedAt,
		UpdatedAt:        v.UpdatedAt,
	}
}

func HostModelToEntity(v *Host) *entity.Host {
	if v == nil {
		return nil
	}
	return &entity.Host{
		HostID:           v.HostID,
		AgentID:          v.AgentID,
		ZoneID:           v.ZoneID,
		ZoneSlug:         v.ZoneSlug,
		DataPlaneID:      v.DataPlaneID,
		Hostname:         v.Hostname,
		PrivateIP:        v.PrivateIP,
		HypervisorType:   v.HypervisorType,
		AgentVersion:     v.AgentVersion,
		CapabilitiesJSON: v.CapabilitiesJSON,
		CPUCores:         v.CPUCores,
		MemoryBytes:      v.MemoryBytes,
		DiskBytes:        v.DiskBytes,
		Status:           v.Status,
		LastSeenAt:       v.LastSeenAt,
		CreatedAt:        v.CreatedAt,
		UpdatedAt:        v.UpdatedAt,
	}
}

func HostOptionModelToEntity(v *HostOption) *entity.HostOption {
	if v == nil {
		return nil
	}
	return &entity.HostOption{
		ID:          v.HostID,
		Label:       v.Label,
		ZoneSlug:    v.ZoneSlug,
		Status:      v.Status,
		DataPlaneID: v.DataPlaneID,
	}
}
