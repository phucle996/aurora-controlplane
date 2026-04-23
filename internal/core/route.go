package core

import (
	"controlplane/internal/config"
	"controlplane/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all admin routes for the core module.
func RegisterRoutes(r *gin.Engine, cfg *config.Config, m *Module) {

	r.GET("/admin/core/zones", m.ZoneHandler.ListZones)
	r.GET("/admin/core/zones/:id", m.ZoneHandler.GetZone)
	r.POST("/admin/core/zones", m.ZoneHandler.CreateZone)
	r.PATCH("/admin/core/zones/:id", m.ZoneHandler.UpdateZoneDescription)
	r.DELETE("/admin/core/zones/:id", m.ZoneHandler.DeleteZone)

	r.GET("/api/v1/workspaces/options",
		middleware.Access(),
		middleware.RequirePermission("workspace:read"),
		m.WorkspaceHandler.ListWorkspaceOptions,
	)

	r.GET("/api/v1/core/tenants",
		middleware.Access(),
		middleware.AdminAPIToken(),
		m.TenantHandler.ListTenants,
	)

	r.GET("/api/v1/core/tenants/:id",
		middleware.Access(),
		middleware.AdminAPIToken(),
		m.TenantHandler.GetTenant,
	)

	r.POST("/api/v1/core/tenants",
		middleware.Access(),
		middleware.AdminAPIToken(),
		m.TenantHandler.CreateTenant,
	)

	r.PATCH("/api/v1/core/tenants/:id",
		middleware.Access(),
		middleware.AdminAPIToken(),
		m.TenantHandler.UpdateTenant)
	r.DELETE("/api/v1/core/tenants/:id", m.TenantHandler.DeleteTenant)

	r.GET("/api/v1/core/workspaces", m.WorkspaceHandler.ListWorkspaces)
	r.GET("/api/v1/core/workspaces/:id", m.WorkspaceHandler.GetWorkspace)
	r.POST("/api/v1/core/workspaces", m.WorkspaceHandler.CreateWorkspace)
	r.PATCH("/api/v1/core/workspaces/:id", m.WorkspaceHandler.UpdateWorkspace)
	r.DELETE("/api/v1/core/workspaces/:id", m.WorkspaceHandler.DeleteWorkspace)
}
