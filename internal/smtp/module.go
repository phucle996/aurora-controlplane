package smtp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"controlplane/internal/app/bootstrap"
	"controlplane/internal/config"
	redisprojection "controlplane/internal/primitive/projection/redis"
	"controlplane/internal/primitive/rebalance"
	smtp_domainsvc "controlplane/internal/smtp/domain/service"
	smtp_repo "controlplane/internal/smtp/repository"
	smtp_svc "controlplane/internal/smtp/service"
	smtpgrpc "controlplane/internal/smtp/transport/grpc"
	smtp_handler "controlplane/internal/smtp/transport/http/handler"
	"controlplane/pkg/logger"
)

type Module struct {
	Cfg   *config.Config
	Infra *bootstrap.Infra
	GRPC  *bootstrap.GRPC

	ConsumerRepo    *smtp_repo.ConsumerRepository
	TemplateRepo    *smtp_repo.TemplateRepository
	GatewayRepo     *smtp_repo.GatewayRepository
	EndpointRepo    *smtp_repo.EndpointRepository
	RuntimeRepo     *smtp_repo.RuntimeRepository
	AggregationRepo *smtp_repo.AggregationRepository

	ConsumerService    smtp_domainsvc.ConsumerService
	TemplateService    smtp_domainsvc.TemplateService
	GatewayService     smtp_domainsvc.GatewayService
	EndpointService    smtp_domainsvc.EndpointService
	RuntimeService     smtp_domainsvc.RuntimeService
	AggregationService smtp_domainsvc.AggregationService

	ConsumerHandler    *smtp_handler.ConsumerHandler
	TemplateHandler    *smtp_handler.TemplateHandler
	GatewayHandler     *smtp_handler.GatewayHandler
	EndpointHandler    *smtp_handler.EndpointHandler
	RuntimeHandler     *smtp_handler.RuntimeHandler
	AggregationHandler *smtp_handler.AggregationHandler

	RuntimeGRPCServer *smtpgrpc.RuntimeHandler

	stopReconciler context.CancelFunc
}

func NewModule(cfg *config.Config,
	infra *bootstrap.Infra,
	grpcServer *bootstrap.GRPC) (*Module, error) {
	m := &Module{
		Cfg:   cfg,
		Infra: infra,
		GRPC:  grpcServer,
	}

	if cfg == nil || infra == nil || infra.DB == nil {
		return nil, fmt.Errorf("smtp module: invalid arguments")
	}
	if strings.TrimSpace(cfg.Security.MasterKey) == "" {
		return nil, fmt.Errorf("smtp module: secret master key is required")
	}

	m.ConsumerRepo = smtp_repo.NewConsumerRepository(infra.DB)
	m.TemplateRepo = smtp_repo.NewTemplateRepository(infra.DB)
	m.GatewayRepo = smtp_repo.NewGatewayRepository(infra.DB)
	m.EndpointRepo = smtp_repo.NewEndpointRepository(infra.DB, cfg.Security.MasterKey)
	m.RuntimeRepo = smtp_repo.NewRuntimeRepository(infra.DB)
	m.AggregationRepo = smtp_repo.NewAggregationRepository(infra.DB)

	m.ConsumerService = smtp_svc.NewConsumerService(m.ConsumerRepo)
	m.TemplateService = smtp_svc.NewTemplateService(m.TemplateRepo, m.ConsumerRepo)
	m.GatewayService = smtp_svc.NewGatewayService(m.GatewayRepo, m.TemplateRepo, m.EndpointRepo, m.ConsumerRepo)
	m.EndpointService = smtp_svc.NewEndpointService(m.EndpointRepo)
	var projection rebalance.ProjectionSink
	if infra.Redis != nil {
		projection = redisprojection.NewSink(infra.Redis.Unwrap(), "smtp:runtime:active")
	}
	m.RuntimeService = smtp_svc.NewRuntimeService(m.RuntimeRepo, m.ConsumerRepo, m.TemplateRepo, m.GatewayRepo, m.EndpointRepo, projection)
	m.AggregationService = smtp_svc.NewAggregationService(m.AggregationRepo)

	m.ConsumerHandler = smtp_handler.NewConsumerHandler(m.ConsumerService)
	m.TemplateHandler = smtp_handler.NewTemplateHandler(m.TemplateService)
	m.GatewayHandler = smtp_handler.NewGatewayHandler(m.GatewayService)
	m.EndpointHandler = smtp_handler.NewEndpointHandler(m.EndpointService)
	m.RuntimeHandler = smtp_handler.NewRuntimeHandler(m.RuntimeService)
	m.AggregationHandler = smtp_handler.NewAggregationHandler(m.AggregationService)

	if grpcServer != nil && grpcServer.Server != nil && m.RuntimeService != nil {
		m.RuntimeGRPCServer = smtpgrpc.NewRuntimeServer(m.RuntimeService)
		smtpgrpc.RegisterSMTPRuntimeServer(grpcServer.Server, m.RuntimeGRPCServer)
	}

	return m, nil
}

func (m *Module) Start() {
	if m == nil || m.RuntimeService == nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.stopReconciler = cancel

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := m.RuntimeService.Reconcile(context.Background()); err != nil {
					logger.SysWarn("smtp.module", "smtp reconcile tick failed")
				}
			}
		}
	}()
}

func (m *Module) Stop() {
	if m == nil || m.stopReconciler == nil {
		return
	}
	m.stopReconciler()
}
