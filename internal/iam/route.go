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
//	1  admin global
//	2  admin tenant
//	3 	other roles special in the tenant : manager ,... .
//
// the admin can define level user in these tenant ( 3 and lower)
//
//	4 	 authenticate user global or user level default in the tenant
//
// (when user just join tenant)
func RegisterRoutes(router *gin.Engine, cfg *config.Config, m *Module) {

	// ----------------------------
	// Global authentication routes
	// ----------------------------

	// đăng kí tài khoản global (không thuộc tenant nào)
	router.POST(
		"/api/v1/auth/register",
		middleware.RateLimit(m.RateLimiter, "auth_register", 5, 5, time.Minute),
		m.AuthHandler.Register,
	)

	// kích hoạt tài khoản global (không thuộc tenant nào)
	router.GET("/api/v1/auth/activate",
		middleware.RateLimit(m.RateLimiter, "auth_activate", 5, 5, time.Minute),
		m.AuthHandler.Activate,
	)

	// đăng nhập tài khoản global (không thuộc tenant nào)
	router.POST(
		"/api/v1/auth/login",
		middleware.RateLimit(m.RateLimiter, "auth_login", 5, 5, time.Minute),
		m.AuthHandler.Login,
	)

	// admin login bằng static admin api key, set cookie `apitoken`.
	router.POST(
		"/admin/auth/login",
		middleware.RateLimit(m.RateLimiter, "auth_admin_api_key_login", 5, 5, time.Minute),
		m.AuthHandler.AdminLogin,
	)

	// quên mật khẩu tài khoản global (không thuộc tenant nào)
	router.POST(
		"/api/v1/auth/forgot-password",
		middleware.RateLimit(m.RateLimiter, "auth_forgot_password", 3, 5, time.Minute),
		m.AuthHandler.ForgotPassword,
	)

	// reset mật khẩu tài khoản global (không thuộc tenant nào)
	router.POST(
		"/api/v1/auth/reset-password",
		middleware.RateLimit(m.RateLimiter, "auth_reset_password", 5, 5, time.Minute),
		m.AuthHandler.ResetPassword,
	)

	// Token rotation — requires device signature; rate-limited to 10 req/min per key.
	router.POST(
		"/api/v1/auth/refresh",
		middleware.RateLimit(m.RateLimiter, "auth_refresh", 10, 10, time.Minute),
		m.TokenHandler.Refresh,
	)

	// Logout — requires valid or near-expired access token to blacklist it.
	router.POST("/api/v1/auth/logout",
		middleware.Access(),
		middleware.RequireDeviceID(),
		m.AuthHandler.Logout,
	)

	router.GET("/api/v1/whoami",
		middleware.Access(),
		middleware.RequireDeviceID(),
		m.AuthHandler.WhoAmI,
	)

	// ----------------------------
	// Device management
	// ----------------------------

	// Self-service: device management (any authenticated user)
	router.GET("/api/v1/me/devices",
		middleware.Access(),
		middleware.RequireDeviceID(),
		m.DeviceHandler.ListMyDevices,
	)

	// delete device self
	router.DELETE("/api/v1/me/devices/:id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		m.DeviceHandler.RevokeDevice,
	)

	// revoke another device , keep device current
	router.DELETE("/api/v1/me/devices/others",
		middleware.Access(),
		middleware.RequireDeviceID(),
		m.DeviceHandler.RevokeOtherDevices,
	)

	// Admin: device management
	router.GET("/admin/devices/:id",
		middleware.AdminAPIToken(),
		m.DeviceHandler.AdminGetDevice,
	)

	router.DELETE("/admin/devices/:id",
		middleware.AdminAPIToken(),
		m.DeviceHandler.AdminForceRevoke,
	)

	router.GET("/admin/devices/:id/quarantine",
		middleware.AdminAPIToken(),
		m.DeviceHandler.Quarantine,
	)
	router.POST("/admin/devices/:id/suspicious",
		middleware.AdminAPIToken(),
		m.DeviceHandler.AdminMarkSuspicious,
	)

	router.POST("/admin/devices/cleanup",
		middleware.AdminAPIToken(),
		m.DeviceHandler.AdminCleanupStale,
	)

	// ── Admin: RBAC ──────────────

	router.GET("/admin/rbac/roles",
		middleware.AdminAPIToken(),
		m.RbacHandler.ListRoles,
	)

	router.POST("/admin/rbac/roles",
		middleware.AdminAPIToken(),
		m.RbacHandler.CreateRole,
	)
	router.GET("/admin/rbac/roles/:id",
		middleware.AdminAPIToken(),
		m.RbacHandler.GetRole,
	)

	router.PUT("/admin/rbac/roles/:id",
		middleware.AdminAPIToken(),
		m.RbacHandler.UpdateRole,
	)

	router.DELETE("/admin/rbac/roles/:id",
		middleware.AdminAPIToken(),
		m.RbacHandler.DeleteRole,
	)

	router.GET("/admin/rbac/permissions",
		middleware.AdminAPIToken(),
		m.RbacHandler.ListPermissions,
	)

	router.POST("/admin/rbac/permissions",
		middleware.AdminAPIToken(),
		m.RbacHandler.CreatePermission,
	)

	router.POST("/admin/rbac/roles/:id/permissions",
		middleware.AdminAPIToken(),
		m.RbacHandler.AssignPermission,
	)

	router.DELETE("/admin/rbac/roles/:id/permissions/:perm_id",
		middleware.AdminAPIToken(),
		m.RbacHandler.RevokePermission,
	)

	router.POST("/admin/rbac/cache/invalidate",
		middleware.AdminAPIToken(),
		m.RbacHandler.InvalidateAll,
	)

	// MFA: public challenge flows (no token — in-flight login)
	router.POST("/api/v1/auth/mfa/verify",
		middleware.RateLimit(m.RateLimiter, "mfa_verify", 5, 1, time.Minute),
		m.MfaHandler.Verify,
	)

	// MFA: self-service management (any authenticated user)
	router.GET("/api/v1/me/mfa",
		middleware.RateLimit(m.RateLimiter, "mfa_list", 3, 1, time.Minute),
		middleware.Access(),
		middleware.RequireDeviceID(),
		m.MfaHandler.ListMethods,
	)

	// otp enroll
	router.POST("/api/v1/me/mfa/totp/enroll",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RateLimit(m.RateLimiter, "mfa_enroll", 10, 1, time.Minute),
		m.MfaHandler.EnrollTOTP,
	)

	// otp confirm
	router.POST("/api/v1/me/mfa/totp/confirm",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RateLimit(m.RateLimiter, "mfa_confirm", 10, 1, time.Minute),
		m.MfaHandler.ConfirmTOTP,
	)

	// otp enable
	router.PATCH("/api/v1/me/mfa/:setting_id/enable",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RateLimit(m.RateLimiter, "mfa_enable", 10, 1, time.Minute),
		m.MfaHandler.EnableMethod,
	)

	// otp disable
	router.PATCH("/api/v1/me/mfa/:setting_id/disable",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RateLimit(m.RateLimiter, "mfa_disable", 10, 1, time.Minute),
		m.MfaHandler.DisableMethod,
	)

	// otp delete
	router.DELETE("/api/v1/me/mfa/:setting_id",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RateLimit(m.RateLimiter, "mfa_delete", 10, 1, time.Minute),
		m.MfaHandler.DeleteMethod,
	)

	// otp generate recovery codes
	router.POST("/api/v1/me/mfa/recovery-codes",
		middleware.Access(),
		middleware.RequireDeviceID(),
		middleware.RateLimit(m.RateLimiter, "mfa_recovery_codes", 10, 1, time.Minute),
		m.MfaHandler.GenerateRecoveryCodes,
	)

}
