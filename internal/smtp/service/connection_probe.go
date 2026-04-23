package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"controlplane/internal/smtp/domain/entity"

	goredis "github.com/redis/go-redis/v9"
	nats "github.com/nats-io/nats.go"
	amqp "github.com/rabbitmq/amqp091-go"
)

const smtpProbeTimeout = 5 * time.Second

func probeConsumerConnection(ctx context.Context, consumer *entity.Consumer) error {
	if consumer == nil {
		return errors.New("smtp probe: consumer is nil")
	}

	connectionCfg := decodeJSONMap(consumer.ConnectionConfig)
	secretCfg := decodeJSONMap(consumer.SecretConfig)
	tlsMode := strings.ToLower(stringValue(connectionCfg, "tls_mode"))
	tlsCfg, err := buildTLSConfig(tlsMode, secretCfg)
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(consumer.TransportType)) {
	case "redis-stream", "redis_stream", "redis":
		return probeRedis(ctx, connectionCfg, secretCfg, tlsCfg)
	case "rabbitmq":
		return probeRabbitMQ(connectionCfg, secretCfg, tlsCfg)
	case "nats":
		return probeNATS(connectionCfg, secretCfg, tlsCfg)
	case "kafka":
		return probeKafka(ctx, connectionCfg, tlsCfg)
	default:
		return fmt.Errorf("smtp probe: unsupported consumer transport %q", consumer.TransportType)
	}
}

func probeEndpointConnection(ctx context.Context, endpoint *entity.Endpoint) error {
	if endpoint == nil {
		return errors.New("smtp probe: endpoint is nil")
	}

	address := net.JoinHostPort(strings.TrimSpace(endpoint.Host), fmt.Sprintf("%d", endpoint.Port))
	serverName := strings.TrimSpace(endpoint.Host)
	mode := strings.ToLower(strings.TrimSpace(endpoint.TLSMode))
	tlsCfg, err := buildEndpointTLSConfig(mode, endpoint, serverName)
	if err != nil {
		return err
	}

	dialer := &net.Dialer{Timeout: smtpProbeTimeout}

	var client *smtp.Client
	switch mode {
	case "tls", "mtls":
		conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsCfg)
		if err != nil {
			return fmt.Errorf("smtp probe: dial tls endpoint: %w", err)
		}
		client, err = smtp.NewClient(conn, serverName)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("smtp probe: create smtp client: %w", err)
		}
	default:
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return fmt.Errorf("smtp probe: dial endpoint: %w", err)
		}
		client, err = smtp.NewClient(conn, serverName)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("smtp probe: create smtp client: %w", err)
		}
	}
	defer client.Close()

	if mode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("smtp probe: server does not support STARTTLS")
		}
		if err := client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("smtp probe: starttls failed: %w", err)
		}
	}

	if endpoint.Username != "" && endpoint.Password != "" {
		if mode == "none" && !isLocalSMTPHost(serverName) {
			return errors.New("smtp probe: plain auth requires TLS or localhost")
		}
		if err := client.Auth(smtp.PlainAuth("", endpoint.Username, endpoint.Password, serverName)); err != nil {
			return fmt.Errorf("smtp probe: smtp auth failed: %w", err)
		}
	}

	if err := client.Noop(); err != nil {
		return fmt.Errorf("smtp probe: smtp noop failed: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp probe: smtp quit failed: %w", err)
	}

	return nil
}

