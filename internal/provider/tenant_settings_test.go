package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// tenantSettingsTestUnmanagedKey is an API field that no resource models. PUT updates must keep it
// (putMerged) or must not send it (request body built from the writable settings).
const tenantSettingsTestUnmanagedKey = "tfTestUnmanaged"

type tenantSettingsTestRequest struct {
	Method       string
	Path         string
	ContentType  string
	Experimental string
	Body         interface{}
}

// tenantSettingsTestAPI is a fake tenant settings API holding one object.
type tenantSettingsTestAPI struct {
	t        *testing.T
	path     string
	doc      map[string]interface{}
	missing  bool     // GET returns 404 until the object is created with POST
	hidden   []string // top-level keys never returned (secrets)
	requests []tenantSettingsTestRequest
}

func (a *tenantSettingsTestAPI) handler(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			a.t.Errorf("invalid request body %s: %s", raw, err)
		}
	}
	a.requests = append(a.requests, tenantSettingsTestRequest{
		Method: r.Method, Path: r.URL.Path, ContentType: r.Header.Get("Content-Type"),
		Experimental: r.Header.Get("X-SailPoint-Experimental"), Body: body,
	})
	if r.URL.Path != a.path {
		a.t.Errorf("unexpected path %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if a.missing {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"messages":[{"text":"not found"}]}`))
			return
		}
	case http.MethodPost:
		a.doc, _ = body.(map[string]interface{})
		a.missing = false
	case http.MethodPut:
		a.doc = tenantSettingsMergeJSON(a.doc, body).(map[string]interface{})
	case http.MethodPatch:
		encoded, _ := json.Marshal(body)
		var ops []jsonPatchOp
		if err := json.Unmarshal(encoded, &ops); err != nil {
			a.t.Errorf("invalid JSON Patch %s", encoded)
		}
		for _, op := range ops {
			if op.Op != "replace" && op.Op != "add" {
				a.t.Errorf("unexpected patch operation %+v", op)
			}
			a.doc = tenantSettingsTestSet(a.doc, strings.Split(strings.TrimPrefix(op.Path, "/"), "/"), op.Value).(map[string]interface{})
		}
	}
	response := map[string]interface{}{}
	for k, v := range a.doc {
		response[k] = v
	}
	for _, k := range a.hidden {
		delete(response, k)
	}
	_ = json.NewEncoder(w).Encode(response)
}

// writes returns the requests that change the object.
func (a *tenantSettingsTestAPI) writes() []tenantSettingsTestRequest {
	var writes []tenantSettingsTestRequest
	for _, r := range a.requests {
		if r.Method != http.MethodGet {
			writes = append(writes, r)
		}
	}
	return writes
}

func tenantSettingsTestSet(node interface{}, segments []string, value interface{}) interface{} {
	if len(segments) == 0 {
		return value
	}
	switch n := node.(type) {
	case []interface{}:
		i, _ := strconv.Atoi(segments[0])
		if i >= len(n) {
			n = append(n, nil)
			i = len(n) - 1
		}
		n[i] = tenantSettingsTestSet(n[i], segments[1:], value)
		return n
	case map[string]interface{}:
		n[segments[0]] = tenantSettingsTestSet(n[segments[0]], segments[1:], value)
		return n
	}
	return map[string]interface{}{segments[0]: tenantSettingsTestSet(nil, segments[1:], value)}
}

func tenantSettingsTestProviderConfig(url string) *Config {
	return &Config{URL: url, Credentials: []ClientCredential{{ClientId: "id", ClientSecret: "secret"}}, MaxClientPoolSize: 1, ClientRequestRateLimit: 1000}
}

// tenantSettingsTestValue returns a sample value of a setting; variants differ from each other.
func tenantSettingsTestValue(f *tenantSettingsField, variant int) attr.Value {
	s := fmt.Sprintf("%s-%d", f.Name, variant)
	switch f.Kind {
	case tenantSettingsBool:
		return types.BoolValue(variant%2 == 1)
	case tenantSettingsInt64, tenantSettingsInt64String:
		return types.Int64Value(int64(10 + variant))
	case tenantSettingsFloat64:
		return types.Float64Value(0.25 * float64(variant))
	case tenantSettingsStringList:
		return types.ListValueMust(types.StringType, []attr.Value{types.StringValue(s)})
	case tenantSettingsStringMap:
		return types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue(s)})
	case tenantSettingsJSONObject:
		return types.StringValue(fmt.Sprintf(`{"tfTest": %q}`, s))
	case tenantSettingsJSONArray:
		return types.StringValue(fmt.Sprintf(`[{"tfTest": %q}]`, s))
	}
	return types.StringValue(s)
}

