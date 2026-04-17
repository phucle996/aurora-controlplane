package bootstrap

import (
	"context"
	"controlplane/internal/config"
)

// Runtime holds shared runtime components (event bus, KV store, coordinators).
type Runtime struct {
	Infra *Infra
	// Add shared runtime components here, e.g.:
	// Bus   *eventbus.Bus
	// Store *kvstore.Store
}

// InitRuntime initializes shared runtime components that depend on infra.
func InitRuntime(ctx context.Context, cfg *config.Config, infra *Infra) (*Runtime, error) {
	return &Runtime{
		Infra: infra,
	}, nil
}

func (r *Runtime) Start() error {
	return nil
}

func (r *Runtime) Stop() error {
	return nil
}
