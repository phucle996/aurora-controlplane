package smtpgrpc

import (
	"context"
	"errors"

	"controlplane/internal/security"
	"controlplane/internal/smtp/domain/entity"
	smtp_domainsvc "controlplane/internal/smtp/domain/service"
	smtp_errorx "controlplane/internal/smtp/errorx"
	"controlplane/pkg/logger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type RuntimeHandler struct {
	UnimplementedSMTPRuntimeServer

	svc smtp_domainsvc.RuntimeService
}

func NewRuntimeServer(svc smtp_domainsvc.RuntimeService) *RuntimeHandler {
	return &RuntimeHandler{svc: svc}
}

func (h *RuntimeHandler) SyncSMTPRuntime(ctx context.Context, req *SyncSMTPRuntimeRequest) (*SyncSMTPRuntimeResponse, error) {
	if h == nil || h.svc == nil {
		return nil, status.Error(codes.Unavailable, "smtp runtime unavailable")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sync request")
	}

	peerDataPlaneID, err := dataPlaneIDFromPeer(ctx)
	if err != nil || peerDataPlaneID != req.GetDataPlaneId() {
		logger.SysWarn("smtp.grpc.sync", "smtp runtime peer validation failed")
		return nil, status.Error(codes.Unauthenticated, "invalid dataplane identity")
	}

	resp, err := h.svc.Sync(ctx, &entity.RuntimeSyncRequest{
		DataPlaneID:  req.GetDataPlaneId(),
		LocalVersion: req.GetLocalVersion(),
		Capacity:     int(req.GetCapacity()),
		GRPCEndpoint: req.GetGrpcEndpoint(),
	})
	if err != nil {
		logger.SysWarn("smtp.grpc.sync", "smtp runtime sync failed")
		return nil, mapRuntimeError(err)
	}

	return runtimeSyncResponseToPB(resp), nil
}

func (h *RuntimeHandler) ReportSMTPRuntime(ctx context.Context, req *ReportSMTPRuntimeRequest) (*ReportSMTPRuntimeResponse, error) {
	if h == nil || h.svc == nil {
		return nil, status.Error(codes.Unavailable, "smtp runtime unavailable")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid report request")
	}

	peerDataPlaneID, err := dataPlaneIDFromPeer(ctx)
	if err != nil || peerDataPlaneID != req.GetDataPlaneId() {
		logger.SysWarn("smtp.grpc.report", "smtp runtime peer validation failed")
		return nil, status.Error(codes.Unauthenticated, "invalid dataplane identity")
	}

	resp, err := h.svc.Report(ctx, &entity.RuntimeReportRequest{
		DataPlaneID:      req.GetDataPlaneId(),
		LocalVersion:     req.GetLocalVersion(),
		Capacity:         int(req.GetCapacity()),
		GRPCEndpoint:     req.GetGrpcEndpoint(),
		ConsumerStatuses: consumerStatusesFromPB(req.GetConsumerStatuses()),
		GatewayStatuses:  gatewayStatusesFromPB(req.GetGatewayStatuses()),
	})
	if err != nil {
		logger.SysWarn("smtp.grpc.report", "smtp runtime report failed")
		return nil, mapRuntimeError(err)
	}

	return &ReportSMTPRuntimeResponse{
		ReportIntervalSeconds: uint32(resp.ReportInterval.Seconds()),
		ForceResync:           resp.ForceResync,
	}, nil
}

