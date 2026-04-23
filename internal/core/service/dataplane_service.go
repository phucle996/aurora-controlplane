package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"controlplane/internal/config"
	"controlplane/internal/core/domain/entity"
	core_domainrepo "controlplane/internal/core/domain/repository"
	core_errorx "controlplane/internal/core/errorx"
	"controlplane/internal/security"
	"controlplane/pkg/id"
)

type DataPlaneService struct {
	repo     core_domainrepo.DataPlaneRepository
	zoneRepo core_domainrepo.ZoneRepository
	cfg      *config.Config
	ca       *security.CertificateAuthority
}

func NewDataPlaneService(repo core_domainrepo.DataPlaneRepository, zoneRepo core_domainrepo.ZoneRepository, cfg *config.Config, ca *security.CertificateAuthority) *DataPlaneService {
	return &DataPlaneService{
		repo:     repo,
		zoneRepo: zoneRepo,
		cfg:      cfg,
		ca:       ca,
	}
}

func (s *DataPlaneService) Enroll(ctx context.Context, dataPlane *entity.DataPlane, bootstrapToken, csrPEM string) (*entity.DataPlaneEnrollResult, error) {
	if s == nil || s.repo == nil || s.zoneRepo == nil || s.cfg == nil || s.ca == nil {
		return nil, core_errorx.ErrDataPlaneUnavailable
	}
	if strings.TrimSpace(bootstrapToken) == "" || bootstrapToken != strings.TrimSpace(s.cfg.GRPC.DataPlaneEnrollToken) {
		return nil, core_errorx.ErrDataPlaneEnrollDenied
	}
	if dataPlane == nil {
		return nil, core_errorx.ErrDataPlaneInvalid
	}

	dataPlane.NodeKey = strings.TrimSpace(dataPlane.NodeKey)
	dataPlane.Name = strings.TrimSpace(dataPlane.Name)
	dataPlane.ZoneSlug = normalizeZoneSlug(dataPlane.ZoneSlug)
	dataPlane.GRPCEndpoint = strings.TrimSpace(dataPlane.GRPCEndpoint)
	dataPlane.Version = strings.TrimSpace(dataPlane.Version)

	if dataPlane.NodeKey == "" || dataPlane.Name == "" || dataPlane.ZoneSlug == "" || dataPlane.GRPCEndpoint == "" {
		return nil, core_errorx.ErrDataPlaneInvalid
	}

	zone, err := s.zoneRepo.GetZoneBySlug(ctx, dataPlane.ZoneSlug)
	if err != nil {
		return nil, core_errorx.ErrDataPlaneInvalid
	}
	dataPlane.ZoneID = zone.ID

	if _, err := security.ParseCertificateRequestPEM(csrPEM); err != nil {
		return nil, core_errorx.ErrDataPlaneCSRInvalid
	}

	existing, err := s.repo.GetByNodeKey(ctx, dataPlane.NodeKey)
	if err != nil && !errors.Is(err, core_errorx.ErrDataPlaneNotFound) {
		return nil, err
	}

	if existing != nil && existing.ID != "" {
		dataPlane.ID = existing.ID
	}

	if dataPlane.ID == "" {
		dataPlaneID, err := id.Generate()
		if err != nil {
			return nil, err
		}
		dataPlane.ID = dataPlaneID
	}

	dataPlane.Status = "healthy"
	now := time.Now().UTC()
	dataPlane.LastSeenAt = &now

	clientCertPEM, certSerial, certNotAfter, err := s.ca.SignDataPlaneClientCertificate(csrPEM, dataPlane.ID, s.cfg.GRPC.DataPlaneClientCertTTL)
	if err != nil {
		return nil, core_errorx.ErrDataPlaneCSRInvalid
	}

	dataPlane.CertSerial = certSerial
	dataPlane.CertNotAfter = &certNotAfter

	saved, err := s.repo.SaveEnrollment(ctx, dataPlane)
	if err != nil {
		return nil, err
	}

	if saved.ID != dataPlane.ID {
		dataPlane.ID = saved.ID
		clientCertPEM, certSerial, certNotAfter, err = s.ca.SignDataPlaneClientCertificate(csrPEM, dataPlane.ID, s.cfg.GRPC.DataPlaneClientCertTTL)
		if err != nil {
			return nil, core_errorx.ErrDataPlaneCSRInvalid
		}
		dataPlane.CertSerial = certSerial
		dataPlane.CertNotAfter = &certNotAfter

		saved, err = s.repo.SaveEnrollment(ctx, dataPlane)
		if err != nil {
			return nil, err
		}
	}

	return &entity.DataPlaneEnrollResult{
		DataPlaneID:       saved.ID,
		ClientCertPEM:     clientCertPEM,
		CACertPEM:         s.ca.CertPEM(),
		CertNotAfter:      certNotAfter,
		HeartbeatInterval: s.cfg.GRPC.DataPlaneHeartbeatInterval,
	}, nil
}

func (s *DataPlaneService) Heartbeat(ctx context.Context, dataPlane *entity.DataPlane, peerDataPlaneID string) (*entity.DataPlaneHeartbeatResult, error) {
	if s == nil || s.repo == nil || s.cfg == nil {
		return nil, core_errorx.ErrDataPlaneUnavailable
	}
	if dataPlane == nil {
		return nil, core_errorx.ErrDataPlaneInvalid
	}

	dataPlane.ID = strings.TrimSpace(dataPlane.ID)
	dataPlane.GRPCEndpoint = strings.TrimSpace(dataPlane.GRPCEndpoint)
	dataPlane.Version = strings.TrimSpace(dataPlane.Version)
	peerDataPlaneID = strings.TrimSpace(peerDataPlaneID)

	if dataPlane.ID == "" || dataPlane.GRPCEndpoint == "" {
		return nil, core_errorx.ErrDataPlaneInvalid
	}
	if peerDataPlaneID == "" || peerDataPlaneID != dataPlane.ID {
		return nil, core_errorx.ErrDataPlanePeerInvalid
	}

	if err := s.repo.UpdateHeartbeat(ctx, dataPlane.ID, dataPlane.GRPCEndpoint, dataPlane.Version, "healthy", time.Now().UTC()); err != nil {
		return nil, err
	}

	return &entity.DataPlaneHeartbeatResult{
		HeartbeatInterval: s.cfg.GRPC.DataPlaneHeartbeatInterval,
	}, nil
}

func (s *DataPlaneService) MarkStale(ctx context.Context, now time.Time) (int64, error) {
	if s == nil || s.repo == nil || s.cfg == nil {
		return 0, core_errorx.ErrDataPlaneUnavailable
	}
	if s.cfg.GRPC.DataPlaneHeartbeatStaleTimeout <= 0 {
		return 0, nil
	}

	return s.repo.MarkStaleBefore(ctx, now.UTC().Add(-s.cfg.GRPC.DataPlaneHeartbeatStaleTimeout))
}

func (s *DataPlaneService) ListHealthyByZoneID(ctx context.Context, zoneID string) ([]*entity.DataPlane, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrDataPlaneUnavailable
	}
	return s.repo.ListHealthyByZoneID(ctx, strings.TrimSpace(zoneID))
}

func normalizeZoneSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	return strings.Trim(value, "-")
}
