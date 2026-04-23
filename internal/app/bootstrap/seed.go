package bootstrap

import (
	"context"

	"controlplane/internal/config"
)

// RunSeed is intentionally a no-op because bootstrap data is now owned by
// embedded SQL migrations. The hook stays here so application startup wiring
// does not break when the binary is built in other environments.
func RunSeed(_ context.Context, _ *config.Config) error {
	return nil
}
