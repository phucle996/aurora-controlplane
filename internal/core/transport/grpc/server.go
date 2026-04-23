package controlplanegrpc

import (
	"context"
	"errors"

	"controlplane/internal/core/domain/entity"
	core_domainsvc "controlplane/internal/core/domain/service"
	core_errorx "controlplane/internal/core/errorx"
	"controlplane/internal/security"
	"controlplane/pkg/logger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DataPlaneRegistryHandler implements the DataPlaneRegistryServer gRPC interface.
type DataPlaneRegistryHandler struct {
	UnimplementedDataPlaneRegistryServer

	svc core_domainsvc.DataPlaneService
}

func NewDataPlaneRegistryServer(svc core_domainsvc.DataPlaneService) *DataPlaneRegistryHandler {
	return &DataPlaneRegistryHandler{svc: svc}
}

func (s *DataPlaneRegistryHandler) EnrollDataPlane(ctx context.Context, req *EnrollDataPlaneRequest) (*EnrollDataPlaneResponse, error) {
	if s == nil || s.svc == nil {
		return nil, status.Error(codes.Unavailable, "registry unavailable")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid enroll request")
	}

	result, err := s.svc.Enroll(ctx, &entity.DataPlane{
		NodeKey:      req.GetNodeKey(),
		Name:         req.GetName(),
		ZoneSlug:     req.GetZoneSlug(),
		GRPCEndpoint: req.GetGrpcEndpoint(),
		Version:      req.GetVersion(),
	}, req.GetBootstrapToken(), req.GetCsrPem())
	if err != nil {
		logger.SysWarn("core.grpc.enroll", "dataplane enroll failed")
		return nil, mapRegistryError(err, true)
	}

	logger.SysInfo("core.grpc.enroll", "dataplane enrolled successfully")
	return &EnrollDataPlaneResponse{
		DataPlaneId:              result.DataPlaneID,
		ClientCertPem:            result.ClientCertPEM,
		CaCertPem:                result.CACertPEM,
		CertNotAfter:             timestamppb.New(result.CertNotAfter),
		HeartbeatIntervalSeconds: uint32(result.HeartbeatInterval.Seconds()),
	}, nil
}

func (s *DataPlaneRegistryHandler) HeartbeatDataPlane(ctx context.Context, req *HeartbeatDataPlaneRequest) (*HeartbeatDataPlaneResponse, error) {
	if s == nil || s.svc == nil {
		return nil, status.Error(codes.Unavailable, "registry unavailable")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid heartbeat request")
	}

	peerDataPlaneID, err := dataPlaneIDFromPeer(ctx)
	if err != nil {
		logger.SysWarn("core.grpc.heartbeat", "dataplane heartbeat peer validation failed")
		return nil, status.Error(codes.Unauthenticated, "invalid dataplane identity")
	}

	result, err := s.svc.Heartbeat(ctx, &entity.DataPlane{
		ID:           req.GetDataPlaneId(),
		GRPCEndpoint: req.GetGrpcEndpoint(),
		Version:      req.GetVersion(),
	}, peerDataPlaneID)
	if err != nil {
		logger.SysWarn("core.grpc.heartbeat", "dataplane heartbeat failed")
		return nil, mapRegistryError(err, false)
	}

	return &HeartbeatDataPlaneResponse{
		HeartbeatIntervalSeconds: uint32(result.HeartbeatInterval.Seconds()),
	}, nil
}

func dataPlaneIDFromPeer(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok || p.AuthInfo == nil {
		return "", core_errorx.ErrDataPlanePeerInvalid
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", core_errorx.ErrDataPlanePeerInvalid
	}
	if len(tlsInfo.State.PeerCertificates) == 0 {
		return "", core_errorx.ErrDataPlanePeerInvalid
	}

	return security.DataPlaneIDFromCertificate(tlsInfo.State.PeerCertificates[0])
}

func mapRegistryError(err error, enroll bool) error {
	switch {
	case errors.Is(err, core_errorx.ErrDataPlaneEnrollDenied):
		return status.Error(codes.Unauthenticated, "invalid enroll credentials")
	case errors.Is(err, core_errorx.ErrDataPlaneCSRInvalid):
		return status.Error(codes.InvalidArgument, "invalid certificate request")
	case errors.Is(err, core_errorx.ErrDataPlanePeerInvalid):
		return status.Error(codes.Unauthenticated, "invalid dataplane identity")
	case errors.Is(err, core_errorx.ErrDataPlaneNotFound):
		return status.Error(codes.NotFound, "data plane not found")
	case errors.Is(err, core_errorx.ErrDataPlaneInvalid):
		if enroll {
			return status.Error(codes.InvalidArgument, "invalid enroll request")
		}
		return status.Error(codes.InvalidArgument, "invalid heartbeat request")
	default:
		return status.Error(codes.Internal, "registry request failed")
	}
}
