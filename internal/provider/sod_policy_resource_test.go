package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const sodPolicyTestResponse = `{"id":"sp-1","name":"AP vs AR","created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z",
"description":null,"ownerRef":{"type":"IDENTITY","id":"o-1","name":"Owner"},"externalPolicyReference":null,
"policyQuery":"@access(id:e-1) AND @access(id:e-2)","compensatingControls":null,"correctionAdvice":null,"state":"NOT_ENFORCED",
"tags":[],"creatorId":"c-1","modifierId":"m-1","violationOwnerAssignmentConfig":{"assignmentRule":"MANAGER","ownerRef":null},
"scheduled":false,"type":"CONFLICTING_ACCESS_BASED","conflictingAccessCriteria":{
"leftCriteria":{"name":"left","criteriaList":[{"type":"ENTITLEMENT","id":"e-1","name":"Payables"}]},
"rightCriteria":{"name":"right","criteriaList":[{"type":"ENTITLEMENT","id":"e-2","name":"Receivables"}]}}}`

const sodPolicyTestCriteria = `{"leftCriteria":{"name":"left","criteriaList":[{"type":"ENTITLEMENT","id":"e-1"}]},"rightCriteria":{"name":"right","criteriaList":[{"type":"ENTITLEMENT","id":"e-2"}]}}`

func sodPolicyTestOwner(t *testing.T, id, name string) types.List {
	t.Helper()
	nameValue := types.StringNull()
	if name != "" {
		nameValue = types.StringValue(name)
	}
	list, diags := types.ListValueFrom(context.Background(), objectInfoObjectType, []OwnerModel{{
		ID: types.StringValue(id), Type: types.StringValue("IDENTITY"), Name: nameValue,
	}})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return list
}

// sodPolicyTestModel returns a planned conflicting access based policy with typed nulls.
func sodPolicyTestModel(t *testing.T) SodPolicyModel {
	return SodPolicyModel{
		ID:                             types.StringValue("sp-1"),
		Name:                           types.StringValue("AP vs AR"),
		Description:                    types.StringNull(),
		OwnerRef:                       sodPolicyTestOwner(t, "o-1", ""),
		ExternalPolicyReference:        types.StringNull(),
		PolicyQuery:                    types.StringValue("@access(id:e-1) AND @access(id:e-2)"),
		CompensatingControls:           types.StringNull(),
		CorrectionAdvice:               types.StringNull(),
		State:                          types.StringValue("NOT_ENFORCED"),
		Tags:                           types.ListNull(types.StringType),
		ViolationOwnerAssignmentConfig: types.ListNull(sodPolicyViolationOwnerConfigObjectType),
		Scheduled:                      types.BoolValue(false),
		Type:                           types.StringValue(sodPolicyTypeConflictingAccess),
		ConflictingAccessCriteriaJSON:  types.StringValue(sodPolicyTestCriteria),
		CreatorID:                      types.StringValue("c-1"),
		ModifierID:                     types.StringValue("m-1"),
		Created:                        types.StringValue("2026-01-01T00:00:00Z"),
		Modified:                       types.StringValue("2026-01-01T00:00:00Z"),
	}
}

