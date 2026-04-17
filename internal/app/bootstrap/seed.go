package bootstrap

import (
	"context"
	"controlplane/internal/config"
)

func RunSeed(ctx context.Context, cfg *config.Config) error {
	// Startup-only bootstrap data
	return nil
}
