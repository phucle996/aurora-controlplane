package service

import (
	"context"
	"strings"
	"time"

	"controlplane/internal/primitive/rebalance"
	"controlplane/internal/smtp/domain/entity"
	smtp_domainrepo "controlplane/internal/smtp/domain/repository"
	smtp_domainsvc "controlplane/internal/smtp/domain/service"
	smtp_errorx "controlplane/internal/smtp/errorx"
)

const (
	defaultSyncInterval   = 15 * time.Second
	defaultReportInterval = 15 * time.Second
)

type RuntimeService struct {
	runtimeRepo  smtp_domainrepo.RuntimeRepository
	consumerRepo smtp_domainrepo.ConsumerRepository
	templateRepo smtp_domainrepo.TemplateRepository
	gatewayRepo  smtp_domainrepo.GatewayRepository
	endpointRepo smtp_domainrepo.EndpointRepository

	gatewayCoordinator  *rebalance.Coordinator
	consumerCoordinator *rebalance.Coordinator
}

func NewRuntimeService(runtimeRepo smtp_domainrepo.RuntimeRepository, consumerRepo smtp_domainrepo.ConsumerRepository, templateRepo smtp_domainrepo.TemplateRepository, gatewayRepo smtp_domainrepo.GatewayRepository, endpointRepo smtp_domainrepo.EndpointRepository, projection rebalance.ProjectionSink) smtp_domainsvc.RuntimeService {
	return &RuntimeService{
		runtimeRepo:  runtimeRepo,
		consumerRepo: consumerRepo,
		templateRepo: templateRepo,
		gatewayRepo:  gatewayRepo,
		endpointRepo: endpointRepo,

		gatewayCoordinator:  newGatewayCoordinator(runtimeRepo, projection),
		consumerCoordinator: newConsumerCoordinator(runtimeRepo, projection),
	}
}

func (s *RuntimeService) ListActivityLogs(ctx context.Context, workspaceID string) ([]*entity.ActivityLog, error) {
	if s == nil || s.runtimeRepo == nil {
		return nil, smtp_errorx.ErrRuntimeInvalid
	}
	return s.runtimeRepo.ListActivityLogs(ctx, workspaceID)
}

func (s *RuntimeService) ListDeliveryAttempts(ctx context.Context, workspaceID string) ([]*entity.DeliveryAttempt, error) {
	if s == nil || s.runtimeRepo == nil {
		return nil, smtp_errorx.ErrRuntimeInvalid
	}
	return s.runtimeRepo.ListDeliveryAttempts(ctx, workspaceID)
}

func (s *RuntimeService) ListRuntimeHeartbeats(ctx context.Context) ([]*entity.RuntimeHeartbeat, error) {
	if s == nil || s.runtimeRepo == nil {
		return nil, smtp_errorx.ErrRuntimeInvalid
	}
	return s.runtimeRepo.ListRuntimeHeartbeats(ctx)
}

func (s *RuntimeService) ListGatewayAssignments(ctx context.Context) ([]*entity.GatewayShardAssignment, error) {
	if s == nil || s.runtimeRepo == nil {
		return nil, smtp_errorx.ErrRuntimeInvalid
	}
	return s.runtimeRepo.ListGatewayAssignments(ctx)
}

func (s *RuntimeService) ListConsumerAssignments(ctx context.Context) ([]*entity.ConsumerShardAssignment, error) {
	if s == nil || s.runtimeRepo == nil {
		return nil, smtp_errorx.ErrRuntimeInvalid
	}
	return s.runtimeRepo.ListConsumerAssignments(ctx)
}

func (s *RuntimeService) Sync(ctx context.Context, req *entity.RuntimeSyncRequest) (*entity.RuntimeSyncResponse, error) {
	if s == nil || s.runtimeRepo == nil || s.consumerRepo == nil || s.templateRepo == nil || s.gatewayRepo == nil || s.endpointRepo == nil {
		return nil, smtp_errorx.ErrRuntimeInvalid
	}
	req = normalizeSyncRequest(req)
	if req == nil {
		return nil, smtp_errorx.ErrRuntimeInvalid
	}

	if _, err := s.runtimeRepo.GetRuntimeDataPlane(ctx, req.DataPlaneID); err != nil {
		return nil, err
	}
	if err := s.Reconcile(ctx); err != nil {
		return nil, err
	}

	return s.buildSyncResponse(ctx, req)
}

