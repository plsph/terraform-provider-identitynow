package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const lifecycleStateTestResponse = `{"id":"ls-1","name":"Active","technicalName":"active","description":"","enabled":true,
"identityCount":7,"emailNotificationOption":{"notifyManagers":false,"notifyAllAdmins":false,"notifySpecificUsers":false,"emailAddressList":[]},
"accountActions":[],"accessProfileIds":[],"identityState":null,"accessActionConfiguration":{"removeAllAccessEnabled":false},
"priority":10,"created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z"}`

func lifecycleStateTestAPI(t *testing.T) *LifecycleState {
	t.Helper()
	var state LifecycleState
	if err := json.Unmarshal([]byte(lifecycleStateTestResponse), &state); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	return &state
}

func lifecycleStateTestModel() LifecycleStateModel {
	return LifecycleStateModel{
		IdentityProfileID:       types.StringValue("ip-1"),
		Name:                    types.StringValue("Active"),
		TechnicalName:           types.StringValue("active"),
		Description:             types.StringNull(),
		Enabled:                 types.BoolUnknown(),
		IdentityState:           types.StringNull(),
		Priority:                types.Int64Unknown(),
		EmailNotificationOption: types.ListNull(lifecycleStateEmailNotificationObjectType),
		AccountActionsJSON:      types.StringNull(),
		AccessProfileIDs:        types.SetNull(types.StringType),
		RemoveAllAccessEnabled:  types.BoolUnknown(),
	}
}

func TestLifecycleStateClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotContentType = r.Method, r.URL.Path, string(body), r.Header.Get("Content-Type")
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"type":"LIFECYCLE_STATE","id":"ls-1","name":"Active"}`))
			return
		}
		_, _ = w.Write([]byte(lifecycleStateTestResponse))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	data := lifecycleStateTestModel()
	data.AccessProfileIDs, _ = types.SetValueFrom(ctx, types.StringType, []string{"ap-1"})
	data.AccountActionsJSON = types.StringValue(`[{"action":"ENABLE","allSources":true}]`)
	created, err := client.CreateLifecycleState(ctx, "ip-1", lifecycleStateFromModel(ctx, data, &diags))
	if err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/identity-profiles/ip-1/lifecycle-states" ||
		gotBody != `{"name":"Active","technicalName":"active","accountActions":[{"action":"ENABLE","allSources":true}],"accessProfileIds":["ap-1"]}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	if created.ID != "ls-1" || *created.IdentityCount != 7 {
		t.Fatalf("unexpected response %+v", created)
	}

	if _, err := client.UpdateLifecycleState(ctx, "ip-1", "ls-1", []jsonPatchOp{{Op: "replace", Path: "/enabled", Value: false}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/identity-profiles/ip-1/lifecycle-states/ls-1" || gotContentType != "application/json-patch+json" ||
		gotBody != `[{"op":"replace","path":"/enabled","value":false}]` {
		t.Fatalf("unexpected update request %s %s %s %s", gotMethod, gotPath, gotContentType, gotBody)
	}

	if err := client.DeleteLifecycleState(ctx, "ip-1", "ls-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/identity-profiles/ip-1/lifecycle-states/ls-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestLifecycleStateApplyAndRead(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	data := lifecycleStateTestModel()
	lifecycleStateResolveApply(&data, lifecycleStateTestAPI(t))
	if data.ID.ValueString() != "ls-1" || !data.Enabled.ValueBool() || data.Priority.ValueInt64() != 10 || data.RemoveAllAccessEnabled.ValueBool() ||
		data.IdentityCount.ValueInt64() != 7 || data.Modified.ValueString() == "" {
		t.Fatalf("unexpected state after apply %+v", data)
	}

	planned := lifecycleStateTestModel()
	planned.Enabled = types.BoolValue(false)
	planned.Priority = types.Int64Value(3)
	planned.RemoveAllAccessEnabled = types.BoolValue(true)
	lifecycleStateResolveApply(&planned, lifecycleStateTestAPI(t))
	if planned.Enabled.ValueBool() || planned.Priority.ValueInt64() != 3 || !planned.RemoveAllAccessEnabled.ValueBool() {
		t.Fatalf("expected planned values to be kept, got %+v", planned)
	}

	// Read keeps null for unset optional attributes and default email options.
	setLifecycleStateState(ctx, &data, lifecycleStateTestAPI(t), false, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.Description.IsNull() || !data.IdentityState.IsNull() || !data.EmailNotificationOption.IsNull() ||
		!data.AccountActionsJSON.IsNull() || !data.AccessProfileIDs.IsNull() {
		t.Fatalf("expected unset attributes to stay null, got %+v", data)
	}

	// Configured values are refreshed and equivalent JSON is kept.
	api := lifecycleStateTestAPI(t)
	api.AccountActions = json.RawMessage(`[{"allSources":true,"action":"DISABLE"}]`)
	api.EmailNotificationOption = &LifecycleStateEmailNotificationOption{NotifyManagers: true}
	data.AccountActionsJSON = types.StringValue(`[{"action": "DISABLE", "allSources": true}]`)
	prior := data.AccountActionsJSON
	setLifecycleStateState(ctx, &data, api, false, &diags)
	if !data.AccountActionsJSON.Equal(prior) {
		t.Fatalf("expected equivalent JSON to be kept, got %s", data.AccountActionsJSON)
	}
	var email []LifecycleStateEmailNotificationModel
	diags.Append(data.EmailNotificationOption.ElementsAs(ctx, &email, false)...)
	if len(email) != 1 || !email[0].NotifyManagers.ValueBool() || !email[0].EmailAddressList.IsNull() {
		t.Fatalf("unexpected email option %+v", email)
	}

	// The data source returns empty values too.
	var ds LifecycleStateModel
	setLifecycleStateState(ctx, &ds, lifecycleStateTestAPI(t), true, &diags)
	if ds.Description.IsNull() || ds.EmailNotificationOption.IsNull() || ds.AccessProfileIDs.IsNull() {
		t.Fatalf("expected data source values, got %+v", ds)
	}
}

func TestLifecycleStatePatches(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := lifecycleStateTestModel()
	state.Enabled = types.BoolValue(true)
	state.Priority = types.Int64Value(10)
	state.RemoveAllAccessEnabled = types.BoolValue(false)
	state.Description = types.StringValue("Old")
	state.AccountActionsJSON = types.StringValue(`[{"action":"ENABLE","allSources":true}]`)
	state.AccessProfileIDs, _ = types.SetValueFrom(ctx, types.StringType, []string{"ap-1"})

	plan := state
	plan.AccountActionsJSON = types.StringValue(`[{"allSources": true, "action": "ENABLE"}]`)
	if ops := lifecycleStatePatches(ctx, plan, state, &diags); len(ops) != 0 {
		t.Fatalf("expected no patches, got %+v", ops)
	}

	plan.Description = types.StringNull()
	plan.Enabled = types.BoolValue(false)
	plan.AccountActionsJSON = types.StringNull()
	plan.AccessProfileIDs = types.SetNull(types.StringType)
	plan.RemoveAllAccessEnabled = types.BoolValue(true)
	email, _ := types.ListValueFrom(ctx, lifecycleStateEmailNotificationObjectType, []LifecycleStateEmailNotificationModel{{
		NotifyManagers: types.BoolValue(true), NotifyAllAdmins: types.BoolValue(false), NotifySpecificUsers: types.BoolValue(false),
		EmailAddressList: types.ListNull(types.StringType),
	}})
	plan.EmailNotificationOption = email
	encoded, _ := json.Marshal(lifecycleStatePatches(ctx, plan, state, &diags))
	want := `[{"op":"remove","path":"/description"},{"op":"replace","path":"/enabled","value":false},` +
		`{"op":"replace","path":"/emailNotificationOption","value":{"notifyManagers":true,"notifyAllAdmins":false,"notifySpecificUsers":false,"emailAddressList":[]}},` +
		`{"op":"replace","path":"/accountActions","value":[]},{"op":"replace","path":"/accessProfileIds","value":[]},` +
		`{"op":"replace","path":"/accessActionConfiguration","value":{"removeAllAccessEnabled":true}}]`
	if string(encoded) != want {
		t.Fatalf("unexpected patches\n%s\nwant\n%s", encoded, want)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestLifecycleStateImport(t *testing.T) {
	ctx := context.Background()
	r := &LifecycleStateResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	newState := func() tfsdk.State {
		return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	}

	resp := &resource.ImportStateResponse{State: newState()}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "ip-1/ls-1"}, resp)
	var profileID, id types.String
	resp.State.GetAttribute(ctx, path.Root("identity_profile_id"), &profileID)
	resp.State.GetAttribute(ctx, path.Root("id"), &id)
	if resp.Diagnostics.HasError() || profileID.ValueString() != "ip-1" || id.ValueString() != "ls-1" {
		t.Fatalf("unexpected import %s %s %v", profileID, id, resp.Diagnostics)
	}

	for _, invalid := range []string{"ls-1", "/ls-1", "ip-1/", "a/b/c"} {
		resp := &resource.ImportStateResponse{State: newState()}
		r.ImportState(ctx, resource.ImportStateRequest{ID: invalid}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatalf("expected an error for import ID %q", invalid)
		}
	}
}
