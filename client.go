package main

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
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/time/rate"
)

type Client struct {
	BaseURL      string
	clientId     string
	clientSecret string
	accessToken  string
	HTTPClient   *http.Client
	tokenExpiry  time.Time
	loggerCtx    context.Context
	rateLimiter  *rate.Limiter
}

type errorResponse struct {
	DetailCode string `json:"detailCode"`
	Messages   []struct {
		Locale       string `json:"locale"`
		LocaleOrigen string `json:"localeOrigin"`
		Text         string `json:"text"`
	} `json:"messages"`
}

func NewClient(ctx context.Context, baseURL string, clientId string, secret string, rateLimit int) *Client {
	subctx := tflog.NewSubsystem(ctx, "identitynow")
	// Mask the client_secret if it ever appears as a field
	subctx = tflog.MaskFieldValuesWithFieldKeys(subctx, "client_secret")

	// Normalize baseURL by removing trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Create rate limiter: [rateLimit] requests per second with burst of 1
	limiter := rate.NewLimiter(rate.Limit(rateLimit), 1)

	return &Client{
		BaseURL:      baseURL,
		clientId:     clientId,
		clientSecret: secret,
		loggerCtx:    subctx,
		rateLimiter:  limiter,
		HTTPClient: &http.Client{
			Timeout: time.Minute,
		},
	}
}