func probeRedis(ctx context.Context, connectionCfg map[string]any, secretCfg map[string]any, tlsCfg *tls.Config) error {
	addr := strings.TrimSpace(stringValue(connectionCfg, "addr"))
	if addr == "" {
		host := strings.TrimSpace(stringValue(connectionCfg, "host"))
		port := strings.TrimSpace(stringValue(connectionCfg, "port"))
		if host == "" || port == "" {
			return errors.New("smtp probe: redis addr is missing")
		}
		addr = net.JoinHostPort(host, port)
	}

	client := goredis.NewClient(&goredis.Options{
		Addr:      addr,
		Username:  strings.TrimSpace(stringValue(secretCfg, "username")),
		Password:  stringValue(secretCfg, "password"),
		TLSConfig: tlsCfg,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, smtpProbeTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("smtp probe: redis ping failed: %w", err)
	}
	return nil
}

func probeRabbitMQ(connectionCfg map[string]any, secretCfg map[string]any, tlsCfg *tls.Config) error {
	host := strings.TrimSpace(stringValue(connectionCfg, "host"))
	port := strings.TrimSpace(stringValue(connectionCfg, "port"))
	if host == "" || port == "" {
		return errors.New("smtp probe: rabbitmq host/port is missing")
	}

	scheme := "amqp"
	if tlsCfg != nil {
		scheme = "amqps"
	}
	username := strings.TrimSpace(stringValue(secretCfg, "username"))
	password := stringValue(secretCfg, "password")
	uri := fmt.Sprintf("%s://%s:%s@%s/", scheme, username, password, net.JoinHostPort(host, port))

	conn, err := amqp.DialConfig(uri, amqp.Config{
		TLSClientConfig: tlsCfg,
		Heartbeat:       smtpProbeTimeout,
		Locale:          "en_US",
	})
	if err != nil {
		return fmt.Errorf("smtp probe: rabbitmq dial failed: %w", err)
	}
	return conn.Close()
}

func probeNATS(connectionCfg map[string]any, secretCfg map[string]any, tlsCfg *tls.Config) error {
	serverURL := strings.TrimSpace(stringValue(connectionCfg, "server_url"))
	if serverURL == "" {
		return errors.New("smtp probe: nats server_url is missing")
	}

	opts := []nats.Option{
		nats.Timeout(smtpProbeTimeout),
	}
	if token := strings.TrimSpace(stringValue(secretCfg, "auth_token")); token != "" {
		opts = append(opts, nats.Token(token))
	}
	if tlsCfg != nil {
		opts = append(opts, nats.Secure(tlsCfg))
	}

	conn, err := nats.Connect(serverURL, opts...)
	if err != nil {
		return fmt.Errorf("smtp probe: nats connect failed: %w", err)
	}
	conn.Close()
	return nil
}

func probeKafka(ctx context.Context, connectionCfg map[string]any, tlsCfg *tls.Config) error {
	brokers := strings.TrimSpace(stringValue(connectionCfg, "brokers"))
	if brokers == "" {
		return errors.New("smtp probe: kafka brokers is missing")
	}

	first := strings.TrimSpace(strings.Split(brokers, ",")[0])
	if first == "" {
		return errors.New("smtp probe: kafka broker is empty")
	}

	dialer := &net.Dialer{Timeout: smtpProbeTimeout}
	if tlsCfg != nil {
		conn, err := tls.DialWithDialer(dialer, "tcp", first, tlsCfg)
		if err != nil {
			return fmt.Errorf("smtp probe: kafka tls dial failed: %w", err)
		}
		return conn.Close()
	}

	conn, err := dialer.DialContext(ctx, "tcp", first)
	if err != nil {
		return fmt.Errorf("smtp probe: kafka dial failed: %w", err)
	}
	return conn.Close()
}

func buildTLSConfig(mode string, secretCfg map[string]any) (*tls.Config, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "", "disabled", "none":
		return nil, nil
	case "tls", "starttls", "mtls":
		cfg := &tls.Config{MinVersion: tls.VersionTLS12}
		if caPEM := strings.TrimSpace(stringValue(secretCfg, "ca_cert_pem")); caPEM != "" {
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM([]byte(caPEM)) {
				return nil, errors.New("smtp probe: invalid ca cert pem")
			}
			cfg.RootCAs = pool
		}
		if mode == "mtls" {
			clientCert := strings.TrimSpace(stringValue(secretCfg, "client_cert_pem"))
			clientKey := strings.TrimSpace(stringValue(secretCfg, "client_key_pem"))
			if clientCert == "" || clientKey == "" {
				return nil, errors.New("smtp probe: client certificate is required for mtls")
			}
			cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
			if err != nil {
				return nil, fmt.Errorf("smtp probe: invalid client certificate: %w", err)
			}
			cfg.Certificates = []tls.Certificate{cert}
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("smtp probe: unsupported tls mode %q", mode)
	}
}

func buildEndpointTLSConfig(mode string, endpoint *entity.Endpoint, serverName string) (*tls.Config, error) {
	cfg, err := buildTLSConfig(mode, map[string]any{
		"ca_cert_pem":     endpoint.CACertPEM,
		"client_cert_pem": endpoint.ClientCertPEM,
		"client_key_pem":  endpoint.ClientKeyPEM,
	})
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = &tls.Config{}
	}
	cfg.MinVersion = tls.VersionTLS12
	cfg.ServerName = serverName
	return cfg, nil
}

func decodeJSONMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	if out == nil {
		return map[string]any{}
	}
	return out
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func isLocalSMTPHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}