// tenantSettingsTestRaw builds an object value of the schema: values are used where given, rest
// returns the value of the other attributes.
func tenantSettingsTestRaw(t *testing.T, sch schema.Schema, values map[string]attr.Value, unknownRest bool) tftypes.Value {
	t.Helper()
	ctx := context.Background()
	objType := sch.Type().TerraformType(ctx).(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for name, typ := range objType.AttributeTypes {
		if v, ok := values[name]; ok && v != nil {
			tv, err := v.ToTerraformValue(ctx)
			if err != nil {
				t.Fatalf("converting %s: %s", name, err)
			}
			vals[name] = tv
		} else if unknownRest {
			vals[name] = tftypes.NewValue(typ, tftypes.UnknownValue)
		} else {
			vals[name] = tftypes.NewValue(typ, nil)
		}
	}
	return tftypes.NewValue(objType, vals)
}

func tenantSettingsTestNoErrors(t *testing.T, step string, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatalf("%s: %v", step, diags)
	}
}

// tenantSettingsTestHarness drives a tenant settings resource through the framework.
type tenantSettingsTestHarness struct {
	t      *testing.T
	r      *tenantSettingsResource
	spec   *tenantSettingsSpec
	schema schema.Schema
	api    *tenantSettingsTestAPI
}

func newTenantSettingsTestHarness(t *testing.T, newResource func() resource.Resource, apiPath, doc string) *tenantSettingsTestHarness {
	t.Helper()
	ctx := context.Background()
	r := newResource().(*tenantSettingsResource)
	if r.spec.Path != apiPath {
		t.Fatalf("unexpected API path %s, want %s", r.spec.Path, apiPath)
	}
	api := &tenantSettingsTestAPI{t: t, path: apiPath}
	if err := json.Unmarshal([]byte(doc), &api.doc); err != nil {
		t.Fatalf("invalid test document: %s", err)
	}
	api.doc[tenantSettingsTestUnmanagedKey] = "keep"
	server := newTestServer(t, api.handler)
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: tenantSettingsTestProviderConfig(server.URL)}, &resource.ConfigureResponse{})
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	return &tenantSettingsTestHarness{t: t, r: r, spec: r.spec, schema: schemaResp.Schema, api: api}
}

func (h *tenantSettingsTestHarness) values(state tfsdk.State) map[string]attr.Value {
	var diags diag.Diagnostics
	values := tenantSettingsValues(context.Background(), state, h.spec, &diags)
	tenantSettingsTestNoErrors(h.t, "reading state", diags)
	return values
}

// create runs Create with the configured values; the other settings are unknown in the plan.
func (h *tenantSettingsTestHarness) create(configured map[string]attr.Value) tfsdk.State {
	h.t.Helper()
	ctx := context.Background()
	req := resource.CreateRequest{
		Config: tfsdk.Config{Schema: h.schema, Raw: tenantSettingsTestRaw(h.t, h.schema, configured, false)},
		Plan:   tfsdk.Plan{Schema: h.schema, Raw: tenantSettingsTestRaw(h.t, h.schema, configured, true)},
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: h.schema, Raw: tenantSettingsTestRaw(h.t, h.schema, nil, false)}}
	h.r.Create(ctx, req, resp)
	tenantSettingsTestNoErrors(h.t, "create", resp.Diagnostics)
	h.checkState("create", resp.State)
	return resp.State
}

// update runs Update with the configured values; the other writable and the stable settings keep
// their state value in the plan (UseStateForUnknown), the other read-only settings are unknown.
func (h *tenantSettingsTestHarness) update(state tfsdk.State, configured map[string]attr.Value) tfsdk.State {
	h.t.Helper()
	resp := h.updateResponse(state, configured)
	tenantSettingsTestNoErrors(h.t, "update", resp.Diagnostics)
	h.checkState("update", resp.State)
	return resp.State
}

