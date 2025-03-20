package main

import (
	"context"
	"fmt"
	"time"
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
func (c *Config) IdentityNowClient() (*Client, error) {
    client := NewClient(c.URL, c.ClientId, c.ClientSecret)
    ctx := context.Background()

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
