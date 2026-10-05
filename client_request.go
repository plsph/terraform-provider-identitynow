package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// maxRequestAttempts is the number of attempts for requests that fail with a retryable status.
	maxRequestAttempts = 4
	// maxRetryAfter caps the delay requested by a Retry-After header.
	maxRetryAfter = time.Minute
	// maxErrorBodyLength caps the raw response body included in error messages.
	maxErrorBodyLength = 500
)

// retryBaseDelay is the delay before the first retry, it grows linearly with each attempt.
var retryBaseDelay = 3 * time.Second

// APIError is an error response from the IdentityNow API.
type APIError struct {
	StatusCode int
	DetailCode string
	Message    string
	Method     string
	URL        string
}

func (e *APIError) Error() string {
	message := e.Message
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if e.DetailCode != "" {
		return fmt.Sprintf("%s %s: %s (HTTP %d, %s)", e.Method, e.URL, message, e.StatusCode, e.DetailCode)
	}
	return fmt.Sprintf("%s %s: %s (HTTP %d)", e.Method, e.URL, message, e.StatusCode)
}

// eqFilter returns an API filter expression that matches field to value, with the value escaped.
func eqFilter(field, value string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
	return fmt.Sprintf(`%s eq "%s"`, field, escaped)
}

// sendRequest sends an authenticated API request and decodes a JSON response body into v.
//
// Requests are rate limited and retried when the API responds with 429, or with 502, 503 or 504
// for GET, PUT and DELETE requests. A 401 response refreshes
// the access token and retries once. A 404 response returns a NotFoundError, other error responses
// return an APIError.
func (c *Client) sendRequest(ctx context.Context, req *http.Request, v interface{}) error {
	req = req.WithContext(ctx)
	tokenRefreshed := false
	retried := false

	for attempt := 1; ; attempt++ {
		if attempt > 1 {
			// httputil.DumpRequestOut replaces a nil body with http.NoBody, which needs no resend.
			if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
				return fmt.Errorf("%s %s: request body cannot be sent again for a retry", req.Method, req.URL.Path)
			}
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return err
				}
				req.Body = body
			}
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiting failed: %w", err)
		}
		if err := c.ensureToken(ctx); err != nil {
			return err
		}
		token := c.token()
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		tflog.Trace(ctx, "Sending HTTP Request", map[string]interface{}{
			"method":  req.Method,
			"url":     req.URL.String(),
			"attempt": attempt,
			"request": dumpRequest(req, token),
		})

		res, err := c.HTTPClient.Do(req)
		if err != nil {
			tflog.Error(ctx, "HTTP client operation failed", map[string]interface{}{"error": err.Error()})
			return err
		}

		tflog.Trace(ctx, "Received HTTP Response", map[string]interface{}{
			"status_code": res.StatusCode,
			"response":    dumpResponse(res),
		})

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return fmt.Errorf("%s %s: reading response body: %w", req.Method, req.URL.Path, err)
		}

		if res.StatusCode == http.StatusUnauthorized && !tokenRefreshed {
			tflog.Debug(ctx, "Access token rejected, refreshing token")
			tokenRefreshed = true
			c.invalidateToken()
			continue
		}

		if attempt < maxRequestAttempts && isRetryableStatus(req.Method, res.StatusCode) {
			delay := retryDelay(res, attempt)
			tflog.Warn(ctx, "Retrying API request", map[string]interface{}{
				"method":      req.Method,
				"url":         req.URL.String(),
				"status_code": res.StatusCode,
				"attempt":     attempt,
				"delay":       delay.String(),
			})
			retried = true
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}

		// A retried DELETE that finds nothing means an earlier attempt already deleted the object.
		if req.Method == http.MethodDelete && res.StatusCode == http.StatusNotFound && retried {
			return nil
		}
		return decodeResponse(req, res.StatusCode, body, v)
	}
}

// dumpRequest returns the full outgoing request for trace logging, with the access token redacted.
// The request body is read and replaced with an identical copy, so the request can still be sent.
func dumpRequest(req *http.Request, token string) string {
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return fmt.Sprintf("<failed to dump request: %s>", err)
	}
	if token == "" {
		return string(dump)
	}
	return strings.ReplaceAll(string(dump), token, "<redacted>")
}

// dumpResponse returns the full response for trace logging. The response body is read and
// replaced with an identical copy, so it can still be read by the caller.
func dumpResponse(res *http.Response) string {
	dump, err := httputil.DumpResponse(res, true)
	if err != nil {
		return fmt.Sprintf("<failed to dump response: %s>", err)
	}
	return string(dump)
}

// isRetryableStatus reports whether a response status is retried. Rate limited requests were not
// processed and are always retried. Gateway errors are only retried for idempotent methods,
// because the request might have been processed already.
func isRetryableStatus(method string, status int) bool {
	switch status {
	case http.StatusTooManyRequests:
		return true
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return method == http.MethodGet || method == http.MethodPut || method == http.MethodDelete
	}
	return false
}

// retryDelay returns the delay requested by a Retry-After header, or a linear backoff.
func retryDelay(res *http.Response, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds >= 0 {
		delay := time.Duration(seconds) * time.Second
		if delay > maxRetryAfter {
			delay = maxRetryAfter
		}
		return delay
	}
	return time.Duration(attempt) * retryBaseDelay
}

// decodeResponse decodes a successful response into v, or converts an error response to an error.
func decodeResponse(req *http.Request, status int, body []byte, v interface{}) error {
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		if v == nil || len(bytes.TrimSpace(body)) == 0 {
			return nil
		}
		if err := json.Unmarshal(body, v); err != nil {
			return fmt.Errorf("%s %s: decoding response: %w", req.Method, req.URL.Path, err)
		}
		return nil
	}

	apiErr := &APIError{StatusCode: status, Method: req.Method, URL: req.URL.Path}
	var errRes errorResponse
	if err := json.Unmarshal(body, &errRes); err == nil {
		apiErr.DetailCode = errRes.DetailCode
		switch {
		case len(errRes.Messages) > 0:
			apiErr.Message = errRes.Messages[0].Text
		case errRes.ErrorDescription != "":
			apiErr.Message = errRes.ErrorDescription
		case errRes.Error != "":
			apiErr.Message = errRes.Error
		}
	}
	if apiErr.Message == "" && apiErr.DetailCode == "" {
		raw := strings.TrimSpace(string(body))
		if len(raw) > maxErrorBodyLength {
			raw = raw[:maxErrorBodyLength] + "..."
		}
		apiErr.Message = raw
	}

	if status == http.StatusNotFound {
		return &NotFoundError{apiErr.Error()}
	}
	return apiErr
}
