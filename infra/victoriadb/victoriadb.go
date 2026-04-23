package victoriadb

import (
	"context"
	"controlplane/internal/config"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps an HTTP client for VictoriaMetrics read/write operations.
type Client struct {
	httpClient   *http.Client
	writeURL     string
	readURL      string
	username     string
	password     string
	writeTimeout time.Duration
	readTimeout  time.Duration
}

// NewVictoriaDB creates a ready-to-use VictoriaMetrics client.
// Flow: build HTTP client → health check → return
func NewVictoriaDB(ctx context.Context, cfg *config.VictoriaDBCfg) (*Client, error) {
	httpClient := &http.Client{
		Timeout: cfg.ReadTimeout, // default; overridden per-request
	}

	client := &Client{
		httpClient:   httpClient,
		writeURL:     cfg.WriteURL,
		readURL:      cfg.ReadURL,
		username:     cfg.Username,
		password:     cfg.Password,
		writeTimeout: cfg.WriteTimeout,
		readTimeout:  cfg.ReadTimeout,
	}

	var lastErr error

	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
		pingCtx, pingCancel := context.WithTimeout(ctx, cfg.PingTimeout)
		lastErr = client.ping(pingCtx)
		pingCancel()

		if lastErr == nil {
			return client, nil
		}

		if attempt < cfg.MaxRetries {
			time.Sleep(cfg.RetryInterval)
		}
	}

	return nil, fmt.Errorf("victoriadb: failed to connect after %d attempts: %w", cfg.MaxRetries, lastErr)
}

// ping verifies connectivity to VictoriaMetrics via /health endpoint.
func (c *Client) ping(ctx context.Context) error {
	// VictoriaMetrics exposes /health on the same base URL
	// Derive base from readURL (strip /api/v1/query suffix)
	healthURL := c.readURL
	if len(healthURL) > 0 {
		// Try the standard /health endpoint on the same host
		// Parse base: assume readURL is like http://host:port/api/v1/query
		healthURL = deriveBaseURL(c.readURL) + "/health"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return fmt.Errorf("victoriadb: failed to create health request: %w", err)
	}

	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("victoriadb: health request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("victoriadb: health check returned status %d", resp.StatusCode)
	}

	return nil
}

// WriteClient returns the HTTP client configured for write operations.
func (c *Client) WriteClient() *http.Client {
	return &http.Client{
		Timeout: c.writeTimeout,
	}
}

// WriteURL returns the configured write endpoint.
func (c *Client) WriteURL() string {
	return c.writeURL
}

// ReadURL returns the configured read/query endpoint.
func (c *Client) ReadURL() string {
	return c.readURL
}

// Close cleans up the HTTP client.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

// deriveBaseURL extracts the scheme://host:port from a full URL.
func deriveBaseURL(rawURL string) string {
	// Simple approach: find third slash
	slashCount := 0
	for i, ch := range rawURL {
		if ch == '/' {
			slashCount++
			if slashCount == 3 {
				return rawURL[:i]
			}
		}
	}
	return rawURL
}