// updateResponse runs Update like update and returns the response without checking it.
func (h *tenantSettingsTestHarness) updateResponse(state tfsdk.State, configured map[string]attr.Value) *resource.UpdateResponse {
	h.t.Helper()
	ctx := context.Background()
	planned := h.values(state)
	planned["id"] = types.StringValue(h.spec.ID)
	for i := range h.spec.Fields {
		f := &h.spec.Fields[i]
		if f.ReadOnly && !f.Stable {
			delete(planned, f.Name)
		}
	}
	for name, v := range configured {
		planned[name] = v
	}
	req := resource.UpdateRequest{
		Config: tfsdk.Config{Schema: h.schema, Raw: tenantSettingsTestRaw(h.t, h.schema, configured, false)},
		Plan:   tfsdk.Plan{Schema: h.schema, Raw: tenantSettingsTestRaw(h.t, h.schema, planned, true)},
		State:  state,
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: h.schema, Raw: state.Raw.Copy()}}
	h.r.Update(ctx, req, resp)
	return resp
}

func (h *tenantSettingsTestHarness) read(state tfsdk.State) tfsdk.State {
	h.t.Helper()
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: h.schema, Raw: state.Raw.Copy()}}
	h.r.Read(context.Background(), resource.ReadRequest{State: state}, resp)
	tenantSettingsTestNoErrors(h.t, "read", resp.Diagnostics)
	return resp.State
}

func (h *tenantSettingsTestHarness) checkState(step string, state tfsdk.State) {
	h.t.Helper()
	if !state.Raw.IsFullyKnown() {
		h.t.Fatalf("%s: state has unknown values: %s", step, state.Raw)
	}
	var id types.String
	tenantSettingsTestNoErrors(h.t, step, state.GetAttribute(context.Background(), path.Root("id"), &id))
	if id.ValueString() != h.spec.ID {
		h.t.Fatalf("%s: unexpected id %s", step, id)
	}
}

func (h *tenantSettingsTestHarness) checkValues(step string, state tfsdk.State, want map[string]attr.Value) {
	h.t.Helper()
	got := h.values(state)
	for name, v := range want {
		if !got[name].Equal(v) {
			h.t.Fatalf("%s: %s = %s, want the planned value %s", step, name, got[name], v)
		}
	}
}

func (h *tenantSettingsTestHarness) writableFields() []*tenantSettingsField {
	var fields []*tenantSettingsField
	for i := range h.spec.Fields {
		if !h.spec.Fields[i].ReadOnly {
			fields = append(fields, &h.spec.Fields[i])
		}
	}
	return fields
}

func (h *tenantSettingsTestHarness) checkWrite(step string, w tenantSettingsTestRequest) {
	h.t.Helper()
	wantMethod := http.MethodPut
	if h.spec.Mode == tenantSettingsPatch {
		wantMethod = http.MethodPatch
	}
	if w.Method != wantMethod {
		h.t.Fatalf("%s: unexpected method %s, want %s", step, w.Method, wantMethod)
	}
	if wantMethod == http.MethodPatch && w.ContentType != "application/json-patch+json" {
		h.t.Fatalf("%s: unexpected content type %s", step, w.ContentType)
	}
	if (w.Experimental == "true") != h.spec.Experimental {
		h.t.Fatalf("%s: unexpected experimental header %q", step, w.Experimental)
	}
	if wantMethod == http.MethodPut {
		_, sent := w.Body.(map[string]interface{})[tenantSettingsTestUnmanagedKey]
		if sent != (h.spec.Mode == tenantSettingsPutMerged) {
			h.t.Fatalf("%s: unmanaged field sent = %v in %v", step, sent, w.Body)
		}
	}
}

