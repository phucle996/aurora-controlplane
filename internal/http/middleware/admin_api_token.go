package middleware

import (
	"context"

	"controlplane/internal/http/response"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

var adminTokenValidator func(ctx context.Context, token string) (bool, error)

// InitAdminToken initializes the global validator for admin API tokens.
func InitAdminToken(v func(ctx context.Context, token string) (bool, error)) {
	adminTokenValidator = v
}

// AdminAPIToken validates a static admin API token from a cookie.
// This is for internal/admin tools and does not use JWT.
func AdminAPIToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if adminTokenValidator == nil {
			response.RespondServiceUnavailable(c, "admin token validator unavailable")
			c.Abort()
			return
		}

		token, err := c.Cookie("apitoken")
		if err != nil {
			logger.HandlerWarn(c, "admin.api-token", err, "admin api token cookie not found")
			response.RespondUnauthorized(c, "unauthorized")
			c.Abort()
			return
		}

		valid, err := adminTokenValidator(c.Request.Context(), token)
		if err != nil {
			logger.HandlerError(c, "admin.api-token", err)
			response.RespondServiceUnavailable(c, "admin api token validation unavailable")
			c.Abort()
			return
		}
		if !valid {
			logger.HandlerWarn(c, "admin.api-token", nil, "invalid admin api token")
			response.RespondUnauthorized(c, "unauthorized")
			c.Abort()
			return
		}

		c.Next()
	}
}
