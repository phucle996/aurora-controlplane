package app

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	httpfs "controlplane/internal/http"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RegisterFrontend mounts the embedded static files from the 'out' directory to the Gin engine.
// It also sets up a catch-all NoRoute handler to support SPA (Single Page Application) routing.
func RegisterFrontend(r *gin.Engine) error {
	distFS, err := fs.Sub(httpfs.FrontendFS, "dist")
	if err != nil {
		logger.SysError("app.frontend", "Failed to initialize embedded frontend directory: "+err.Error())
		return err
	}

	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		logger.SysError("app.frontend", "Failed to read embedded index.html: "+err.Error())
		return err
	}

	// Serve frontend static assets cleanly skipping API routes
	fsHandler := http.FileServer(http.FS(distFS))
	serveIndex := func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	}

	r.GET("/", func(c *gin.Context) {
		serveIndex(c)
	})
	r.HEAD("/", func(c *gin.Context) {
		serveIndex(c)
	})

	r.Use(func(c *gin.Context) {
		// Ignore API and health check paths
		if isBackendPath(c.Request.URL.Path) || c.Request.URL.Path == "/" {
			c.Next()
			return
		}

		if serveExportedAsset(c, distFS, fsHandler) {
			c.Abort()
			return
		}

		// If no file found, we fall through to standard NoRoute handler logic
		c.Next()
	})

	// SPA Fallback NoRoute Hook
	r.NoRoute(func(c *gin.Context) {
		// Don't intercept backend 404s
		if isBackendPath(c.Request.URL.Path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
			return
		}

		// For frontend paths, fallback to index.html for client-side routing
		if c.Request.Method == http.MethodGet {
			serveIndex(c)
			return
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	return nil
}

func isBackendPath(path string) bool {
	return strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/admin")
}

func serveExportedAsset(c *gin.Context, distFS fs.FS, fsHandler http.Handler) bool {
	if c == nil || c.Request == nil {
		return false
	}

	candidates := staticPathCandidates(c.Request.URL.Path)
	for _, candidate := range candidates {
		info, err := fs.Stat(distFS, candidate)
		if err != nil || info.IsDir() {
			continue
		}

		req := c.Request.Clone(c.Request.Context())
		req.URL.Path = "/" + candidate
		fsHandler.ServeHTTP(c.Writer, req)
		return true
	}

	return false
}

func staticPathCandidates(rawPath string) []string {
	cleanPath := strings.TrimPrefix(path.Clean("/"+rawPath), "/")
	if cleanPath == "" || cleanPath == "." {
		return []string{"index.html"}
	}

	candidates := []string{cleanPath}
	if !strings.Contains(path.Base(cleanPath), ".") {
		candidates = append(candidates, cleanPath+".html")
		candidates = append(candidates, path.Join(cleanPath, "index.html"))
	}

	return candidates
}