func (c *Client) GetToken(ctx context.Context) error {
	// Apply rate limiting before making any API requests
	if err := c.rateLimiter.Wait(ctx); err != nil {
		tflog.Debug(ctx, "Rate limiting wait failed", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("rate limiting failed: %w", err)
	}

	tflog.Debug(ctx, "Obtaining OAuth token from IdentityNow", map[string]interface{}{
		"base_url":  c.BaseURL,
		"client_id": c.clientId,
	})

	tokenURL := fmt.Sprintf("%s/oauth/token?grant_type=client_credentials&client_id=%s&client_secret=%s", c.BaseURL, c.clientId, c.clientSecret)
	tflog.Debug(ctx, "Creating HTTP request for OAuth token", map[string]interface{}{
		"method": "POST",
		"url":    c.BaseURL + "/oauth/token", // Don't log credentials in URL
	})
	req, err := http.NewRequest("POST", tokenURL, nil)
	if err != nil {
		return err
	}

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
		if err == nil && len(errRes.Messages) > 0 {
			return errors.New(errRes.Messages[0].Text)
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

	c.accessToken = tokenRes.AccessToken

	// Set expiry with a safety margin and better validation
	if tokenRes.ExpiresIn > 0 {
		expirationDuration := time.Duration(tokenRes.ExpiresIn) * time.Second
		// Subtract 5 minutes as safety margin to refresh before actual expiry
		safetyMargin := 5 * time.Minute
		if expirationDuration > safetyMargin {
			expirationDuration -= safetyMargin
		}
		c.tokenExpiry = time.Now().Add(expirationDuration)

		tflog.Debug(ctx, "Token expiry set", map[string]interface{}{
			"expires_in_seconds": tokenRes.ExpiresIn,
			"token_expiry":       c.tokenExpiry.Format(time.RFC3339),
		})
	} else {
		// Fallback: set a default expiry of 1 hour if expires_in is missing or invalid
		c.tokenExpiry = time.Now().Add(1 * time.Hour)
		tflog.Warn(ctx, "expires_in field missing or invalid, using default 1 hour expiry", map[string]interface{}{
			"expires_in_received": tokenRes.ExpiresIn,
			"default_expiry":      c.tokenExpiry.Format(time.RFC3339),
		})
	}

	return nil
}

func (c *Client) GetSourceByName(ctx context.Context, name string) (*Source, error) {
	filter := fmt.Sprintf("name eq \"%s\"", name)
	sourceURL := fmt.Sprintf("%s/v2025/sources?filters=%s", c.BaseURL, url.QueryEscape(filter))
	tflog.Debug(ctx, "Creating HTTP request to get source", map[string]interface{}{
		"method":    "GET",
		"url":       sourceURL,
		"source_name": name,
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

func (c *Client) GetSource(ctx context.Context, id string) (*Source, error) {
	sourceURL := fmt.Sprintf("%s/beta/sources/%s", c.BaseURL, id)
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
	sourceURL := fmt.Sprintf("%s/beta/sources", c.BaseURL)
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
	tflog.Debug(ctx, "UpdateSource request body", map[string]interface{}{
		"body": string(body),
	})

	// Create the HTTP PATCH request
	patchURL := fmt.Sprintf("%s/v3/sources/%s", c.BaseURL, source.ID)
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

	resBody, _ := json.MarshalIndent(res, "", "  ")
	tflog.Debug(ctx, "Response body received", map[string]interface{}{
		"response": string(resBody),
	})

	return &res, nil
}

func (c *Client) CreateSource(ctx context.Context, source *Source) (*Source, error) {
	var res *Source

	if source.Connector == "Microsoft-Entra" {
		newSource := *source
		newSource.ConnectorAttributes = nil

		// Create source request
		sourceResponse, err := c.CreateSourceRequest(ctx, &newSource)
		if err != nil {
			return nil, err
		}
		source.ID = sourceResponse.ID
		// Add connector attributes
		res, err = c.AddConnectorAttributesToMicrosoftEntraSource(ctx, source)
		if err != nil {
			return nil, err
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

func (c *Client) UpdateSource(ctx context.Context, source *Source) (*Source, error) {
	body, err := json.Marshal(&source)
	if err != nil {
		return nil, err
	}
	updateURL := fmt.Sprintf("%s/beta/sources/%s", c.BaseURL, source.ID)
	tflog.Debug(ctx, "Creating HTTP request to update source", map[string]interface{}{
		"method":    "PUT",
		"url":       updateURL,
		"source_id": source.ID,
	})
	req, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(body))
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
		tflog.Error(ctx, "Failed source update", map[string]interface{}{
			"error":    err.Error(),
			"response": fmt.Sprintf("%+v", res),
		})
		return nil, err
	}

	return &res, nil
}

func (c *Client) DeleteSource(ctx context.Context, source *Source) error {
	deleteURL := fmt.Sprintf("%s/beta/sources/%s", c.BaseURL, source.ID)
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

func (c *Client) GetAccessProfile(ctx context.Context, id string) (*AccessProfile, error) {
	profileURL := fmt.Sprintf("%s/v3/access-profiles/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to get access profile", map[string]interface{}{
		"method":     "GET",
		"url":        profileURL,
		"profile_id": id,
	})
	maxRetries := 3
	retryDelay := 3 * time.Second
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", profileURL, nil)
		if err != nil {
			tflog.Error(ctx, "Failed to create HTTP request for access profile", map[string]interface{}{
				"profile_id": id,
				"error":      err.Error(),
			})
			return nil, err
		}

		req = req.WithContext(ctx)

		res := AccessProfile{}
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Failed to get access profile", map[string]interface{}{
				"profile_id": id,
				"error":      err.Error(),
			})
			if ((attempt < maxRetries) &&
			    (err.Error() == "rate limit exceeded (429)" || err.Error() == "Gateway Timeout error (504)")) {
				backoffDelay := time.Duration(attempt) * retryDelay
				time.Sleep(backoffDelay)
				continue
			}
			return nil, err
		}
		tflog.Debug(ctx, "Successfully retrieved access profile", map[string]interface{}{
			"profile_id": id,
		})

		return &res, nil
	}
	return nil, errors.New("dead code")
}

func (c *Client) GetSourceEntitlements(ctx context.Context, id string) ([]*SourceEntitlement, error) {
	entitlementsURL := fmt.Sprintf("%s/beta/entitlements?filters=source.id", c.BaseURL) + url.QueryEscape(" eq ") + fmt.Sprintf("\"%s\"", id)
	tflog.Debug(ctx, "Creating HTTP request to get source entitlements", map[string]interface{}{
		"method":    "GET",
		"url":       entitlementsURL,
		"source_id": id,
	})
	req, err := http.NewRequest("GET", entitlementsURL, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	req = req.WithContext(ctx)

	var res []*SourceEntitlement
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}
	return res, nil
}

func (c *Client) GetSourceEntitlement(ctx context.Context, id string, nameFilter string) ([]*SourceEntitlement, error) {
	filter := fmt.Sprintf("source.id eq \"%s\" and (name eq \"%s\")", id, nameFilter)
	entitlementURL := fmt.Sprintf("%s/v2024/entitlements?filters=%s", c.BaseURL, url.QueryEscape(filter))
	tflog.Debug(ctx, "Creating HTTP request to get source entitlement", map[string]interface{}{
		"method":      "GET",
		"url":         entitlementURL,
		"source_id":   id,
		"name_filter": nameFilter,
	})

	maxRetries := 3
	retryDelay := 3 * time.Second
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", entitlementURL, nil)
		if err != nil {
			tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error(), "attempt": attempt})
			return nil, err
		}

		req = req.WithContext(ctx)

		var res []*SourceEntitlement
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
			if ((attempt < maxRetries) &&
			    (err.Error() == "rate limit exceeded (429)" || err.Error() == "Gateway Timeout error (504)")) {
				backoffDelay := time.Duration(attempt) * retryDelay
				time.Sleep(backoffDelay)
				continue
			}
			return nil, err
		}
		return res, nil
	}
	return nil, errors.New("dead code")
}

