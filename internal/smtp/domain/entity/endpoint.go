package entity

import "time"

type Endpoint struct {
	ID                   string
	WorkspaceID          string
	OwnerUserID          string
	Name                 string
	ProviderKind         string
	Host                 string
	Port                 int
	Username             string
	Priority             int
	Weight               int
	MaxConnections       int
	MaxParallelSends     int
	MaxMessagesPerSecond int
	Burst                int
	WarmupState          string
	Status               string
	TLSMode              string
	RuntimeVersion       int64
	Password             string
	CACertPEM            string
	ClientCertPEM        string
	ClientKeyPEM         string
	SecretRef            string
	SecretVersion        int64
	SecretProvider       string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
