package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestConnectorClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotContentType = r.Method, r.URL.Path, string(body), r.Header.Get("Content-Type")
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"name":"My Connector","type":"custom My Connector","className":"sailpoint.connector.OpenConnectorAdapter","scriptName":"my-connector","directConnect":true,"status":"DEVELOPMENT","applicationXml":"<xml/>","fileUpload":false,"connectorMetadata":{"supportedUI":"EXTJS"}}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	plan := ConnectorModel{
		Name: types.StringValue("My Connector"), Type: types.StringUnknown(), ClassName: types.StringValue("sailpoint.connector.OpenConnectorAdapter"),
		DirectConnect: types.BoolUnknown(), Status: types.StringValue("DEVELOPMENT"),
	}
	created, err := client.CreateConnector(ctx, connectorCreateRequestFromModel(plan))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/connectors" || gotBody != `{"name":"My Connector","className":"sailpoint.connector.OpenConnectorAdapter","status":"DEVELOPMENT"}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	if created.ScriptName != "my-connector" {
		t.Fatalf("unexpected response %+v", created)
	}

	ops := []jsonPatchOp{{Op: "replace", Path: "/applicationXml", Value: "app"}}
	if _, err := client.PatchConnector(ctx, "my-connector", ops); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/connectors/my-connector" || gotContentType != "application/json-patch+json" ||
		gotBody != `[{"op":"replace","path":"/applicationXml","value":"app"}]` {
		t.Fatalf("unexpected patch request %s %s %s %s", gotMethod, gotPath, gotContentType, gotBody)
	}
	if _, err := client.GetConnector(ctx, "my-connector"); err != nil || gotMethod != http.MethodGet || gotPath != "/v2026/connectors/my-connector" {
		t.Fatalf("unexpected get request %s %s (%v)", gotMethod, gotPath, err)
	}
	if err := client.DeleteConnector(ctx, "my-connector"); err != nil || gotMethod != http.MethodDelete {
		t.Fatalf("unexpected delete request %s (%v)", gotMethod, err)
	}
}

func TestConnectorPatchOnlyChangedConfiguredValues(t *testing.T) {
	state := ConnectorModel{
		ConnectorMetadataJSON: types.StringValue(`{"a":1}`), ApplicationXML: types.StringValue("<a/>"),
		CorrelationConfigXML: types.StringValue("<c/>"), SourceConfigXML: types.StringValue("<s/>"),
	}
	plan := state
	plan.ConnectorMetadataJSON = types.StringValue(`{ "a": 1 }`)
	plan.SourceConfigXML = types.StringValue("<s2/>")
	ops, err := connectorPatch(plan, state)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(ops) != 1 || ops[0].Path != "/sourceConfigXml" || ops[0].Value != "<s2/>" {
		t.Fatalf("unexpected operations %+v", ops)
	}

	// On create, only configured values are patched.
	created := ConnectorModel{ConnectorMetadataJSON: types.StringValue(`{"a":2}`), ApplicationXML: types.StringUnknown(), CorrelationConfigXML: types.StringUnknown(), SourceConfigXML: types.StringValue("<s/>")}
	unset := ConnectorModel{ConnectorMetadataJSON: types.StringNull(), ApplicationXML: types.StringNull(), CorrelationConfigXML: types.StringNull(), SourceConfigXML: types.StringNull()}
	ops, err = connectorPatch(created, unset)
	if err != nil || len(ops) != 2 || ops[0].Path != "/connectorMetadata" || ops[1].Path != "/sourceConfigXml" {
		t.Fatalf("unexpected operations %+v (%v)", ops, err)
	}
}

