package iam

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"controlplane/internal/app/bootstrap"
	"controlplane/internal/config"
	"controlplane/internal/http/middleware"
	iam_repo "controlplane/internal/iam/repository"
	iam_svc "controlplane/internal/iam/service"
	iam_handler "controlplane/internal/iam/transport/http/handler"
	"controlplane/internal/ratelimit"
	"controlplane/internal/security"

	"github.com/redis/go-redis/v9"
)

const (
	adminBootstrapTokenPath = "/var/lib/aurora-controlplane/admin-api-token"
	adminBootstrapTokenName = "aurora-controlplane-admin-api-token"
)

// Module encapsulates all IAM dependencies.
type Module struct {
	Cfg         *config.Config
	Infra       *bootstrap.Infra
	RateLimiter *ratelimit.Bucket
	Registry    *middleware.RoleRegistry
	Secrets     security.SecretProvider

	// Repos
	UserRepo          *iam_repo.UserRepository
	DeviceRepo        *iam_repo.DeviceRepository
	TokenRepo         *iam_repo.TokenRepository
	MfaRepo           *iam_repo.MfaRepository
	RbacRepo          *iam_repo.RbacRepository
	AdminAPITokenRepo *iam_repo.AdminAPITokenRepository

	// Services
	DeviceService        *iam_svc.DeviceService
	TokenService         *iam_svc.TokenService
	MfaService           *iam_svc.MfaService
	AuthService          *iam_svc.AuthService
	RbacService          *iam_svc.RbacService
	AdminAPITokenService *iam_svc.AdminAPITokenService

	// Handlers
	AuthHandler   *iam_handler.AuthHandler
	DeviceHandler *iam_handler.DeviceHandler
	TokenHandler  *iam_handler.TokenHandler
	MfaHandler    *iam_handler.MfaHandler
	RbacHandler   *iam_handler.RbacHandler

	stopCleanup context.CancelFunc
	cleanupDone chan struct{}
	rbacSync    *iam_svc.RbacCacheSync
	stopOnce    sync.Once
}

// NewModule wires all IAM dependencies and starts background jobs.
func NewModule(
	cfg *config.Config,
	infra *bootstrap.Infra,
	secrets security.SecretProvider,
) (*Module, error) {
	registry := middleware.NewRoleRegistry()

	m := &Module{
		Cfg:      cfg,
		Infra:    infra,
		Registry: registry,
		Secrets:  secrets,
	}

	if cfg == nil || infra == nil || secrets == nil {
		return nil, fmt.Errorf("iam module: invalid arguments")
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
	m.AdminAPITokenRepo = iam_repo.NewAdminAPITokenRepository(infra.DB)

	// ── Services ──────────────────────────────────────────────────────────────
	m.DeviceService = iam_svc.NewDeviceService(m.DeviceRepo)
	m.TokenService = iam_svc.NewTokenService(m.TokenRepo, m.DeviceRepo, m.UserRepo, rdb, cfg, m.Secrets)
	m.MfaService = iam_svc.NewMfaService(m.MfaRepo, m.UserRepo, rdb, cfg)
	m.RbacService = iam_svc.NewRbacService(m.RbacRepo, registry, iam_svc.NewRedisRbacCacheBus(rdb))
	m.AdminAPITokenService = iam_svc.NewAdminAPITokenService(m.AdminAPITokenRepo, m.Secrets)
	if err := m.ensureAdminBootstrapToken(context.Background()); err != nil {
		return nil, fmt.Errorf("iam module: ensure admin bootstrap token: %w", err)
	}
	m.AuthService = iam_svc.NewAuthService(m.UserRepo, m.DeviceService,
		m.TokenService, m.MfaService, m.AdminAPITokenService, rdb, cfg, m.Secrets)

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

	if rdb != nil {
		m.rbacSync = iam_svc.NewRbacCacheSync(rdb, registry)
		m.rbacSync.Start(context.Background())
	}

	m.startCleanupWorker(context.Background(), registry)

	return m, nil
}

func (m *Module) ensureAdminBootstrapToken(ctx context.Context) error {
	if m == nil || m.AdminAPITokenService == nil {
		return nil
	}

	token, created, err := m.AdminAPITokenService.EnsureBootstrapToken(ctx)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}

	if err := writeAdminBootstrapTokenFile(adminBootstrapTokenPath, token); err == nil {
		return nil
	}

	fallbackPath := filepath.Join(os.TempDir(), adminBootstrapTokenName)
	if err := writeAdminBootstrapTokenFile(fallbackPath, token); err != nil {
		return err
	}

	return nil
}

func writeAdminBootstrapTokenFile(path string, token string) error {
	if path == "" {
		return fmt.Errorf("iam module: admin bootstrap token path is empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(token+"\n"), 0o600)
}

func (m *Module) startCleanupWorker(parent context.Context, registry *middleware.RoleRegistry) {
	if m == nil || m.TokenService == nil {
		return
	}
	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)
	m.stopCleanup = cancel
	m.cleanupDone = make(chan struct{})

	go func() {
		defer close(m.cleanupDone)
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if registry != nil {
					registry.EvictExpired()
				}
				deleted, err := m.TokenService.CleanupExpired(ctx)
				if err != nil {
					slog.Warn("iam: token cleanup failed", "err", err)
					continue
				}
				if deleted > 0 {
					slog.Info("iam: token cleanup completed", "deleted", deleted)
				}
			}
		}
	}()
}

// Stop shuts down module-level workers and is safe to call more than once.
func (m *Module) Stop() {
	if m == nil {
		return
	}

	m.stopOnce.Do(func() {
		if m.stopCleanup != nil {
			m.stopCleanup()
			if m.cleanupDone != nil {
				<-m.cleanupDone
			}
		}
		if m.rbacSync != nil {
			m.rbacSync.Stop()
		}
	})
}
