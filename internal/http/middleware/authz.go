package middleware

import (
	"context"
	"time"

	"controlplane/internal/http/response"

	"github.com/gin-gonic/gin"
)

// RoleResolver is implemented by the RBAC service.
// It performs cache-aside: check in-memory, miss → load DB → re-cache.
// Defined here to avoid importing the rbac package from middleware.
type RoleResolver interface {
	GetRoleEntry(ctx context.Context, role string) (RoleEntry, error)
}

// RequireLevel returns a middleware that compares the authenticated user's
// security level (from the JWT claim, injected by Access middleware) against
// the required minimum.
//
// Level semantics (same as User.SecurityLevel):
//
//	0   = highest privilege  (super-admin)
//	N   = lower privilege the higher the number  (e.g. user = 100)
//
// RequireLevel(50) passes users with level 0..50 and rejects 51+.
//
// IMPORTANT: RequireLevel only reads from gin context — it does NOT call
// RbacService or the database. It must come before RequirePermission in the
// middleware chain so that low-level users cannot reach higher-privilege
// permission checks.
//
// Usage:
//
//	router.DELETE("/admin/users/:id",
//	    middleware.Access(secret),          // inject level from JWT
//	    middleware.RequireLevel(10),        // gate on raw level
//	    middleware.RequirePermission(svc, "iam:users:delete"), // perm check
//	    handler)
func RequireLevel(minLevel int) gin.HandlerFunc {
	return func(c *gin.Context) {
		level, exists := c.Get(CtxKeyLevel)
		if !exists {
			// Access middleware did not run — misconfigured route.
			response.RespondForbidden(c, "missing level claim")
			c.Abort()
			return
		}

		userLevel, ok := level.(int)
		if !ok {
			response.RespondForbidden(c, "invalid level claim")
			c.Abort()
			return
		}

		if userLevel > minLevel {
			response.RespondForbidden(c, "insufficient privilege level")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission returns a middleware that verifies the authenticated user's
// role has the given permission string via the RoleResolver (cache-aside).
//
// On cache-miss the resolver falls back to Postgres and re-caches — the
// request is NOT rejected due to a cold cache.
//
// Usage:
//
//	middleware.RequirePermission(rbacSvc, "iam:users:delete")
func RequirePermission(resolver RoleResolver, perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString(CtxKeyRole)
		if role == "" {
			response.RespondForbidden(c, "missing role claim")
			c.Abort()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		entry, err := resolver.GetRoleEntry(ctx, role)
		if err != nil {
			response.RespondForbidden(c, "role resolution failed")
			c.Abort()
			return
		}

		if !hasPermission(entry.Permissions, perm) {
			response.RespondForbidden(c, "insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func hasPermission(permissions []string, perm string) bool {
	for _, p := range permissions {
		if p == perm || p == "*" {
			return true
		}
	}
	return false
}