func (s *RuntimeService) Report(ctx context.Context, req *entity.RuntimeReportRequest) (*entity.RuntimeReportResponse, error) {
	if s == nil || s.runtimeRepo == nil {
		return nil, smtp_errorx.ErrRuntimeInvalid
	}
	req = normalizeReportRequest(req)
	if req == nil {
		return nil, smtp_errorx.ErrRuntimeInvalid
	}

	if _, err := s.runtimeRepo.GetRuntimeDataPlane(ctx, req.DataPlaneID); err != nil {
		return nil, err
	}

	heartbeat := &entity.RuntimeHeartbeat{
		DataPlaneID:   req.DataPlaneID,
		LocalVersion:  req.LocalVersion,
		GatewayCount:  len(req.GatewayStatuses),
		ConsumerCount: len(req.ConsumerStatuses),
		MemberState:   "ready",
		Capacity:      maxInt(req.Capacity, 1),
		GRPCAddr:      req.GRPCEndpoint,
	}
	if err := s.runtimeRepo.UpsertRuntimeHeartbeat(ctx, heartbeat); err != nil {
		return nil, err
	}
	if err := s.runtimeRepo.ReplaceGatewayStatuses(ctx, req.DataPlaneID, req.GatewayStatuses); err != nil {
		return nil, err
	}
	if err := s.runtimeRepo.ReplaceConsumerStatuses(ctx, req.DataPlaneID, req.ConsumerStatuses); err != nil {
		return nil, err
	}
	currentVersion, err := s.runtimeRepo.GetRuntimeVersionByDataPlane(ctx, req.DataPlaneID)
	if err != nil {
		return nil, err
	}

	return &entity.RuntimeReportResponse{
		ReportInterval: defaultReportInterval,
		ForceResync:    currentVersion != req.LocalVersion,
	}, nil
}

func (s *RuntimeService) Reconcile(ctx context.Context) error {
	if s == nil || s.runtimeRepo == nil {
		return smtp_errorx.ErrRuntimeInvalid
	}
	if err := s.runtimeRepo.EnsureDesiredShards(ctx); err != nil {
		return err
	}
	if s.gatewayCoordinator != nil {
		if err := s.gatewayCoordinator.Reconcile(ctx); err != nil {
			return err
		}
	}
	if s.consumerCoordinator != nil {
		if err := s.consumerCoordinator.Reconcile(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *RuntimeService) buildSyncResponse(ctx context.Context, req *entity.RuntimeSyncRequest) (*entity.RuntimeSyncResponse, error) {
	consumerAssignments, err := s.runtimeRepo.ListConsumerAssignmentsByDataPlane(ctx, req.DataPlaneID)
	if err != nil {
		return nil, err
	}
	gatewayAssignments, err := s.runtimeRepo.ListGatewayAssignmentsByDataPlane(ctx, req.DataPlaneID)
	if err != nil {
		return nil, err
	}

	consumerIDSet := make(map[string]struct{}, len(consumerAssignments))
	gatewayIDSet := make(map[string]struct{}, len(gatewayAssignments))
	templateIDSet := make(map[string]struct{})
	endpointIDSet := make(map[string]struct{})
	workspaceIDs, err := s.runtimeRepo.ListWorkspaceIDsByDataPlane(ctx, req.DataPlaneID)
	if err != nil {
		return nil, err
	}

	for _, assignment := range consumerAssignments {
		if assignment == nil || assignment.ConsumerID == "" {
			continue
		}
		if _, seen := consumerIDSet[assignment.ConsumerID]; seen {
			continue
		}
		consumerIDSet[assignment.ConsumerID] = struct{}{}
	}

	for _, assignment := range gatewayAssignments {
		if assignment == nil || assignment.GatewayID == "" {
			continue
		}
		if _, seen := gatewayIDSet[assignment.GatewayID]; seen {
			continue
		}
		gatewayIDSet[assignment.GatewayID] = struct{}{}
	}

	consumers := make([]*entity.Consumer, 0, len(consumerAssignments))
	gateways := make([]*entity.Gateway, 0, len(gatewayAssignments))
	for _, workspaceID := range workspaceIDs {
		items, err := s.consumerRepo.ListConsumersByWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item == nil {
				continue
			}
			if _, ok := consumerIDSet[item.ID]; !ok {
				continue
			}
			consumers = append(consumers, item)
		}
	}

	for _, workspaceID := range workspaceIDs {
		items, err := s.gatewayRepo.ListGatewaysByWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item == nil {
				continue
			}
			if _, ok := gatewayIDSet[item.ID]; !ok {
				continue
			}
			gateways = append(gateways, item)
			for _, templateID := range item.TemplateIDs {
				templateIDSet[templateID] = struct{}{}
			}
			for _, endpointID := range item.EndpointIDs {
				endpointIDSet[endpointID] = struct{}{}
			}
		}
	}

	templates := make([]*entity.Template, 0)
	for _, workspaceID := range workspaceIDs {
		items, err := s.templateRepo.ListTemplatesByWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item == nil {
				continue
			}
			if _, ok := templateIDSet[item.ID]; ok {
				templates = append(templates, item)
				continue
			}
			if _, ok := consumerIDSet[item.ConsumerID]; ok {
				templateIDSet[item.ID] = struct{}{}
				templates = append(templates, item)
			}
		}
	}

	endpoints := make([]*entity.Endpoint, 0, len(endpointIDSet))
	for _, workspaceID := range workspaceIDs {
		items, err := s.endpointRepo.ListEndpointsByWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item == nil {
				continue
			}
			if _, ok := endpointIDSet[item.ID]; !ok {
				continue
			}
			endpoints = append(endpoints, item)
		}
	}

	runtimeVersion := maxRuntimeVersion(req.LocalVersion, consumers, templates, gateways, endpoints, consumerAssignments, gatewayAssignments)
	fullResync := runtimeVersion != req.LocalVersion

	resp := &entity.RuntimeSyncResponse{
		RuntimeVersion:      runtimeVersion,
		ConsumerAssignments: consumerAssignments,
		GatewayAssignments:  gatewayAssignments,
		SyncInterval:        defaultSyncInterval,
		FullResync:          fullResync,
	}
	if fullResync {
		resp.Consumers = consumers
		resp.Templates = templates
		resp.Gateways = gateways
		resp.Endpoints = endpoints
	}
	return resp, nil
}

