package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestSearchAttributeConfigClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType, gotExperimental string
	missing := false
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		gotContentType, gotExperimental = r.Header.Get("Content-Type"), r.Header.Get("X-SailPoint-Experimental")
		switch {
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodDelete || missing:
			w.WriteHeader(http.StatusNoContent)
		default:
			_, _ = w.Write([]byte(`{"name":"newMail","displayName":"New Mail","applicationAttributes":{"src-1":"mail"}}`))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	attributes, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{"src-1": "mail"})
	data := SearchAttributeConfigModel{Name: types.StringValue("newMail"), DisplayName: types.StringValue("New Mail"), ApplicationAttributes: attributes}
	if err := client.CreateSearchAttributeConfig(ctx, searchAttributeConfigFromModel(ctx, data, &diags)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/accounts/search-attribute-config" || gotExperimental != "true" ||
		gotBody != `{"name":"newMail","displayName":"New Mail","applicationAttributes":{"src-1":"mail"}}` {
		t.Fatalf("unexpected create request %s %s %s %s", gotMethod, gotPath, gotExperimental, gotBody)
	}

	config, err := client.GetSearchAttributeConfig(ctx, "newMail")
	if err != nil || gotPath != "/v2026/accounts/search-attribute-config/newMail" || gotExperimental != "true" || config.ApplicationAttributes["src-1"] != "mail" {
		t.Fatalf("unexpected read %s %s %+v (%v)", gotPath, gotExperimental, config, err)
	}

	if _, err := client.UpdateSearchAttributeConfig(ctx, "newMail", []jsonPatchOp{{Op: "replace", Path: "/displayName", Value: "Mail"}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotContentType != "application/json-patch+json" || gotExperimental != "true" {
		t.Fatalf("unexpected update request %s %s %s", gotMethod, gotContentType, gotExperimental)
	}

	if err := client.DeleteSearchAttributeConfig(ctx, "newMail"); err != nil || gotMethod != http.MethodDelete || gotExperimental != "true" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotExperimental, err)
	}

	// A 204 response without a body means the configuration does not exist.
	missing = true
	if _, err := client.GetSearchAttributeConfig(ctx, "newMail"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestSearchAttributeConfigStateAndPatches(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	var data SearchAttributeConfigModel
	setSearchAttributeConfigState(ctx, &data, &SearchAttributeConfig{Name: "newMail", DisplayName: "New Mail", ApplicationAttributes: map[string]string{"src-1": "mail"}}, &diags)
	if diags.HasError() || data.ID.ValueString() != "newMail" || len(data.ApplicationAttributes.Elements()) != 1 {
		t.Fatalf("unexpected state %+v (%v)", data, diags)
	}

	plan := data
	if ops := searchAttributeConfigPatches(ctx, plan, data, &diags); len(ops) != 0 {
		t.Fatalf("expected no patches, got %+v", ops)
	}
	plan.Name = types.StringValue("otherMail")
	plan.ApplicationAttributes, _ = types.MapValueFrom(ctx, types.StringType, map[string]string{"src-1": "mail", "src-2": "email"})
	encoded, _ := json.Marshal(searchAttributeConfigPatches(ctx, plan, data, &diags))
	want := `[{"op":"replace","path":"/name","value":"otherMail"},{"op":"replace","path":"/applicationAttributes","value":{"src-1":"mail","src-2":"email"}}]`
	if string(encoded) != want {
		t.Fatalf("unexpected patches\n%s\nwant\n%s", encoded, want)
	}
}

// searchAttributeConfigTestFastPolling shortens the create polling for a test.
func searchAttributeConfigTestFastPolling(t *testing.T, timeout time.Duration) {
	t.Helper()
	oldTimeout, oldInterval, oldMax := searchAttributeConfigCreateTimeout, searchAttributeConfigPollInterval, searchAttributeConfigMaxPollInterval
	searchAttributeConfigCreateTimeout, searchAttributeConfigPollInterval, searchAttributeConfigMaxPollInterval = timeout, time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() {
		searchAttributeConfigCreateTimeout, searchAttributeConfigPollInterval, searchAttributeConfigMaxPollInterval = oldTimeout, oldInterval, oldMax
	})
}

// searchAttributeConfigTestServer accepts creates with 202 and answers reads with 204 until
// availableAfter reads were made.
func searchAttributeConfigTestServer(t *testing.T, availableAfter int32) (*Config, *int32) {
	t.Helper()
	var reads int32
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			if atomic.AddInt32(&reads, 1) <= availableAfter {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			_, _ = w.Write([]byte(`{"name":"newMail","displayName":"New Mail","applicationAttributes":{"src-1":"mail"}}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return &Config{URL: server.URL, Credentials: []ClientCredential{{ClientId: "id", ClientSecret: "secret"}}, MaxClientPoolSize: 1, ClientRequestRateLimit: 1000}, &reads
}

func searchAttributeConfigTestCreate(t *testing.T, cfg *Config) *resource.CreateResponse {
	t.Helper()
	ctx := context.Background()
	r := &SearchAttributeConfigResource{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: cfg}, &resource.ConfigureResponse{})
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	attributes, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{"src-1": "mail"})
	plan := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	if diags := plan.Set(ctx, &SearchAttributeConfigModel{ID: types.StringValue("newMail"), Name: types.StringValue("newMail"),
		DisplayName: types.StringValue("New Mail"), ApplicationAttributes: attributes}); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}, resp)
	return resp
}

func TestSearchAttributeConfigCreateWaitsForAsyncCreate(t *testing.T) {
	searchAttributeConfigTestFastPolling(t, 5*time.Second)
	cfg, reads := searchAttributeConfigTestServer(t, 3)
	resp := searchAttributeConfigTestCreate(t, cfg)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if got := atomic.LoadInt32(reads); got != 4 {
		t.Fatalf("expected Create to read until the configuration exists, got %d reads", got)
	}
	var data SearchAttributeConfigModel
	resp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "newMail" || data.DisplayName.ValueString() != "New Mail" {
		t.Fatalf("unexpected state %+v", data)
	}
}

func TestSearchAttributeConfigCreateTimeout(t *testing.T) {
	searchAttributeConfigTestFastPolling(t, 30*time.Millisecond)
	cfg, _ := searchAttributeConfigTestServer(t, 1<<30)
	resp := searchAttributeConfigTestCreate(t, cfg)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "did not become available") {
		t.Fatalf("expected a timeout error, got %v", resp.Diagnostics)
	}
	// The accepted configuration is saved, so Terraform taints it instead of forgetting it.
	var data SearchAttributeConfigModel
	resp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "newMail" {
		t.Fatalf("expected the state to be saved, got %+v", data)
	}
}

func TestSearchAttributeConfigWaitHonoursContext(t *testing.T) {
	searchAttributeConfigTestFastPolling(t, time.Minute)
	cfg, _ := searchAttributeConfigTestServer(t, 1<<30)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	client, err := cfg.IdentityNowClient(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	start := time.Now()
	if _, err := client.WaitForSearchAttributeConfig(ctx, "newMail", time.Minute); err == nil || time.Since(start) > 10*time.Second {
		t.Fatalf("expected the wait to stop with the context, got %v after %s", err, time.Since(start))
	}
}