func (c *Client) CreateAccessProfile(ctx context.Context, accessProfile *AccessProfile) (*AccessProfile, error) {
	body, err := json.Marshal(&accessProfile)
	if err != nil {
		return nil, err
	}

	createURL := fmt.Sprintf("%s/v3/access-profiles", c.BaseURL)
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
	updateURL := fmt.Sprintf("%s/v3/access-profiles/%s", c.BaseURL, id)
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
	deleteURL := fmt.Sprintf("%s/v3/access-profiles/%s", c.BaseURL, accessProfile.ID)
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
	roleURL := fmt.Sprintf("%s/v3/roles/%s", c.BaseURL, id)
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

	createURL := fmt.Sprintf("%s/v3/roles", c.BaseURL)
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
	updateURL := fmt.Sprintf("%s/v3/roles/%s", c.BaseURL, id)
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
	deleteURL := fmt.Sprintf("%s/v3/role/%s", c.BaseURL, role.ID)
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
	identityURL := fmt.Sprintf("%s/v2024/identities?filters=alias", c.BaseURL) + url.QueryEscape(" eq ") + fmt.Sprintf("\"%s\"", alias)
	tflog.Debug(ctx, "Creating HTTP request to get identity by alias", map[string]interface{}{
		"method": "GET",
		"url":    identityURL,
		"alias":  alias,
	})
	req, err := http.NewRequest("GET", identityURL, nil)

	if err != nil {
		tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
		return nil, err
	}
	tflog.Debug(ctx, "GetIdentity request details", map[string]interface{}{"request": fmt.Sprintf("%+v", req)})

	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	var res []*Identity
	if err := c.sendRequest(ctx, req, &res); err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return nil, err
	}

	tflog.Debug(ctx, "GetIdentity response details", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})

	return res, nil
}

func (c *Client) GetIdentityByEmail(ctx context.Context, email string) ([]*Identity, error) {
	identityURL := fmt.Sprintf("%s/v2024/identities?filters=email", c.BaseURL) + url.QueryEscape(" eq ") + fmt.Sprintf("\"%s\"", email)
	tflog.Debug(ctx, "Creating HTTP request to get identity by email", map[string]interface{}{
		"method": "GET",
		"url":    identityURL,
		"email":  email,
	})

	maxRetries := 3
	retryDelay := 3 * time.Second
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", identityURL, nil)
		if err != nil {
			tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
			return nil, err
		}
		tflog.Debug(ctx, "GetIdentity request details", map[string]interface{}{"request": fmt.Sprintf("%+v", req)})

		req.Header.Set("Accept", "application/json; charset=utf-8")

		req = req.WithContext(ctx)

		var res []*Identity
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
			if ((attempt < maxRetries) &&
			    (err.Error() == "rate limit exceeded (429)" || err.Error() == "Gateway Timeout error (504)")) {
				backoffDelay := time.Duration(attempt) * retryDelay
				time.Sleep(backoffDelay)
				continue
			}
			return nil, err
		}
		tflog.Debug(ctx, "GetIdentity response details", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})

		return res, nil
	}
	return nil, errors.New("dead code")
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

	return &res[0], nil
}

