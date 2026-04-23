package core_model

import (
	"time"

	"controlplane/internal/core/domain/entity"
)

// DataPlane mirrors core.data_planes.
type DataPlane struct {
	ID           string     `db:"id"`
	NodeKey      string     `db:"node_key"`
	Name         string     `db:"name"`
	ZoneID       string     `db:"zone_id"`
	GRPCEndpoint string     `db:"grpc_endpoint"`
	Version      string     `db:"version"`
	CertSerial   string     `db:"cert_serial"`
	CertNotAfter *time.Time `db:"cert_not_after"`
	Status       string     `db:"status"`
	LastSeenAt   *time.Time `db:"last_seen_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

func DataPlaneEntityToModel(v *entity.DataPlane) *DataPlane {
	if v == nil {
		return nil
	}

	return &DataPlane{
		ID:           v.ID,
		NodeKey:      v.NodeKey,
		Name:         v.Name,
		ZoneID:       v.ZoneID,
		GRPCEndpoint: v.GRPCEndpoint,
		Version:      v.Version,
		CertSerial:   v.CertSerial,
		CertNotAfter: v.CertNotAfter,
		Status:       v.Status,
		LastSeenAt:   v.LastSeenAt,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    v.UpdatedAt,
	}
}

func DataPlaneModelToEntity(v *DataPlane) *entity.DataPlane {
	if v == nil {
		return nil
	}

	return &entity.DataPlane{
		ID:           v.ID,
		NodeKey:      v.NodeKey,
		Name:         v.Name,
		ZoneID:       v.ZoneID,
		GRPCEndpoint: v.GRPCEndpoint,
		Version:      v.Version,
		CertSerial:   v.CertSerial,
		CertNotAfter: v.CertNotAfter,
		Status:       v.Status,
		LastSeenAt:   v.LastSeenAt,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    v.UpdatedAt,
	}
}
