package virtualmachinegrpc

import (
	"context"
	"errors"
	"strings"
	"time"

	"controlplane/internal/security"
	vm_domainentity "controlplane/internal/virtualmachine/domain/entity"
	vm_domainsvc "controlplane/internal/virtualmachine/domain/service"
	vm_errorx "controlplane/internal/virtualmachine/errorx"
	"controlplane/pkg/logger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// HostRegistryHandler implements the VirtualMachineRegistryServer gRPC interface.
type HostRegistryHandler struct {
	UnimplementedVirtualMachineRegistryServer

	svc vm_domainsvc.HostService
}

func NewHostRegistryServer(svc vm_domainsvc.HostService) *HostRegistryHandler {
	return &HostRegistryHandler{svc: svc}
}

func (s *HostRegistryHandler) UpsertHost(ctx context.Context, req *UpsertHostRequest) (*UpsertHostResponse, error) {
	if s == nil || s.svc == nil {
		return nil, status.Error(codes.Unavailable, "virtual machine registry unavailable")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid host request")
	}

	peerDataPlaneID, err := dataPlaneIDFromPeer(ctx)
	if err != nil || peerDataPlaneID != strings.TrimSpace(req.GetDataPlaneId()) {
		logger.SysWarn("virtual-machine.grpc.upsert", "peer dataplane validation failed")
		return nil, status.Error(codes.Unauthenticated, "invalid dataplane identity")
	}

	result, err := s.svc.UpsertHost(ctx, &vm_domainentity.Host{
		HostID:           req.GetHostId(),
		AgentID:          req.GetAgentId(),
		ZoneSlug:         req.GetZoneSlug(),
		DataPlaneID:      req.GetDataPlaneId(),
		Hostname:         req.GetHostname(),
		PrivateIP:        req.GetPrivateIp(),
		HypervisorType:   req.GetHypervisorType(),
		AgentVersion:     req.GetAgentVersion(),
		CapabilitiesJSON: req.GetCapabilitiesJson(),
		CPUCores:         req.GetCpuCores(),
		MemoryBytes:      req.GetMemoryBytes(),
		DiskBytes:        req.GetDiskBytes(),
		Status:           req.GetStatus(),
		LastSeenAt:       timestamppbToTime(req.GetLastSeenAt()),
	})
	if err != nil {
		logger.SysWarn("virtual-machine.grpc.upsert", "host upsert failed")
		return nil, mapHostError(err)
	}

	return &UpsertHostResponse{Host: hostToPB(result)}, nil
}

func (s *HostRegistryHandler) UpdateHostStatus(ctx context.Context, req *UpdateHostStatusRequest) (*UpdateHostStatusResponse, error) {
	if s == nil || s.svc == nil {
		return nil, status.Error(codes.Unavailable, "virtual machine registry unavailable")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid host status request")
	}

	peerDataPlaneID, err := dataPlaneIDFromPeer(ctx)
	if err != nil || peerDataPlaneID != strings.TrimSpace(req.GetDataPlaneId()) {
		return nil, status.Error(codes.Unauthenticated, "invalid dataplane identity")
	}

	lastSeenAt := time.Now().UTC()
	if ts := timestamppbToTime(req.GetLastSeenAt()); ts != nil {
		lastSeenAt = *ts
	}

	result, err := s.svc.UpdateHostStatus(ctx, req.GetHostId(), req.GetDataPlaneId(), req.GetStatus(), lastSeenAt)
	if err != nil {
		return nil, mapHostError(err)
	}

	return &UpdateHostStatusResponse{Host: hostToPB(result)}, nil
}

func (s *HostRegistryHandler) GetHostBinding(ctx context.Context, req *GetHostBindingRequest) (*GetHostBindingResponse, error) {
	if s == nil || s.svc == nil {
		return nil, status.Error(codes.Unavailable, "virtual machine registry unavailable")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid binding request")
	}

	peerDataPlaneID, err := dataPlaneIDFromPeer(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid dataplane identity")
	}
	_ = peerDataPlaneID

	result, err := s.svc.GetHostBinding(ctx, req.GetHostId(), req.GetAgentId())
	if err != nil {
		return nil, mapHostError(err)
	}

	resp := &GetHostBindingResponse{
		Allowed: result.Allowed,
	}
	if result.Current != nil {
		resp.Host = hostToPB(result.Current)
	}
	return resp, nil
}

func dataPlaneIDFromPeer(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok || p.AuthInfo == nil {
		return "", vm_errorx.ErrHostForbidden
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", vm_errorx.ErrHostForbidden
	}
	if len(tlsInfo.State.PeerCertificates) == 0 {
		return "", vm_errorx.ErrHostForbidden
	}

	return security.DataPlaneIDFromCertificate(tlsInfo.State.PeerCertificates[0])
}

func mapHostError(err error) error {
	switch {
	case errors.Is(err, vm_errorx.ErrHostNotFound):
		return status.Error(codes.NotFound, "host not found")
	case errors.Is(err, vm_errorx.ErrHostInvalid):
		return status.Error(codes.InvalidArgument, "invalid host request")
	case errors.Is(err, vm_errorx.ErrHostConflict):
		return status.Error(codes.AlreadyExists, "host already bound to another agent")
	case errors.Is(err, vm_errorx.ErrHostForbidden):
		return status.Error(codes.PermissionDenied, "host access denied")
	default:
		return status.Error(codes.Internal, "virtual machine registry request failed")
	}
}

func hostToPB(item *vm_domainentity.Host) *Host {
	if item == nil {
		return nil
	}
	return &Host{
		HostId:           item.HostID,
		AgentId:          item.AgentID,
		ZoneId:           item.ZoneID,
		ZoneSlug:         item.ZoneSlug,
		DataPlaneId:      item.DataPlaneID,
		Hostname:         item.Hostname,
		PrivateIp:        item.PrivateIP,
		HypervisorType:   item.HypervisorType,
		AgentVersion:     item.AgentVersion,
		CapabilitiesJson: item.CapabilitiesJSON,
		CpuCores:         item.CPUCores,
		MemoryBytes:      item.MemoryBytes,
		DiskBytes:        item.DiskBytes,
		Status:           item.Status,
		LastSeenAt:       timeToPB(item.LastSeenAt),
		CreatedAt:        timestamppb.New(item.CreatedAt),
		UpdatedAt:        timestamppb.New(item.UpdatedAt),
	}
}

func timestamppbToTime(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	value := ts.AsTime().UTC()
	return &value
}

func timeToPB(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(t.UTC())
}
