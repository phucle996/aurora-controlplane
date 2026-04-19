package httpfs

import "embed"

// FrontendFS embeds the exported static frontend that UI build scripts place in internal/http/dist.
//
//go:embed all:dist
var FrontendFS embed.FS
