package main

import (
	"context"
	"fmt"
	"sync"
	"time"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Config is the configuration parameters for an IdentityNow API
type ClientCredential struct {
    ClientId     string `json:"client_id"`
    ClientSecret string `json:"client_secret"`
}

type Config struct {
	URL                   string `json:"url"`
	ClientId              string `json:"client_id,omitempty"`
	ClientSecret          string `json:"client_secret,omitempty"`
        Credentials           []ClientCredential `json:"credentials,omitempty"`
	MaxClientPoolSize     int    `json:"max_client_pool_size,omitempty" default:"1"`
	DefaultClientPoolSize int    `json:"default_client_pool_size,omitempty" default:"1"`
	ClientRequestRateLimit int   `json:"client_request_rate_limit" default:"10"`

	// Client pool for round-robin token management
	clients        []*Client
	clientIndex    int
	clientPoolSize int
	clientMux      sync.Mutex
}

func (c *Client) IsTokenValid(ctx context.Context) bool {
	tflog.Debug(ctx, "Checking if token is valid", map[string]interface{}{
		"token_expiry": c.tokenExpiry,
		"now":          time.Now(),
	})
	return time.Now().Before(c.tokenExpiry)
}

// initializeClientPool ensures the client pool is properly initialized
func (cfg *Config) initializeClientPool() {
	if cfg.clientPoolSize == 0 {
		cfg.clientPoolSize = cfg.DefaultClientPoolSize
	}
	if cfg.clientPoolSize > cfg.MaxClientPoolSize {
		cfg.clientPoolSize = cfg.MaxClientPoolSize
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
	tflog.Debug(ctx, "Client pool stats", cfg.GetClientPoolStats(ctx))
	cfg.clientMux.Lock()
	defer cfg.clientMux.Unlock()

	cfg.initializeClientPool()

	// Get the next client using round-robin
	clientIndex := cfg.getNextClientIndex()

	tflog.Debug(ctx, "Selecting client from pool", map[string]interface{}{
		"client_index": clientIndex,
		"pool_size":    cfg.clientPoolSize,
	})

	// Check if we have a client at this index and if its token is valid
	if cfg.clients[clientIndex] != nil && cfg.clients[clientIndex].IsTokenValid(ctx) {
		tflog.Debug(ctx, "Reusing existing client with valid token", map[string]interface{}{
			"client_index": clientIndex,
			"token_expiry": cfg.clients[clientIndex].tokenExpiry.Format(time.RFC3339),
		})
		return cfg.clients[clientIndex], nil
	}

	// Create new client if we don't have one at this index
	if cfg.clients[clientIndex] == nil {
		tflog.Debug(ctx, "Creating new IdentityNow client in pool", map[string]interface{}{
			"client_index": clientIndex,
			"base_url":     cfg.URL,
			"client_id":    cfg.Credentials[clientIndex % len(cfg.Credentials)].ClientId,
		})
		cfg.clients[clientIndex] = NewClient(ctx, cfg.URL, cfg.Credentials[clientIndex % len(cfg.Credentials)].ClientId,
		cfg.Credentials[clientIndex % len(cfg.Credentials)].ClientSecret, cfg.ClientRequestRateLimit)
	} else {
		tflog.Debug(ctx, "Token expired, refreshing token for client", map[string]interface{}{
			"client_index": clientIndex,
			"expired_at":   cfg.clients[clientIndex].tokenExpiry.Format(time.RFC3339),
			"now":          time.Now().Format(time.RFC3339),
		})
	}

	// Get/refresh the token for this client
	if err := cfg.clients[clientIndex].GetToken(ctx); err != nil {
		tflog.Error(ctx, "Failed to get OAuth token for client", map[string]interface{}{
			"client_index": clientIndex,
			"error":        err.Error(),
		})
		return nil, err
	}

	if len(cfg.clients[clientIndex].accessToken) == 0 {
		tflog.Error(ctx, "Access token is empty after successful token request", map[string]interface{}{
			"client_index": clientIndex,
		})
		return nil, fmt.Errorf("access token is empty for client %d", clientIndex)
	}

	tflog.Debug(ctx, "Client ready with valid token", map[string]interface{}{
		"client_index": clientIndex,
		"token_expiry": cfg.clients[clientIndex].tokenExpiry.Format(time.RFC3339),
	})

	return cfg.clients[clientIndex], nil
}

// ResetClient clears all cached clients, forcing creation of new ones on next request
func (cfg *Config) ResetClient() {
	cfg.clientMux.Lock()
	defer cfg.clientMux.Unlock()
	cfg.clients = nil
	cfg.clientIndex = 0
}

// SetClientPoolSize sets the size of the client pool (max 10)
func (cfg *Config) SetClientPoolSize(size int) {
	cfg.clientMux.Lock()
	defer cfg.clientMux.Unlock()

	if size > cfg.MaxClientPoolSize {
		size = cfg.MaxClientPoolSize
	}
	if size < 1 {
		size = 1
	}

	cfg.clientPoolSize = size
	// Reset the pool when size changes
	cfg.clients = nil
	cfg.clientIndex = 0
}

// GetClientPoolStats returns statistics about the client pool
func (cfg *Config) GetClientPoolStats(ctx context.Context) map[string]interface{} {
	cfg.clientMux.Lock()
	defer cfg.clientMux.Unlock()

	if cfg.clients == nil {
		return map[string]interface{}{
			"pool_size":      cfg.clientPoolSize,
			"active_clients": 0,
			"valid_tokens":   0,
		}
	}

	activeClients := 0
	validTokens := 0

	for i, client := range cfg.clients {
		if client != nil {
			activeClients++
			if client.IsTokenValid(ctx) {
				validTokens++
			}
		}
		tflog.Debug(ctx, "Client pool status", map[string]interface{}{
			"index":       i,
			"has_client":  client != nil,
			"valid_token": client != nil && client.IsTokenValid(ctx),
			"token_expiry": func() string {
				if client != nil {
					return client.tokenExpiry.Format(time.RFC3339)
				}
				return "N/A"
			}(),
		})
	}

	return map[string]interface{}{
		"pool_size":      len(cfg.clients),
		"active_clients": activeClients,
		"valid_tokens":   validTokens,
		"current_index":  cfg.clientIndex,
	}
}
