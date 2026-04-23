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

type GatewayService struct {
	repo         smtp_domainrepo.GatewayRepository
	templateRepo smtp_domainrepo.TemplateRepository
	endpointRepo smtp_domainrepo.EndpointRepository
	consumerRepo smtp_domainrepo.ConsumerRepository
}

func NewGatewayService(repo smtp_domainrepo.GatewayRepository, templateRepo smtp_domainrepo.TemplateRepository, endpointRepo smtp_domainrepo.EndpointRepository, consumerRepo smtp_domainrepo.ConsumerRepository) smtp_domainsvc.GatewayService {
	return &GatewayService{
		repo:         repo,
		templateRepo: templateRepo,
		endpointRepo: endpointRepo,
		consumerRepo: consumerRepo,
	}
}

func (s *GatewayService) ListGatewayItems(ctx context.Context, workspaceID string) ([]*entity.GatewayListItem, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.ListGatewayItemsByWorkspace(ctx, workspaceID)
}

func (s *GatewayService) GetGatewayDetail(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.GetGatewayDetail(ctx, workspaceID, strings.TrimSpace(gatewayID))
}

func (s *GatewayService) ListGateways(ctx context.Context, workspaceID string) ([]*entity.Gateway, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.ListGatewaysByWorkspace(ctx, workspaceID)
}

func (s *GatewayService) GetGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.Gateway, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.GetGateway(ctx, workspaceID, strings.TrimSpace(gatewayID))
}

func (s *GatewayService) UpdateGatewayTemplates(ctx context.Context, workspaceID, gatewayID string, templateIDs []string) (*entity.GatewayDetail, error) {
	gateway, err := s.loadGatewayForMutation(ctx, workspaceID, gatewayID)
	if err != nil {
		return nil, err
	}
	gateway.TemplateIDs = normalizeStringSlice(templateIDs)
	if err := s.normalizeGateway(ctx, gateway); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateGateway(ctx, gateway); err != nil {
		return nil, err
	}
	return s.repo.GetGatewayDetail(ctx, gateway.WorkspaceID, gateway.ID)
}

func (s *GatewayService) UpdateGatewayEndpoints(ctx context.Context, workspaceID, gatewayID string, endpointIDs []string) (*entity.GatewayDetail, error) {
	gateway, err := s.loadGatewayForMutation(ctx, workspaceID, gatewayID)
	if err != nil {
		return nil, err
	}
	gateway.EndpointIDs = normalizeStringSlice(endpointIDs)
	if err := s.normalizeGateway(ctx, gateway); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateGateway(ctx, gateway); err != nil {
		return nil, err
	}
	return s.repo.GetGatewayDetail(ctx, gateway.WorkspaceID, gateway.ID)
}

func (s *GatewayService) StartGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error) {
	return s.setGatewayStatus(ctx, workspaceID, gatewayID, "active")
}

func (s *GatewayService) DrainGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error) {
	return s.setGatewayStatus(ctx, workspaceID, gatewayID, "draining")
}

func (s *GatewayService) DisableGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error) {
	return s.setGatewayStatus(ctx, workspaceID, gatewayID, "disabled")
}

func (s *GatewayService) CreateGateway(ctx context.Context, gateway *entity.Gateway) error {
	if s == nil || s.repo == nil || gateway == nil {
		return smtp_errorx.ErrInvalidResource
	}
	if err := s.normalizeGateway(ctx, gateway); err != nil {
		return err
	}
	gatewayID, err := id.Generate()
	if err != nil {
		return err
	}
	gateway.ID = gatewayID
	return s.repo.CreateGateway(ctx, gateway)
}

func (s *GatewayService) UpdateGateway(ctx context.Context, gateway *entity.Gateway) error {
	if s == nil || s.repo == nil || gateway == nil {
		return smtp_errorx.ErrInvalidResource
	}
	if strings.TrimSpace(gateway.ID) == "" {
		return smtp_errorx.ErrInvalidResource
	}
	if err := s.normalizeGateway(ctx, gateway); err != nil {
		return err
	}
	return s.repo.UpdateGateway(ctx, gateway)
}

