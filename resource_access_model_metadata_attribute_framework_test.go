package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const accessModelMetadataAttributeTestResponse = `{"key":"iscPrivacy","name":"Privacy","multiselect":false,"status":"active","type":"governance",
"objectTypes":["all"],"description":"Privacy level","values":[{"value":"public","name":"Public","status":"active"},{"value":"private","name":"Private","status":"active"}]}`

func accessModelMetadataAttributeTestValues(t *testing.T, values ...AccessModelMetadataAttributeValueModel) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(context.Background(), accessModelMetadataAttributeValueObjectType, values)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return list
}

func accessModelMetadataAttributeTestAPI(t *testing.T) *AccessModelMetadataAttributeDTO {
	t.Helper()
	var attribute AccessModelMetadataAttributeDTO
	if err := json.Unmarshal([]byte(accessModelMetadataAttributeTestResponse), &attribute); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	return &attribute
}

func TestAccessModelMetadataAttributeClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotContentType = r.Method, r.URL.Path, string(body), r.Header.Get("Content-Type")
		_, _ = w.Write([]byte(accessModelMetadataAttributeTestResponse))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := AccessModelMetadataAttributeResourceModel{
		Key: types.StringUnknown(), Name: types.StringValue("Privacy"), Type: types.StringValue("governance"),
		ObjectTypes: types.ListUnknown(types.StringType), Description: types.StringNull(), Multiselect: types.BoolUnknown(), Status: types.StringUnknown(),
		Values: accessModelMetadataAttributeTestValues(t, AccessModelMetadataAttributeValueModel{
			Value: types.StringValue("public"), Name: types.StringValue("Public"), Status: types.StringUnknown(),
		}),
	}
	created, err := client.CreateAccessModelMetadataAttribute(ctx, accessModelMetadataAttributeFromModel(ctx, plan, &diags))
	if err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/access-model-metadata/attributes" ||
		gotBody != `{"name":"Privacy","type":"governance","values":[{"value":"public","name":"Public"}]}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	if created.Key != "iscPrivacy" || len(created.Values) != 2 {
		t.Fatalf("unexpected response %+v", created)
	}

	if _, err := client.UpdateAccessModelMetadataAttribute(ctx, "iscPrivacy", []jsonPatchOp{{Op: "replace", Path: "/name", Value: "New"}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/access-model-metadata/attributes/iscPrivacy" || gotContentType != "application/json-patch+json" {
		t.Fatalf("unexpected update request %s %s %s", gotMethod, gotPath, gotContentType)
	}
}

func TestAccessModelMetadataAttributeState(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	plan := AccessModelMetadataAttributeResourceModel{
		Key: types.StringUnknown(), Name: types.StringValue("Privacy"), Type: types.StringUnknown(),
		ObjectTypes: types.ListUnknown(types.StringType), Description: types.StringNull(), Multiselect: types.BoolUnknown(), Status: types.StringUnknown(),
		Values: accessModelMetadataAttributeTestValues(t,
			AccessModelMetadataAttributeValueModel{Value: types.StringValue("private"), Name: types.StringValue("Private"), Status: types.StringUnknown()},
			AccessModelMetadataAttributeValueModel{Value: types.StringValue("new"), Name: types.StringValue("New"), Status: types.StringUnknown()},
		),
	}
	accessModelMetadataAttributeResolveApply(ctx, &plan, accessModelMetadataAttributeTestAPI(t), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if plan.ID.ValueString() != "iscPrivacy" || plan.Key.ValueString() != "iscPrivacy" || plan.Type.ValueString() != "governance" ||
		plan.Status.ValueString() != "active" || plan.Multiselect.IsUnknown() || len(plan.ObjectTypes.Elements()) != 1 || !plan.Description.IsNull() {
		t.Fatalf("unexpected state after apply %+v", plan)
	}
	var values []AccessModelMetadataAttributeValueModel
	diags.Append(plan.Values.ElementsAs(ctx, &values, false)...)
	if len(values) != 2 || values[0].Value.ValueString() != "private" || values[0].Status.ValueString() != "active" ||
		values[1].Status.ValueString() != accessModelMetadataAttributeDefaultValueStatus {
		t.Fatalf("expected planned values with resolved statuses, got %+v", values)
	}

	// Read orders values like the prior state and refreshes drift.
	setAccessModelMetadataAttributeState(ctx, &plan, accessModelMetadataAttributeTestAPI(t), &diags)
	values = nil
	diags.Append(plan.Values.ElementsAs(ctx, &values, false)...)
	if len(values) != 2 || values[0].Value.ValueString() != "private" || values[1].Value.ValueString() != "public" || plan.Description.ValueString() != "Privacy level" {
		t.Fatalf("unexpected values after read %+v %s", values, plan.Description)
	}

	// An unset description stays null when the API has none.
	api := accessModelMetadataAttributeTestAPI(t)
	api.Description = ""
	plan.Description = types.StringNull()
	setAccessModelMetadataAttributeState(ctx, &plan, api, &diags)
	if !plan.Description.IsNull() {
		t.Fatalf("expected description to stay null, got %s", plan.Description)
	}
}

func TestAccessModelMetadataAttributePatches(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := AccessModelMetadataAttributeResourceModel{
		Key: types.StringValue("iscPrivacy"), Name: types.StringValue("Privacy"), Description: types.StringValue("Old"),
		Multiselect: types.BoolValue(false),
		Values: accessModelMetadataAttributeTestValues(t,
			AccessModelMetadataAttributeValueModel{Value: types.StringValue("public"), Name: types.StringValue("Public"), Status: types.StringValue("active")},
		),
	}
	plan := state
	plan.Values = accessModelMetadataAttributeTestValues(t,
		AccessModelMetadataAttributeValueModel{Value: types.StringValue("public"), Name: types.StringValue("Public"), Status: types.StringUnknown()},
	)
	if ops := accessModelMetadataAttributePatches(ctx, plan, state, &diags); len(ops) != 0 {
		t.Fatalf("expected no patches, got %+v", ops)
	}

	plan.Name = types.StringValue("Data Privacy")
	plan.Description = types.StringNull()
	plan.Multiselect = types.BoolValue(true)
	plan.Values = accessModelMetadataAttributeTestValues(t,
		AccessModelMetadataAttributeValueModel{Value: types.StringValue("public"), Name: types.StringValue("Public Data"), Status: types.StringUnknown()},
	)
	encoded, _ := json.Marshal(accessModelMetadataAttributePatches(ctx, plan, state, &diags))
	want := `[{"op":"replace","path":"/name","value":"Data Privacy"},{"op":"replace","path":"/description","value":""},` +
		`{"op":"replace","path":"/multiselect","value":true},{"op":"replace","path":"/values","value":[{"value":"public","name":"Public Data"}]}]`
	if string(encoded) != want {
		t.Fatalf("unexpected patches\n%s\nwant\n%s", encoded, want)
	}
}

func TestAccessModelMetadataAttributeDeleteWarns(t *testing.T) {
	resp := &resource.DeleteResponse{}
	(&AccessModelMetadataAttributeResource{}).Delete(context.Background(), resource.DeleteRequest{}, resp)
	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("expected a single warning, got %v", resp.Diagnostics)
	}
}

// accessModelMetadataAttributeTestRaw encodes a model with the resource schema; nil encodes null.
func accessModelMetadataAttributeTestRaw(t *testing.T, data *AccessModelMetadataAttributeResourceModel) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&AccessModelMetadataAttributeResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	if data != nil {
		if diags := state.Set(ctx, data); diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
	}
	return state
}

func TestAccessModelMetadataAttributeModifyPlan(t *testing.T) {
	ctx := context.Background()
	objectTypes, _ := types.ListValueFrom(ctx, types.StringType, []string{"all"})
	prior := AccessModelMetadataAttributeResourceModel{
		ID: types.StringValue("iscPrivacy"), Key: types.StringValue("iscPrivacy"), Name: types.StringValue("Privacy"),
		Type: types.StringValue("governance"), ObjectTypes: objectTypes, Description: types.StringNull(),
		Multiselect: types.BoolValue(false), Status: types.StringValue("active"), Values: types.ListNull(accessModelMetadataAttributeValueObjectType),
	}
	run := func(plan, state *AccessModelMetadataAttributeResourceModel) *resource.ModifyPlanResponse {
		planState := accessModelMetadataAttributeTestRaw(t, plan)
		req := resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: planState.Schema, Raw: planState.Raw}, State: accessModelMetadataAttributeTestRaw(t, state)}
		resp := &resource.ModifyPlanResponse{Plan: req.Plan}
		(&AccessModelMetadataAttributeResource{}).ModifyPlan(ctx, req, resp)
		return resp
	}

	// Create, destroy and updatable changes are allowed.
	created := prior
	created.ID, created.Key, created.Type, created.Status, created.ObjectTypes = types.StringUnknown(), types.StringUnknown(), types.StringUnknown(), types.StringUnknown(), types.ListUnknown(types.StringType)
	updated := prior
	updated.Name = types.StringValue("New name")
	updated.Multiselect = types.BoolValue(true)
	for name, resp := range map[string]*resource.ModifyPlanResponse{
		"create": run(&created, nil), "destroy": run(nil, &prior), "update": run(&updated, &prior),
	} {
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: unexpected diagnostics: %v", name, resp.Diagnostics)
		}
	}

	// Changes of attributes that can neither be updated nor replaced fail the plan.
	changed := prior
	changed.Key = types.StringValue("other")
	changed.Type = types.StringValue("custom")
	changed.Status = types.StringValue("inactive")
	changed.ObjectTypes, _ = types.ListValueFrom(ctx, types.StringType, []string{"entitlement"})
	if resp := run(&changed, &prior); resp.Diagnostics.ErrorsCount() != 4 {
		t.Fatalf("expected an error per changed attribute, got %v", resp.Diagnostics)
	}
	// Unknown planned values are checked again during apply.
	unknown := prior
	unknown.Key = types.StringUnknown()
	if got := accessModelMetadataAttributeImmutableChanges(unknown, prior); len(got) != 0 {
		t.Fatalf("expected unknown values to be skipped, got %v", got)
	}
	if got := accessModelMetadataAttributeImmutableChanges(changed, prior); len(got) != 4 || got[0] != "key" || got[3] != "object_types" {
		t.Fatalf("unexpected changed attributes %v", got)
	}
}