// tenantSettingsTestLifecycle checks the behaviour shared by all tenant settings resources:
//   - Create without configured settings writes nothing and takes every value from the API.
//   - Create with every writable setting configured sends them in one request and keeps the
//     planned values; a following Read shows no drift.
//   - Update of one setting sends only that setting (JSON Patch) or keeps the other values (PUT).
//   - Delete only warns, Import sets the fixed ID.
func tenantSettingsTestLifecycle(t *testing.T, newResource func() resource.Resource, apiPath, doc string) *tenantSettingsTestHarness {
	t.Helper()
	h := newTenantSettingsTestHarness(t, newResource, apiPath, doc)

	h.create(nil)
	if writes := h.api.writes(); len(writes) != 0 {
		t.Fatalf("create without settings must not write, got %+v", writes)
	}
	for _, r := range h.api.requests {
		if (r.Experimental == "true") != h.spec.Experimental {
			t.Fatalf("unexpected experimental header %q on %s", r.Experimental, r.Method)
		}
	}

	configured := map[string]attr.Value{}
	for _, f := range h.writableFields() {
		configured[f.Name] = tenantSettingsTestValue(f, 1)
	}
	h.api.requests = nil
	state := h.create(configured)
	writes := h.api.writes()
	if len(writes) != 1 {
		t.Fatalf("create: expected one write, got %+v", writes)
	}
	h.checkWrite("create", writes[0])
	h.checkValues("create", state, configured)

	refreshed := h.read(state)
	if !refreshed.Raw.Equal(state.Raw) {
		t.Fatalf("read after create shows drift:\n%s\n%s", state.Raw, refreshed.Raw)
	}

	// Update changes the first setting; a second configured setting is unchanged and not sent.
	first := h.writableFields()[0]
	changed := map[string]attr.Value{first.Name: tenantSettingsTestValue(first, 2)}
	updateConfig := map[string]attr.Value{first.Name: changed[first.Name]}
	if fields := h.writableFields(); len(fields) > 1 {
		updateConfig[fields[1].Name] = configured[fields[1].Name]
	}
	h.api.requests = nil
	updated := h.update(refreshed, updateConfig)
	writes = h.api.writes()
	if len(writes) != 1 {
		t.Fatalf("update: expected one write, got %+v", writes)
	}
	h.checkWrite("update", writes[0])
	h.checkValues("update", updated, changed)
	want, _ := tenantSettingsToAPI(first, changed[first.Name])
	if h.spec.Mode == tenantSettingsPatch {
		ops, _ := writes[0].Body.([]interface{})
		if len(ops) != 1 {
			t.Fatalf("update: expected one patch operation, got %v", writes[0].Body)
		}
		op := ops[0].(map[string]interface{})
		if op["path"] != tenantSettingsPointer(first.Path) {
			t.Fatalf("update: unexpected patch operation %v", op)
		}
	} else if got, _ := tenantSettingsLookup(writes[0].Body, first.Path); !tenantSettingsTestJSONEqual(got, want) {
		t.Fatalf("update: %v sent for %s, want %v", got, first.Name, want)
	}
	// The other settings keep the values from the previous apply.
	unchanged := h.values(refreshed)
	delete(unchanged, first.Name)
	for i := range h.spec.Fields {
		if h.spec.Fields[i].ReadOnly {
			delete(unchanged, h.spec.Fields[i].Name)
		}
	}
	h.checkValues("update", updated, unchanged)

	deleteResp := &resource.DeleteResponse{}
	h.r.Delete(context.Background(), resource.DeleteRequest{State: updated}, deleteResp)
	if deleteResp.Diagnostics.HasError() || deleteResp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("delete: expected a warning, got %v", deleteResp.Diagnostics)
	}

	imported := h.read(h.imported())
	h.checkState("import", imported)
	return h
}

// imported returns the state after ImportState, before the following Read.
func (h *tenantSettingsTestHarness) imported() tfsdk.State {
	h.t.Helper()
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: h.schema, Raw: tenantSettingsTestRaw(h.t, h.schema, nil, false)}}
	h.r.ImportState(context.Background(), resource.ImportStateRequest{ID: "anything"}, resp)
	tenantSettingsTestNoErrors(h.t, "import", resp.Diagnostics)
	var id types.String
	tenantSettingsTestNoErrors(h.t, "import", resp.State.GetAttribute(context.Background(), path.Root("id"), &id))
	if id.ValueString() != h.spec.ID {
		h.t.Fatalf("import: unexpected id %s", id)
	}
	return resp.State
}

