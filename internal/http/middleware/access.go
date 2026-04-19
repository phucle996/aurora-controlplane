package middleware

import (
	"context"
	"errors"

	"controlplane/internal/http/response"
	"controlplane/internal/security"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Context keys injected by the Access middleware.
// Use these constants in handlers instead of string literals.
const (
	jwtClaimsContextKey = "jwt_claims" // full security.Claims object

	CtxKeyUserID = "user_id" // string — JWT subject
	CtxKeyRole   = "role"    // string — user role
	CtxKeyJTI    = "jti"     // string — token ID
	CtxKeyStatus = "status"  // string — account status
	CtxKeyLevel  = "level"   // int    — security level (0=highest)
)

type TokenBlacklist interface {
	IsBlacklisted(ctx context.Context, jti string) bool
}

// Access validates a Bearer JWT and injects parsed claims into the gin context.
// It also checks the token's JTI against a blacklist (e.g. for logged-out tokens).
func Access(secret string, blacklist TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := security.ExtractBearerToken(c.GetHeader("Authorization"))
		if !ok {
			c.Header("WWW-Authenticate", "Bearer")
			response.RespondUnauthorized(c, "unauthorized")
			c.Abort()
			return
		}

		claims, err := security.Parse(token, secret)
		if err != nil {
			if errors.Is(err, security.ErrEmptySecret) {
				response.RespondServiceUnavailable(c, "authentication temporarily unavailable")
				c.Abort()
				return
			}

			c.Header("WWW-Authenticate", "Bearer")
			response.RespondUnauthorized(c, "unauthorized")
			c.Abort()
			return
		}

		if blacklist != nil && blacklist.IsBlacklisted(c.Request.Context(), claims.TokenID) {
			logger.HandlerWarn(c, "iam.access", nil, "token is blacklisted")
			c.Header("WWW-Authenticate", "Bearer")
			response.RespondUnauthorized(c, "token has been revoked")
			c.Abort()
			return
		}

		// Store full claims for callers that need everything.
		c.Set(jwtClaimsContextKey, claims)

		// Inject individual identity fields as flat keys.
		c.Set(CtxKeyUserID, claims.Subject)
		c.Set(CtxKeyRole, claims.Role)
		c.Set(CtxKeyJTI, claims.TokenID)
		c.Set(CtxKeyStatus, claims.Status)
		c.Set(CtxKeyLevel, claims.Level) // int — read directly by RequireLevel

		// Piggyback on logger key so request logs include user_id automatically.
		if claims.Subject != "" {
			c.Set(logger.KeyUserID, claims.Subject)
		}

		c.Next()
	}
}

// JWTClaims returns the full parsed JWT claims from the gin context.
// Prefer c.GetString(middleware.CtxKeyUserID) for simple identity lookups.
func JWTClaims(c *gin.Context) (security.Claims, bool) {
	if c == nil {
		return security.Claims{}, false
	}

	v, ok := c.Get(jwtClaimsContextKey)
	if !ok {
		return security.Claims{}, false
	}

	claims, ok := v.(security.Claims)
	if !ok {
		return security.Claims{}, false
	}

	return claims, true
}
