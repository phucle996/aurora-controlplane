package smtp_model

import (
	"time"

	"controlplane/internal/smtp/domain/entity"
)

type Endpoint struct {
	ID                   string    `db:"id"`
	WorkspaceID          *string   `db:"workspace_id"`
	OwnerUserID          *string   `db:"owner_user_id"`
	Name                 string    `db:"name"`
	ProviderKind         string    `db:"provider_kind"`
	Host                 string    `db:"host"`
	Port                 int       `db:"port"`
	Username             string    `db:"username"`
	Priority             int       `db:"priority"`
	Weight               int       `db:"weight"`
	MaxConnections       int       `db:"max_connections"`
	MaxParallelSends     int       `db:"max_parallel_sends"`
	MaxMessagesPerSecond int       `db:"max_messages_per_second"`
	Burst                int       `db:"burst"`
	WarmupState          string    `db:"warmup_state"`
	Status               string    `db:"status"`
	TLSMode              string    `db:"tls_mode"`
	RuntimeVersion       int64     `db:"runtime_version"`
	Password             string    `db:"password"`
	CACertPEM            string    `db:"ca_cert_pem"`
	ClientCertPEM        string    `db:"client_cert_pem"`
	ClientKeyPEM         string    `db:"client_key_pem"`
	SecretRef            string    `db:"secret_ref"`
	SecretVersion        int64     `db:"secret_version"`
	SecretProvider       string    `db:"provider"`
	CreatedAt            time.Time `db:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"`
}

func EndpointEntityToModel(v *entity.Endpoint) *Endpoint {
	if v == nil {
		return nil
	}

	return &Endpoint{
		ID:                   v.ID,
		WorkspaceID:          stringPtr(v.WorkspaceID),
		OwnerUserID:          stringPtr(v.OwnerUserID),
		Name:                 v.Name,
		ProviderKind:         v.ProviderKind,
		Host:                 v.Host,
		Port:                 v.Port,
		Username:             v.Username,
		Priority:             v.Priority,
		Weight:               v.Weight,
		MaxConnections:       v.MaxConnections,
		MaxParallelSends:     v.MaxParallelSends,
		MaxMessagesPerSecond: v.MaxMessagesPerSecond,
		Burst:                v.Burst,
		WarmupState:          v.WarmupState,
		Status:               v.Status,
		TLSMode:              v.TLSMode,
		RuntimeVersion:       v.RuntimeVersion,
		Password:             v.Password,
		CACertPEM:            v.CACertPEM,
		ClientCertPEM:        v.ClientCertPEM,
		ClientKeyPEM:         v.ClientKeyPEM,
		SecretRef:            v.SecretRef,
		SecretVersion:        v.SecretVersion,
		SecretProvider:       v.SecretProvider,
		CreatedAt:            v.CreatedAt,
		UpdatedAt:            v.UpdatedAt,
	}
}

func EndpointModelToEntity(v *Endpoint) *entity.Endpoint {
	if v == nil {
		return nil
	}

	return &entity.Endpoint{
		ID:                   v.ID,
		WorkspaceID:          stringValue(v.WorkspaceID),
		OwnerUserID:          stringValue(v.OwnerUserID),
		Name:                 v.Name,
		ProviderKind:         v.ProviderKind,
		Host:                 v.Host,
		Port:                 v.Port,
		Username:             v.Username,
		Priority:             v.Priority,
		Weight:               v.Weight,
		MaxConnections:       v.MaxConnections,
		MaxParallelSends:     v.MaxParallelSends,
		MaxMessagesPerSecond: v.MaxMessagesPerSecond,
		Burst:                v.Burst,
		WarmupState:          v.WarmupState,
		Status:               v.Status,
		TLSMode:              v.TLSMode,
		RuntimeVersion:       v.RuntimeVersion,
		Password:             v.Password,
		CACertPEM:            v.CACertPEM,
		ClientCertPEM:        v.ClientCertPEM,
		ClientKeyPEM:         v.ClientKeyPEM,
		SecretRef:            v.SecretRef,
		SecretVersion:        v.SecretVersion,
		SecretProvider:       v.SecretProvider,
		CreatedAt:            v.CreatedAt,
		UpdatedAt:            v.UpdatedAt,
	}
}
