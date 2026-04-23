package virtualmachine

import (
	"controlplane/internal/app/bootstrap"
	"controlplane/internal/config"
	vm_repo "controlplane/internal/virtualmachine/repository"
	vm_svc "controlplane/internal/virtualmachine/service"
	vm_grpc "controlplane/internal/virtualmachine/transport/grpc"
	vm_handler "controlplane/internal/virtualmachine/transport/http/handler"
)

type Module struct {
	Cfg   *config.Config
	Infra *bootstrap.Infra
	GRPC  *bootstrap.GRPC

	HostRepo    *vm_repo.HostRepository
	HostService *vm_svc.HostService
	HostHandler *vm_handler.HostHandler
	GRPCServer  *vm_grpc.HostRegistryHandler
}

func NewModule(cfg *config.Config, infra *bootstrap.Infra, grpcServer *bootstrap.GRPC) (*Module, error) {
	m := &Module{
		Cfg:   cfg,
		Infra: infra,
		GRPC:  grpcServer,
	}

	m.HostRepo = vm_repo.NewHostRepository(infra.DB)
	m.HostService = vm_svc.NewHostService(m.HostRepo)
	m.HostHandler = vm_handler.NewHostHandler(m.HostService)

	m.GRPCServer = vm_grpc.NewHostRegistryServer(m.HostService)
	vm_grpc.RegisterVirtualMachineRegistryServer(grpcServer.Server, m.GRPCServer)

	return m, nil
}