func TestSodPolicyClient(t *testing.T) {
	var requests []string
	var putBody, patchContentType, deleteQuery string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodPut:
			putBody = string(body)
		case http.MethodPatch:
			patchContentType = r.Header.Get("Content-Type")
		case http.MethodDelete:
			deleteQuery = r.URL.RawQuery
			w.WriteHeader(http.StatusNoContent)
			return
		case http.MethodPost:
			if string(body) != `{"name":"AP vs AR","ownerRef":{"type":"IDENTITY","id":"o-1"},"state":"NOT_ENFORCED","type":"CONFLICTING_ACCESS_BASED","conflictingAccessCriteria":{"leftCriteria":{"criteriaList":[{"id":"e-1","type":"ENTITLEMENT"}],"name":"left"},"rightCriteria":{"criteriaList":[{"id":"e-2","type":"ENTITLEMENT"}],"name":"right"}}}` {
				t.Errorf("unexpected create body %s", body)
			}
		}
		_, _ = w.Write([]byte(sodPolicyTestResponse))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := sodPolicyTestModel(t)
	plan.PolicyQuery, plan.Scheduled = types.StringUnknown(), types.BoolUnknown()
	created, err := client.CreateSodPolicy(ctx, sodPolicyFromModel(ctx, plan, &diags))
	if err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	setSodPolicyComputed(&plan, created)
	if plan.PolicyQuery.ValueString() != created.PolicyQuery || plan.Scheduled.IsUnknown() || !plan.Description.IsNull() {
		t.Fatalf("unexpected state after create %+v", plan)
	}

	// PUT keeps fields that are not managed, e.g. the modifier and the violation owner configuration.
	general := sodPolicyTestModel(t)
	general.Type = types.StringValue(sodPolicyTypeGeneral)
	general.ConflictingAccessCriteriaJSON = types.StringNull()
	general.PolicyQuery = types.StringValue("@access(id:e-9)")
	general.Description = types.StringNull()
	if _, err := client.UpdateSodPolicy(ctx, "sp-1", sodPolicyManagedFields(ctx, general, &diags)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	var sent map[string]interface{}
	_ = json.Unmarshal([]byte(putBody), &sent)
	if sent["policyQuery"] != "@access(id:e-9)" || sent["modifierId"] != "m-1" || sent["description"] != nil || sent["type"] != sodPolicyTypeGeneral {
		t.Fatalf("unexpected merged PUT body %s", putBody)
	}
	if config, ok := sent["violationOwnerAssignmentConfig"].(map[string]interface{}); !ok || config["assignmentRule"] != "MANAGER" {
		t.Fatalf("expected unmanaged violation owner configuration to be kept, got %s", putBody)
	}

	if _, err := client.PatchSodPolicy(ctx, "sp-1", []jsonPatchOp{{Op: "replace", Path: "/name", Value: "x"}}); err != nil || patchContentType != "application/json-patch+json" {
		t.Fatalf("unexpected patch request %s (%v)", patchContentType, err)
	}
	if err := client.DeleteSodPolicy(ctx, "sp-1"); err != nil || deleteQuery != "logical=false" {
		t.Fatalf("unexpected delete request %s (%v)", deleteQuery, err)
	}
	want := "POST /v2026/sod-policies,GET /v2026/sod-policies/sp-1,PUT /v2026/sod-policies/sp-1,PATCH /v2026/sod-policies/sp-1,DELETE /v2026/sod-policies/sp-1"
	if got := strings.Join(requests, ","); got != want {
		t.Fatalf("unexpected requests %s", got)
	}
}

