package response

import (
	"time"

	"controlplane/internal/virtualmachine/domain/entity"
)

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type Host struct {
	HostID           string     `json:"host_id"`
	AgentID          string     `json:"agent_id"`
	ZoneID           string     `json:"zone_id"`
	ZoneSlug         string     `json:"zone_slug"`
	DataPlaneID      string     `json:"data_plane_id"`
	Hostname         string     `json:"hostname"`
	PrivateIP        string     `json:"private_ip"`
	HypervisorType   string     `json:"hypervisor_type"`
	AgentVersion     string     `json:"agent_version"`
	CapabilitiesJSON string     `json:"capabilities_json"`
	CPUCores         int32      `json:"cpu_cores"`
	MemoryBytes      int64      `json:"memory_bytes"`
	DiskBytes        int64      `json:"disk_bytes"`
	Status           string     `json:"status"`
	LastSeenAt       *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type HostOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	ZoneSlug    string `json:"zone_slug"`
	Status      string `json:"status"`
	DataPlaneID string `json:"data_plane_id"`
}

type HostPage struct {
	Items      []*Host    `json:"items"`
	Pagination Pagination `json:"pagination"`
}

func HostFromEntity(item *entity.Host) *Host {
	if item == nil {
		return nil
	}
	return &Host{
		HostID:           item.HostID,
		AgentID:          item.AgentID,
		ZoneID:           item.ZoneID,
		ZoneSlug:         item.ZoneSlug,
		DataPlaneID:      item.DataPlaneID,
		Hostname:         item.Hostname,
		PrivateIP:        item.PrivateIP,
		HypervisorType:   item.HypervisorType,
		AgentVersion:     item.AgentVersion,
		CapabilitiesJSON: item.CapabilitiesJSON,
		CPUCores:         item.CPUCores,
		MemoryBytes:      item.MemoryBytes,
		DiskBytes:        item.DiskBytes,
		Status:           item.Status,
		LastSeenAt:       item.LastSeenAt,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func HostPageFromEntity(page *entity.HostPage) *HostPage {
	if page == nil {
		return &HostPage{
			Items:      []*Host{},
			Pagination: Pagination{},
		}
	}

	items := make([]*Host, 0, len(page.Items))
	for _, item := range page.Items {
		if mapped := HostFromEntity(item); mapped != nil {
			items = append(items, mapped)
		}
	}

	return &HostPage{
		Items: items,
		Pagination: Pagination{
			Page:       page.Pagination.Page,
			Limit:      page.Pagination.Limit,
			Total:      page.Pagination.Total,
			TotalPages: page.Pagination.TotalPages,
		},
	}
}

func HostOptionFromEntity(item *entity.HostOption) *HostOption {
	if item == nil {
		return nil
	}
	return &HostOption{
		ID:          item.ID,
		Label:       item.Label,
		ZoneSlug:    item.ZoneSlug,
		Status:      item.Status,
		DataPlaneID: item.DataPlaneID,
	}
}
