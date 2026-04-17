package app

import (
	"io/fs"
	"net/http"
	"strings"

	"controlplane/pkg/logger"
	"controlplane/ui"

	"github.com/gin-gonic/gin"
)

// RegisterFrontend mounts the embedded static files from the 'out' directory to the Gin engine.
// It also sets up a catch-all NoRoute handler to support SPA (Single Page Application) routing.
func RegisterFrontend(r *gin.Engine) error {
	distFS, err := fs.Sub(ui.FrontendFS, "out")
	if err != nil {
		logger.SysError("app.frontend", "embed_failed", "Failed to initialize embedded frontend directory", err.Error())
		return err
	}

	// Serve frontend static assets cleanly skipping API routes
	fsHandler := http.FileServer(http.FS(distFS))

	r.Use(func(c *gin.Context) {
		// Ignore API and health check paths
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.Next()
			return
		}

		// Try providing static file first
		file, err := distFS.Open(strings.TrimPrefix(c.Request.URL.Path, "/"))
		if err == nil {
			file.Close()
			fsHandler.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		// If no file found, we fall through to standard NoRoute handler logic
		c.Next()
	})

	// SPA Fallback NoRoute Hook
	r.NoRoute(func(c *gin.Context) {
		// Don't intercept API 404s
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
			return
		}

		// For frontend paths, fallback to index.html for client-side routing
		if c.Request.Method == http.MethodGet {
			indexFile, err := distFS.Open("index.html")
			if err == nil {
				indexFile.Close()
				c.FileFromFS("index.html", http.FS(distFS))
				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	return nil
}