func runtimeSyncResponseToPB(resp *entity.RuntimeSyncResponse) *SyncSMTPRuntimeResponse {
	if resp == nil {
		return &SyncSMTPRuntimeResponse{}
	}

	out := &SyncSMTPRuntimeResponse{
		RuntimeVersion:      resp.RuntimeVersion,
		SyncIntervalSeconds: uint32(resp.SyncInterval.Seconds()),
		FullResync:          resp.FullResync,
		ConsumerAssignments: make([]*ConsumerShardAssignment, 0, len(resp.ConsumerAssignments)),
		GatewayAssignments:  make([]*GatewayShardAssignment, 0, len(resp.GatewayAssignments)),
	}
	for _, item := range resp.Consumers {
		out.Consumers = append(out.Consumers, consumerToPB(item))
	}
	for _, item := range resp.Templates {
		out.Templates = append(out.Templates, templateToPB(item))
	}
	for _, item := range resp.Gateways {
		out.Gateways = append(out.Gateways, gatewayToPB(item))
	}
	for _, item := range resp.Endpoints {
		out.Endpoints = append(out.Endpoints, endpointToPB(item))
	}
	for _, item := range resp.ConsumerAssignments {
		if item == nil {
			continue
		}
		out.ConsumerAssignments = append(out.ConsumerAssignments, &ConsumerShardAssignment{
			ConsumerId:                item.ConsumerID,
			ShardId:                   int32(item.ShardID),
			DataPlaneId:               item.DataPlaneID,
			Generation:                item.Generation,
			AssignmentState:           item.AssignmentState,
			DesiredState:              item.DesiredState,
			LeaseExpiresAt:            timestamppb.New(item.LeaseExpiresAt),
			TargetGatewayId:           item.TargetGatewayID,
			TargetGatewayShardId:      int32(item.TargetShardID),
			TargetGatewayDataPlaneId:  item.TargetPlaneID,
			TargetGatewayGrpcEndpoint: item.TargetGRPCAddr,
		})
	}
	for _, item := range resp.GatewayAssignments {
		if item == nil {
			continue
		}
		out.GatewayAssignments = append(out.GatewayAssignments, &GatewayShardAssignment{
			GatewayId:       item.GatewayID,
			ShardId:         int32(item.ShardID),
			DataPlaneId:     item.DataPlaneID,
			Generation:      item.Generation,
			AssignmentState: item.AssignmentState,
			DesiredState:    item.DesiredState,
			LeaseExpiresAt:  timestamppb.New(item.LeaseExpiresAt),
			GrpcEndpoint:    item.GRPCEndpoint,
		})
	}
	return out
}

func consumerToPB(item *entity.Consumer) *ConsumerResource {
	if item == nil {
		return nil
	}
	return &ConsumerResource{
		Id:                   item.ID,
		WorkspaceId:          item.WorkspaceID,
		ZoneId:               item.ZoneID,
		Name:                 item.Name,
		TransportType:        item.TransportType,
		Source:               item.Source,
		ConsumerGroup:        item.ConsumerGroup,
		WorkerConcurrency:    int32(item.WorkerConcurrency),
		AckTimeoutSeconds:    int32(item.AckTimeoutSeconds),
		BatchSize:            int32(item.BatchSize),
		Status:               item.Status,
		Note:                 item.Note,
		ConnectionConfigJson: item.ConnectionConfig,
		SecretConfigJson:     item.SecretConfig,
		SecretRef:            item.SecretRef,
		SecretVersion:        item.SecretVersion,
		DesiredShardCount:    int32(item.DesiredShardCount),
		RuntimeVersion:       item.RuntimeVersion,
	}
}

func templateToPB(item *entity.Template) *TemplateResource {
	if item == nil {
		return nil
	}
	return &TemplateResource{
		Id:                  item.ID,
		WorkspaceId:         item.WorkspaceID,
		Name:                item.Name,
		Category:            item.Category,
		TrafficClass:        item.TrafficClass,
		Subject:             item.Subject,
		FromEmail:           item.FromEmail,
		ToEmail:             item.ToEmail,
		Status:              item.Status,
		Variables:           item.Variables,
		ConsumerId:          item.ConsumerID,
		ActiveVersion:       int32(item.ActiveVersion),
		RetryMaxAttempts:    int32(item.RetryMaxAttempts),
		RetryBackoffSeconds: int32(item.RetryBackoffSeconds),
		TextBody:            item.TextBody,
		HtmlBody:            item.HTMLBody,
		RuntimeVersion:      item.RuntimeVersion,
	}
}

func gatewayToPB(item *entity.Gateway) *GatewayResource {
	if item == nil {
		return nil
	}
	return &GatewayResource{
		Id:                item.ID,
		WorkspaceId:       item.WorkspaceID,
		ZoneId:            item.ZoneID,
		Name:              item.Name,
		TrafficClass:      item.TrafficClass,
		Status:            item.Status,
		RoutingMode:       item.RoutingMode,
		Priority:          int32(item.Priority),
		FallbackGatewayId: item.FallbackGatewayID,
		DesiredShardCount: int32(item.DesiredShardCount),
		TemplateIds:       item.TemplateIDs,
		EndpointIds:       item.EndpointIDs,
		RuntimeVersion:    item.RuntimeVersion,
	}
}

