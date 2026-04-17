package app

import (
	"context"
	"controlplane/internal/app/bootstrap"
	"controlplane/internal/config"
)

// globalModules handles global module assembly.
// Receives infra and runtime to wire repositories, services, and handlers.
func globalModules(ctx context.Context, cfg *config.Config, infra *bootstrap.Infra, rt *bootstrap.Runtime) error {
	// Example wiring pattern:
	//
	// iamRepo := iam.NewRepository(infra.DB)
	// iamSvc  := iam.NewService(iamRepo)
	// iamHandler := iam.NewHandler(iamSvc)
	//
	// Attach to a GlobalModules struct if needed for route registration.

	return nil
}
