package app

import (
	"controlplane/internal/config"
	"controlplane/internal/core"
	"controlplane/internal/http/handler"
	"controlplane/internal/iam"
	"controlplane/internal/smtp"
	"controlplane/internal/virtualmachine"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes is the top-level HTTP route composition root.
// Composes root route tree only — no business logic, no handler implementation.
func RegisterRoutes(r *gin.Engine, cfg *config.Config,
	health *handler.HealthHandler, m *GlobalModules) {
	// Health endpoints
	r.GET("/api/health/liveness", health.Liveness)
	r.GET("/api/health/readiness", health.Readiness)
	r.GET("/api/health/startup", health.Startup)

	if m.Core != nil {
		core.RegisterRoutes(r, cfg, m.Core)
	}

	if m.IAM != nil {
		iam.RegisterRoutes(r, cfg, m.IAM)
	}

	if m.VirtualMachine != nil {
		virtualmachine.RegisterRoutes(r, cfg, m.VirtualMachine)
	}

	if m.SMTP != nil {
		smtp.RegisterRoutes(r, cfg, m.SMTP)
	}

	// Register Frontend SPA fallback and static files (ignoring the /api prefix)
	if err := RegisterFrontend(r); err != nil {
		// Logged internally, frontend disabled
	}
}
