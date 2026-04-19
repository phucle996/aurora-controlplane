package iam

import (
	"time"

	"controlplane/internal/config"
	"controlplane/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers HTTP routes for the IAM module.
//
// Level matrix (lower number = higher privilege):
//
//	0   super-admin
//	10  admin
//	50  operator
//	100 regular user (any authenticated)
func (m *Module) RegisterRoutes(router *gin.RouterGroup, cfg *config.Config) {
	access := middleware.Access(cfg.Security.AccessSecretKey, m.TokenService)

	auth := router.Group("/auth")
	m.registerAuthRoutes(auth, access)

	// ── Self-service: device management (any authenticated user) ──────────────
	meDevices := router.Group("/me/devices",
		access,
	)
	{
		meDevices.GET("", m.DeviceHandler.ListMyDevices)
		meDevices.DELETE("/:id", m.DeviceHandler.RevokeDevice)
		meDevices.DELETE("", m.DeviceHandler.RevokeOtherDevices)
	}

	// ── Admin: device management ──────────────
	adminDevices := router.Group("/admin/devices",
		access,
		middleware.RequireLevel(50),
		middleware.RequirePermission(m.RbacService, "admin:device:read"),
	)
	{
		adminDevices.GET("/:id", m.DeviceHandler.AdminGetDevice)
		adminDevices.DELETE("/:id",
			middleware.RequirePermission(m.RbacService, "admin:device:revoke"),
			m.DeviceHandler.AdminForceRevoke,
		)
		adminDevices.POST("/:id/suspicious",
			middleware.RequirePermission(m.RbacService, "admin:device:quarantine"),
			m.DeviceHandler.AdminMarkSuspicious,
		)
		adminDevices.POST("/cleanup",
			middleware.RequirePermission(m.RbacService, "admin:device:cleanup"),
			m.DeviceHandler.AdminCleanupStale,
		)
	}

	// ── Admin: RBAC ──────────────
	if m.RbacHandler != nil {
		rbacAdmin := router.Group("/admin/rbac",
			access,
			middleware.RequireLevel(0), // Super admin
		)
		{
			rbacAdmin.GET("/roles", middleware.RequirePermission(m.RbacService, "rbac:role:read"), m.RbacHandler.ListRoles)
			rbacAdmin.POST("/roles", middleware.RequirePermission(m.RbacService, "rbac:role:create"), m.RbacHandler.CreateRole)
			rbacAdmin.GET("/roles/:id", middleware.RequirePermission(m.RbacService, "rbac:role:read"), m.RbacHandler.GetRole)
			rbacAdmin.PUT("/roles/:id", middleware.RequirePermission(m.RbacService, "rbac:role:update"), m.RbacHandler.UpdateRole)
			rbacAdmin.DELETE("/roles/:id", middleware.RequirePermission(m.RbacService, "rbac:role:delete"), m.RbacHandler.DeleteRole)

			rbacAdmin.GET("/permissions", middleware.RequirePermission(m.RbacService, "rbac:permission:read"), m.RbacHandler.ListPermissions)
			rbacAdmin.POST("/permissions", middleware.RequirePermission(m.RbacService, "rbac:permission:create"), m.RbacHandler.CreatePermission)

			rbacAdmin.POST("/roles/:id/permissions", middleware.RequirePermission(m.RbacService, "rbac:permission:assign"), m.RbacHandler.AssignPermission)
			rbacAdmin.DELETE("/roles/:id/permissions/:perm_id", middleware.RequirePermission(m.RbacService, "rbac:permission:revoke"), m.RbacHandler.RevokePermission)

			rbacAdmin.POST("/cache/invalidate", middleware.RequirePermission(m.RbacService, "rbac:cache:flush"), m.RbacHandler.InvalidateAll)
		}
	}

	// ── MFA: public challenge flows (no token — in-flight login) ─────────────
	if m.MfaHandler != nil {
		mfaPublic := router.Group("/auth/mfa")
		{
			mfaPublic.POST("/verify",
				middleware.RateLimit(m.RateLimiter, "mfa_verify", 5, 3, time.Minute),
				m.MfaHandler.Verify,
			)

		}

		// ── MFA: self-service management (any authenticated user) ─────────────
		mfaMe := router.Group("/me/mfa",
			access,
		)
		{
			mfaMe.GET("", m.MfaHandler.ListMethods)
			mfaMe.POST("/totp/enroll",
				middleware.RateLimit(m.RateLimiter, "mfa_enroll", 3, 5, time.Minute),
				m.MfaHandler.EnrollTOTP,
			)
			mfaMe.POST("/totp/confirm",
				middleware.RateLimit(m.RateLimiter, "mfa_confirm", 5, 5, time.Minute),
				m.MfaHandler.ConfirmTOTP,
			)
			mfaMe.PATCH("/:setting_id/enable", m.MfaHandler.EnableMethod)
			mfaMe.PATCH("/:setting_id/disable", m.MfaHandler.DisableMethod)
			mfaMe.DELETE("/:setting_id", m.MfaHandler.DeleteMethod)
			mfaMe.POST("/recovery-codes", m.MfaHandler.GenerateRecoveryCodes)
		}
	}
}

// ── Auth routes (public & protected) ──────────────────────────────────────────
func (m *Module) registerAuthRoutes(router *gin.RouterGroup, access gin.HandlerFunc) {

	router.POST(
		"/register",
		middleware.RateLimit(m.RateLimiter, "auth_register", 5, 5, time.Minute),
		m.AuthHandler.Register,
	)
	router.GET("/activate",
		middleware.RateLimit(m.RateLimiter, "auth_activate", 5, 5, time.Minute),
		m.AuthHandler.Activate)

	router.POST(
		"/login",
		middleware.RateLimit(m.RateLimiter, "auth_login", 5, 5, time.Minute),
		m.AuthHandler.Login,
	)
	router.POST(
		"/forgot-password",
		middleware.RateLimit(m.RateLimiter, "auth_forgot_password", 3, 5, time.Minute),
		m.AuthHandler.ForgotPassword,
	)
	router.POST(
		"/reset-password",
		middleware.RateLimit(m.RateLimiter, "auth_reset_password", 5, 5, time.Minute),
		m.AuthHandler.ResetPassword,
	)

	// Token rotation — requires device signature; rate-limited to 10 req/min per key.
	router.POST(
		"/refresh",
		middleware.RateLimit(m.RateLimiter, "auth_refresh", 10, 10, time.Minute),
		m.TokenHandler.Refresh,
	)

	// Logout — requires valid or near-expired access token to blacklist it.
	router.POST("/logout", access, m.AuthHandler.Logout)
}
