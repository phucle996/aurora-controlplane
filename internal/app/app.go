package app

import (
	"context"
	"controlplane/internal/app/bootstrap"
	"controlplane/internal/config"
	"controlplane/internal/http/handler"
	"controlplane/internal/http/middleware"
	"controlplane/pkg/logger"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type App struct {
	ctx        context.Context
	cancel     context.CancelFunc
	cfg        *config.Config
	infra      *bootstrap.Infra
	health     *handler.HealthHandler
	httpServer *http.Server
	grpc       *bootstrap.GRPC
	modules    *GlobalModules
}

func NewApplication(cfg *config.Config) (*App, error) {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Init infra
	infra, err := bootstrap.InitInfra(ctx, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	// Run migrations
	if err := bootstrap.RunMigrations(ctx, infra.DB); err != nil {
		cancel()
		return nil, err
	}

	// Run seed
	if err := bootstrap.RunSeed(ctx, cfg); err != nil {
		cancel()
		return nil, err
	}

	// Init HealthHandler
	health := handler.NewHealthHandler(infra.DB, infra.Redis.Unwrap())

	// Init gRPC (server + client manager)
	g, err := bootstrap.InitGRPC(ctx, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	// Init Gin engine and register routes
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(
		gin.Recovery(),
		middleware.AccessLog(),
		middleware.RequestID(),
	)

	// Build modules
	m, err := globalModules(cfg, infra, g)
	if err != nil {
		cancel()
		return nil, err
	}

	// Initialize middlewares that require dependencies.
	middleware.Init(m.Core.SecretService, infra.Redis.Unwrap())
	middleware.InitAuthz(m.IAM.Registry, m.IAM.RbacService.LoadRole)
	middleware.InitAdminToken(m.IAM.AdminAPITokenService.Validate)

	RegisterRoutes(engine, cfg, health, m)

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.HTTPPort),
		Handler: engine,
	}

	return &App{
		ctx:        ctx,
		cancel:     cancel,
		cfg:        cfg,
		infra:      infra,
		health:     health,
		httpServer: httpSrv,
		grpc:       g,
		modules:    m,
	}, nil
}

func (a *App) Start(cfg *config.Config) error {
	// Start gRPC server
	go func() {
		if err := a.grpc.Start(); err != nil {
			logger.SysError("app", fmt.Sprintf("gRPC server stopped: %v", err))
		}
	}()

	if a.modules != nil && a.modules.SMTP != nil {
		a.modules.SMTP.Start()
	}

	// Start HTTP server
	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.SysError("app", fmt.Sprintf("HTTP server stopped: %v", err))
		}
	}()

	// Mark application as ready to serve traffic
	a.health.MarkReady()
	logger.SysInfo("app", "Application is ready to receive traffic")

	return nil
}

func (a *App) Stop() {
	// 1. Mark as not ready to drain incoming traffic from load balancers
	a.health.MarkNotReady()

	// Optional: add a small sleep here if deployed behind a cloud load balancer (e.g. AWS ALB)
	// to allow time for the unregistered target state to propagate.

	// Stop HTTP server
	if err := a.httpServer.Shutdown(context.Background()); err != nil {
		logger.SysError("app", fmt.Sprintf("HTTP server shutdown error: %v", err))
	}

	// Stop gRPC (server + close all client connections)
	a.grpc.Stop()

	if a.modules != nil && a.modules.IAM != nil {
		a.modules.IAM.Stop()
	}

	// Cancel root context
	a.cancel()

	// Close infra connections
	bootstrap.CloseInfra()
}
