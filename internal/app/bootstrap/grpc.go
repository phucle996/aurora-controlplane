package bootstrap

import (
	"context"
	"controlplane/internal/config"
	"controlplane/pkg/logger"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPC holds both inbound server and outbound client manager.
type GRPC struct {
	Server  *grpc.Server
	Clients *GRPCClientManager
	lis     net.Listener
	cfg     *config.GRPCCfg
}

// GRPCClientManager manages reusable outbound gRPC connections.
// Connections are stored by service name — no per-request dial.
type GRPCClientManager struct {
	mu    sync.RWMutex
	conns map[string]*grpc.ClientConn
	cfg   *config.GRPCCfg
}

// InitGRPC initializes both gRPC server and client manager.
func InitGRPC(ctx context.Context, cfg *config.Config) (*GRPC, error) {
	// Server
	server := grpc.NewServer()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPC.ServerPort))
	if err != nil {
		return nil, fmt.Errorf("grpc: failed to listen on port %s: %w", cfg.GRPC.ServerPort, err)
	}

	// Client manager
	clients := &GRPCClientManager{
		conns: make(map[string]*grpc.ClientConn),
		cfg:   &cfg.GRPC,
	}

	logger.SysInfo("grpc", "start", fmt.Sprintf("grpc: server listener ready on :%s", cfg.GRPC.ServerPort))

	return &GRPC{
		Server:  server,
		Clients: clients,
		lis:     lis,
		cfg:     &cfg.GRPC,
	}, nil
}

// Start begins serving gRPC (blocking — run in goroutine).
func (g *GRPC) Start() error {
	logger.SysInfo("grpc", "starting", "grpc: server starting...")
	return g.Server.Serve(g.lis)
}

// Stop gracefully stops the gRPC server and closes all client connections.
func (g *GRPC) Stop() {
	logger.SysInfo("grpc", "stopping", "grpc: stopping server...")
	g.Server.GracefulStop()

	logger.SysInfo("grpc", "closing_clients", "grpc: closing client connections...")
	g.Clients.CloseAll()
}

// --- Client Manager ---

// Dial creates or reuses a gRPC client connection for a named service.
// Uses config-driven targets. Connection is reused on subsequent calls.
func (m *GRPCClientManager) Dial(ctx context.Context, serviceName, target string) (*grpc.ClientConn, error) {
	m.mu.RLock()
	if conn, ok := m.conns[serviceName]; ok {
		m.mu.RUnlock()
		return conn, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if conn, ok := m.conns[serviceName]; ok {
		return conn, nil
	}

	// Build credentials
	var creds credentials.TransportCredentials
	if m.cfg.ClientTLSEnabled {
		tlsConfig, err := buildGRPCClientTLSConfig(m.cfg)
		if err != nil {
			return nil, fmt.Errorf("grpc: failed to build client TLS config: %w", err)
		}
		creds = credentials.NewTLS(tlsConfig)
	} else {
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc: failed to dial %s (%s): %w", serviceName, target, err)
	}

	m.conns[serviceName] = conn
	logger.SysInfo("grpc_client", "connected", fmt.Sprintf("grpc: client connected to %s (%s)", serviceName, target))
	return conn, nil
}

// Get returns an existing connection by service name, or nil if not found.
func (m *GRPCClientManager) Get(serviceName string) *grpc.ClientConn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conns[serviceName]
}

// CloseAll closes all client connections.
func (m *GRPCClientManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, conn := range m.conns {
		if err := conn.Close(); err != nil {
			logger.SysError("grpc_client", "close_error", fmt.Sprintf("grpc: error closing client %s: %v", name, err), "")
		}
	}
	m.conns = make(map[string]*grpc.ClientConn)
}

// buildGRPCClientTLSConfig constructs TLS config from typed config.
func buildGRPCClientTLSConfig(cfg *config.GRPCCfg) (*tls.Config, error) {
	tlsCfg := &tls.Config{}

	if cfg.ClientCACertPath != "" {
		caCert, err := os.ReadFile(cfg.ClientCACertPath)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caCert)
		tlsCfg.RootCAs = pool
	}

	if cfg.ClientCertPath != "" && cfg.ClientKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load client cert/key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return tlsCfg, nil
}
