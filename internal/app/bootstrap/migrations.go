package bootstrap

import (
	"context"
	"controlplane/internal/config"
)

func RunMigrations(ctx context.Context, cfg *config.Config) error {
	// Run migrations before serving traffic
	return nil
}