func endpointToPB(item *entity.Endpoint) *EndpointResource {
	if item == nil {
		return nil
	}
	return &EndpointResource{
		Id:                   item.ID,
		WorkspaceId:          item.WorkspaceID,
		Name:                 item.Name,
		ProviderKind:         item.ProviderKind,
		Host:                 item.Host,
		Port:                 int32(item.Port),
		Username:             item.Username,
		Priority:             int32(item.Priority),
		Weight:               int32(item.Weight),
		MaxConnections:       int32(item.MaxConnections),
		MaxParallelSends:     int32(item.MaxParallelSends),
		MaxMessagesPerSecond: int32(item.MaxMessagesPerSecond),
		Burst:                int32(item.Burst),
		WarmupState:          item.WarmupState,
		Status:               item.Status,
		TlsMode:              item.TLSMode,
		Password:             item.Password,
		CaCertPem:            item.CACertPEM,
		ClientCertPem:        item.ClientCertPEM,
		ClientKeyPem:         item.ClientKeyPEM,
		SecretRef:            item.SecretRef,
		SecretVersion:        item.SecretVersion,
		RuntimeVersion:       item.RuntimeVersion,
	}
}

func consumerStatusesFromPB(items []*ConsumerShardStatus) []*entity.ConsumerShardStatus {
	out := make([]*entity.ConsumerShardStatus, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, &entity.ConsumerShardStatus{
			ConsumerID:      item.GetConsumerId(),
			ShardID:         int(item.GetShardId()),
			GatewayID:       item.GetGatewayId(),
			Status:          item.GetStatus(),
			InflightCount:   item.GetInflightCount(),
			BrokerLag:       item.GetBrokerLag(),
			OldestUnackedMS: item.GetOldestUnackedAgeMs(),
			DesiredWorkers:  int(item.GetDesiredWorkers()),
			ActiveWorkers:   int(item.GetActiveWorkers()),
			RelayQueueDepth: item.GetRelayQueueDepth(),
			LastError:       item.GetLastError(),
			Generation:      item.GetGeneration(),
			AssignmentState: item.GetAssignmentState(),
			RevokingDone:    item.GetRevokingDone(),
		})
	}
	return out
}

func gatewayStatusesFromPB(items []*GatewayShardStatus) []*entity.GatewayShardStatus {
	out := make([]*entity.GatewayShardStatus, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, &entity.GatewayShardStatus{
			GatewayID:       item.GetGatewayId(),
			ShardID:         int(item.GetShardId()),
			Status:          item.GetStatus(),
			InflightCount:   item.GetInflightCount(),
			DesiredWorkers:  int(item.GetDesiredWorkers()),
			ActiveWorkers:   int(item.GetActiveWorkers()),
			RelayQueueDepth: item.GetRelayQueueDepth(),
			PoolOpenConns:   int(item.GetPoolOpenConns()),
			PoolBusyConns:   int(item.GetPoolBusyConns()),
			SendRate:        item.GetSendRatePerSecond(),
			Backpressure:    item.GetBackpressureState(),
			LastError:       item.GetLastError(),
			Generation:      item.GetGeneration(),
			AssignmentState: item.GetAssignmentState(),
			RevokingDone:    item.GetRevokingDone(),
		})
	}
	return out
}

func dataPlaneIDFromPeer(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok || p.AuthInfo == nil {
		return "", smtp_errorx.ErrDataPlaneNotReady
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", smtp_errorx.ErrDataPlaneNotReady
	}
	if len(tlsInfo.State.PeerCertificates) == 0 {
		return "", smtp_errorx.ErrDataPlaneNotReady
	}

	return security.DataPlaneIDFromCertificate(tlsInfo.State.PeerCertificates[0])
}

func mapRuntimeError(err error) error {
	switch {
	case errors.Is(err, smtp_errorx.ErrRuntimeInvalid):
		return status.Error(codes.InvalidArgument, "invalid runtime request")
	case errors.Is(err, smtp_errorx.ErrDataPlaneNotFound):
		return status.Error(codes.NotFound, "dataplane not found")
	case errors.Is(err, smtp_errorx.ErrDataPlaneNotReady):
		return status.Error(codes.FailedPrecondition, "dataplane not ready")
	default:
		return status.Error(codes.Internal, "smtp runtime request failed")
	}
}