func normalizeSyncRequest(req *entity.RuntimeSyncRequest) *entity.RuntimeSyncRequest {
	if req == nil {
		return nil
	}
	req.DataPlaneID = strings.TrimSpace(req.DataPlaneID)
	req.GRPCEndpoint = strings.TrimSpace(req.GRPCEndpoint)
	req.Capacity = maxInt(req.Capacity, 1)
	if req.DataPlaneID == "" {
		return nil
	}
	return req
}

func normalizeReportRequest(req *entity.RuntimeReportRequest) *entity.RuntimeReportRequest {
	if req == nil {
		return nil
	}
	req.DataPlaneID = strings.TrimSpace(req.DataPlaneID)
	req.GRPCEndpoint = strings.TrimSpace(req.GRPCEndpoint)
	req.Capacity = maxInt(req.Capacity, 1)
	if req.DataPlaneID == "" {
		return nil
	}
	return req
}

func maxRuntimeVersion(base int64, consumers []*entity.Consumer, templates []*entity.Template, gateways []*entity.Gateway, endpoints []*entity.Endpoint, consumerAssignments []*entity.ConsumerShardAssignment, gatewayAssignments []*entity.GatewayShardAssignment) int64 {
	version := base
	for _, item := range consumers {
		if item != nil && item.RuntimeVersion > version {
			version = item.RuntimeVersion
		}
	}
	for _, item := range templates {
		if item != nil && item.RuntimeVersion > version {
			version = item.RuntimeVersion
		}
	}
	for _, item := range gateways {
		if item != nil && item.RuntimeVersion > version {
			version = item.RuntimeVersion
		}
	}
	for _, item := range endpoints {
		if item != nil && item.RuntimeVersion > version {
			version = item.RuntimeVersion
		}
	}
	for _, item := range consumerAssignments {
		if item != nil && item.Generation > version {
			version = item.Generation
		}
	}
	for _, item := range gatewayAssignments {
		if item != nil && item.Generation > version {
			version = item.Generation
		}
	}
	return version
}
