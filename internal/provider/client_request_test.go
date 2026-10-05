package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func withFastRetries(t *testing.T) {
	t.Helper()
	previous := retryBaseDelay
	retryBaseDelay = time.Millisecond
	t.Cleanup(func() { retryBaseDelay = previous })
}

func TestSendRequestRetriesRateLimitedRequests(t *testing.T) {
	withFastRetries(t)
	var calls int32
	var bodies []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		if atomic.AddInt32(&calls, 1) < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"id":"form-1","name":"form"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)

	form, err := client.CreateFormDefinition(context.Background(), &FormDefinition{Name: "form"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if form.ID != "form-1" || calls != 3 {
		t.Fatalf("expected success after 3 calls, got %d calls (%+v)", calls, form)
	}
	for _, body := range bodies {
		if body != `{"name":"form"}` {
			t.Fatalf("expected the request body to be resent on retry, got %q", body)
		}
	}
}

func TestSendRequestRetriesRequestsWithoutBody(t *testing.T) {
	withFastRetries(t)
	var calls int32
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"id":"form-1"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)

	if _, err := client.GetFormDefinition(context.Background(), "form-1"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestSendRequestRetriesOnlyIdempotentRequestsOnGatewayTimeout(t *testing.T) {
	withFastRetries(t)
	var calls int32
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte(`upstream timeout`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)

	_, err := client.CreateFormDefinition(context.Background(), &FormDefinition{Name: "form"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusGatewayTimeout || apiErr.Message != "upstream timeout" {
		t.Fatalf("expected API error with status 504, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected a single POST, got %d", calls)
	}

	// PATCH requests are not retried either.
	calls = 0
	if _, err := client.UpdateFormDefinition(context.Background(), "form-1", nil); err == nil || calls != 1 {
		t.Fatalf("expected a single PATCH, got %d (%v)", calls, err)
	}

	// GET requests are retried on 504.
	calls = 0
	_, err = client.GetFormDefinition(context.Background(), "form-1")
	if err == nil || calls != maxRequestAttempts {
		t.Fatalf("expected %d attempts, got %d (%v)", maxRequestAttempts, calls, err)
	}
}

func TestSendRequestRefreshesTokenOnUnauthorized(t *testing.T) {
	var tokenCalls, apiCalls, alwaysUnauthorized int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			n := atomic.AddInt32(&tokenCalls, 1)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"access_token":"token-%d","expires_in":720}`, n)))
			return
		}
		if atomic.AddInt32(&apiCalls, 1) == 1 || atomic.LoadInt32(&alwaysUnauthorized) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"JWT validation failed"}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer token-2" {
			t.Errorf("expected refreshed token, got %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"id":"form-1"}`))
	}))
	defer server.Close()
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)

	if _, err := client.GetFormDefinition(context.Background(), "form-1"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if tokenCalls != 2 || apiCalls != 2 {
		t.Fatalf("expected 2 token and 2 API calls, got %d and %d", tokenCalls, apiCalls)
	}

	// A second 401 in a row is returned as an error with the OAuth error message.
	atomic.StoreInt32(&alwaysUnauthorized, 1)
	_, err := client.GetFormDefinition(context.Background(), "form-1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized || !strings.Contains(err.Error(), "JWT validation failed") {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestSendRequestNotFoundWithoutJSONBody(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<html>not found</html>`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	if _, err := client.GetFormDefinition(context.Background(), "form-1"); !isNotFound(err) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestSendRequestAcceptsEmptySuccessBody(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	err := client.DeleteAccessProfileAttachment(context.Background(), &AccessProfileAttachment{SourceAppId: "app-1", AccessProfiles: []string{"ap-1"}})
	if err != nil {
		t.Fatalf("unexpected error for empty 200 body: %s", err)
	}
}

func TestIdentityLookupEscapesFilter(t *testing.T) {
	var gotFilter string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotFilter = r.URL.Query().Get("filters")
		_, _ = w.Write([]byte(`[]`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	if _, err := client.GetIdentityByEmail(context.Background(), `john+test@example.com`); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotFilter != `email eq "john+test@example.com"` {
		t.Fatalf("unexpected filter %q", gotFilter)
	}
	if got := eqFilter("name", `a "quoted" \ name`); got != `name eq "a \"quoted\" \\ name"` {
		t.Fatalf("unexpected escaped filter %q", got)
	}
}

func TestGetSegmentsPagesAndRejectsDuplicates(t *testing.T) {
	var offsets []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		offsets = append(offsets, offset)
		if offset == "0" {
			segments := make([]string, 250)
			for i := range segments {
				segments[i] = `{"id":"s","name":"other"}`
			}
			_, _ = w.Write([]byte("[" + strings.Join(segments, ",") + "]"))
			return
		}
		_, _ = w.Write([]byte(`[{"id":"s1","name":"target"},{"id":"s2","name":"dup"},{"id":"s3","name":"dup"}]`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)

	segment, err := client.GetSegmentByName(context.Background(), "target")
	if err != nil || segment.ID != "s1" {
		t.Fatalf("expected segment from second page, got %+v (%v)", segment, err)
	}
	if strings.Join(offsets, ",") != "0,250" {
		t.Fatalf("unexpected offsets %v", offsets)
	}
	if _, err := client.GetSegmentByName(context.Background(), "dup"); err == nil || isNotFound(err) {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
}

func TestSendRequestRetriedDeleteNotFoundIsSuccess(t *testing.T) {
	withFastRetries(t)
	var calls int32
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	if err := client.DeleteFormDefinition(context.Background(), "form-1"); err != nil {
		t.Fatalf("expected retried delete to succeed, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}