func (c *Client) ManageAccountAggregationSchedule(ctx context.Context, scheduleAggregation *AccountAggregationSchedule, enable bool) (*AccountAggregationSchedule, error) {
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
	schemaURL := fmt.Sprintf("%s/v3/sources/%s/schemas/%s", c.BaseURL, sourceId, id)
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

//func (c *Client) CreateAccountSchema(ctx context.Context, accountSchema *AccountSchema) (*AccountSchema, error) {
//for _, value := range updateAccountSchema {
//	log.Printf("arrBody: %+v, value: %+v", value, value.Value)
//}
//log.Printf("arrBody type: %+v", reflect.TypeOf(updateAccountSchema))
//body, err := json.Marshal(&updateAccountSchema)
//log.Printf("body: %+v", string(body))
//
//if err != nil {
//	return nil, err
//}
//req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/v3/sources/%s/schemas/%s", c.BaseURL, sourceId, schemaId), bytes.NewBuffer(body))
//if err != nil {
//	tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
//	return nil, err
//}
//
//req.Header.Set("Content-Type", "application/json-patch+json; charset=utf-8")
//req.Header.Set("Accept", "application/json; charset=utf-8")
//
//req = req.WithContext(ctx)
//res := AccountSchema{}
//if err := c.sendRequest(ctx, req, &res); err != nil {
//	log.Printf("get body: %+v\n", req.GetBody)
//
//	tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
//	// Error already logged above
//	return nil, err
//}
//for _, value := range updateAccountSchema {
//	log.Printf("arrBody: %+v, value: %+v", value, value.Value)
//}
//return &res, nil
//}

func (c *Client) UpdateAccountSchema(ctx context.Context, accountSchema *AccountSchema) (*AccountSchema, error) {
	body, err := json.Marshal(&accountSchema)
	if err != nil {
		return nil, err
	}
	schemaURL := fmt.Sprintf("%s/v3/sources/%s/schemas/%s", c.BaseURL, accountSchema.SourceID, accountSchema.ID)
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

func (c *Client) DeleteAccountSchema(ctx context.Context, accountSchema *AccountSchema) error {
	endpoint := fmt.Sprintf("%s/v3/sources/%s/schemas/%s", c.BaseURL, accountSchema.SourceID, accountSchema.ID)

	client := &http.Client{}

	tflog.Debug(ctx, "Creating HTTP request to delete account schema", map[string]interface{}{
		"method":    "DELETE",
		"url":       endpoint,
		"source_id": accountSchema.SourceID,
		"schema_id": accountSchema.ID,
	})
	req, err := http.NewRequest("DELETE", endpoint, nil)

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)
	res, err := client.Do(req)

	if err != nil {
		tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
		// Error already logged above
		return err
	}

	return nil
}

func (c *Client) CreatePasswordPolicy(ctx context.Context, passwordPolicy *PasswordPolicy) (*PasswordPolicy, error) {
	body, err := json.Marshal(&passwordPolicy)
	if err != nil {
		return nil, err
	}
	policyURL := fmt.Sprintf("%s/beta/password-policies", c.BaseURL)
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
	policyURL := fmt.Sprintf("%s/beta/password-policies", c.BaseURL)
	tflog.Debug(ctx, "Creating HTTP request to update password policy", map[string]interface{}{
		"method": "PUT",
		"url":    policyURL,
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
	policyURL := fmt.Sprintf("%s/beta/password-policies/%s", c.BaseURL, passwordPolicyId)
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
	endpoint := fmt.Sprintf("%s/beta/password-policies/%s", c.BaseURL, passwordPolicyId)

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

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		tflog.Error(ctx, "HTTP client operation failed", map[string]interface{}{"error": err.Error()})
		return err
	}

	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		var errRes errorResponse
		err = json.NewDecoder(res.Body).Decode(&errRes)
		if err == nil {
			if res.StatusCode == http.StatusNotFound {
				return &NotFoundError{errRes.Messages[0].Text}
			} else {
				return errors.New(errRes.Messages[0].Text)
			}
		}

		return fmt.Errorf("unknown error, status code: %d", res.StatusCode)
	}

	return nil
}

func (c *Client) CreateGovernanceGroup(ctx context.Context, governanceGroup *GovernanceGroup) (*GovernanceGroup, error) {
	body, err := json.Marshal(&governanceGroup)
	if err != nil {
		return nil, err
	}

	workgroupURL := fmt.Sprintf("%s/v2024/workgroups", c.BaseURL)
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

func (c *Client) GetGovernanceGroup(ctx context.Context, id string) (*GovernanceGroup, error) {
	workgroupURL := fmt.Sprintf("%s/v2024/workgroups/%s", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to get governance group", map[string]interface{}{
		"method":   "GET",
		"url":      workgroupURL,
		"group_id": id,
	})
	maxRetries := 3
	retryDelay := 3 * time.Second
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", workgroupURL, nil)
		if err != nil {
			tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
			return nil, err
		}

		req.Header.Set("Accept", "application/json; charset=utf-8")
		req.Header.Set("X-SailPoint-Experimental", "true")

		req = req.WithContext(ctx)

		res := GovernanceGroup{}
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
			if ((attempt < maxRetries) &&
			    (err.Error() == "rate limit exceeded (429)" || err.Error() == "Gateway Timeout error (504)")) {
				backoffDelay := time.Duration(attempt) * retryDelay
				time.Sleep(backoffDelay)
				continue
			}
			return nil, err
		}
		tflog.Debug(ctx, "GetGovernanceGroup response details", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})

		return &res, nil
	}
	return nil, errors.New("dead code")
}