func TestSodPolicyReadState(t *testing.T) {
	ctx := context.Background()
	var policy SodPolicy
	if err := json.Unmarshal([]byte(sodPolicyTestResponse), &policy); err != nil {
		t.Fatal(err)
	}
	var diags diag.Diagnostics
	data := sodPolicyTestModel(t)
	prior := data
	setSodPolicyState(ctx, &data, &policy, false, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.ConflictingAccessCriteriaJSON.Equal(prior.ConflictingAccessCriteriaJSON) {
		t.Fatalf("expected criteria with resolved names to keep the prior value, got %s", data.ConflictingAccessCriteriaJSON)
	}
	if !data.OwnerRef.Equal(prior.OwnerRef) || !data.Tags.IsNull() || !data.Description.IsNull() || !data.ViolationOwnerAssignmentConfig.IsNull() {
		t.Fatalf("expected unset optional values to stay null, got %+v", data)
	}

	// The data source reports everything the API returns.
	all := SodPolicyModel{
		OwnerRef: types.ListNull(objectInfoObjectType), Tags: types.ListNull(types.StringType),
		ViolationOwnerAssignmentConfig: types.ListNull(sodPolicyViolationOwnerConfigObjectType),
	}
	setSodPolicyState(ctx, &all, &policy, true, &diags)
	var configs []SodPolicyViolationOwnerConfigModel
	diags.Append(all.ViolationOwnerAssignmentConfig.ElementsAs(ctx, &configs, false)...)
	var owners []OwnerModel
	diags.Append(all.OwnerRef.ElementsAs(ctx, &owners, false)...)
	if diags.HasError() || len(configs) != 1 || configs[0].AssignmentRule.ValueString() != "MANAGER" || len(owners) != 1 || owners[0].Name.ValueString() != "Owner" {
		t.Fatalf("unexpected data source state %+v (%v)", all, diags)
	}

	// A configured violation owner configuration is refreshed.
	configured, d := types.ListValueFrom(ctx, sodPolicyViolationOwnerConfigObjectType, []SodPolicyViolationOwnerConfigModel{{
		AssignmentRule: types.StringValue("STATIC"), OwnerRef: sodPolicyTestOwner(t, "g-1", ""),
	}})
	diags.Append(d...)
	data.ViolationOwnerAssignmentConfig = configured
	setSodPolicyState(ctx, &data, &policy, false, &diags)
	diags.Append(data.ViolationOwnerAssignmentConfig.ElementsAs(ctx, &configs, false)...)
	if diags.HasError() || len(configs) != 1 || configs[0].AssignmentRule.ValueString() != "MANAGER" || !configs[0].OwnerRef.IsNull() {
		t.Fatalf("expected drift of the configured block to be reported, got %+v (%v)", configs, diags)
	}
}

func TestSodPolicyPatchOps(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := sodPolicyTestModel(t)
	plan := sodPolicyTestModel(t)
	plan.ModifierID, plan.Modified = types.StringUnknown(), types.StringUnknown()
	plan.ConflictingAccessCriteriaJSON = types.StringValue(`{"rightCriteria": {"criteriaList": [{"id": "e-2", "type": "ENTITLEMENT"}], "name": "right"}, "leftCriteria": {"criteriaList": [{"id": "e-1", "type": "ENTITLEMENT"}], "name": "left"}}`)
	if ops := sodPolicyPatchOps(ctx, plan, state, &diags); len(ops) != 0 {
		t.Fatalf("expected no ops for reformatted JSON, got %+v", ops)
	}

	state.Description = types.StringValue("old")
	plan.Tags, _ = types.ListValueFrom(ctx, types.StringType, []string{"SOX"})
	plan.ConflictingAccessCriteriaJSON = types.StringValue(`{"leftCriteria":{"name":"left","criteriaList":[{"type":"ENTITLEMENT","id":"e-3"}]},"rightCriteria":{"name":"right","criteriaList":[{"type":"ENTITLEMENT","id":"e-2"}]}}`)
	plan.PolicyQuery = types.StringUnknown()
	ops := sodPolicyPatchOps(ctx, plan, state, &diags)
	if diags.HasError() || len(ops) != 3 || ops[0].Op != "remove" || ops[0].Path != "/description" || ops[1].Path != "/tags" || ops[2].Path != "/conflictingAccessCriteria" {
		t.Fatalf("unexpected ops %+v (%v)", ops, diags)
	}
}

func sodPolicyTestSchemaState(t *testing.T, data SodPolicyModel) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&SodPolicyResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, &data); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return state
}

func TestSodPolicyQueryPlanModifier(t *testing.T) {
	ctx := context.Background()
	prior := sodPolicyTestModel(t)
	state := sodPolicyTestSchemaState(t, prior)

	run := func(criteria string) types.String {
		planned := prior
		planned.PolicyQuery = types.StringUnknown()
		planned.ConflictingAccessCriteriaJSON = types.StringValue(criteria)
		plan := sodPolicyTestSchemaState(t, planned)
		req := planmodifier.StringRequest{
			ConfigValue: types.StringNull(), StateValue: prior.PolicyQuery, PlanValue: types.StringUnknown(),
			Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}, State: state,
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		sodPolicyQueryPlanModifier{}.PlanModifyString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		return resp.PlanValue
	}
	if got := run(sodPolicyTestCriteria); !got.Equal(prior.PolicyQuery) {
		t.Fatalf("expected the generated query to be kept for unchanged criteria, got %s", got)
	}
	if got := run(`{"leftCriteria":{"name":"left","criteriaList":[{"type":"ENTITLEMENT","id":"e-3"}]}}`); !got.IsUnknown() {
		t.Fatalf("expected the generated query to be unknown for changed criteria, got %s", got)
	}
}

