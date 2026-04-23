package core

import (
	"context"
	"fmt"
	"time"

	"controlplane/internal/app/bootstrap"
	"controlplane/internal/config"
	core_repo "controlplane/internal/core/repository"
	core_svc "controlplane/internal/core/service"
	core_grpc "controlplane/internal/core/transport/grpc"
	core_handler "controlplane/internal/core/transport/http/handler"
	"controlplane/internal/security"
	"controlplane/pkg/logger"
)

// Module encapsulates core dependencies and wiring.
type Module struct {
	Cfg   *config.Config
	Infra *bootstrap.Infra
	GRPC  *bootstrap.GRPC

	ZoneRepo    *core_repo.ZoneRepository
	ZoneService *core_svc.ZoneService
	ZoneHandler *core_handler.ZoneHandler

	TenantRepo    *core_repo.TenantRepository
	TenantService *core_svc.TenantService
	TenantHandler *core_handler.TenantHandler

	WorkspaceRepo    *core_repo.WorkspaceRepository
	WorkspaceService *core_svc.WorkspaceService
	WorkspaceHandler *core_handler.WorkspaceHandler

	SecretRepo    *core_repo.SecretKeyVersionRepository
	SecretService *core_svc.SecretService

	DataPlaneRepo       *core_repo.DataPlaneRepository
	DataPlaneService    *core_svc.DataPlaneService
	DataPlaneGRPCServer *core_grpc.DataPlaneRegistryHandler

	stopStale  context.CancelFunc
	stopSecret context.CancelFunc
}

// NewModule constructs the core module dependency graph.
func NewModule(cfg *config.Config, infra *bootstrap.Infra, grpcServer *bootstrap.GRPC) (*Module, error) {
	m := &Module{
		Cfg:   cfg,
		Infra: infra,
		GRPC:  grpcServer,
	}

	m.SecretRepo = core_repo.NewSecretKeyVersionRepository(infra.DB)
	m.SecretService = core_svc.NewSecretService(m.SecretRepo, cfg.Security.MasterKey, 168*time.Hour)
	if err := m.SecretService.Bootstrap(context.Background()); err != nil {
		return nil, fmt.Errorf("core module: bootstrap secrets: %w", err)
	}

	m.ZoneRepo = core_repo.NewZoneRepository(infra.DB)
	m.ZoneService = core_svc.NewZoneService(m.ZoneRepo)
	m.ZoneHandler = core_handler.NewZoneHandler(m.ZoneService)

	m.TenantRepo = core_repo.NewTenantRepository(infra.DB)
	m.TenantService = core_svc.NewTenantService(m.TenantRepo)
	m.TenantHandler = core_handler.NewTenantHandler(m.TenantService)

	m.WorkspaceRepo = core_repo.NewWorkspaceRepository(infra.DB)
	m.WorkspaceService = core_svc.NewWorkspaceService(m.WorkspaceRepo)
	m.WorkspaceHandler = core_handler.NewWorkspaceHandler(m.WorkspaceService)

	m.DataPlaneRepo = core_repo.NewDataPlaneRepository(infra.DB)
	m.startSecretLoop()

	if cfg.GRPC.DataPlaneClientCACertPath == "" || cfg.GRPC.DataPlaneClientCAKeyPath == "" || cfg.GRPC.DataPlaneEnrollToken == "" {
		logger.SysWarn("core.module", "dataplane registry disabled: missing gRPC dataplane CA or enroll token config")
		return m, nil
	}

	ca, err := security.LoadCertificateAuthority(cfg.GRPC.DataPlaneClientCACertPath, cfg.GRPC.DataPlaneClientCAKeyPath)
	if err != nil {
		logger.SysError("core.module", "failed to load dataplane client certificate authority: "+err.Error())
		return m, nil
	}

	m.DataPlaneService = core_svc.NewDataPlaneService(m.DataPlaneRepo, m.ZoneRepo, cfg, ca)
	m.DataPlaneGRPCServer = core_grpc.NewDataPlaneRegistryServer(m.DataPlaneService)

	core_grpc.RegisterDataPlaneRegistryServer(grpcServer.Server, m.DataPlaneGRPCServer)
	m.startStaleMarker()
	return m, nil
}

func (m *Module) Stop() {
	if m.stopStale != nil {
		m.stopStale()
	}
	if m.stopSecret != nil {
		m.stopSecret()
	}
}

func (m *Module) startStaleMarker() {
	if m == nil || m.DataPlaneService == nil || m.Cfg == nil {
		return
	}
	if m.Cfg.GRPC.DataPlaneHeartbeatStaleTimeout <= 0 {
		return
	}

	interval := m.Cfg.GRPC.DataPlaneHeartbeatInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if interval > time.Minute {
		interval = time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.stopStale = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				updated, err := m.DataPlaneService.MarkStale(context.Background(), time.Now().UTC())
				if err != nil {
					logger.SysWarn("core.module", "failed to mark stale data planes")
					continue
				}
				if updated > 0 {
					logger.SysInfo("core.module", "marked stale data planes")
				}
			}
		}
	}()
}

func (m *Module) startSecretLoop() {
	if m == nil || m.SecretService == nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.stopSecret = cancel

	go func() {
		syncTicker := time.NewTicker(10 * time.Second)
		rotateTicker := time.NewTicker(time.Minute)
		defer syncTicker.Stop()
		defer rotateTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-syncTicker.C:
				if err := m.SecretService.RefreshChangedFamilies(context.Background()); err != nil {
					logger.SysWarn("core.module", "failed to refresh secret cache")
				}
			case <-rotateTicker.C:
				if err := m.SecretService.RotateDue(context.Background()); err != nil {
					logger.SysWarn("core.module", "failed to rotate due secrets")
				}
			}
		}
	}()
}
