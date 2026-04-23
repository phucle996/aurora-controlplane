package smtp

import (
	"controlplane/internal/config"
	"controlplane/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes is a package-level function to register all routes for the smtp module.
func RegisterRoutes(router *gin.Engine, cfg *config.Config, m *Module) {

	// -----------------------------------------
	// smtp aggregation
	// -----------------------------------------

	// router smtp aggregation
	router.GET("/api/v1/smtp/aggregation",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:aggregate"),
		m.AggregationHandler.GetWorkspaceAggregation,
	)

	// -----------------------------------------
	// smtp consumers
	// -----------------------------------------

	// get all smtp consumers
	router.GET("/api/v1/smtp/consumers",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:consumer:read"),
		m.ConsumerHandler.ListConsumers,
	)

	// get 1 smtp consumer detail
	router.GET("/api/v1/smtp/consumers/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:consumer:read"),
		m.ConsumerHandler.GetConsumer,
	)

	// test connection for smtp consumer
	router.POST("/api/v1/smtp/consumers/try-connect",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:consumer:write"),
		m.ConsumerHandler.TryConnect,
	)

	// create new smtp consumer
	router.POST("/api/v1/smtp/consumers",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:consumer:write"),
		m.ConsumerHandler.CreateConsumer,
	)

	// update smtp consumer
	router.PUT("/api/v1/smtp/consumers/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:consumer:write"),
		m.ConsumerHandler.UpdateConsumer,
	)

	// delete smtp consumer
	router.DELETE("/api/v1/smtp/consumers/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:consumer:write"),
		m.ConsumerHandler.DeleteConsumer,
	)

	// get consumer options for template
	router.GET("/api/v1/smtp/consumers/options",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:consumer:read"),
		m.ConsumerHandler.ListConsumerOptions,
	)

	// -----------------------------------------
	// smtp templates
	// -----------------------------------------

	// get all smtp templates
	router.GET("/api/v1/smtp/templates",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:template:read"),
		m.TemplateHandler.ListTemplates,
	)

	// get 1 smtp template detail
	router.GET("/api/v1/smtp/templates/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:template:read"),
		m.TemplateHandler.GetTemplate,
	)

	// create new smtp template
	router.POST("/api/v1/smtp/templates",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:template:write"),
		m.TemplateHandler.CreateTemplate,
	)

	// update smtp template
	router.PUT("/api/v1/smtp/templates/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:template:write"),
		m.TemplateHandler.UpdateTemplate,
	)

	// delete smtp template
	router.DELETE("/api/v1/smtp/templates/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:template:write"),
		m.TemplateHandler.DeleteTemplate,
	)

	// -----------------------------------------
	// smtp gateways
	// -----------------------------------------

	// get all smtp gateways
	router.GET("/api/v1/smtp/gateways",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:read"),
		m.GatewayHandler.ListGateways,
	)

	// get 1 smtp gateway detail
	router.GET("/api/v1/smtp/gateways/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:read"),
		m.GatewayHandler.GetGateway,
	)

	// get 1 smtp gateway detail
	router.GET("/api/v1/smtp/gateways/:id/detail",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:read"),
		m.GatewayHandler.GetGatewayDetail,
	)

	// update smtp gateway templates
	router.PUT("/api/v1/smtp/gateways/:id/templates",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:write"),
		m.GatewayHandler.UpdateGatewayTemplates,
	)

	// update smtp gateway endpoints
	router.PUT("/api/v1/smtp/gateways/:id/endpoints",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:write"),
		m.GatewayHandler.UpdateGatewayEndpoints,
	)

	// start smtp gateway
	router.POST("/api/v1/smtp/gateways/:id/start",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:write"),
		m.GatewayHandler.StartGateway,
	)

	// drain smtp gateway
	router.POST("/api/v1/smtp/gateways/:id/drain",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:write"),
		m.GatewayHandler.DrainGateway,
	)

	// disable smtp gateway
	router.POST("/api/v1/smtp/gateways/:id/disable",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:write"),
		m.GatewayHandler.DisableGateway,
	)

	// create new smtp gateway
	router.POST("/api/v1/smtp/gateways",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:write"),
		m.GatewayHandler.CreateGateway,
	)

	// update smtp gateway
	router.PUT("/api/v1/smtp/gateways/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:write"),
		m.GatewayHandler.UpdateGateway,
	)

	// delete smtp gateway
	router.DELETE("/api/v1/smtp/gateways/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:gateway:write"),
		m.GatewayHandler.DeleteGateway,
	)

	// -----------------------------------------
	// smtp endpoints
	// -----------------------------------------

	// get all smtp endpoints
	router.GET("/api/v1/smtp/endpoints",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:endpoint:read"),
		m.EndpointHandler.ListEndpoints,
	)

	// get 1 smtp endpoint detail
	router.GET("/api/v1/smtp/endpoints/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:endpoint:read"),
		m.EndpointHandler.GetEndpoint,
	)

	// test connection for smtp endpoint
	router.POST("/api/v1/smtp/endpoints/try-connect",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:endpoint:write"),
		m.EndpointHandler.TryConnect,
	)

	// create new smtp endpoint
	router.POST("/api/v1/smtp/endpoints",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:endpoint:write"),
		m.EndpointHandler.CreateEndpoint,
	)

	// update smtp endpoint
	router.PUT("/api/v1/smtp/endpoints/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:endpoint:write"),
		m.EndpointHandler.UpdateEndpoint,
	)

	// delete smtp endpoint
	router.DELETE("/api/v1/smtp/endpoints/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:endpoint:write"),
		m.EndpointHandler.DeleteEndpoint,
	)

	// -----------------------------------------
	// smtp runtime
	// -----------------------------------------

	// list runtime activity logs
	router.GET("/api/v1/smtp/runtime/activity-logs",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:runtime:read"),
		m.RuntimeHandler.ListActivityLogs,
	)

	// list runtime delivery attempts
	router.GET("/api/v1/smtp/runtime/delivery-attempts",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:runtime:read"),
		m.RuntimeHandler.ListDeliveryAttempts,
	)

	// list runtime heartbeats
	router.GET("/api/v1/smtp/runtime/heartbeats",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:runtime:read"),
		m.RuntimeHandler.ListRuntimeHeartbeats,
	)

	// list runtime gateway assignments
	router.GET("/api/v1/smtp/runtime/gateway-assignments",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:runtime:read"),
		m.RuntimeHandler.ListGatewayAssignments,
	)

	// list runtime consumer assignments
	router.GET("/api/v1/smtp/runtime/consumer-assignments",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:runtime:read"),
		m.RuntimeHandler.ListConsumerAssignments,
	)

	// reconcile runtime
	router.POST("/api/v1/smtp/runtime/reconcile",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RequirePermission("smtp:runtime:write"),
		m.RuntimeHandler.Reconcile,
	)
}
