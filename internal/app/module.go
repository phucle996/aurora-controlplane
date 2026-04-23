package app

import (
	"controlplane/internal/app/bootstrap"
	"controlplane/internal/config"
	core "controlplane/internal/core"
	"controlplane/internal/iam"
	smtp "controlplane/internal/smtp"
	virtualmachine "controlplane/internal/virtualmachine"
)

type GlobalModules struct {
	IAM            *iam.Module
	Core           *core.Module
	VirtualMachine *virtualmachine.Module
	SMTP           *smtp.Module
}

// globalModules handles global module assembly.
// Receives infra and runtime to wire repositories, services, and handlers.
func globalModules(
	cfg *config.Config,
	infra *bootstrap.Infra,
	grpc *bootstrap.GRPC,
) (*GlobalModules, error) {
	// Example wiring pattern:
	//
	// iamRepo := iam.NewRepository(infra.DB)
	// iamSvc  := iam.NewService(iamRepo)
	// iamHandler := iam.NewHandler(iamSvc)
	//
	// Attach to a GlobalModules struct if needed for route registration.

	coreModule, err := core.NewModule(cfg, infra, grpc)
	if err != nil {
		return nil, err
	}

	iamModule, err := iam.NewModule(cfg, infra, coreModule.SecretService)
	if err != nil {
		return nil, err
	}

	virtualmachineModule, err := virtualmachine.NewModule(cfg, infra, grpc)
	if err != nil {
		return nil, err
	}

	smtpModule, err := smtp.NewModule(cfg, infra, grpc)
	if err != nil {
		return nil, err
	}

	return &GlobalModules{
		IAM:            iamModule,
		Core:           coreModule,
		VirtualMachine: virtualmachineModule,
		SMTP:           smtpModule,
	}, nil
}
