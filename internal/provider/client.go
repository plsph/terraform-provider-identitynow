package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/time/rate"
)

type Client struct {
	BaseURL      string
	clientId     string
	clientSecret string
	HTTPClient   *http.Client
	rateLimiter  *rate.Limiter

	// tokenMux guards accessToken and tokenExpiry, refreshMux serializes token refreshes.
	tokenMux    sync.Mutex
	refreshMux  sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

type errorResponse struct {
	DetailCode string `json:"detailCode"`
	// Error and ErrorDescription are set by OAuth and authentication errors instead of Messages.
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Messages         []struct {
		Locale       string `json:"locale"`
		LocaleOrigen string `json:"localeOrigin"`
		Text         string `json:"text"`
	} `json:"messages"`
}

func NewClient(ctx context.Context, baseURL string, clientId string, secret string, rateLimit int) *Client {
	// Normalize baseURL by removing trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	if rateLimit < 1 {
		rateLimit = 1
	}
	// Create rate limiter: [rateLimit] requests per second with burst of 1
	limiter := rate.NewLimiter(rate.Limit(rateLimit), 1)

	return &Client{
		BaseURL:      baseURL,
		clientId:     clientId,
		clientSecret: secret,
		rateLimiter:  limiter,
		HTTPClient: &http.Client{
			Timeout: time.Minute,
		},
	}
}

// token returns the current access token.
func (c *Client) token() string {
	c.tokenMux.Lock()
	defer c.tokenMux.Unlock()
	return c.accessToken
}

// isTokenValid reports whether the access token is set and not expired.
func (c *Client) isTokenValid() bool {
	c.tokenMux.Lock()
	defer c.tokenMux.Unlock()
	return c.accessToken != "" && time.Now().Before(c.tokenExpiry)
}

// invalidateToken forces the next ensureToken call to fetch a new token.
func (c *Client) invalidateToken() {
	c.tokenMux.Lock()
	defer c.tokenMux.Unlock()
	c.tokenExpiry = time.Time{}
}

// ensureToken fetches a new access token if the current one is missing or expired.
func (c *Client) ensureToken(ctx context.Context) error {
	c.refreshMux.Lock()
	defer c.refreshMux.Unlock()
	if c.isTokenValid() {
		return nil
	}
	return c.GetToken(ctx)
}