func (s *GatewayService) DeleteGateway(ctx context.Context, workspaceID, gatewayID string) error {
	if s == nil || s.repo == nil {
		return smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.DeleteGateway(ctx, workspaceID, strings.TrimSpace(gatewayID))
}

func (s *GatewayService) setGatewayStatus(ctx context.Context, workspaceID, gatewayID, status string) (*entity.GatewayDetail, error) {
	gateway, err := s.loadGatewayForMutation(ctx, workspaceID, gatewayID)
	if err != nil {
		return nil, err
	}
	gateway.Status = status
	if err := s.normalizeGateway(ctx, gateway); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateGateway(ctx, gateway); err != nil {
		return nil, err
	}
	return s.repo.GetGatewayDetail(ctx, gateway.WorkspaceID, gateway.ID)
}

func (s *GatewayService) loadGatewayForMutation(ctx context.Context, workspaceID, gatewayID string) (*entity.Gateway, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.GetGateway(ctx, workspaceID, strings.TrimSpace(gatewayID))
}

func (s *GatewayService) normalizeGateway(ctx context.Context, gateway *entity.Gateway) error {
	gateway.WorkspaceID = trimString(gateway.WorkspaceID)
	gateway.OwnerUserID = trimString(gateway.OwnerUserID)
	gateway.ZoneID = trimString(gateway.ZoneID)
	gateway.Name = trimString(gateway.Name)
	gateway.TrafficClass = defaultString(gateway.TrafficClass, "transactional")
	gateway.Status = defaultString(gateway.Status, "disabled")
	gateway.RoutingMode = defaultString(gateway.RoutingMode, "round_robin")
	gateway.FallbackGatewayID = trimString(gateway.FallbackGatewayID)
	gateway.DesiredShardCount = maxInt(gateway.DesiredShardCount, 1)
	gateway.TemplateIDs = normalizeStringSlice(gateway.TemplateIDs)
	gateway.EndpointIDs = normalizeStringSlice(gateway.EndpointIDs)

	if gateway.WorkspaceID == "" {
		return smtp_errorx.ErrWorkspaceRequired
	}
	if gateway.ZoneID == "" {
		return smtp_errorx.ErrZoneRequired
	}
	if gateway.Name == "" {
		return smtp_errorx.ErrInvalidResource
	}

	for _, templateID := range gateway.TemplateIDs {
		template, err := s.templateRepo.GetTemplateByID(ctx, templateID)
		if err != nil {
			return err
		}
		if template.WorkspaceID != gateway.WorkspaceID {
			return smtp_errorx.ErrWorkspaceMismatch
		}
		if template.ConsumerID != "" {
			consumer, err := s.consumerRepo.GetConsumerByID(ctx, template.ConsumerID)
			if err != nil {
				return err
			}
			if consumer.WorkspaceID != gateway.WorkspaceID {
				return smtp_errorx.ErrWorkspaceMismatch
			}
			if consumer.ZoneID != gateway.ZoneID {
				return smtp_errorx.ErrZoneMismatch
			}
		}
	}

	for _, endpointID := range gateway.EndpointIDs {
		endpoint, err := s.endpointRepo.GetEndpointByID(ctx, endpointID)
		if err != nil {
			return err
		}
		if endpoint.WorkspaceID != gateway.WorkspaceID {
			return smtp_errorx.ErrWorkspaceMismatch
		}
	}

	if gateway.FallbackGatewayID != "" {
		fallback, err := s.repo.GetGatewayByID(ctx, gateway.FallbackGatewayID)
		if err != nil {
			return err
		}
		if fallback.WorkspaceID != gateway.WorkspaceID {
			return smtp_errorx.ErrWorkspaceMismatch
		}
		if fallback.ZoneID != gateway.ZoneID {
			return smtp_errorx.ErrZoneMismatch
		}
	}

	return nil
}