func TestSodPolicyValidateConfig(t *testing.T) {
	ctx := context.Background()
	validate := func(data SodPolicyModel) diag.Diagnostics {
		state := sodPolicyTestSchemaState(t, data)
		resp := &resource.ValidateConfigResponse{}
		(&SodPolicyResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: state.Schema, Raw: state.Raw}}, resp)
		return resp.Diagnostics
	}
	conflicting := sodPolicyTestModel(t)
	if diags := validate(conflicting); !diags.HasError() {
		t.Fatalf("expected an error for a configured query of a conflicting access based policy")
	}
	conflicting.PolicyQuery = types.StringNull()
	if diags := validate(conflicting); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	general := sodPolicyTestModel(t)
	general.Type = types.StringNull()
	general.PolicyQuery = types.StringNull()
	general.ConflictingAccessCriteriaJSON = types.StringNull()
	if diags := validate(general); !diags.HasError() {
		t.Fatalf("expected an error for a general policy without query")
	}
}

// With the MANAGER rule the API may return an owner reference without ID ({"type": "MANAGER"}).
// It is not shown when no owner_ref block is configured, so it does not cause a perpetual diff.
func TestSodPolicyViolationOwnerManagerRefWithoutBlock(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	rule := "MANAGER"
	api := &SodPolicyViolationOwnerConfig{AssignmentRule: &rule, OwnerRef: &SodPolicyRef{Type: "MANAGER"}}
	configured, d := types.ListValueFrom(ctx, sodPolicyViolationOwnerConfigObjectType, []SodPolicyViolationOwnerConfigModel{{
		AssignmentRule: types.StringValue("MANAGER"), OwnerRef: types.ListNull(objectInfoObjectType),
	}})
	diags.Append(d...)
	state := sodPolicyViolationOwnerConfigState(ctx, api, configured, false, &diags)
	if diags.HasError() || !state.Equal(configured) {
		t.Fatalf("expected the configured block to be kept, got %s (%v)", state, diags)
	}
	// The data source does not report the reference without ID either.
	all := sodPolicyViolationOwnerConfigState(ctx, api, types.ListNull(sodPolicyViolationOwnerConfigObjectType), true, &diags)
	var configs []SodPolicyViolationOwnerConfigModel
	diags.Append(all.ElementsAs(ctx, &configs, false)...)
	if diags.HasError() || len(configs) != 1 || configs[0].AssignmentRule.ValueString() != "MANAGER" || !configs[0].OwnerRef.IsNull() {
		t.Fatalf("unexpected data source state %+v (%v)", configs, diags)
	}

	// A STATIC owner reference is still refreshed.
	static := "STATIC"
	api = &SodPolicyViolationOwnerConfig{AssignmentRule: &static, OwnerRef: &SodPolicyRef{ID: "g-2", Type: "GOVERNANCE_GROUP"}}
	configured, d = types.ListValueFrom(ctx, sodPolicyViolationOwnerConfigObjectType, []SodPolicyViolationOwnerConfigModel{{
		AssignmentRule: types.StringValue("STATIC"), OwnerRef: sodPolicyTestOwner(t, "g-1", ""),
	}})
	diags.Append(d...)
	state = sodPolicyViolationOwnerConfigState(ctx, api, configured, false, &diags)
	diags.Append(state.ElementsAs(ctx, &configs, false)...)
	var owners []OwnerModel
	diags.Append(configs[0].OwnerRef.ElementsAs(ctx, &owners, false)...)
	if diags.HasError() || len(owners) != 1 || owners[0].ID.ValueString() != "g-2" {
		t.Fatalf("expected owner drift to be reported, got %+v (%v)", owners, diags)
	}
}