func tenantSettingsTestJSONEqual(a, b interface{}) bool {
	ea, _ := json.Marshal(a)
	eb, _ := json.Marshal(b)
	var da, db interface{}
	_ = json.Unmarshal(ea, &da)
	_ = json.Unmarshal(eb, &db)
	return reflect.DeepEqual(da, db)
}

func TestTenantSettingsMergeAndProjectJSON(t *testing.T) {
	var current, configured interface{}
	_ = json.Unmarshal([]byte(`{"a":{"x":1,"y":2},"b":[1,2],"c":"keep"}`), &current)
	_ = json.Unmarshal([]byte(`{"a":{"x":3},"b":[3]}`), &configured)
	merged := tenantSettingsMergeJSON(current, configured)
	if !tenantSettingsTestJSONEqual(merged, map[string]interface{}{"a": map[string]interface{}{"x": 3, "y": 2}, "b": []interface{}{3}, "c": "keep"}) {
		t.Fatalf("unexpected merge %v", merged)
	}

	projected := tenantSettingsProjectJSON(configured, merged, false)
	if !tenantSettingsTestJSONEqual(projected, configured) {
		t.Fatalf("unexpected projection %v", projected)
	}
	var secret interface{}
	_ = json.Unmarshal([]byte(`{"skey":"s3cr3t","ikey":"i"}`), &secret)
	kept := tenantSettingsProjectJSON(secret, map[string]interface{}{"ikey": "i"}, true)
	if !tenantSettingsTestJSONEqual(kept, secret) {
		t.Fatalf("secret keys must be kept, got %v", kept)
	}
	dropped := tenantSettingsProjectJSON(secret, map[string]interface{}{"ikey": "i"}, false)
	if !tenantSettingsTestJSONEqual(dropped, map[string]interface{}{"ikey": "i"}) {
		t.Fatalf("removed keys must show as drift, got %v", dropped)
	}
}

// Arrays of the same length are projected element by element: keys the API adds to array elements
// (e.g. approverId: null in approval schemes) are not drift.
func TestTenantSettingsProjectJSONArrays(t *testing.T) {
	var prior, api interface{}
	_ = json.Unmarshal([]byte(`{"a":{"schemes":[{"type":"MANAGER"},{"type":"OWNER","enabled":false}]}}`), &prior)
	_ = json.Unmarshal([]byte(`{"a":{"schemes":[{"type":"MANAGER","id":null},{"type":"OWNER","id":"x"}],"other":1},"b":2}`), &api)
	if projected := tenantSettingsProjectJSON(prior, api, false); !tenantSettingsTestJSONEqual(projected, prior) {
		t.Fatalf("keys added to array elements must not be drift, got %v", projected)
	}
	_ = json.Unmarshal([]byte(`{"a":{"schemes":[{"type":"GOVERNANCE_GROUP","id":"g"},{"type":"OWNER"}]}}`), &api)
	want := map[string]interface{}{"a": map[string]interface{}{"schemes": []interface{}{map[string]interface{}{"type": "GOVERNANCE_GROUP"}, map[string]interface{}{"type": "OWNER", "enabled": false}}}}
	if projected := tenantSettingsProjectJSON(prior, api, false); !tenantSettingsTestJSONEqual(projected, want) {
		t.Fatalf("changed configured keys of array elements must be drift, got %v", projected)
	}
	_ = json.Unmarshal([]byte(`{"a":{"schemes":[{"type":"MANAGER"}]}}`), &api)
	if projected := tenantSettingsProjectJSON(prior, api, false); !tenantSettingsTestJSONEqual(projected, api) {
		t.Fatalf("arrays of a different length are returned as is, got %v", projected)
	}

	field := &tenantSettingsField{Name: "x_json", Kind: tenantSettingsJSONObject}
	priorValue := types.StringValue(`{"a": {"schemes": [{"type": "MANAGER"}], "required": false}}`)
	raw := map[string]interface{}{"a": map[string]interface{}{"schemes": []interface{}{map[string]interface{}{"type": "MANAGER", "id": nil}}}}
	if got := tenantSettingsFromAPI(field, priorValue, raw, true); !got.Equal(priorValue) {
		t.Fatalf("added keys and omitted false values must keep the prior value, got %s", got)
	}
}

