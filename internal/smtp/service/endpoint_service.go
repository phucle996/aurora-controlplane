package service

import (
	"context"
	"strings"

	"controlplane/internal/smtp/domain/entity"
	smtp_domainrepo "controlplane/internal/smtp/domain/repository"
	smtp_domainsvc "controlplane/internal/smtp/domain/service"
	smtp_errorx "controlplane/internal/smtp/errorx"
	"controlplane/pkg/id"
)

type EndpointService struct {
	repo smtp_domainrepo.EndpointRepository
}

func NewEndpointService(repo smtp_domainrepo.EndpointRepository) smtp_domainsvc.EndpointService {
	return &EndpointService{repo: repo}
}

func (s *EndpointService) ListEndpointViews(ctx context.Context, workspaceID string) ([]*entity.EndpointView, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.ListEndpointViewsByWorkspace(ctx, workspaceID)
}

func (s *EndpointService) GetEndpointView(ctx context.Context, workspaceID, endpointID string) (*entity.EndpointView, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.GetEndpointView(ctx, workspaceID, strings.TrimSpace(endpointID))
}

func (s *EndpointService) ListEndpoints(ctx context.Context, workspaceID string) ([]*entity.Endpoint, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.ListEndpointsByWorkspace(ctx, workspaceID)
}

func (s *EndpointService) GetEndpoint(ctx context.Context, workspaceID, endpointID string) (*entity.Endpoint, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.GetEndpoint(ctx, workspaceID, strings.TrimSpace(endpointID))
}

func (s *EndpointService) TryConnect(ctx context.Context, endpoint *entity.Endpoint) error {
	if s == nil || endpoint == nil {
		return smtp_errorx.ErrInvalidResource
	}
	if err := normalizeEndpoint(endpoint); err != nil {
		return err
	}
	return probeEndpointConnection(ctx, endpoint)
}

func (s *EndpointService) CreateEndpoint(ctx context.Context, endpoint *entity.Endpoint) error {
	if s == nil || s.repo == nil || endpoint == nil {
		return smtp_errorx.ErrInvalidResource
	}
	if err := normalizeEndpoint(endpoint); err != nil {
		return err
	}
	endpointID, err := id.Generate()
	if err != nil {
		return err
	}
	endpoint.ID = endpointID
	return s.repo.CreateEndpoint(ctx, endpoint)
}

func (s *EndpointService) UpdateEndpoint(ctx context.Context, endpoint *entity.Endpoint) error {
	if s == nil || s.repo == nil || endpoint == nil {
		return smtp_errorx.ErrInvalidResource
	}
	if strings.TrimSpace(endpoint.ID) == "" {
		return smtp_errorx.ErrInvalidResource
	}
	existing, err := s.repo.GetEndpoint(ctx, strings.TrimSpace(endpoint.WorkspaceID), strings.TrimSpace(endpoint.ID))
	if err != nil {
		return err
	}
	if err := normalizeEndpoint(endpoint); err != nil {
		return err
	}
	preserveEndpointSecrets(endpoint, existing)
	return s.repo.UpdateEndpoint(ctx, endpoint)
}

func (s *EndpointService) DeleteEndpoint(ctx context.Context, workspaceID, endpointID string) error {
	if s == nil || s.repo == nil {
		return smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.DeleteEndpoint(ctx, workspaceID, strings.TrimSpace(endpointID))
}

func normalizeEndpoint(endpoint *entity.Endpoint) error {
	endpoint.WorkspaceID = trimString(endpoint.WorkspaceID)
	endpoint.OwnerUserID = trimString(endpoint.OwnerUserID)
	endpoint.Name = trimString(endpoint.Name)
	endpoint.ProviderKind = defaultString(endpoint.ProviderKind, "smtp")
	endpoint.Host = trimString(endpoint.Host)
	endpoint.Username = trimString(endpoint.Username)
	endpoint.WarmupState = defaultString(endpoint.WarmupState, "stable")
	endpoint.Status = defaultString(endpoint.Status, "disabled")
	endpoint.TLSMode = defaultString(endpoint.TLSMode, "starttls")
	endpoint.Password = strings.TrimSpace(endpoint.Password)
	endpoint.CACertPEM = strings.TrimSpace(endpoint.CACertPEM)
	endpoint.ClientCertPEM = strings.TrimSpace(endpoint.ClientCertPEM)
	endpoint.ClientKeyPEM = strings.TrimSpace(endpoint.ClientKeyPEM)
	endpoint.SecretRef = trimString(endpoint.SecretRef)
	endpoint.SecretProvider = defaultString(endpoint.SecretProvider, "postgresql")
	endpoint.Port = maxInt(endpoint.Port, 1)
	endpoint.Priority = maxInt(endpoint.Priority, 0)
	endpoint.Weight = maxInt(endpoint.Weight, 1)
	endpoint.MaxConnections = maxInt(endpoint.MaxConnections, 1)
	endpoint.MaxParallelSends = maxInt(endpoint.MaxParallelSends, 1)
	endpoint.MaxMessagesPerSecond = maxInt(endpoint.MaxMessagesPerSecond, 0)
	endpoint.Burst = maxInt(endpoint.Burst, 0)

	if endpoint.WorkspaceID == "" {
		return smtp_errorx.ErrWorkspaceRequired
	}
	if endpoint.Name == "" || endpoint.Host == "" {
		return smtp_errorx.ErrInvalidResource
	}
	return nil
}

func preserveEndpointSecrets(endpoint, existing *entity.Endpoint) {
	if endpoint == nil || existing == nil {
		return
	}

	if endpoint.Password == "" {
		endpoint.Password = existing.Password
	}
	if endpoint.SecretRef == "" {
		endpoint.SecretRef = existing.SecretRef
	}
	if endpoint.SecretProvider == "" {
		endpoint.SecretProvider = existing.SecretProvider
	}

	switch endpoint.TLSMode {
	case "none", "starttls":
		endpoint.CACertPEM = ""
		endpoint.ClientCertPEM = ""
		endpoint.ClientKeyPEM = ""
	case "tls":
		if endpoint.CACertPEM == "" {
			endpoint.CACertPEM = existing.CACertPEM
		}
		endpoint.ClientCertPEM = ""
		endpoint.ClientKeyPEM = ""
	case "mtls":
		if endpoint.CACertPEM == "" {
			endpoint.CACertPEM = existing.CACertPEM
		}
		if endpoint.ClientCertPEM == "" {
			endpoint.ClientCertPEM = existing.ClientCertPEM
		}
		if endpoint.ClientKeyPEM == "" {
			endpoint.ClientKeyPEM = existing.ClientKeyPEM
		}
	}
}
