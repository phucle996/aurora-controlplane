package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"controlplane/internal/virtualmachine/domain/entity"
	core_repo "controlplane/internal/virtualmachine/domain/repository"
	vm_errorx "controlplane/internal/virtualmachine/errorx"
)

type HostService struct {
	repo core_repo.HostRepository
}

func NewHostService(repo core_repo.HostRepository) *HostService {
	return &HostService{repo: repo}
}

func (s *HostService) ListHosts(ctx context.Context, filter entity.HostListFilter) (*entity.HostPage, error) {
	if s == nil || s.repo == nil {
		return nil, vm_errorx.ErrHostUnavailable
	}

	filter.Page, filter.Limit = normalizePagination(filter.Page, filter.Limit)
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.ZoneSlug = strings.TrimSpace(filter.ZoneSlug)
	if filter.Status != "" && !isAllowedStatus(filter.Status, hostStatuses) {
		return nil, vm_errorx.ErrHostInvalid
	}

	return s.repo.ListHosts(ctx, filter)
}

func (s *HostService) GetHost(ctx context.Context, hostID string) (*entity.Host, error) {
	if s == nil || s.repo == nil {
		return nil, vm_errorx.ErrHostUnavailable
	}

	hostID = strings.TrimSpace(hostID)
	if hostID == "" {
		return nil, vm_errorx.ErrHostInvalid
	}

	return s.repo.GetHostByID(ctx, hostID)
}

func (s *HostService) GetHostBinding(ctx context.Context, hostID, agentID string) (*entity.HostBinding, error) {
	if s == nil || s.repo == nil {
		return nil, vm_errorx.ErrHostUnavailable
	}

	hostID = strings.TrimSpace(hostID)
	agentID = strings.TrimSpace(agentID)
	if hostID == "" || agentID == "" {
		return nil, vm_errorx.ErrHostInvalid
	}

	binding := &entity.HostBinding{
		HostID:           hostID,
		RequestedAgentID: agentID,
		Allowed:          true,
	}

	host, err := s.repo.GetHostByID(ctx, hostID)
	if err != nil {
		if errors.Is(err, vm_errorx.ErrHostNotFound) {
			return binding, nil
		}
		return nil, err
	}

	binding.Current = host
	binding.BoundAgentID = host.AgentID
	binding.Allowed = strings.TrimSpace(host.AgentID) == agentID
	return binding, nil
}

func (s *HostService) UpsertHost(ctx context.Context, host *entity.Host) (*entity.Host, error) {
	if s == nil || s.repo == nil || host == nil {
		return nil, vm_errorx.ErrHostInvalid
	}

	host.HostID = strings.TrimSpace(host.HostID)
	host.AgentID = strings.TrimSpace(host.AgentID)
	host.ZoneSlug = strings.TrimSpace(host.ZoneSlug)
	host.DataPlaneID = strings.TrimSpace(host.DataPlaneID)
	host.Hostname = strings.TrimSpace(host.Hostname)
	host.PrivateIP = strings.TrimSpace(host.PrivateIP)
	host.HypervisorType = strings.TrimSpace(host.HypervisorType)
	host.AgentVersion = strings.TrimSpace(host.AgentVersion)
	host.CapabilitiesJSON = strings.TrimSpace(host.CapabilitiesJSON)
	host.Status = strings.ToLower(strings.TrimSpace(host.Status))

	if host.HostID == "" || host.AgentID == "" || host.ZoneSlug == "" || host.DataPlaneID == "" || host.Hostname == "" || host.HypervisorType == "" {
		return nil, vm_errorx.ErrHostInvalid
	}
	if host.Status == "" {
		host.Status = "online"
	}
	if !isAllowedStatus(host.Status, hostStatuses) {
		return nil, vm_errorx.ErrHostInvalid
	}
	if host.CapabilitiesJSON == "" {
		host.CapabilitiesJSON = "{}"
	}
	if host.LastSeenAt == nil {
		now := time.Now().UTC()
		host.LastSeenAt = &now
	}

	existingByAgent, err := s.repo.GetHostByAgentID(ctx, host.AgentID)
	if err != nil && !errors.Is(err, vm_errorx.ErrHostNotFound) {
		return nil, err
	}
	if existingByAgent != nil && existingByAgent.HostID != host.HostID {
		return nil, vm_errorx.ErrHostConflict
	}

	return s.repo.UpsertHost(ctx, host)
}

func (s *HostService) UpdateHostStatus(ctx context.Context, hostID, dataPlaneID, status string, lastSeenAt time.Time) (*entity.Host, error) {
	if s == nil || s.repo == nil {
		return nil, vm_errorx.ErrHostUnavailable
	}

	hostID = strings.TrimSpace(hostID)
	dataPlaneID = strings.TrimSpace(dataPlaneID)
	status = strings.ToLower(strings.TrimSpace(status))
	if hostID == "" || dataPlaneID == "" {
		return nil, vm_errorx.ErrHostInvalid
	}
	if status == "" {
		status = "online"
	}
	if !isAllowedStatus(status, hostStatuses) {
		return nil, vm_errorx.ErrHostInvalid
	}
	if lastSeenAt.IsZero() {
		lastSeenAt = time.Now().UTC()
	}

	return s.repo.UpdateHostStatus(ctx, hostID, dataPlaneID, status, lastSeenAt.UTC())
}

func (s *HostService) ListHostOptions(ctx context.Context, filter entity.HostListFilter) ([]*entity.HostOption, error) {
	if s == nil || s.repo == nil {
		return nil, vm_errorx.ErrHostUnavailable
	}

	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.ZoneSlug = strings.TrimSpace(filter.ZoneSlug)
	if filter.Status != "" && !isAllowedStatus(filter.Status, hostStatuses) {
		return nil, vm_errorx.ErrHostInvalid
	}

	return s.repo.ListHostOptions(ctx, filter)
}

var hostStatuses = map[string]struct{}{
	"online":      {},
	"offline":     {},
	"degraded":    {},
	"quarantined": {},
}

func normalizePagination(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func isAllowedStatus(status string, allowed map[string]struct{}) bool {
	_, ok := allowed[status]
	return ok
}