// Default JSON Patch operations replace existing members and add missing ones.
func TestTenantSettingsPatchOp(t *testing.T) {
	current := map[string]interface{}{"present": nil, "nested": map[string]interface{}{"x": 1}, "null": nil}
	for _, tc := range []struct {
		path []string
		want jsonPatchOp
	}{
		{[]string{"present"}, jsonPatchOp{Op: "replace", Path: "/present", Value: "v"}},
		{[]string{"missing"}, jsonPatchOp{Op: "add", Path: "/missing", Value: "v"}},
		{[]string{"nested", "x"}, jsonPatchOp{Op: "replace", Path: "/nested/x", Value: "v"}},
		{[]string{"nested", "y"}, jsonPatchOp{Op: "add", Path: "/nested/y", Value: "v"}},
		{[]string{"absent", "y"}, jsonPatchOp{Op: "add", Path: "/absent", Value: map[string]interface{}{"y": "v"}}},
		{[]string{"null", "y"}, jsonPatchOp{Op: "replace", Path: "/null", Value: map[string]interface{}{"y": "v"}}},
	} {
		if got := tenantSettingsPatchOp(current, tc.path, "v"); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v: got %+v, want %+v", tc.path, got, tc.want)
		}
	}
}

func TestTenantSettingsFromAPIKeepsEquivalentValues(t *testing.T) {
	jsonField := &tenantSettingsField{Name: "x_json", Kind: tenantSettingsJSONObject}
	prior := types.StringValue(`{"a": {"x": 1}}`)
	api := map[string]interface{}{"a": map[string]interface{}{"x": float64(1), "y": float64(2)}, "b": true}
	if got := tenantSettingsFromAPI(jsonField, prior, api, true); !got.Equal(prior) {
		t.Fatalf("configured keys that match must keep the prior value, got %s", got)
	}
	drift := map[string]interface{}{"a": map[string]interface{}{"x": float64(5)}}
	if got := tenantSettingsFromAPI(jsonField, prior, drift, true); got.(types.String).ValueString() != `{"a":{"x":5}}` {
		t.Fatalf("drift of configured keys must be shown, got %s", got)
	}
	if got := tenantSettingsFromAPI(jsonField, nil, api, true); !jsonSemanticallyEqual([]byte(got.(types.String).ValueString()), []byte(`{"a":{"x":1,"y":2},"b":true}`)) {
		t.Fatalf("without a prior value the whole object is returned, got %s", got)
	}

	floatField := &tenantSettingsField{Name: "f", Kind: tenantSettingsFloat64}
	if got := tenantSettingsFromAPI(floatField, types.Float64Value(0.7), float64(float32(0.7)), true); !got.Equal(types.Float64Value(0.7)) {
		t.Fatalf("single precision API values must keep the prior value, got %s", got)
	}

	listField := &tenantSettingsField{Name: "l", Kind: tenantSettingsStringList}
	priorList := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("b"), types.StringValue("a")})
	if got := tenantSettingsFromAPI(listField, priorList, []interface{}{"a", "b"}, true); !got.Equal(priorList) {
		t.Fatalf("reordered lists must keep the prior order, got %s", got)
	}

	secretField := &tenantSettingsField{Name: "s", Kind: tenantSettingsString, WriteOnly: true}
	if got := tenantSettingsFromAPI(secretField, types.StringValue("s3cr3t"), nil, false); !got.Equal(types.StringValue("s3cr3t")) {
		t.Fatalf("secrets not returned by the API must be kept, got %s", got)
	}
	if got := tenantSettingsFromAPI(secretField, nil, nil, false); !got.IsNull() {
		t.Fatalf("unknown secrets not returned by the API must be null, got %s", got)
	}
}

// Trigger settings keep the known value when the API resets them.
func TestTenantSettingsFromAPITriggerKeepsPrior(t *testing.T) {
	field := &tenantSettingsField{Name: "t", Kind: tenantSettingsBool, Trigger: true}
	if got := tenantSettingsFromAPI(field, types.BoolValue(true), false, true); !got.Equal(types.BoolValue(true)) {
		t.Fatalf("a reset trigger must keep the prior value, got %s", got)
	}
	if got := tenantSettingsFromAPI(field, nil, true, true); !got.Equal(types.BoolValue(true)) {
		t.Fatalf("without a prior value the API value is used, got %s", got)
	}
}