func TestConnectorState(t *testing.T) {
	directConnect, fileUpload := true, false
	api := &Connector{
		Name: "My Connector", Type: "custom My Connector", ClassName: "cls", ScriptName: "my-connector", DirectConnect: &directConnect,
		Status: "DEVELOPMENT", ApplicationXML: "<xml/>\n", FileUpload: &fileUpload, ConnectorMetadata: map[string]interface{}{"a": 1.0},
	}

	// After create, planned values are kept and unknown values are resolved.
	data := ConnectorModel{
		ID: types.StringValue("my-connector"), ScriptName: types.StringValue("my-connector"), Name: types.StringValue("My Connector"), Type: types.StringUnknown(),
		ClassName: types.StringValue("cls"), DirectConnect: types.BoolUnknown(), Status: types.StringValue("DEVELOPMENT"),
		ConnectorMetadataJSON: types.StringUnknown(), ApplicationXML: types.StringValue("<xml/>"), CorrelationConfigXML: types.StringUnknown(),
		SourceConfigXML: types.StringUnknown(), FileUpload: types.BoolUnknown(),
	}
	setConnectorState(&data, api, false)
	if data.Type.ValueString() != "custom My Connector" || !data.DirectConnect.ValueBool() || data.FileUpload.ValueBool() ||
		data.ApplicationXML.ValueString() != "<xml/>" || data.ConnectorMetadataJSON.ValueString() != `{"a":1}` ||
		data.CorrelationConfigXML.IsUnknown() || data.SourceConfigXML.IsUnknown() {
		t.Fatalf("unexpected state after create %+v", data)
	}

	// Refresh keeps text that differs only in trailing whitespace and detects drift.
	api.ApplicationXML = "<xml/>\r\n"
	api.Status = "RELEASED"
	setConnectorState(&data, api, true)
	if data.ApplicationXML.ValueString() != "<xml/>" || data.Status.ValueString() != "RELEASED" {
		t.Fatalf("unexpected state after refresh %+v", data)
	}

	unknown := ConnectorModel{Type: types.StringUnknown(), Status: types.StringUnknown(), ConnectorMetadataJSON: types.StringUnknown(), ApplicationXML: types.StringUnknown(),
		CorrelationConfigXML: types.StringUnknown(), SourceConfigXML: types.StringUnknown(), DirectConnect: types.BoolUnknown(), FileUpload: types.BoolUnknown()}
	connectorResolveUnknown(&unknown)
	if unknown.Type.IsUnknown() || unknown.DirectConnect.IsUnknown() || unknown.SourceConfigXML.IsUnknown() {
		t.Fatalf("expected unknown values to be resolved, got %+v", unknown)
	}
}

// connectorTestCreate runs Create against a server whose create response has no script name.
// list is the response of the connector list.
func connectorTestCreate(t *testing.T, list string) (*resource.CreateResponse, []string) {
	t.Helper()
	ctx := context.Background()
	var requests []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path+" "+r.URL.Query().Get("filters"))
		switch {
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"name":"My Connector","className":"sailpoint.connector.OpenConnectorAdapter"}`))
		case r.URL.Path == "/v2026/connectors":
			_, _ = w.Write([]byte(list))
		default:
			_, _ = w.Write([]byte(`{"name":"My Connector","type":"custom My Connector","className":"sailpoint.connector.OpenConnectorAdapter","scriptName":"my-connector","directConnect":true,"status":"DEVELOPMENT","fileUpload":false}`))
		}
	})
	r := &ConnectorResource{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: &Config{URL: server.URL, Credentials: []ClientCredential{{ClientId: "id", ClientSecret: "secret"}}, MaxClientPoolSize: 1, ClientRequestRateLimit: 1000}}, &resource.ConfigureResponse{})
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	null := tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)
	plan := tfsdk.State{Schema: schemaResp.Schema, Raw: null}
	if diags := plan.Set(ctx, &ConnectorModel{
		ID: types.StringUnknown(), ScriptName: types.StringUnknown(), Name: types.StringValue("My Connector"), Type: types.StringUnknown(),
		ClassName: types.StringValue("sailpoint.connector.OpenConnectorAdapter"), DirectConnect: types.BoolUnknown(), Status: types.StringUnknown(),
		ConnectorMetadataJSON: types.StringUnknown(), ApplicationXML: types.StringUnknown(), CorrelationConfigXML: types.StringUnknown(),
		SourceConfigXML: types.StringUnknown(), FileUpload: types.BoolUnknown(),
	}); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: null}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}, resp)
	return resp, requests
}

func TestConnectorCreateFindsScriptNameByName(t *testing.T) {
	resp, requests := connectorTestCreate(t, `[{"name":"My Connector 2","scriptName":"my-connector-2"},{"name":"My Connector","scriptName":"my-connector"}]`)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if len(requests) != 3 || requests[1] != `GET /v2026/connectors name sw "My Connector"` || requests[2] != "GET /v2026/connectors/my-connector " {
		t.Fatalf("unexpected requests %q", requests)
	}
	var data ConnectorModel
	resp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "my-connector" || data.ScriptName.ValueString() != "my-connector" || data.Status.ValueString() != "DEVELOPMENT" {
		t.Fatalf("unexpected state %+v", data)
	}
}

func TestConnectorCreateWithoutScriptNameFails(t *testing.T) {
	resp, _ := connectorTestCreate(t, `[{"name":"My Connector 2","scriptName":"my-connector-2"}]`)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "could not be found by name") {
		t.Fatalf("expected an error, got %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatalf("expected no state without a script name, got %s", resp.State.Raw)
	}
}
