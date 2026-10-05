package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// newTestServer returns a server that answers OAuth token requests and passes other requests to handler.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token-1","token_type":"bearer","expires_in":720}`))
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestGetTokenSendsCredentialsInBody(t *testing.T) {
	var gotQuery, gotSecret, gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotContentType = r.Header.Get("Content-Type")
		_ = r.ParseForm()
		gotSecret = r.PostForm.Get("client_secret")
		_, _ = w.Write([]byte(`{"access_token":"token-1","expires_in":720}`))
	}))
	defer server.Close()

	client := NewClient(context.Background(), server.URL, "id", "s3cr3t+&#%", 10)
	if err := client.GetToken(context.Background()); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotQuery != "" {
		t.Errorf("expected no query string, got %q", gotQuery)
	}
	if gotSecret != "s3cr3t+&#%" {
		t.Errorf("expected secret to survive form encoding, got %q", gotSecret)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("unexpected content type %q", gotContentType)
	}
	if client.token() != "token-1" || !client.isTokenValid() {
		t.Errorf("expected valid token after GetToken")
	}
}

func TestGetTokenErrorDoesNotLeakSecret(t *testing.T) {
	client := NewClient(context.Background(), "http://127.0.0.1:1", "id", "very-secret", 10)
	err := client.GetToken(context.Background())
	if err == nil {
		t.Fatal("expected connection error")
	}
	if strings.Contains(err.Error(), "very-secret") {
		t.Fatalf("error leaks client secret: %s", err)
	}
}

func TestIdentityNowClientGuardsPoolSize(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	cfg := &Config{
		URL:                    server.URL,
		Credentials:            []ClientCredential{{ClientId: "id", ClientSecret: "secret"}},
		MaxClientPoolSize:      0,
		DefaultClientPoolSize:  0,
		ClientRequestRateLimit: 0,
	}
	client, err := cfg.IdentityNowClient(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if client.token() != "token-1" {
		t.Fatalf("expected token, got %q", client.token())
	}
}

func TestClientTokenIsSafeForConcurrentUse(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	cfg := &Config{
		URL:                    server.URL,
		Credentials:            []ClientCredential{{ClientId: "id", ClientSecret: "secret"}},
		MaxClientPoolSize:      1,
		DefaultClientPoolSize:  1,
		ClientRequestRateLimit: 1000,
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := cfg.IdentityNowClient(context.Background())
			if err != nil {
				t.Errorf("unexpected error: %s", err)
				return
			}
			client.invalidateToken()
			if err := client.DeleteWorkflow(context.Background(), "wf-1"); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		}()
	}
	wg.Wait()
}

func TestGetAccountAggregationScheduleEmptyIsNotFound(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 100)
	_, err := client.GetAccountAggregationSchedule(context.Background(), "source-1")
	if !isNotFound(err) {
		t.Fatalf("expected not found error, got %v", err)
	}
	if _, err := client.ManageAccountAggregationSchedule(context.Background(), &AccountAggregationSchedule{SourceID: "source-1"}, true); err == nil {
		t.Fatal("expected error for missing cron expression")
	}
}

func TestPasswordPolicyRequests(t *testing.T) {
	var gotMethod, gotPath string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"detailCode":"404 Not found"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"policy-1","name":"Policy"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 100)

	if _, err := client.UpdatePasswordPolicy(context.Background(), &PasswordPolicy{ID: "policy-1", Name: "Policy"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != "PUT" || gotPath != "/v2026/password-policies/policy-1" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}

	// A 404 body without messages used to panic.
	if err := client.DeletePasswordPolicy(context.Background(), "policy-1"); !isNotFound(err) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestUpdateSourceSendsJSONPatch(t *testing.T) {
	var gotMethod, gotPath, gotContentType, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotContentType, gotBody = r.Method, r.URL.Path, r.Header.Get("Content-Type"), string(body)
		_, _ = w.Write([]byte(`{"id":"source-1","name":"Source"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 100)
	_, err := client.UpdateSource(context.Background(), "source-1", []*UpdateSource{{Op: "replace", Path: "/description", Value: "new"}})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != "PATCH" || gotPath != "/v2026/sources/source-1" || !strings.HasPrefix(gotContentType, "application/json-patch+json") {
		t.Fatalf("unexpected request %s %s %s", gotMethod, gotPath, gotContentType)
	}
	if gotBody != `[{"op":"replace","path":"/description","value":"new"}]` {
		t.Fatalf("unexpected body %s", gotBody)
	}
}

func TestCreateMicrosoftEntraSourceWithoutConnectorAttributes(t *testing.T) {
	var calls []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method)
		_, _ = w.Write([]byte(`{"id":"source-1","name":"Entra","connector":"Microsoft-Entra"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 100)
	source, err := client.CreateSource(context.Background(), &Source{Name: "Entra", Connector: "Microsoft-Entra"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if source.ID != "source-1" || len(calls) != 1 || calls[0] != "POST" {
		t.Fatalf("expected a single POST, got %v (%+v)", calls, source)
	}
}

func TestSetWorkflowEnabled(t *testing.T) {
	var gotMethod, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotBody = r.Method, string(body)
		_, _ = w.Write([]byte(`{"id":"wf-1","enabled":true}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 100)
	workflow, err := client.SetWorkflowEnabled(context.Background(), "wf-1", true)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != "PATCH" || gotBody != `[{"op":"replace","path":"/enabled","value":true}]` || !*workflow.Enabled {
		t.Fatalf("unexpected request %s %s", gotMethod, gotBody)
	}
}
