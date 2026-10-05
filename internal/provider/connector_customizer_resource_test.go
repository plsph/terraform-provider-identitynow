package provider

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestConnectorCustomizerClient(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/connector-customizers":
			_, _ = w.Write([]byte(`[{"id":"cc-0","name":"other"},{"id":"cc-1","name":"custom","imageVersion":2,"imageID":"img","tenantID":"t-1","created":"c"}]`))
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"cc-1","name":"custom","tenantID":"t-1","created":"c"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"cc-1","name":"renamed","imageVersion":2,"imageID":"img","tenantID":"t-1","created":"c"}`))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	created, err := client.CreateConnectorCustomizer(ctx, &ConnectorCustomizerRequest{Name: "custom"})
	if err != nil || gotMethod != http.MethodPost || gotPath != "/v2026/connector-customizers" || gotBody != `{"name":"custom"}` {
		t.Fatalf("unexpected create request %s %s %s (%v)", gotMethod, gotPath, gotBody, err)
	}
	data := ConnectorCustomizerModel{ID: types.StringUnknown(), Name: types.StringValue("custom"), ImageVersion: types.Int64Unknown(),
		ImageID: types.StringUnknown(), TenantID: types.StringUnknown(), Created: types.StringUnknown()}
	setConnectorCustomizerState(&data, created, false)
	if data.ID.ValueString() != "cc-1" || !data.ImageVersion.IsNull() || !data.ImageID.IsNull() || data.TenantID.ValueString() != "t-1" {
		t.Fatalf("unexpected state after create %+v", data)
	}

	updated, err := client.UpdateConnectorCustomizer(ctx, "cc-1", &ConnectorCustomizerRequest{Name: "renamed"})
	if err != nil || gotMethod != http.MethodPut || gotPath != "/v2026/connector-customizers/cc-1" || gotBody != `{"name":"renamed"}` {
		t.Fatalf("unexpected update request %s %s %s (%v)", gotMethod, gotPath, gotBody, err)
	}
	// Update keeps the planned (prior) computed values, refresh takes them from the API.
	data.Name = types.StringValue("renamed")
	setConnectorCustomizerState(&data, updated, false)
	if !data.ImageVersion.IsNull() {
		t.Fatalf("expected planned image version to be kept, got %s", data.ImageVersion)
	}
	setConnectorCustomizerState(&data, updated, true)
	if data.ImageVersion.ValueInt64() != 2 || data.ImageID.ValueString() != "img" {
		t.Fatalf("unexpected state after refresh %+v", data)
	}

	found, err := client.GetConnectorCustomizerByName(ctx, "custom")
	if err != nil || found.ID != "cc-1" {
		t.Fatalf("unexpected lookup result %+v (%v)", found, err)
	}
	if _, err := client.GetConnectorCustomizerByName(ctx, "missing"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if err := client.DeleteConnectorCustomizer(ctx, "cc-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/connector-customizers/cc-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

// connectorCustomizerTestUpdate renames the customizer "custom" to "renamed" against a server
// that reports storedName after the PUT.
func connectorCustomizerTestUpdate(t *testing.T, storedName string) (*resource.UpdateResponse, []string) {
	t.Helper()
	ctx := context.Background()
	var requests []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		// The PUT response echoes the request, the read returns what was stored.
		name := storedName
		if r.Method == http.MethodPut {
			name = "renamed"
		}
		_, _ = w.Write([]byte(`{"id":"cc-1","name":"` + name + `","tenantID":"t-1","created":"c"}`))
	})
	r := &ConnectorCustomizerResource{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: &Config{URL: server.URL, Credentials: []ClientCredential{{ClientId: "id", ClientSecret: "secret"}}, MaxClientPoolSize: 1, ClientRequestRateLimit: 1000}}, &resource.ConfigureResponse{})
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	encode := func(name string) tfsdk.State {
		state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
		if diags := state.Set(ctx, &ConnectorCustomizerModel{ID: types.StringValue("cc-1"), Name: types.StringValue(name), ImageVersion: types.Int64Null(),
			ImageID: types.StringNull(), TenantID: types.StringValue("t-1"), Created: types.StringValue("c")}); diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		return state
	}
	prior, plan := encode("custom"), encode("renamed")
	resp := &resource.UpdateResponse{State: prior}
	r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}, State: prior}, resp)
	return resp, requests
}

func TestConnectorCustomizerUpdateVerifiesRename(t *testing.T) {
	resp, requests := connectorCustomizerTestUpdate(t, "renamed")
	if resp.Diagnostics.HasError() || len(requests) != 2 || requests[0] != "PUT /v2026/connector-customizers/cc-1" || requests[1] != "GET /v2026/connector-customizers/cc-1" {
		t.Fatalf("unexpected result %v %q", resp.Diagnostics, requests)
	}
	var data ConnectorCustomizerModel
	resp.State.Get(context.Background(), &data)
	if data.Name.ValueString() != "renamed" {
		t.Fatalf("expected the new name in state, got %+v", data)
	}

	// The API ignored the rename: the apply fails and the prior state is kept.
	resp, _ = connectorCustomizerTestUpdate(t, "custom")
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error when the rename is ignored")
	}
	resp.State.Get(context.Background(), &data)
	if data.Name.ValueString() != "custom" {
		t.Fatalf("expected the prior name in state, got %+v", data)
	}
}
