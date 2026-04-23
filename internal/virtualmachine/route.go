package virtualmachine

import (
	"controlplane/internal/config"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes is a package-level function to register all routes for the virtualmachine module.
func RegisterRoutes(r *gin.Engine, cfg *config.Config, m *Module) {
	if r == nil || m == nil {
		return
	}

	api := r.Group("/api")
	v1 := api.Group("/v1")

	m.RegisterAPIRoutes(v1)
}

// RegisterRoutes is kept for symmetry with other modules.
func (m *Module) RegisterAPIRoutes(router *gin.RouterGroup, middleware ...gin.HandlerFunc) {
	if router == nil {
		return
	}
	hosts := router.Group("/virtual-machine/hosts", middleware...)
	{
		hosts.GET("", m.HostHandler.ListHosts)
		hosts.GET("/options", m.HostHandler.ListHostOptions)
		hosts.GET("/:host_id", m.HostHandler.GetHost)
	}
}
