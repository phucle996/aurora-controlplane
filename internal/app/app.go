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
	runtime    *bootstrap.Runtime
	health     *handler.HealthHandler
	httpServer *http.Server
	grpc       *bootstrap.GRPC
}

func NewApplication(cfg *config.Config) (*App, error) {
	logger.SysInfo("app", "Initializing application...")

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Init infra
	infra, err := bootstrap.InitInfra(ctx, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	// Run migrations
	if err := bootstrap.RunMigrations(ctx, cfg); err != nil {
		cancel()
		return nil, err
	}

	// Run seed
	if err := bootstrap.RunSeed(ctx, cfg); err != nil {
		cancel()
		return nil, err
	}

	// Init runtime
	rt, err := bootstrap.InitRuntime(ctx, cfg, infra)
	if err != nil {
		cancel()
		return nil, err
	}

	// Build modules
	if err := globalModules(ctx, cfg, infra, rt); err != nil {
		cancel()
		return nil, err
	}

	// Init HealthHandler
	health := handler.NewHealthHandler(infra.DB, infra.Redis.Unwrap())

	// Init Gin engine and register routes
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery(), middleware.AccessLog(), middleware.RequestID())

	RegisterRoutes(engine, cfg, rt, health)

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.HTTPPort),
		Handler: engine,
	}

	// Init gRPC (server + client manager)
	g, err := bootstrap.InitGRPC(ctx, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	return &App{
		ctx:        ctx,
		cancel:     cancel,
		cfg:        cfg,
		infra:      infra,
		runtime:    rt,
		health:     health,
		httpServer: httpSrv,
		grpc:       g,
	}, nil
}

func (a *App) Start(cfg *config.Config) error {
	logger.SysInfo("app", "Starting application components...")

	// Start runtime
	if err := a.runtime.Start(); err != nil {
		return err
	}

	// Start gRPC server
	go func() {
		if err := a.grpc.Start(); err != nil {
			logger.SysError("app", fmt.Sprintf("gRPC server stopped: %v", err))
		}
	}()

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
	logger.SysInfo("app", "Stopping application gracefully...")

	// 1. Mark as not ready to drain incoming traffic from load balancers
	a.health.MarkNotReady()
	logger.SysInfo("app", "Application marked as not ready (draining traffic)")

	// Optional: add a small sleep here if deployed behind a cloud load balancer (e.g. AWS ALB)
	// to allow time for the unregistered target state to propagate.

	// Stop HTTP server
	if err := a.httpServer.Shutdown(context.Background()); err != nil {
		logger.SysError("app", fmt.Sprintf("HTTP server shutdown error: %v", err))
	}

	// Stop gRPC (server + close all client connections)
	a.grpc.Stop()

	// Stop runtime
	a.runtime.Stop()

	// Cancel root context
	a.cancel()

	// Close infra connections
	bootstrap.CloseInfra()
}
