package iam

import (
	"context"
	"log/slog"
	"time"

	"controlplane/internal/app/bootstrap"
	"controlplane/internal/config"
	"controlplane/internal/http/middleware"
	iam_repo "controlplane/internal/iam/repository"
	iam_svc "controlplane/internal/iam/service"
	iam_handler "controlplane/internal/iam/transport/http/handler"
	"controlplane/internal/ratelimit"

	"github.com/redis/go-redis/v9"
)

// Module encapsulates all IAM dependencies.
type Module struct {
	Cfg         *config.Config
	Infra       *bootstrap.Infra
	Runtime     *bootstrap.Runtime
	RateLimiter *ratelimit.Bucket
	Registry    *middleware.RoleRegistry

	// Repos
	UserRepo   *iam_repo.UserRepository
	DeviceRepo *iam_repo.DeviceRepository
	TokenRepo  *iam_repo.TokenRepository
	MfaRepo    *iam_repo.MfaRepository
	RbacRepo   *iam_repo.RbacRepository

	// Services
	DeviceService *iam_svc.DeviceService
	TokenService  *iam_svc.TokenService
	MfaService    *iam_svc.MfaService
	AuthService   *iam_svc.AuthService
	RbacService   *iam_svc.RbacService

	// Handlers
	AuthHandler   *iam_handler.AuthHandler
	DeviceHandler *iam_handler.DeviceHandler
	TokenHandler  *iam_handler.TokenHandler
	MfaHandler    *iam_handler.MfaHandler
	RbacHandler   *iam_handler.RbacHandler

	stopEvict context.CancelFunc
}

// NewModule wires all IAM dependencies and starts background jobs.
func NewModule(cfg *config.Config, infra *bootstrap.Infra, rt *bootstrap.Runtime, registry *middleware.RoleRegistry) *Module {
	if registry == nil {
		registry = middleware.NewRoleRegistry()
	}

	m := &Module{
		Cfg:      cfg,
		Infra:    infra,
		Runtime:  rt,
		Registry: registry,
	}

	if infra == nil {
		return m
	}

	var rdb *redis.Client
	if infra.Redis != nil {
		rdb = infra.Redis.Unwrap()
		m.RateLimiter = ratelimit.NewBucket(rdb)
	}

	// ── Repositories ──────────────────────────────────────────────────────────
	m.UserRepo = iam_repo.NewUserRepository(infra.DB)
	m.DeviceRepo = iam_repo.NewDeviceRepository(infra.DB)
	m.TokenRepo = iam_repo.NewTokenRepository(infra.DB)
	m.MfaRepo = iam_repo.NewMfaRepository(infra.DB)
	m.RbacRepo = iam_repo.NewRbacRepository(infra.DB)

	// ── Services ──────────────────────────────────────────────────────────────
	m.DeviceService = iam_svc.NewDeviceService(m.DeviceRepo)
	m.TokenService = iam_svc.NewTokenService(m.TokenRepo, m.DeviceRepo, m.UserRepo, rdb, cfg)
	m.MfaService = iam_svc.NewMfaService(m.MfaRepo, m.UserRepo, rdb, cfg)
	m.RbacService = iam_svc.NewRbacService(m.RbacRepo, registry)
	m.AuthService = iam_svc.NewAuthService(m.UserRepo, m.DeviceService, m.TokenService, m.MfaService, rdb, cfg)

	// ── Handlers ──────────────────────────────────────────────────────────────
	m.AuthHandler = iam_handler.NewAuthHandler(m.AuthService)
	m.DeviceHandler = iam_handler.NewDeviceHandler(m.DeviceService)
	m.TokenHandler = iam_handler.NewTokenHandler(m.TokenService)
	m.MfaHandler = iam_handler.NewMfaHandler(m.MfaService, m.TokenService)
	m.RbacHandler = iam_handler.NewRbacHandler(m.RbacService)

	// ── RBAC warm-up (best-effort, non-blocking) ──────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.RbacService.WarmUp(ctx); err != nil {
		slog.Warn("iam: rbac warm-up failed", "err", err)
	}

	// ── Background TTL eviction every 5 minutes ───────────────────────────────
	evictCtx, evictCancel := context.WithCancel(context.Background())
	m.stopEvict = evictCancel
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-evictCtx.Done():
				return
			case <-ticker.C:
				registry.EvictExpired()
			}
		}
	}()

	return m
}

// Stop shuts down background goroutines (call during graceful shutdown).
func (m *Module) Stop() {
	if m.stopEvict != nil {
		m.stopEvict()
	}
}
