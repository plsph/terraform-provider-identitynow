package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Config is the configuration parameters for an IdentityNow API
type Config struct {
	URL          string `json:"url"`
	ClientId     string `json:"cacert"`
	ClientSecret string `json:"tokenKey"`
}

func (c *Client) IsTokenValid() bool {
	return time.Now().Before(c.tokenExpiry)
}

// IdentityNowClient returns a Client with a valid access token
func (cfg *Config) IdentityNowClient(ctx context.Context) (*Client, error) {
	client := NewClient(ctx, cfg.URL, cfg.ClientId, cfg.ClientSecret)

	// Optionally log configuration (without secrets)
	tflog.Info(ctx, "Initializing IdentityNow client", map[string]any{
		"base_url":  cfg.URL,
		"client_id": cfg.ClientId,
	})

	// Check if the current token is valid before attempting to get a new one
	if !client.IsTokenValid() {
		if err := client.GetToken(ctx); err != nil {
			return nil, err
		}
	}

	if len(client.accessToken) == 0 {
		return nil, fmt.Errorf("access token is empty")
	}
	return client, nil
}
