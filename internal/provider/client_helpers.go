package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// requestOption customizes a request built by doJSON.
type requestOption func(*http.Request)

// withExperimental sets the header required by experimental API endpoints.
func withExperimental() requestOption {
	return func(req *http.Request) {
		req.Header.Set("X-SailPoint-Experimental", "true")
	}
}

// withJSONPatch sends the body with the JSON Patch content type.
func withJSONPatch() requestOption {
	return func(req *http.Request) {
		req.Header.Set("Content-Type", "application/json-patch+json")
	}
}

// jsonPatchOp is a single JSON Patch operation. Value is omitted for remove operations.
type jsonPatchOp struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// MarshalJSON omits the value of remove operations. Other operations always send a value, also
// when it is null, since add and replace require one.
func (o jsonPatchOp) MarshalJSON() ([]byte, error) {
	if o.Op == "remove" {
		return json.Marshal(struct {
			Op   string `json:"op"`
			Path string `json:"path"`
		}{o.Op, o.Path})
	}
	type plain jsonPatchOp
	return json.Marshal(plain(o))
}

// apiPath formats an API path, escaping each segment argument.
func apiPath(format string, segments ...string) string {
	args := make([]interface{}, len(segments))
	for i, segment := range segments {
		args[i] = url.PathEscape(segment)
	}
	return fmt.Sprintf(format, args...)
}

// doJSON sends a request to path, relative to the API base URL, with body encoded as JSON, and
// decodes the JSON response into out. body and out may be nil.
func (c *Client) doJSON(ctx context.Context, method, path string, body, out interface{}, opts ...requestOption) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body for %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, opt := range opts {
		opt(req)
	}
	tflog.Debug(ctx, "Sending API request", map[string]interface{}{"method": method, "url": req.URL.String()})
	return c.sendRequest(ctx, req, out)
}

// listPageSize is the page size used by listAllPages, the maximum most list endpoints accept.
const listPageSize = 250

// listAllPages fetches all pages of a list endpoint that supports limit and offset.
func listAllPages[T any](ctx context.Context, c *Client, path string, query url.Values, opts ...requestOption) ([]T, error) {
	return listAllPagesSized[T](ctx, c, path, query, listPageSize, opts...)
}

// listAllPagesSized is listAllPages for endpoints that accept a smaller maximum page size.
func listAllPagesSized[T any](ctx context.Context, c *Client, path string, query url.Values, pageSize int, opts ...requestOption) ([]T, error) {
	var items []T
	for offset := 0; ; offset += pageSize {
		pageQuery := url.Values{}
		for k, v := range query {
			pageQuery[k] = v
		}
		pageQuery.Set("limit", fmt.Sprintf("%d", pageSize))
		pageQuery.Set("offset", fmt.Sprintf("%d", offset))
		var page []T
		if err := c.doJSON(ctx, http.MethodGet, path+"?"+pageQuery.Encode(), nil, &page, opts...); err != nil {
			return nil, err
		}
		items = append(items, page...)
		if len(page) < pageSize {
			return items, nil
		}
	}
}

// findByName lists path with a name filter and returns the single item whose name matches exactly.
// It returns a NotFoundError when nothing matches and an error when several items match.
func findByName[T any](ctx context.Context, c *Client, path, kind, name string, nameOf func(T) string, opts ...requestOption) (*T, error) {
	items, err := listAllPages[T](ctx, c, path, url.Values{"filters": {eqFilter("name", name)}}, opts...)
	if err != nil {
		return nil, err
	}
	var match *T
	for i := range items {
		if nameOf(items[i]) != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple %s objects are named %q", kind, name)
		}
		match = &items[i]
	}
	if match == nil {
		return nil, &NotFoundError{fmt.Sprintf("%s with name %q not found", kind, name)}
	}
	return match, nil
}

// putMerged reads the object at path, sets the managed fields and sends it back with PUT.
// Fields that are not managed by the provider are kept, so the full replacement semantics of PUT
// do not clear configuration made outside Terraform. The response is decoded into out.
func (c *Client) putMerged(ctx context.Context, path string, managed map[string]interface{}, out interface{}, opts ...requestOption) error {
	current := map[string]interface{}{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &current, opts...); err != nil {
		return err
	}
	for key, value := range managed {
		current[key] = value
	}
	return c.doJSON(ctx, http.MethodPut, path, current, out, opts...)
}
