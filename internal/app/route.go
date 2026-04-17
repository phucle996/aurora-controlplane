package app

import (
	"controlplane/internal/app/bootstrap"
	"controlplane/internal/config"
	"controlplane/internal/http/handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes is the top-level HTTP route composition root.
// Composes root route tree only — no business logic, no handler implementation.
func RegisterRoutes(r *gin.Engine, cfg *config.Config, rt *bootstrap.Runtime, health *handler.HealthHandler) {
	api := r.Group("/api")
	{
		// Health endpoints
		api.GET("/health/liveness", health.Liveness)
		api.GET("/health/readiness", health.Readiness)
		api.GET("/health/startup", health.Startup)

		// Versioned API group
		// v1 := api.Group("/v1")
		// {
		// 	iam.RegisterRoutes(v1.Group("/iam"), &m.IamModule)
		// }
	}

	// Register Frontend SPA fallback and static files (ignoring the /api prefix)
	if err := RegisterFrontend(r); err != nil {
		// Logged internally, frontend disabled
	}
}
