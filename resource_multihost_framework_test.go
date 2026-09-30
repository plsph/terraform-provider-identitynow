package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func multihostTestRef(t *testing.T, id string, name types.String) types.List {
	t.Helper()
	list, d := types.ListValueFrom(context.Background(), multihostRefObjectType, []MultihostRefModel{{ID: types.StringValue(id), Name: name}})
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}
	return list
}

func TestMultihostClient(t *testing.T) {
	const multihostJSON = `{"id":"mh-1","name":"SQL","description":"d","owner":{"type":"IDENTITY","id":"owner-1","name":"John"},"connector":"multihost-microsoft-sql-server","connectorAttributes":{"authType":"SQLAuthentication","password":"2:ENC:abc","maxSourcesPerAggGroup":10,"multihost_status":"ready"},"type":"Multi-Host - Microsoft SQL Server","created":"c","modified":"m"}`
	var gotMethod, gotPath, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotContentType = r.Method, r.URL.Path, string(body), r.Header.Get("Content-Type")
		switch r.Method {
		case http.MethodDelete, http.MethodPatch:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(multihostJSON))
		default:
			_, _ = w.Write([]byte(multihostJSON))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	data := MultihostModel{
		ID: types.StringUnknown(), Name: types.StringValue("SQL"), Description: types.StringValue("d"), Connector: types.StringValue("multihost-microsoft-sql-server"),
		ConnectorAttributesJSON: types.StringValue(`{"authType":"SQLAuthentication","password":"secret"}`), MaxSourcesPerAggGroup: types.Int64Value(10),
		MaxAllowedSources: types.Int64Unknown(), Type: types.StringUnknown(), Created: types.StringUnknown(), Modified: types.StringUnknown(),
		Owner: multihostTestRef(t, "owner-1", types.StringUnknown()), Cluster: types.ListNull(multihostRefObjectType), ManagementWorkgroup: types.ListNull(multihostRefObjectType),
	}
	request := multihostFromModel(ctx, data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	created, err := client.CreateMultihost(ctx, request)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := `{"name":"SQL","description":"d","owner":{"type":"IDENTITY","id":"owner-1"},"connector":"multihost-microsoft-sql-server","connectorAttributes":{"authType":"SQLAuthentication","maxSourcesPerAggGroup":10,"password":"secret"}}`
	if gotMethod != http.MethodPost || gotPath != "/v2026/multihosts" || gotBody != want {
		t.Fatalf("unexpected create request %s %s\n got %s\nwant %s", gotMethod, gotPath, gotBody, want)
	}

	// Create keeps planned values and resolves unknown ones, including the owner name.
	setMultihostState(ctx, &data, created, false, &diags)
	var owners []MultihostRefModel
	diags.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if data.ID.ValueString() != "mh-1" || !data.MaxAllowedSources.IsNull() || data.MaxSourcesPerAggGroup.ValueInt64() != 10 ||
		data.Type.ValueString() != "Multi-Host - Microsoft SQL Server" || len(owners) != 1 || owners[0].Name.ValueString() != "John" ||
		data.ConnectorAttributesJSON.ValueString() != `{"authType":"SQLAuthentication","password":"secret"}` {
		t.Fatalf("unexpected state after create %+v (%v)", data, diags)
	}

	if err := client.PatchMultihost(ctx, "mh-1", []jsonPatchOp{{Op: "replace", Path: "/description", Value: "new"}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/multihosts/mh-1" || gotContentType != "application/json-patch+json" {
		t.Fatalf("unexpected patch request %s %s %s", gotMethod, gotPath, gotContentType)
	}
	if err := client.DeleteMultihost(ctx, "mh-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/multihosts/mh-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestMultihostPatch(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := MultihostModel{
		Name: types.StringValue("SQL"), Description: types.StringValue("d"),
		ConnectorAttributesJSON: types.StringValue(`{"user":"a","url":"jdbc:x","old":"o"}`),
		MaxSourcesPerAggGroup:   types.Int64Value(10), MaxAllowedSources: types.Int64Value(300),
		Owner:               multihostTestRef(t, "owner-1", types.StringValue("John")),
		Cluster:             multihostTestRef(t, "cluster-1", types.StringValue("VA")),
		ManagementWorkgroup: types.ListNull(multihostRefObjectType),
	}
	plan := state
	plan.Description = types.StringValue("new")
	plan.ConnectorAttributesJSON = types.StringValue(`{"url":"jdbc:x","user":"b","a/b":1}`)
	plan.MaxSourcesPerAggGroup = types.Int64Value(20)
	// Names are computed and unknown in the plan; an unchanged ID does not produce an operation.
	plan.Owner = multihostTestRef(t, "owner-1", types.StringUnknown())
	plan.Cluster = types.ListNull(multihostRefObjectType)
	plan.ManagementWorkgroup = multihostTestRef(t, "wg-1", types.StringUnknown())

	ops, removed := multihostPatch(ctx, plan, state, map[string]interface{}{"user": "a", "url": "jdbc:x", "old": "o"}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	encoded, _ := json.Marshal(ops)
	want := `[{"op":"replace","path":"/description","value":"new"},` +
		`{"op":"replace","path":"/cluster","value":null},` +
		`{"op":"replace","path":"/managementWorkgroup","value":{"type":"GOVERNANCE_GROUP","id":"wg-1"}},` +
		`{"op":"add","path":"/connectorAttributes/a~1b","value":1},` +
		`{"op":"add","path":"/connectorAttributes/user","value":"b"},` +
		`{"op":"add","path":"/connectorAttributes/maxSourcesPerAggGroup","value":20}]`
	if string(encoded) != want {
		t.Fatalf("unexpected operations\n got %s\nwant %s", encoded, want)
	}
	if len(removed) != 1 || removed[0] != "old" {
		t.Fatalf("unexpected removed keys %v", removed)
	}
	if ops, removed := multihostPatch(ctx, state, state, nil, &diags); len(ops) != 0 || len(removed) != 0 {
		t.Fatalf("expected no operations without changes, got %+v %v", ops, removed)
	}
}

func TestMultihostConnectorAttributesState(t *testing.T) {
	prior := types.StringValue(`{"user": "a", "password": "secret", "port": 1433}`)
	api := map[string]interface{}{"user": "a", "password": "2:ENC:xyz", "port": 1433.0, "multihost_status": "ready"}
	if got := multihostConnectorAttributesState(prior, api); !got.Equal(prior) {
		t.Fatalf("expected configured attributes to be kept, got %s", got)
	}
	api["password"] = "********"
	if got := multihostConnectorAttributesState(prior, api); !got.Equal(prior) {
		t.Fatalf("expected masked value to be ignored, got %s", got)
	}
	delete(api, "password")
	api["user"] = "changed"
	if got := multihostConnectorAttributesState(prior, api); got.ValueString() != `{"password":"secret","port":1433,"user":"changed"}` {
		t.Fatalf("expected drift of a configured key, got %s", got)
	}
	if got := multihostConnectorAttributesState(types.StringNull(), api); !got.IsNull() {
		t.Fatalf("expected unset attributes to stay null, got %s", got)
	}
}

func TestMultihostPatchNestedAttributes(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := MultihostModel{
		Name: types.StringValue("SQL"), Description: types.StringValue("d"),
		ConnectorAttributesJSON: types.StringValue(`{"multiHostAttributes":{"user":"u","password":"secret","authType":"SQLAuthentication"}}`),
		MaxSourcesPerAggGroup:   types.Int64Null(), MaxAllowedSources: types.Int64Null(),
		Owner:   multihostTestRef(t, "owner-1", types.StringValue("John")),
		Cluster: types.ListNull(multihostRefObjectType), ManagementWorkgroup: types.ListNull(multihostRefObjectType),
	}
	current := map[string]interface{}{"multiHostAttributes": map[string]interface{}{
		"user": "u", "password": "****", "authType": "SQLAuthentication", "connector_files": "x.jar"}}

	// An unchanged nested object produces no operation, although the API masks the password.
	if ops, removed := multihostPatch(ctx, state, state, current, &diags); len(ops) != 0 || len(removed) != 0 {
		t.Fatalf("expected no operations without changes, got %+v %v", ops, removed)
	}

	// Only the changed leaf is sent, so connector_files and the password are not overwritten, and
	// a nested key removed from the configuration is reported.
	plan := state
	plan.ConnectorAttributesJSON = types.StringValue(`{"multiHostAttributes":{"user":"u2","password":"secret"}}`)
	ops, removed := multihostPatch(ctx, plan, state, current, &diags)
	encoded, _ := json.Marshal(ops)
	if want := `[{"op":"add","path":"/connectorAttributes/multiHostAttributes/user","value":"u2"}]`; string(encoded) != want {
		t.Fatalf("unexpected operations\n got %s\nwant %s", encoded, want)
	}
	if len(removed) != 1 || removed[0] != "multiHostAttributes/authType" {
		t.Fatalf("unexpected removed keys %v", removed)
	}

	// A newly configured nested object that already exists in IdentityNow is patched per value.
	plan.ConnectorAttributesJSON = types.StringValue(`{"multiHostAttributes":{"user":"u"},"other":{"a":1}}`)
	state.ConnectorAttributesJSON = types.StringNull()
	ops, _ = multihostPatch(ctx, plan, state, current, &diags)
	encoded, _ = json.Marshal(ops)
	if want := `[{"op":"add","path":"/connectorAttributes/multiHostAttributes/user","value":"u"},{"op":"add","path":"/connectorAttributes/other","value":{"a":1}}]`; string(encoded) != want {
		t.Fatalf("unexpected operations\n got %s\nwant %s", encoded, want)
	}

	// Without the nested object in IdentityNow, the whole configured object is added.
	ops, _ = multihostPatch(ctx, plan, state, nil, &diags)
	encoded, _ = json.Marshal(ops)
	if want := `[{"op":"add","path":"/connectorAttributes/multiHostAttributes","value":{"user":"u"}},{"op":"add","path":"/connectorAttributes/other","value":{"a":1}}]`; string(encoded) != want {
		t.Fatalf("unexpected operations\n got %s\nwant %s", encoded, want)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestMultihostConnectorAttributesStateNested(t *testing.T) {
	prior := types.StringValue(`{"multiHostAttributes":{"user":"u","password":"secret"}}`)
	api := map[string]interface{}{"multihost_status": "ready", "multiHostAttributes": map[string]interface{}{
		"user": "u", "password": "****", "connector_files": "x.jar"}}
	if got := multihostConnectorAttributesState(prior, api); !got.Equal(prior) {
		t.Fatalf("expected configured nested attributes to be kept, got %s", got)
	}
	api["multiHostAttributes"].(map[string]interface{})["password"] = "2:ENC:abc"
	if got := multihostConnectorAttributesState(prior, api); !got.Equal(prior) {
		t.Fatalf("expected encrypted nested value to be ignored, got %s", got)
	}
	delete(api["multiHostAttributes"].(map[string]interface{}), "password")
	if got := multihostConnectorAttributesState(prior, api); !got.Equal(prior) {
		t.Fatalf("expected a nested value the API omits to be kept, got %s", got)
	}
	api["multiHostAttributes"].(map[string]interface{})["user"] = "changed"
	if got := multihostConnectorAttributesState(prior, api); got.ValueString() != `{"multiHostAttributes":{"password":"secret","user":"changed"}}` {
		t.Fatalf("expected drift of a configured nested key, got %s", got)
	}
	api["multiHostAttributes"] = "****"
	if got := multihostConnectorAttributesState(prior, api); !got.Equal(prior) {
		t.Fatalf("expected a masked nested object to be ignored, got %s", got)
	}
	arrays := types.StringValue(`{"hosts":[{"name":"a","password":"p"}]}`)
	if got := multihostConnectorAttributesState(arrays, map[string]interface{}{"hosts": []interface{}{map[string]interface{}{"name": "a", "password": "***", "port": 1.0}}}); !got.Equal(arrays) {
		t.Fatalf("expected masked values in arrays to be ignored, got %s", got)
	}
}

func TestMultihostRefreshState(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	data := MultihostModel{ID: types.StringValue("mh-1"), ConnectorAttributesJSON: types.StringNull(), Owner: types.ListNull(multihostRefObjectType),
		Cluster: types.ListNull(multihostRefObjectType), ManagementWorkgroup: types.ListNull(multihostRefObjectType)}
	setMultihostState(ctx, &data, &Multihost{
		ID: "mh-1", Name: "SQL", Description: "d", Connector: "c", Owner: &MultihostRef{ID: "owner-1", Name: "John"},
		Cluster:             &MultihostRef{ID: "cluster-1", Name: "VA"},
		ConnectorAttributes: map[string]interface{}{"maxAllowedSources": 300.0},
	}, true, &diags)
	if diags.HasError() || !data.ConnectorAttributesJSON.IsNull() || data.MaxAllowedSources.ValueInt64() != 300 || !data.MaxSourcesPerAggGroup.IsNull() ||
		len(data.Cluster.Elements()) != 1 || !data.ManagementWorkgroup.IsNull() || !data.Created.IsNull() {
		t.Fatalf("unexpected refreshed state %+v (%v)", data, diags)
	}

	unknown := MultihostModel{Type: types.StringUnknown(), Created: types.StringUnknown(), Modified: types.StringUnknown(), MaxSourcesPerAggGroup: types.Int64Unknown(),
		MaxAllowedSources: types.Int64Unknown(), Owner: multihostTestRef(t, "owner-1", types.StringUnknown()), Cluster: types.ListNull(multihostRefObjectType),
		ManagementWorkgroup: types.ListNull(multihostRefObjectType)}
	multihostResolveUnknown(ctx, &unknown, &diags)
	var owners []MultihostRefModel
	diags.Append(unknown.Owner.ElementsAs(ctx, &owners, false)...)
	if unknown.Type.IsUnknown() || unknown.MaxAllowedSources.IsUnknown() || owners[0].Name.IsUnknown() {
		t.Fatalf("expected unknown values to be resolved, got %+v", unknown)
	}
}
