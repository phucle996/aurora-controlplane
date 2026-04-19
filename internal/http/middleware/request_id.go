package middleware

import (
	"controlplane/pkg/id"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

const (
	// HeaderXRequestID is the standard header for Request ID.
	HeaderXRequestID = "X-Request-ID"
)

// RequestID generates a unique ULID for every incoming request,
// injects it into the gin.Context for logging, and attaches it to
// the HTTP response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := id.MustGenerate()

		// Inject into gin.Context using the key expected by pkg/logger.
		c.Set(logger.KeyRequestID, reqID)

		// Set the header in the response so the client gets it too.
		c.Header(HeaderXRequestID, reqID)

		c.Next()
	}
}