func (c *Client) UpdateGovernanceGroup(ctx context.Context, governanceGroup []*UpdateGovernanceGroup, id interface{}) (*GovernanceGroup, error) {
	body, err := json.Marshal(&governanceGroup)
	if err != nil {
		return nil, err
	}
	updateURL := fmt.Sprintf("%s/v2024/workgroups/%s", c.BaseURL, id)
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
	deleteURL := fmt.Sprintf("%s/v2024/workgroups/%s", c.BaseURL, governanceGroup.ID)
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

func (c *Client) GetSourceAppByName(ctx context.Context, name string) ([]*SourceApp, error) {
	filter := fmt.Sprintf("name eq \"%s\"", name)
	sourceAppURL := fmt.Sprintf("%s/v2025/source-apps?filters=%s", c.BaseURL, url.QueryEscape(filter))
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
	sourceAppURL := fmt.Sprintf("%s/v2025/source-apps/%s", c.BaseURL, id)
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

	sourceAppURL := fmt.Sprintf("%s/v2025/source-apps", c.BaseURL)
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
	updateURL := fmt.Sprintf("%s/v2025/source-apps/%s", c.BaseURL, id)
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
	deleteURL := fmt.Sprintf("%s/v2025/source-apps/%s", c.BaseURL, sourceApp.ID)
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
		url := fmt.Sprintf("%s/v2025/source-apps/%s/access-profiles?limit=%d&offset=%d", c.BaseURL, id, limit, offset)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
			return nil, err
		}

		req.Header.Set("X-SailPoint-Experimental", "true")

		req = req.WithContext(ctx)

		var res []AccessProfileFromSourceApp
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
			// Error already logged above
			return nil, err
		}

		for _, ap := range res {
			accessProfiles = append(accessProfiles, ap.ID)
		}

		if len(res) < limit-1 {
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
	updateURL := fmt.Sprintf("%s/v2025/source-apps/%s", c.BaseURL, id)
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

	deleteURL := fmt.Sprintf("%s/v2025/source-apps/%s/access-profiles/bulk-remove", c.BaseURL, accessProfileAttachment.SourceAppId)
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
	createURL := fmt.Sprintf("%s/v2025/workgroups/%s/members/bulk-add", c.BaseURL, id)
	tflog.Debug(ctx, "Creating HTTP request to create governance group members", map[string]interface{}{
		"method": "POST",
		"url":    createURL,
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
		url := fmt.Sprintf("%s/v2025/workgroups/%s/members?limit=%d&offset=%d", c.BaseURL, id, limit, offset)
		tflog.Debug(ctx, "Creating HTTP request to get governance group members", map[string]interface{}{
			"method": "GET",
			"url":    url,
			"governance_group_id": id,
		})
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			tflog.Error(ctx, "Failed to create new HTTP request", map[string]interface{}{"error": err.Error()})
			return nil, err
		}

		req.Header.Set("X-SailPoint-Experimental", "true")

		req = req.WithContext(ctx)

		res := []GovernanceGroupMembersMembers{}
		if err := c.sendRequest(ctx, req, &res); err != nil {
			tflog.Error(ctx, "Request failed", map[string]interface{}{"response": fmt.Sprintf("%+v", res)})
			// Error already logged above
			return nil, err
		}

		for _, govgmem := range res {
			governanceGroupMembersMembers = append(governanceGroupMembersMembers, &govgmem)
		}

		if len(res) < limit-1 {
			break
		}

		offset += limit
	}

	governanceGroupMembers := GovernanceGroupMembers{
		GovernanceGroupId:    id,
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
			GovernanceGroupId:    id,
			GovernanceGroupMembersMembers: onlyInActual,
		}

		body, err := json.Marshal(membersOnlyInActual.GovernanceGroupMembersMembers)
		if err != nil {
			return nil, err
		}

		deleteURL := fmt.Sprintf("%s/v2025/workgroups/%s/members/bulk-delete", c.BaseURL, governanceGroupMembers.GovernanceGroupId)
		tflog.Debug(ctx, "Creating HTTP request to update governance group members", map[string]interface{}{
			"method":        "POST",
			"url":           deleteURL,
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
			GovernanceGroupId:    id,
			GovernanceGroupMembersMembers: onlyInDesired,
		}

		body, err := json.Marshal(membersOnlyInDesired.GovernanceGroupMembersMembers)
		if err != nil {
			return nil, err
		}
		createURL := fmt.Sprintf("%s/v2025/workgroups/%s/members/bulk-add", c.BaseURL, id)
		tflog.Debug(ctx, "Creating HTTP request to update governance group members", map[string]interface{}{
			"method": "POST",
			"url":    createURL,
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

	return governanceGroupMembers,nil 
	}

func (c *Client) DeleteGovernanceGroupMembers(ctx context.Context, governanceGroupMembers *GovernanceGroupMembers) error {
	body, err := json.Marshal(governanceGroupMembers.GovernanceGroupMembersMembers)
	if err != nil {
		return err
	}

	deleteURL := fmt.Sprintf("%s/v2025/workgroups/%s/members/bulk-delete", c.BaseURL, governanceGroupMembers.GovernanceGroupId)
	tflog.Debug(ctx, "Creating HTTP request to delete governance group members", map[string]interface{}{
		"method":        "POST",
		"url":           deleteURL,
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

	allSuccessful := true
	for _, member := range res {
		if member.Status != 204 {
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

func (c *Client) sendRequest(ctx context.Context, req *http.Request, v interface{}) error {
	// Apply rate limiting before making any API requests
	tflog.Trace(ctx, "Before rate limiter", map[string]interface{}{
		"url":           req.URL.String(),
	})
	if err := c.rateLimiter.Wait(ctx); err != nil {
		tflog.Debug(ctx, "Rate limiting wait failed", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("rate limiting failed: %w", err)
	}

	tflog.Trace(ctx, "Sending HTTP Request", map[string]interface{}{
		"method":        req.Method,
		"url":           req.URL.String(),
		"headers":       req.Header,
		"request_body":  req.Body,
	})

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		tflog.Error(ctx, "HTTP client operation failed", map[string]interface{}{"error": err.Error()})
		return err
	}

	defer res.Body.Close()

	tflog.Trace(ctx, "Received HTTP Response", map[string]interface{}{
		"status_code":    res.StatusCode,
		"response_headers": res.Header,
	})

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		var errRes errorResponse
		err = json.NewDecoder(res.Body).Decode(&errRes)
		if err == nil {
			if res.StatusCode == http.StatusTooManyRequests {
				tflog.Error(ctx, "API Rate limit exceeded", map[string]interface{}{
					"status_code": res.StatusCode,
				})
				return errors.New("rate limit exceeded (429)")
			}
			if res.StatusCode == http.StatusNotFound {
				// on the return statement, an interface value of type error is created by the compiler and bound to the pointer to satisfy the return argument.
				return &NotFoundError{"status not found"}
			}
			if len(errRes.Messages) == 0 {
				tflog.Error(ctx, "Unknown error occurred", map[string]interface{}{
					"status_code": res.StatusCode,
				})
				return errors.New("unknown error")
			}
			return errors.New(errRes.Messages[0].Text)
		}

		// Handle 429 without error response body
		if res.StatusCode == http.StatusTooManyRequests {
			tflog.Error(ctx, "Rate limit exceeded", map[string]interface{}{
				"status_code": res.StatusCode,
			})
			return errors.New("rate limit exceeded (429)")
		}

		if res.StatusCode == http.StatusGatewayTimeout {
			tflog.Error(ctx, "Gateway Timeout error", map[string]interface{}{
				"status_code": res.StatusCode,
			})
			return errors.New("Gateway Timeout error (504)")
		}

		tflog.Error(ctx, "Unknown error occurred", map[string]interface{}{
			"status_code": res.StatusCode,
		})
		return errors.New("unknown error")
	}

	if res.StatusCode == 204 && req.Method == "DELETE" {
		tflog.Debug(ctx, "Resource deleted successfully")
		return nil
	}

	if err = json.NewDecoder(res.Body).Decode(&v); err != nil {
		tflog.Error(ctx, "JSON decoder error", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	tflog.Trace(ctx, "Parsed HTTP Response", map[string]interface{}{
		"response_body":  v,
	})
	return nil
}