func (c *Client) GetToken(ctx context.Context) error {
	// Apply rate limiting before making any API requests
	if err := c.rateLimiter.Wait(ctx); err != nil {
		tflog.Debug(ctx, "Rate limiting wait failed", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("rate limiting failed: %w", err)
	}

	tokenURL := fmt.Sprintf("%s/oauth/token", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request for OAuth token", map[string]interface{}{
		"method":    "POST",
		"url":       tokenURL,
		"client_id": c.clientId,
	})
	// Credentials go in the form body, so they are never part of a URL that may appear in errors or logs.
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.clientId)
	form.Set("client_secret", c.clientSecret)
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	// Don't use sendRequest here as it requires an access token
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		tflog.Error(ctx, "Failed to get OAuth token", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		var errRes errorResponse
		err = json.NewDecoder(res.Body).Decode(&errRes)
		if err == nil {
			switch {
			case len(errRes.Messages) > 0:
				return fmt.Errorf("failed to get token (HTTP %d): %s", res.StatusCode, errRes.Messages[0].Text)
			case errRes.ErrorDescription != "":
				return fmt.Errorf("failed to get token (HTTP %d): %s", res.StatusCode, errRes.ErrorDescription)
			case errRes.Error != "":
				return fmt.Errorf("failed to get token (HTTP %d): %s", res.StatusCode, errRes.Error)
			}
		}
		tflog.Debug(ctx, "Failed to get OAuth token", map[string]interface{}{
			"status_code": res.StatusCode,
		})
		return fmt.Errorf("failed to get token, status code: %d", res.StatusCode)
	}

	var tokenRes OauthToken
	if err := json.NewDecoder(res.Body).Decode(&tokenRes); err != nil {
		tflog.Error(ctx, "Failed to decode OAuth token response", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	tflog.Debug(ctx, "OAuth token response received", map[string]interface{}{
		"expires_in": tokenRes.ExpiresIn,
		"token_type": tokenRes.TokenType,
	})

	if tokenRes.AccessToken == "" {
		return errors.New("access token is empty in OAuth token response")
	}

	// Set expiry with a safety margin and better validation
	var tokenExpiry time.Time
	if tokenRes.ExpiresIn > 0 {
		expirationDuration := time.Duration(tokenRes.ExpiresIn) * time.Second
		// Subtract 5 minutes as safety margin to refresh before actual expiry
		safetyMargin := 5 * time.Minute
		if expirationDuration > safetyMargin {
			expirationDuration -= safetyMargin
		}
		tokenExpiry = time.Now().Add(expirationDuration)

		tflog.Debug(ctx, "Token expiry set", map[string]interface{}{
			"expires_in_seconds": tokenRes.ExpiresIn,
			"token_expiry":       tokenExpiry.Format(time.RFC3339),
		})
	} else {
		// Fallback: set a default expiry of 1 hour if expires_in is missing or invalid
		tokenExpiry = time.Now().Add(1 * time.Hour)
		tflog.Warn(ctx, "expires_in field missing or invalid, using default 1 hour expiry", map[string]interface{}{
			"expires_in_received": tokenRes.ExpiresIn,
			"default_expiry":      tokenExpiry.Format(time.RFC3339),
		})
	}

	c.tokenMux.Lock()
	c.accessToken = tokenRes.AccessToken
	c.tokenExpiry = tokenExpiry
	c.tokenMux.Unlock()

	return nil
}

func (c *Client) GetSourceByName(ctx context.Context, name string) ([]*Source, error) {
	sourceURL := fmt.Sprintf("%s/v2026/sources?filters=%s", c.BaseURL, url.QueryEscape(eqFilter("name", name)))
	tflog.Debug(ctx, "Creating HTTP request to get source", map[string]interface{}{
		"method":      "GET",
		"url":         sourceURL,
		"source_name": name,
	})
	req, err := http.NewRequest("GET", sourceURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	var res []*Source
	if err := c.sendRequest(ctx, req, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) GetSource(ctx context.Context, id string) (*Source, error) {
	sourceURL := fmt.Sprintf("%s/v2026/sources/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to get source", map[string]interface{}{
		"method":    "GET",
		"url":       sourceURL,
		"source_id": id,
	})
	req, err := http.NewRequest("GET", sourceURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := Source{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) CreateSourceRequest(ctx context.Context, source *Source) (*Source, error) {
	body, err := json.Marshal(&source)
	if err != nil {
		return nil, err
	}
	sourceURL := fmt.Sprintf("%s/v2026/sources", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request to create source", map[string]interface{}{
		"method": "POST",
		"url":    sourceURL,
	})
	req, err := http.NewRequest("POST", sourceURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := Source{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Failed source creation", map[string]interface{}{
			"error":    err.Error(),
			"response": fmt.Sprintf("%+v", res),
		})
		return nil, err
	}
	return &res, nil
}

func (c *Client) AddConnectorAttributesToMicrosoftEntraSource(ctx context.Context, source *Source) (*Source, error) {
	if source == nil || source.ConnectorAttributes == nil {
		return nil, fmt.Errorf("source or ConnectorAttributes cannot be nil")
	}

	var updateSource []*UpdateSource

	// Reflect on the ConnectorAttributes struct
	val := reflect.ValueOf(source.ConnectorAttributes).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldName := getJSONFieldName(field)
		if fieldName == "" { // Skip fields without valid JSON tags
			continue
		}

		fieldValue := val.Field(i).Interface()

		// Skip empty or nil values
		if isEmptyValue(fieldValue) {
			continue
		}

		// Create the update source object
		updateSource = append(updateSource, &UpdateSource{
			Op:    "add",
			Path:  "/connectorAttributes/" + fieldName,
			Value: fieldValue,
		})
	}

	if len(updateSource) == 0 {
		tflog.Debug(ctx, "No attributes to update")
		return source, nil // Return the original source if nothing to update
	}

	// Marshal the updateSource to JSON
	body, err := json.MarshalIndent(updateSource, "", "  ")
	if err != nil {
		tflog.Error(ctx, "Failed to marshal updateSource", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to marshal updateSource: %w", err)
	}
	// Create the HTTP PATCH request
	patchURL := fmt.Sprintf("%s/v2026/sources/%s", c.BaseURL, source.ID)
	tflog.Debug(ctx, "Creating HTTP request to add connector attributes to Microsoft Entra source", map[string]interface{}{
		"method":    "PATCH",
		"url":       patchURL,
		"source_id": source.ID,
	})
	req, err := http.NewRequest("PATCH", patchURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create HTTP request", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	// Send the request and handle the response
	var res Source
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Failed updating source", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to update source: %w", err)
	}

	return &res, nil
}

func (c *Client) CreateSource(ctx context.Context, source *Source) (*Source, error) {
	var res *Source

	if source.Connector == "Microsoft-Entra" && source.ConnectorAttributes != nil {
		newSource := *source
		newSource.ConnectorAttributes = nil

		// Create source request
		sourceResponse, err := c.CreateSourceRequest(ctx, &newSource)
		if err != nil {
			return nil, err
		}
		source.ID = sourceResponse.ID
		// Add connector attributes. On failure the created source is returned with the error,
		// so the caller can track it instead of leaving it orphaned.
		res, err = c.AddConnectorAttributesToMicrosoftEntraSource(ctx, source)
		if err != nil {
			return sourceResponse, err
		}
	} else {
		var err error
		res, err = c.CreateSourceRequest(ctx, source)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (c *Client) UpdateSource(ctx context.Context, id string, patches []*UpdateSource) (*Source, error) {
	body, err := json.Marshal(patches)
	if err != nil {
		return nil, err
	}
	updateURL := fmt.Sprintf("%s/v2026/sources/%s", c.BaseURL, url.PathEscape(id))
	tflog.Debug(ctx, "Creating HTTP request to update source", map[string]interface{}{
		"method":    "PATCH",
		"url":       updateURL,
		"source_id": id,
	})
	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := Source{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteSource(ctx context.Context, source *Source) error {
	deleteURL := fmt.Sprintf("%s/v2026/sources/%s", c.BaseURL, source.ID)
	tflog.Debug(ctx, "Creating HTTP request to delete source", map[string]interface{}{
		"method":    "DELETE",
		"url":       deleteURL,
		"source_id": source.ID,
	})
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	var res interface{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Failed source deletion", map[string]interface{}{
			"error":    err.Error(),
			"response": fmt.Sprintf("%+v", res),
		})
		return err
	}

	return nil
}

func (c *Client) GetAccessProfileByName(ctx context.Context, name string) ([]*AccessProfile, error) {
	requestURL := fmt.Sprintf("%s/v2026/access-profiles?filters=%s", c.BaseURL, url.QueryEscape(eqFilter("name", name)))
	tflog.Debug(ctx, "Creating HTTP request for GetAccessProfileByName", map[string]interface{}{
		"method": "GET",
		"url":    requestURL,
		"name":   name,
	})
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	res := []*AccessProfile{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	return res, nil
}

func (c *Client) GetAccessProfile(ctx context.Context, id string) (*AccessProfile, error) {
	requestURL := fmt.Sprintf("%s/v2026/access-profiles/%s", c.BaseURL, url.PathEscape(id))
	tflog.Debug(ctx, "Creating HTTP request for GetAccessProfile", map[string]interface{}{
		"method":     "GET",
		"url":        requestURL,
		"profile_id": id,
	})
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	res := AccessProfile{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetSourceEntitlement(ctx context.Context, id string, nameFilter string) ([]*SourceEntitlement, error) {
	requestURL := fmt.Sprintf("%s/v2026/entitlements?filters=%s", c.BaseURL, url.QueryEscape(fmt.Sprintf("%s and (%s)", eqFilter("source.id", id), eqFilter("name", nameFilter))))
	tflog.Debug(ctx, "Creating HTTP request for GetSourceEntitlement", map[string]interface{}{
		"method":      "GET",
		"url":         requestURL,
		"source_id":   id,
		"name_filter": nameFilter,
	})
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	res := []*SourceEntitlement{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	return res, nil
}

func (c *Client) CreateAccessProfile(ctx context.Context, accessProfile *AccessProfile) (*AccessProfile, error) {
	body, err := json.Marshal(&accessProfile)
	if err != nil {
		return nil, err
	}

	createURL := fmt.Sprintf("%s/v2026/access-profiles", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request to create access profile", map[string]interface{}{
		"method": "POST",
		"url":    createURL,
	})
	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := AccessProfile{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) UpdateAccessProfile(ctx context.Context, accessProfile []*UpdateAccessProfile, id interface{}) (*AccessProfile, error) {
	body, err := json.Marshal(&accessProfile)
	if err != nil {
		return nil, err
	}
	updateURL := fmt.Sprintf("%s/v2026/access-profiles/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to update access profile", map[string]interface{}{
		"method":     "PATCH",
		"url":        updateURL,
		"profile_id": id,
	})
	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := AccessProfile{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteAccessProfile(ctx context.Context, accessProfile *AccessProfile) error {
	deleteURL := fmt.Sprintf("%s/v2026/access-profiles/%s", c.BaseURL, accessProfile.ID)
	tflog.Debug(ctx, "Creating HTTP request to delete access profile", map[string]interface{}{
		"method":     "DELETE",
		"url":        deleteURL,
		"profile_id": accessProfile.ID,
	})
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	var res interface{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return err
	}

	return nil
}

func (c *Client) GetRole(ctx context.Context, id string) (*Role, error) {
	roleURL := fmt.Sprintf("%s/v2026/roles/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to get role", map[string]interface{}{
		"method":  "GET",
		"url":     roleURL,
		"role_id": id,
	})
	req, err := http.NewRequest("GET", roleURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := Role{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) CreateRole(ctx context.Context, role *Role) (*Role, error) {
	body, err := json.Marshal(&role)
	if err != nil {
		return nil, err
	}

	createURL := fmt.Sprintf("%s/v2026/roles", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request to create role", map[string]interface{}{
		"method": "POST",
		"url":    createURL,
	})
	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}
	tflog.Debug(ctx, "Role request details", map[string]interface{}{
		"request": fmt.Sprintf("%v", req),
	})

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := Role{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) UpdateRole(ctx context.Context, role []*UpdateRole, id interface{}) (*Role, error) {
	body, err := json.Marshal(&role)
	if err != nil {
		return nil, err
	}
	updateURL := fmt.Sprintf("%s/v2026/roles/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to update role", map[string]interface{}{
		"method":  "PATCH",
		"url":     updateURL,
		"role_id": id,
	})
	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := Role{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteRole(ctx context.Context, role *Role) (*Role, error) {
	body, err := json.Marshal(&role)
	if err != nil {
		return nil, err
	}
	deleteURL := fmt.Sprintf("%s/v2026/roles/%s", c.BaseURL, role.ID)
	tflog.Debug(ctx, "Creating HTTP request to delete role", map[string]interface{}{
		"method":  "DELETE",
		"url":     deleteURL,
		"role_id": role.ID,
	})
	req, err := http.NewRequest("DELETE", deleteURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := Role{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetIdentityByAlias(ctx context.Context, alias string) ([]*Identity, error) {
	requestURL := fmt.Sprintf("%s/v2026/identities?filters=%s", c.BaseURL, url.QueryEscape(eqFilter("alias", alias)))
	tflog.Debug(ctx, "Creating HTTP request for GetIdentityByAlias", map[string]interface{}{
		"method": "GET",
		"url":    requestURL,
		"alias":  alias,
	})
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	res := []*Identity{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	return res, nil
}

func (c *Client) GetIdentityByEmail(ctx context.Context, email string) ([]*Identity, error) {
	requestURL := fmt.Sprintf("%s/v2026/identities?filters=%s", c.BaseURL, url.QueryEscape(eqFilter("email", email)))
	tflog.Debug(ctx, "Creating HTTP request for GetIdentityByEmail", map[string]interface{}{
		"method": "GET",
		"url":    requestURL,
		"email":  email,
	})
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	res := []*Identity{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	return res, nil
}

func (c *Client) GetAccountAggregationSchedule(ctx context.Context, id string) (*AccountAggregationSchedule, error) {
	scheduleURL := fmt.Sprintf("%s/cc/api/source/getAggregationSchedules/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to get account aggregation schedule", map[string]interface{}{
		"method":    "GET",
		"url":       scheduleURL,
		"source_id": id,
	})
	req, err := http.NewRequest("GET", scheduleURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req = req.WithContext(ctx)

	res := []AccountAggregationSchedule{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	if len(res) == 0 {
		return nil, &NotFoundError{fmt.Sprintf("no account aggregation schedule for source %q", id)}
	}
	return &res[0], nil
}

func (c *Client) ManageAccountAggregationSchedule(ctx context.Context, scheduleAggregation *AccountAggregationSchedule, enable bool) (*AccountAggregationSchedule, error) {
	if len(scheduleAggregation.CronExpressions) == 0 {
		return nil, errors.New("at least one cron expression is required")
	}
	endpoint := fmt.Sprintf("%s/cc/api/source/scheduleAggregation/%s", c.BaseURL, scheduleAggregation.SourceID)
	data := url.Values{}
	data.Set("enable", fmt.Sprintf("%t", enable))
	data.Set("cronExp", scheduleAggregation.CronExpressions[0])
	tflog.Debug(ctx, "Creating HTTP request to manage account aggregation schedule", map[string]interface{}{
		"method":    "POST",
		"url":       endpoint,
		"source_id": scheduleAggregation.SourceID,
		"enable":    enable,
	})
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	req = req.WithContext(ctx)

	res := AccountAggregationSchedule{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetAccountSchema(ctx context.Context, sourceId string, id string) (*AccountSchema, error) {
	schemaURL := fmt.Sprintf("%s/v2026/sources/%s/schemas/%s", c.BaseURL, sourceId, id)
	tflog.Debug(ctx, "Creating HTTP request to get account schema", map[string]interface{}{
		"method":    "GET",
		"url":       schemaURL,
		"source_id": sourceId,
		"schema_id": id,
	})
	req, err := http.NewRequest("GET", schemaURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req = req.WithContext(ctx)

	res := AccountSchema{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}
	res.SourceID = sourceId

	return &res, nil
}

func (c *Client) UpdateAccountSchema(ctx context.Context, accountSchema *AccountSchema) (*AccountSchema, error) {
	body, err := json.Marshal(&accountSchema)
	if err != nil {
		return nil, err
	}
	schemaURL := fmt.Sprintf("%s/v2026/sources/%s/schemas/%s", c.BaseURL, accountSchema.SourceID, accountSchema.ID)
	tflog.Debug(ctx, "Creating HTTP request to update account schema", map[string]interface{}{
		"method":    "PUT",
		"url":       schemaURL,
		"source_id": accountSchema.SourceID,
		"schema_id": accountSchema.ID,
	})
	req, err := http.NewRequest("PUT", schemaURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)
	res := AccountSchema{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) CreatePasswordPolicy(ctx context.Context, passwordPolicy *PasswordPolicy) (*PasswordPolicy, error) {
	body, err := json.Marshal(&passwordPolicy)
	if err != nil {
		return nil, err
	}
	policyURL := fmt.Sprintf("%s/v2026/password-policies", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request to create password policy", map[string]interface{}{
		"method": "POST",
		"url":    policyURL,
	})
	req, err := http.NewRequest("POST", policyURL, bytes.NewBuffer(body))

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := PasswordPolicy{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) UpdatePasswordPolicy(ctx context.Context, passwordPolicy *PasswordPolicy) (*PasswordPolicy, error) {

	body, err := json.Marshal(&passwordPolicy)
	if err != nil {
		return nil, err
	}
	policyURL := fmt.Sprintf("%s/v2026/password-policies/%s", c.BaseURL, url.PathEscape(passwordPolicy.ID))
	tflog.Debug(ctx, "Creating HTTP request to update password policy", map[string]interface{}{
		"method":    "PUT",
		"url":       policyURL,
		"policy_id": passwordPolicy.ID,
	})
	req, err := http.NewRequest("PUT", policyURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := PasswordPolicy{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetPasswordPolicy(ctx context.Context, passwordPolicyId string) (*PasswordPolicy, error) {
	policyURL := fmt.Sprintf("%s/v2026/password-policies/%s", c.BaseURL, passwordPolicyId)
	tflog.Debug(ctx, "Creating HTTP request to get password policy", map[string]interface{}{
		"method":    "GET",
		"url":       policyURL,
		"policy_id": passwordPolicyId,
	})
	req, err := http.NewRequest("GET", policyURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req = req.WithContext(ctx)

	res := PasswordPolicy{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeletePasswordPolicy(ctx context.Context, passwordPolicyId string) error {
	endpoint := fmt.Sprintf("%s/v2026/password-policies/%s", c.BaseURL, passwordPolicyId)

	tflog.Debug(ctx, "Creating HTTP request to delete password policy", map[string]interface{}{
		"method":    "DELETE",
		"url":       endpoint,
		"policy_id": passwordPolicyId,
	})
	req, err := http.NewRequest("DELETE", endpoint, nil)

	if err != nil {
		return err
	}

	req.Header.Set("Accept", "*/*")

	req = req.WithContext(ctx)

	var res interface{}
	return c.sendRequest(ctx, req, &res)
}

func (c *Client) CreateGovernanceGroup(ctx context.Context, governanceGroup *GovernanceGroup) (*GovernanceGroup, error) {
	body, err := json.Marshal(&governanceGroup)
	if err != nil {
		return nil, err
	}

	workgroupURL := fmt.Sprintf("%s/v2026/workgroups", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request to create governance group", map[string]interface{}{
		"method": "POST",
		"url":    workgroupURL,
	})
	req, err := http.NewRequest("POST", workgroupURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("X-SailPoint-Experimental", "true")

	req = req.WithContext(ctx)

	res := GovernanceGroup{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetGovernanceGroupByName(ctx context.Context, name string) ([]*GovernanceGroup, error) {
	requestURL := fmt.Sprintf("%s/v2026/workgroups?filters=%s", c.BaseURL, url.QueryEscape(eqFilter("name", name)))
	tflog.Debug(ctx, "Creating HTTP request for GetGovernanceGroupByName", map[string]interface{}{
		"method": "GET",
		"url":    requestURL,
		"name":   name,
	})
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("X-SailPoint-Experimental", "true")

	res := []*GovernanceGroup{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	return res, nil
}

func (c *Client) GetGovernanceGroups(ctx context.Context, id string) (*GovernanceGroup, error) {
	requestURL := fmt.Sprintf("%s/v2026/workgroups?filters=%s", c.BaseURL, url.QueryEscape(eqFilter("id", id)))
	tflog.Debug(ctx, "Creating HTTP request for GetGovernanceGroups", map[string]interface{}{
		"method":   "GET",
		"url":      requestURL,
		"group_id": id,
	})
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("X-SailPoint-Experimental", "true")

	res := []*GovernanceGroup{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if len(res) == 0 {
		return nil, &NotFoundError{fmt.Sprintf("governance group %q not found", id)}
	}
	return res[0], nil
}

func (c *Client) UpdateGovernanceGroup(ctx context.Context, governanceGroup []*UpdateGovernanceGroup, id interface{}) (*GovernanceGroup, error) {
	body, err := json.Marshal(&governanceGroup)
	if err != nil {
		return nil, err
	}
	updateURL := fmt.Sprintf("%s/v2026/workgroups/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to update governance group", map[string]interface{}{
		"method":   "PATCH",
		"url":      updateURL,
		"group_id": id,
	})
	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("X-SailPoint-Experimental", "true")

	req = req.WithContext(ctx)

	res := GovernanceGroup{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteGovernanceGroup(ctx context.Context, governanceGroup *GovernanceGroup) error {
	deleteURL := fmt.Sprintf("%s/v2026/workgroups/%s", c.BaseURL, governanceGroup.ID)
	tflog.Debug(ctx, "Creating HTTP request to delete governance group", map[string]interface{}{
		"method":   "DELETE",
		"url":      deleteURL,
		"group_id": governanceGroup.ID,
	})
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("X-SailPoint-Experimental", "true")

	req = req.WithContext(ctx)

	var res interface{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return err
	}

	return nil
}

func (c *Client) GetSourceAppsAll(ctx context.Context) ([]*SourceApp, error) {
	var allApps []*SourceApp
	offset := 0
	limit := 250
	for {
		sourceAppURL := fmt.Sprintf("%s/v2026/source-apps/all?limit=%d&offset=%d", c.BaseURL, limit, offset)
		tflog.Debug(ctx, "Creating HTTP request to get source apps", map[string]interface{}{
			"method": "GET",
			"url":    sourceAppURL,
		})
		req, err := http.NewRequest("GET", sourceAppURL, nil)
		if err != nil {
			tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
			return nil, err
		}

		req.Header.Set("X-SailPoint-Experimental", "true")

		req = req.WithContext(ctx)

		var res []*SourceApp
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
			return nil, err
		}

		allApps = append(allApps, res...)

		if len(res) < limit-1 {
			break
		}

		offset += limit
	}

	return allApps, nil
}

func (c *Client) GetSourceAppByName(ctx context.Context, name string) ([]*SourceApp, error) {
	sourceAppURL := fmt.Sprintf("%s/v2026/source-apps/all?filters=%s", c.BaseURL, url.QueryEscape(eqFilter("name", name)))
	tflog.Debug(ctx, "Creating HTTP request to get source app by name", map[string]interface{}{
		"method":   "GET",
		"url":      sourceAppURL,
		"app_name": name,
	})
	req, err := http.NewRequest("GET", sourceAppURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("X-SailPoint-Experimental", "true")

	req = req.WithContext(ctx)

	var res []*SourceApp
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return res, nil
}

func (c *Client) GetSourceApp(ctx context.Context, id string) (*SourceApp, error) {
	sourceAppURL := fmt.Sprintf("%s/v2026/source-apps/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to get source app", map[string]interface{}{
		"method": "GET",
		"url":    sourceAppURL,
		"app_id": id,
	})
	req, err := http.NewRequest("GET", sourceAppURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("X-SailPoint-Experimental", "true")

	req = req.WithContext(ctx)

	res := SourceApp{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) CreateSourceApp(ctx context.Context, sourceApp *SourceApp) (*SourceApp, error) {
	body, err := json.Marshal(&sourceApp)
	if err != nil {
		return nil, err
	}

	sourceAppURL := fmt.Sprintf("%s/v2026/source-apps", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request to create source app", map[string]interface{}{
		"method": "POST",
		"url":    sourceAppURL,
	})
	req, err := http.NewRequest("POST", sourceAppURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("X-SailPoint-Experimental", "true")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := SourceApp{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) UpdateSourceApp(ctx context.Context, sourceApp []*UpdateSourceApp, id interface{}) (*SourceApp, error) {
	body, err := json.Marshal(&sourceApp)
	if err != nil {
		return nil, err
	}
	updateURL := fmt.Sprintf("%s/v2026/source-apps/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to update source app", map[string]interface{}{
		"method": "PATCH",
		"url":    updateURL,
		"app_id": id,
	})
	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("X-SailPoint-Experimental", "true")
	req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := SourceApp{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteSourceApp(ctx context.Context, sourceApp *SourceApp) error {
	deleteURL := fmt.Sprintf("%s/v2026/source-apps/%s", c.BaseURL, sourceApp.ID)
	tflog.Debug(ctx, "Creating HTTP request to delete source app", map[string]interface{}{
		"method": "DELETE",
		"url":    deleteURL,
		"app_id": sourceApp.ID,
	})
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return err
	}

	req.Header.Set("X-SailPoint-Experimental", "true")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	var res interface{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return err
	}

	return nil
}

func (c *Client) GetAccessProfileAttachment(ctx context.Context, id string) (*AccessProfileAttachment, error) {
	var accessProfiles []string
	offset := 0
	limit := 250
	for {
		pageURL := fmt.Sprintf("%s/v2026/source-apps/%s/access-profiles?limit=%d&offset=%d", c.BaseURL, url.PathEscape(id), limit, offset)
		req, err := http.NewRequest("GET", pageURL, nil)
		if err != nil {
			tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
			return nil, err
		}
		req.Header.Set("X-SailPoint-Experimental", "true")

		var res []AccessProfileFromSourceApp
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Request failed", map[string]interface{}{"error": err.Error()})
			return nil, err
		}

		for _, ap := range res {
			accessProfiles = append(accessProfiles, ap.ID)
		}

		if len(res) < limit {
			break
		}
		offset += limit
	}

	accessProfileAttachment := AccessProfileAttachment{
		SourceAppId:    id,
		AccessProfiles: accessProfiles,
	}

	return &accessProfileAttachment, nil
}

func (c *Client) UpdateAccessProfileAttachment(ctx context.Context, accessProfileAttachment *AccessProfileAttachment, id string) (*AccessProfileAttachment, error) {
	//var accessProfiles []string

	//	for _, apa := range UpdateAccessProfileAttachment {
	//		accessProfiles = append(accessProfiles, apa.AccessProfiles...)
	//	}

	updateAccessProfileAttachment := UpdateAccessProfileAttachment{
		Op:    "replace",
		Path:  "/accessProfiles",
		Value: accessProfileAttachment.AccessProfiles,
		//Value: accessProfiles,
	}
	updates := []UpdateAccessProfileAttachment{updateAccessProfileAttachment}

	body, err := json.Marshal(updates)
	if err != nil {
		return nil, err
	}
	updateURL := fmt.Sprintf("%s/v2026/source-apps/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to update access profile attachment", map[string]interface{}{
		"method": "PATCH",
		"url":    updateURL,
		"app_id": id,
	})
	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("X-SailPoint-Experimental", "true")
	req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := AccessProfileAttachment{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteAccessProfileAttachment(ctx context.Context, accessProfileAttachment *AccessProfileAttachment) error {
	body, err := json.Marshal(accessProfileAttachment.AccessProfiles)
	if err != nil {
		return err
	}

	deleteURL := fmt.Sprintf("%s/v2026/source-apps/%s/access-profiles/bulk-remove", c.BaseURL, accessProfileAttachment.SourceAppId)
	tflog.Debug(ctx, "Creating HTTP request to delete access profile attachment", map[string]interface{}{
		"method":        "POST",
		"url":           deleteURL,
		"source_app_id": accessProfileAttachment.SourceAppId,
	})
	req, err := http.NewRequest("POST", deleteURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return err
	}

	req.Header.Set("X-SailPoint-Experimental", "true")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	var res interface{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return err
	}

	return nil
}

func (c *Client) CreateGovernanceGroupMembers(ctx context.Context, governanceGroupMembers *GovernanceGroupMembers, id string) (*GovernanceGroupMembers, error) {
	body, err := json.Marshal(governanceGroupMembers.GovernanceGroupMembersMembers)
	if err != nil {
		return nil, err
	}
	createURL := fmt.Sprintf("%s/v2026/workgroups/%s/members/bulk-add", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to create governance group members", map[string]interface{}{
		"method":              "POST",
		"url":                 createURL,
		"governance_group_id": id,
	})
	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("X-SailPoint-Experimental", "true")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := []GovernanceGroupMembersResponse{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	allSuccessful := true
	for _, member := range res {
		if member.Status != 201 {
			allSuccessful = false
			break
		}
	}

	if !allSuccessful {
		tflog.Error(ctx, "Creating governance group members failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, fmt.Errorf("not all governance group members were added successfully")
	}
	return governanceGroupMembers, nil
}

func (c *Client) GetGovernanceGroupMembers(ctx context.Context, id string) (*GovernanceGroupMembers, error) {
	governanceGroupMembersMembers := []*GovernanceGroupMembersMembers{}
	offset := 0
	limit := 50
	for {
		pageURL := fmt.Sprintf("%s/v2026/workgroups/%s/members?limit=%d&offset=%d", c.BaseURL, url.PathEscape(id), limit, offset)
		tflog.Debug(ctx, "Creating HTTP request to get governance group members", map[string]interface{}{
			"method":              "GET",
			"url":                 pageURL,
			"governance_group_id": id,
		})
		req, err := http.NewRequest("GET", pageURL, nil)
		if err != nil {
			tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
			return nil, err
		}
		req.Header.Set("X-SailPoint-Experimental", "true")

		var res []GovernanceGroupMembersMembers
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Request failed", map[string]interface{}{"error": err.Error()})
			return nil, err
		}

		for i := range res {
			governanceGroupMembersMembers = append(governanceGroupMembersMembers, &res[i])
		}

		if len(res) < limit {
			break
		}
		offset += limit
	}

	governanceGroupMembers := GovernanceGroupMembers{
		GovernanceGroupId:             id,
		GovernanceGroupMembersMembers: governanceGroupMembersMembers,
	}

	return &governanceGroupMembers, nil
}

func (c *Client) UpdateGovernanceGroupMembers(ctx context.Context, governanceGroupMembers *GovernanceGroupMembers, governanceGroupMembersActual *GovernanceGroupMembers, id string) (*GovernanceGroupMembers, error) {
	// Create maps for efficient lookup
	desiredMap := make(map[string]*GovernanceGroupMembersMembers)
	actualMap := make(map[string]*GovernanceGroupMembersMembers)
	var onlyInDesired []*GovernanceGroupMembersMembers
	var onlyInActual []*GovernanceGroupMembersMembers

	// Map desired items by ID
	for _, item := range governanceGroupMembers.GovernanceGroupMembersMembers {
		desiredMap[item.ID] = item
	}

	// Map actual items by ID
	for _, item := range governanceGroupMembersActual.GovernanceGroupMembersMembers {
		actualMap[item.ID] = item
	}

	// Find items only in desired list
	for _, item := range governanceGroupMembers.GovernanceGroupMembersMembers {
		if _, exists := actualMap[item.ID]; !exists {
			onlyInDesired = append(onlyInDesired, item)
		}
	}

	// Find items only in actual list
	for _, item := range governanceGroupMembersActual.GovernanceGroupMembersMembers {
		if _, exists := desiredMap[item.ID]; !exists {
			onlyInActual = append(onlyInActual, item)
		}
	}

	if len(onlyInActual) > 0 {
		// Delete actual members no longer desired
		membersOnlyInActual := GovernanceGroupMembers{
			GovernanceGroupId:             id,
			GovernanceGroupMembersMembers: onlyInActual,
		}

		body, err := json.Marshal(membersOnlyInActual.GovernanceGroupMembersMembers)
		if err != nil {
			return nil, err
		}

		deleteURL := fmt.Sprintf("%s/v2026/workgroups/%s/members/bulk-delete", c.BaseURL, governanceGroupMembers.GovernanceGroupId)
		tflog.Debug(ctx, "Creating HTTP request to update governance group members", map[string]interface{}{
			"method":              "POST",
			"url":                 deleteURL,
			"governance_group_id": governanceGroupMembers.GovernanceGroupId,
		})
		req, err := http.NewRequest("POST", deleteURL, bytes.NewBuffer(body))
		if err != nil {
			tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
			return nil, err
		}

		req.Header.Set("X-SailPoint-Experimental", "true")
		req.Header.Set("Accept", "application/json; charset=utf-8")
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		req = req.WithContext(ctx)

		res := []GovernanceGroupMembersResponse{}
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
			// Error already logged above
			return nil, err
		}

		allSuccessful := true
		for _, member := range res {
			if member.Status != 204 {
				allSuccessful = false
				break
			}
		}

		if !allSuccessful {
			tflog.Error(ctx, "Updating governance group members during remove phase failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
			return nil, fmt.Errorf("not all governance group members were removed successfully")
		}
	}

	if len(onlyInDesired) > 0 {
		// Create new members
		membersOnlyInDesired := GovernanceGroupMembers{
			GovernanceGroupId:             id,
			GovernanceGroupMembersMembers: onlyInDesired,
		}

		body, err := json.Marshal(membersOnlyInDesired.GovernanceGroupMembersMembers)
		if err != nil {
			return nil, err
		}
		createURL := fmt.Sprintf("%s/v2026/workgroups/%s/members/bulk-add", c.BaseURL, id)
		tflog.Debug(ctx, "Creating HTTP request to update governance group members", map[string]interface{}{
			"method":              "POST",
			"url":                 createURL,
			"governance_group_id": id,
		})
		req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(body))
		if err != nil {
			tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
			return nil, err
		}

		req.Header.Set("X-SailPoint-Experimental", "true")
		req.Header.Set("Accept", "application/json; charset=utf-8")
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		req = req.WithContext(ctx)

		res := []GovernanceGroupMembersResponse{}
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
			// Error already logged above
			return nil, err
		}

		allSuccessful := true
		for _, member := range res {
			if member.Status != 201 {
				allSuccessful = false
				break
			}
		}

		if !allSuccessful {
			tflog.Error(ctx, "Updating governance group members during add phase failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
			return nil, fmt.Errorf("not all governance group members were added successfully")
		}
	}

	return governanceGroupMembers, nil
}

func (c *Client) DeleteGovernanceGroupMembers(ctx context.Context, governanceGroupMembers *GovernanceGroupMembers) error {
	body, err := json.Marshal(governanceGroupMembers.GovernanceGroupMembersMembers)
	if err != nil {
		return err
	}

	deleteURL := fmt.Sprintf("%s/v2026/workgroups/%s/members/bulk-delete", c.BaseURL, governanceGroupMembers.GovernanceGroupId)
	tflog.Debug(ctx, "Creating HTTP request to delete governance group members", map[string]interface{}{
		"method":              "POST",
		"url":                 deleteURL,
		"governance_group_id": governanceGroupMembers.GovernanceGroupId,
	})
	req, err := http.NewRequest("POST", deleteURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return err
	}

	req.Header.Set("X-SailPoint-Experimental", "true")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := []GovernanceGroupMembersResponse{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return err
	}

	// Members that are already gone (404) count as removed.
	allSuccessful := true
	for _, member := range res {
		if member.Status != http.StatusNoContent && member.Status != http.StatusNotFound {
			allSuccessful = false
			break
		}
	}

	if !allSuccessful {
		tflog.Error(ctx, "Deleting governance group members during failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return fmt.Errorf("not all governance group members were removed successfully")
	}

	return nil
}

func (c *Client) GetTaggedObject(ctx context.Context, objectType string, objectID string) (*TaggedObject, error) {
	taggedObjectURL := fmt.Sprintf("%s/v2026/tagged-objects/%s/%s", c.BaseURL, url.PathEscape(objectType), url.PathEscape(objectID))
	tflog.Debug(ctx, "Creating HTTP request to get tagged object", map[string]interface{}{
		"method":      "GET",
		"url":         taggedObjectURL,
		"object_type": objectType,
		"object_id":   objectID,
	})

	req, err := http.NewRequest("GET", taggedObjectURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")

	res := TaggedObject{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	return &res, nil
}

func (c *Client) SetTaggedObject(ctx context.Context, taggedObject *TaggedObject) (*TaggedObject, error) {
	body, err := json.Marshal(taggedObject)
	if err != nil {
		return nil, err
	}

	taggedObjectURL := fmt.Sprintf("%s/v2026/tagged-objects/%s/%s", c.BaseURL, url.PathEscape(taggedObject.ObjectRef.Type), url.PathEscape(taggedObject.ObjectRef.ID))
	tflog.Debug(ctx, "Creating HTTP request to set tagged object", map[string]interface{}{
		"method":      "PUT",
		"url":         taggedObjectURL,
		"object_type": taggedObject.ObjectRef.Type,
		"object_id":   taggedObject.ObjectRef.ID,
	})
	req, err := http.NewRequest("PUT", taggedObjectURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := TaggedObject{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteTaggedObject(ctx context.Context, objectType string, objectID string) error {
	taggedObjectURL := fmt.Sprintf("%s/v2026/tagged-objects/%s/%s", c.BaseURL, url.PathEscape(objectType), url.PathEscape(objectID))
	tflog.Debug(ctx, "Creating HTTP request to delete tagged object", map[string]interface{}{
		"method":      "DELETE",
		"url":         taggedObjectURL,
		"object_type": objectType,
		"object_id":   objectID,
	})
	req, err := http.NewRequest("DELETE", taggedObjectURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	var res interface{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return err
	}

	return nil
}

func (c *Client) GetDimension(ctx context.Context, roleId string, dimensionId string) (*Dimension, error) {
	dimensionURL := fmt.Sprintf("%s/v2026/roles/%s/dimensions/%s", c.BaseURL, roleId, dimensionId)
	tflog.Debug(ctx, "Creating HTTP request to get dimension", map[string]interface{}{
		"method":       "GET",
		"url":          dimensionURL,
		"role_id":      roleId,
		"dimension_id": dimensionId,
	})
	req, err := http.NewRequest("GET", dimensionURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := Dimension{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetSegment(ctx context.Context, id string) (*Segment, error) {
	segmentURL := fmt.Sprintf("%s/v2026/segments/%s", c.BaseURL, url.PathEscape(id))
	req, err := http.NewRequest("GET", segmentURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := Segment{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) GetSegments(ctx context.Context) ([]*Segment, error) {
	const limit = 250
	var segments []*Segment
	for offset := 0; ; offset += limit {
		segmentURL := fmt.Sprintf("%s/v2026/segments?limit=%d&offset=%d", c.BaseURL, limit, offset)
		req, err := http.NewRequest("GET", segmentURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json; charset=utf-8")

		var page []*Segment
		if err := c.sendRequest(ctx, req, &page); err != nil {
			return nil, err
		}
		segments = append(segments, page...)
		if len(page) < limit {
			return segments, nil
		}
	}
}

func (c *Client) GetSegmentByName(ctx context.Context, name string) (*Segment, error) {
	segments, err := c.GetSegments(ctx)
	if err != nil {
		return nil, err
	}
	var match *Segment
	for _, segment := range segments {
		if segment.Name == name {
			if match != nil {
				return nil, fmt.Errorf("multiple segments are named %q", name)
			}
			match = segment
		}
	}
	if match == nil {
		return nil, &NotFoundError{fmt.Sprintf("segment with name %q not found", name)}
	}
	return match, nil
}

func (c *Client) CreateSegment(ctx context.Context, segment *Segment) (*Segment, error) {
	body, err := json.Marshal(segment)
	if err != nil {
		return nil, err
	}
	segmentURL := fmt.Sprintf("%s/v2026/segments", c.BaseURL)
	req, err := http.NewRequest("POST", segmentURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := Segment{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) UpdateSegment(ctx context.Context, id string, patches []*UpdateSegment) (*Segment, error) {
	body, err := json.Marshal(patches)
	if err != nil {
		return nil, err
	}
	segmentURL := fmt.Sprintf("%s/v2026/segments/%s", c.BaseURL, url.PathEscape(id))
	req, err := http.NewRequest("PATCH", segmentURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := Segment{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) DeleteSegment(ctx context.Context, id string) error {
	segmentURL := fmt.Sprintf("%s/v2026/segments/%s", c.BaseURL, url.PathEscape(id))
	req, err := http.NewRequest("DELETE", segmentURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	var res interface{}
	return c.sendRequest(ctx, req, &res)
}

func (c *Client) CreateDimension(ctx context.Context, roleId string, dimension *Dimension) (*Dimension, error) {
	body, err := json.Marshal(&dimension)
	if err != nil {
		return nil, err
	}

	createURL := fmt.Sprintf("%s/v2026/roles/%s/dimensions", c.BaseURL, roleId)
	tflog.Debug(ctx, "Creating HTTP request to create dimension", map[string]interface{}{
		"method":  "POST",
		"url":     createURL,
		"role_id": roleId,
	})
	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := Dimension{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) UpdateDimension(ctx context.Context, roleId string, dimensionId string, patches []*UpdateDimension) (*Dimension, error) {
	body, err := json.Marshal(&patches)
	if err != nil {
		return nil, err
	}
	updateURL := fmt.Sprintf("%s/v2026/roles/%s/dimensions/%s", c.BaseURL, roleId, dimensionId)
	tflog.Debug(ctx, "Creating HTTP request to update dimension", map[string]interface{}{
		"method":       "PATCH",
		"url":          updateURL,
		"role_id":      roleId,
		"dimension_id": dimensionId,
	})
	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := Dimension{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteDimension(ctx context.Context, roleId string, dimensionId string) error {
	deleteURL := fmt.Sprintf("%s/v2026/roles/%s/dimensions/%s", c.BaseURL, roleId, dimensionId)
	tflog.Debug(ctx, "Creating HTTP request to delete dimension", map[string]interface{}{
		"method":       "DELETE",
		"url":          deleteURL,
		"role_id":      roleId,
		"dimension_id": dimensionId,
	})
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	var res interface{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return err
	}

	return nil
}

func (c *Client) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	workflowURL := fmt.Sprintf("%s/v2026/workflows/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to get workflow", map[string]interface{}{
		"method":      "GET",
		"url":         workflowURL,
		"workflow_id": id,
	})
	req, err := http.NewRequest("GET", workflowURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := Workflow{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetWorkflowByName(ctx context.Context, name string) (*Workflow, error) {
	workflowURL := fmt.Sprintf("%s/v2026/workflows", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request to list workflows", map[string]interface{}{
		"method": "GET",
		"url":    workflowURL,
		"name":   name,
	})
	req, err := http.NewRequest("GET", workflowURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	var res []*Workflow
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	var match *Workflow
	for _, w := range res {
		if w.Name == name {
			if match != nil {
				return nil, fmt.Errorf("multiple workflows are named %q", name)
			}
			match = w
		}
	}
	if match == nil {
		return nil, &NotFoundError{fmt.Sprintf("workflow with name %q not found", name)}
	}
	return match, nil
}

func (c *Client) CreateWorkflow(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	body, err := json.Marshal(workflow)
	if err != nil {
		return nil, err
	}

	createURL := fmt.Sprintf("%s/v2026/workflows", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request to create workflow", map[string]interface{}{
		"method": "POST",
		"url":    createURL,
	})
	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := Workflow{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) UpdateWorkflow(ctx context.Context, id string, workflow *Workflow) (*Workflow, error) {
	body, err := json.Marshal(workflow)
	if err != nil {
		return nil, err
	}

	updateURL := fmt.Sprintf("%s/v2026/workflows/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to update workflow", map[string]interface{}{
		"method":      "PUT",
		"url":         updateURL,
		"workflow_id": id,
	})
	req, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	res := Workflow{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) SetWorkflowEnabled(ctx context.Context, id string, enabled bool) (*Workflow, error) {
	body, err := json.Marshal([]map[string]interface{}{{"op": "replace", "path": "/enabled", "value": enabled}})
	if err != nil {
		return nil, err
	}

	patchURL := fmt.Sprintf("%s/v2026/workflows/%s", c.BaseURL, url.PathEscape(id))
	tflog.Debug(ctx, "Creating HTTP request to set workflow enabled", map[string]interface{}{
		"method":      "PATCH",
		"url":         patchURL,
		"workflow_id": id,
		"enabled":     enabled,
	})
	req, err := http.NewRequest("PATCH", patchURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := Workflow{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteWorkflow(ctx context.Context, id string) error {
	deleteURL := fmt.Sprintf("%s/v2026/workflows/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to delete workflow", map[string]interface{}{
		"method":      "DELETE",
		"url":         deleteURL,
		"workflow_id": id,
	})
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	var res interface{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return err
	}

	return nil
}

func (c *Client) GetFormDefinition(ctx context.Context, id string) (*FormDefinition, error) {
	formURL := fmt.Sprintf("%s/v2026/form-definitions/%s", c.BaseURL, url.PathEscape(id))
	tflog.Debug(ctx, "Creating HTTP request to get form definition", map[string]interface{}{
		"method":             "GET",
		"url":                formURL,
		"form_definition_id": id,
	})
	req, err := http.NewRequest("GET", formURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := FormDefinition{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) GetFormDefinitionByName(ctx context.Context, name string) (*FormDefinition, error) {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(name)
	query := url.Values{}
	query.Set("filters", fmt.Sprintf(`name eq "%s"`, escaped))
	query.Set("limit", "250")
	query.Set("offset", "0")
	formURL := fmt.Sprintf("%s/v2026/form-definitions?%s", c.BaseURL, query.Encode())
	tflog.Debug(ctx, "Creating HTTP request to list form definitions", map[string]interface{}{
		"method": "GET",
		"url":    formURL,
		"name":   name,
	})
	req, err := http.NewRequest("GET", formURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := ListFormDefinitionsResponse{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	for _, form := range res.Results {
		if form.Name == name {
			return form, nil
		}
	}

	return nil, &NotFoundError{fmt.Sprintf("form definition with name %q not found", name)}
}

func (c *Client) CreateFormDefinition(ctx context.Context, form *FormDefinition) (*FormDefinition, error) {
	body, err := json.Marshal(form)
	if err != nil {
		return nil, err
	}

	createURL := fmt.Sprintf("%s/v2026/form-definitions", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request to create form definition", map[string]interface{}{
		"method": "POST",
		"url":    createURL,
	})
	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := FormDefinition{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) UpdateFormDefinition(ctx context.Context, id string, patches []*UpdateFormDefinition) (*FormDefinition, error) {
	body, err := json.Marshal(patches)
	if err != nil {
		return nil, err
	}

	updateURL := fmt.Sprintf("%s/v2026/form-definitions/%s", c.BaseURL, url.PathEscape(id))
	tflog.Debug(ctx, "Creating HTTP request to update form definition", map[string]interface{}{
		"method":             "PATCH",
		"url":                updateURL,
		"form_definition_id": id,
	})
	req, err := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(body))
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	res := FormDefinition{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteFormDefinition(ctx context.Context, id string) error {
	deleteURL := fmt.Sprintf("%s/v2026/form-definitions/%s", c.BaseURL, url.PathEscape(id))
	tflog.Debug(ctx, "Creating HTTP request to delete form definition", map[string]interface{}{
		"method":             "DELETE",
		"url":                deleteURL,
		"form_definition_id": id,
	})
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return err
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	var res interface{}
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		return err
	}

	return nil
}
