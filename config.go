package main

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ClientCredential struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Config is the configuration parameters for an IdentityNow API
type Config struct {
	URL                    string             `json:"url"`
	Credentials            []ClientCredential `json:"credentials,omitempty"`
	MaxClientPoolSize      int                `json:"max_client_pool_size,omitempty"`
	DefaultClientPoolSize  int                `json:"default_client_pool_size,omitempty"`
	ClientRequestRateLimit int                `json:"client_request_rate_limit"`

	// Client pool for round-robin token management
	clients        []*Client
	clientIndex    int
	clientPoolSize int
	clientMux      sync.Mutex
}

// initializeClientPool ensures the client pool is properly initialized
func (cfg *Config) initializeClientPool() {
	if cfg.clientPoolSize == 0 {
		cfg.clientPoolSize = cfg.DefaultClientPoolSize
	}
	if cfg.clientPoolSize > cfg.MaxClientPoolSize {
		cfg.clientPoolSize = cfg.MaxClientPoolSize
	}
	if cfg.clientPoolSize < 1 {
		cfg.clientPoolSize = 1
	}
	if cfg.clients == nil {
		cfg.clients = make([]*Client, cfg.clientPoolSize)
	}
}

// getNextClientIndex returns the next client index using round-robin
func (cfg *Config) getNextClientIndex() int {
	currentIndex := cfg.clientIndex
	cfg.clientIndex = (cfg.clientIndex + 1) % cfg.clientPoolSize
	return currentIndex
}

// IdentityNowClient returns a Client with a valid access token using round-robin selection
func (cfg *Config) IdentityNowClient(ctx context.Context) (*Client, error) {
	cfg.clientMux.Lock()
	cfg.initializeClientPool()
	clientIndex := cfg.getNextClientIndex()
	if cfg.clients[clientIndex] == nil {
		credential := cfg.Credentials[clientIndex%len(cfg.Credentials)]
		tflog.Debug(ctx, "Creating new IdentityNow client in pool", map[string]interface{}{
			"client_index": clientIndex,
			"base_url":     cfg.URL,
			"client_id":    credential.ClientId,
		})
		cfg.clients[clientIndex] = NewClient(ctx, cfg.URL, credential.ClientId, credential.ClientSecret, cfg.ClientRequestRateLimit)
	}
	client := cfg.clients[clientIndex]
	cfg.clientMux.Unlock()

	tflog.Debug(ctx, "Selected client from pool", map[string]interface{}{
		"client_index": clientIndex,
		"pool_size":    cfg.clientPoolSize,
	})

	// The token is refreshed per client, so other clients in the pool are not blocked meanwhile.
	if err := client.ensureToken(ctx); err != nil {
		tflog.Error(ctx, "Failed to get OAuth token for client", map[string]interface{}{
			"client_index": clientIndex,
			"error":        err.Error(),
		})
		return nil, err
	}

	return client, nil
}
