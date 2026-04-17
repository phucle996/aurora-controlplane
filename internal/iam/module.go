package iam

import (
	"controlplane/internal/app/bootstrap"
	"controlplane/internal/config"
	iam_repoImple "controlplane/internal/iam/repository"
	iam_svcImple "controlplane/internal/iam/service"
	iam_handler "controlplane/internal/iam/transport/http/handler"
)

// Module encapsulates the IAM module's dependencies and config.
type Module struct {
	Cfg         *config.Config
	Infra       *bootstrap.Infra
	Runtime     *bootstrap.Runtime
	AuthHandler *iam_handler.AuthHandler
}

// NewModule initializes the IAM module and wires its dependencies.
func NewModule(cfg *config.Config, infra *bootstrap.Infra, rt *bootstrap.Runtime) *Module {
	userRepo := iam_repoImple.NewUserRepoImple(infra.DB)
	authSvc := iam_svcImple.NewAuthSvcImple(userRepo)
	authHandler := iam_handler.NewAuthHandler(authSvc)

	return &Module{
		Cfg:         cfg,
		Infra:       infra,
		Runtime:     rt,
		AuthHandler: authHandler,
	}
}
