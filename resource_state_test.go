package main

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func intPtr(v int) *int { return &v }

func TestPasswordPolicyStateKeepsPlanOnApplyAndRefreshesOnRead(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	api := &PasswordPolicy{ID: "pp-1", Name: "Policy", MinLength: intPtr(12), MaxLength: intPtr(64), UseDictionary: boolPtr(true)}

	planned := PasswordPolicyResourceModel{
		Name:          types.StringValue("Policy"),
		Description:   types.StringNull(),
		MinLength:     types.Int64Value(10),
		MaxLength:     types.Int64Unknown(),
		UseDictionary: types.BoolUnknown(),
		SourceIDs:     types.ListNull(types.StringType),
		DateCreated:   types.StringUnknown(),
		LastUpdated:   types.StringUnknown(),
	}
	(&PasswordPolicyResource{}).setStateFromAPI(ctx, &planned, api, false, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if planned.MinLength.ValueInt64() != 10 {
		t.Errorf("expected planned min_length to be kept, got %s", planned.MinLength)
	}
	if planned.MaxLength.ValueInt64() != 64 || !planned.UseDictionary.ValueBool() {
		t.Errorf("expected unknown values to be resolved from the API, got %s %s", planned.MaxLength, planned.UseDictionary)
	}
	if planned.MinAlpha.IsUnknown() || planned.DateCreated.IsUnknown() || planned.LastUpdated.IsUnknown() {
		t.Errorf("expected no unknown values after apply")
	}
	if !planned.Description.IsNull() {
		t.Errorf("expected description to stay null, got %s", planned.Description)
	}

	(&PasswordPolicyResource{}).setStateFromAPI(ctx, &planned, api, true, &diags)
	if planned.MinLength.ValueInt64() != 12 {
		t.Errorf("expected Read to refresh min_length from the API, got %s", planned.MinLength)
	}
	if !planned.Description.IsNull() || !planned.SourceIDs.IsNull() {
		t.Errorf("expected unset description and source_ids to stay null")
	}
}

func TestAccountSchemaMergeKeepsUnmanagedFields(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	schemaType := accountSchemaAttributeSchemaObjectType()
	emptySchema, _ := types.ListValue(schemaType, []attr.Value{})
	attrs, d := types.ListValueFrom(ctx, accountSchemaAttributeObjectType(), []AccountSchemaAttributeModel{
		{Name: types.StringValue("department"), Type: types.StringUnknown(), Description: types.StringValue("Department"),
			IsGroup: types.BoolUnknown(), IsMultiValued: types.BoolValue(true), IsEntitlement: types.BoolUnknown(), Schema: emptySchema},
		{Name: types.StringValue("department"), Type: types.StringValue("STRING"), Description: types.StringNull(),
			IsGroup: types.BoolNull(), IsMultiValued: types.BoolNull(), IsEntitlement: types.BoolNull(), Schema: emptySchema},
	})
	diags.Append(d...)
	existing := []*AccountSchemaAttribute{
		{Name: "department", NativeName: "dept", Type: "STRING"},
		{Name: "unmanaged", NativeName: "x"},
	}

	merged := (&AccountSchemaResource{}).mergeAttributes(ctx, AccountSchemaResourceModel{Attributes: attrs}, existing, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(merged) != 1 {
		t.Fatalf("expected duplicates and unconfigured attributes to be dropped, got %d", len(merged))
	}
	got := merged[0]
	if got.NativeName != "dept" || got.Type != "STRING" || got.Description != "Department" || !got.IsMultiValued {
		t.Fatalf("unexpected merged attribute %+v", got)
	}
	body, _ := json.Marshal(got)
	if !strings.Contains(string(body), `"isMulti":true`) || !strings.Contains(string(body), `"nativeName":"dept"`) {
		t.Fatalf("unexpected JSON %s", body)
	}

	// Without attribute blocks (a null list at runtime) the existing attributes are kept.
	none := types.ListNull(accountSchemaAttributeObjectType())
	if got := (&AccountSchemaResource{}).mergeAttributes(ctx, AccountSchemaResourceModel{Attributes: none}, existing, &diags); len(got) != 2 {
		t.Fatalf("expected existing attributes to be kept, got %d", len(got))
	}
}

func TestRolePatchesRemoveMembership(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	membership, d := types.ListValueFrom(ctx, membershipObjectType(), []MembershipModel{{Type: types.StringValue("STANDARD"), Criteria: criteriaEmptyList()}})
	diags.Append(d...)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	state := RoleResourceModel{
		Name:                types.StringValue("Role"),
		Description:         types.StringValue("Old"),
		Owner:               types.ListNull(objectInfoObjectType),
		AccessProfiles:      types.ListNull(objectInfoObjectType),
		Entitlements:        types.ListNull(objectInfoObjectType),
		AccessRequestConfig: types.ListNull(roleAccessRequestConfigObjectType()),
		AccessModelMetadata: types.ListNull(accessModelMetadataObjectType()),
		Membership:          membership,
		Enabled:             types.BoolValue(true),
		Requestable:         types.BoolValue(true),
		Dimensional:         types.BoolValue(false),
	}
	plan := state
	plan.Description = types.StringNull()
	plan.Membership = types.ListNull(membershipObjectType())

	patches := rolePatches(plan, state, &Role{Name: "Role", Enabled: boolPtr(true)})
	var ops []string
	for _, p := range patches {
		ops = append(ops, p.Op+" "+p.Path)
	}
	want := []string{"replace /description", "remove /membership"}
	if !reflect.DeepEqual(ops, want) {
		t.Fatalf("got %v, want %v", ops, want)
	}
	if patches[0].Value != "" {
		t.Fatalf("expected description to be cleared, got %v", patches[0].Value)
	}
}

func TestCaseInsensitiveSetSemanticEquals(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	set := func(values ...string) CaseInsensitiveStringSetValue {
		return stringSliceToCaseInsensitiveSet(ctx, values, &diags)
	}
	tests := []struct {
		prior, next CaseInsensitiveStringSetValue
		want        bool
	}{
		{set("prod", "dev"), set("PROD", "DEV"), true},
		{set("a", "b"), set("a", "A"), false},
		{set("a"), set("a", "b"), false},
	}
	for _, tt := range tests {
		got, d := tt.prior.SetSemanticEquals(ctx, tt.next)
		diags.Append(d...)
		if got != tt.want {
			t.Errorf("SetSemanticEquals(%s, %s) = %v, want %v", tt.prior, tt.next, got, tt.want)
		}
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := toUpperSlice([]string{"prod", "PROD", "dev"}); !reflect.DeepEqual(got, []string{"PROD", "DEV"}) {
		t.Fatalf("expected deduplicated upper-case tags, got %v", got)
	}
	_ = basetypes.SetType{}
}

func TestJSONStringState(t *testing.T) {
	steps := map[string]interface{}{"b": 1.0, "a": "x"}
	if got := jsonStringState(types.StringValue(`{ "a": "x", "b": 1 }`), steps); got.ValueString() != `{ "a": "x", "b": 1 }` {
		t.Errorf("expected semantically equal prior to be kept, got %s", got)
	}
	if got := jsonStringState(types.StringValue(`{"a":"y"}`), steps); got.ValueString() != `{"a":"x","b":1}` {
		t.Errorf("expected normalized API value, got %s", got)
	}
	if got := jsonStringState(types.StringNull(), map[string]interface{}{}); !got.IsNull() {
		t.Errorf("expected empty object to be null, got %s", got)
	}
	if got := jsonStringState(types.StringValue(`{}`), nil); got.ValueString() != `{}` {
		t.Errorf("expected configured empty object to be kept, got %s", got)
	}
}

func TestSegmentVisibilityJSONState(t *testing.T) {
	criteria := &SegmentVisibilityCriteria{Expression: &SegmentVisibilityExpression{Operator: "EQUALS", Attribute: "location", Value: &SegmentVisibilityValue{Type: "STRING", Value: "Austin"}}}
	prior := types.StringValue(`{"expression": {"value": {"value": "Austin", "type": "STRING"}, "attribute": "location", "operator": "EQUALS", "children": []}}`)
	if got := segmentVisibilityJSONState(prior, criteria); !got.Equal(prior) {
		t.Errorf("expected equivalent prior to be kept, got %s", got)
	}
	criteria.Expression.Value.Value = "Boston"
	if got := segmentVisibilityJSONState(prior, criteria); !strings.Contains(got.ValueString(), "Boston") {
		t.Errorf("expected drift to be detected, got %s", got)
	}
}

func TestListWithDefaultString(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	list, d := types.ListValueFrom(ctx, objectInfoObjectType, []OwnerModel{
		{ID: types.StringValue("1"), Name: types.StringValue("a"), Type: types.StringUnknown()},
		{ID: types.StringValue("2"), Name: types.StringValue("b"), Type: types.StringValue("GOVERNANCE_GROUP")},
	})
	diags.Append(d...)
	got := listWithDefaultString(ctx, list, "type", "IDENTITY", &diags)
	var owners []OwnerModel
	diags.Append(got.ElementsAs(ctx, &owners, false)...)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if owners[0].Type.ValueString() != "IDENTITY" || owners[1].Type.ValueString() != "GOVERNANCE_GROUP" {
		t.Fatalf("unexpected types %s, %s", owners[0].Type, owners[1].Type)
	}
}

func TestRoleAccessRequestConfigReconcileOmitsDefaults(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	prior := types.ListNull(roleAccessRequestConfigObjectType())
	got := roleAccessRequestConfigReconcile(ctx, prior, &RoleAccessRequestConfig{CommentsRequired: boolPtr(false), DenialCommentsRequired: boolPtr(false)}, &diags)
	if !got.IsNull() {
		t.Fatalf("expected default config to be omitted, got %s", got)
	}
	got = roleAccessRequestConfigReconcile(ctx, prior, &RoleAccessRequestConfig{CommentsRequired: boolPtr(true)}, &diags)
	if got.IsNull() || len(got.Elements()) != 1 {
		t.Fatalf("expected non-default config to be detected, got %s", got)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestEqualIgnoringUnknown(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state, d := types.ListValueFrom(ctx, objectInfoObjectType, []OwnerModel{{ID: types.StringValue("1"), Type: types.StringValue("IDENTITY"), Name: types.StringValue("a")}})
	diags.Append(d...)
	planUnknownName, d := types.ListValueFrom(ctx, objectInfoObjectType, []OwnerModel{{ID: types.StringValue("1"), Type: types.StringValue("IDENTITY"), Name: types.StringUnknown()}})
	diags.Append(d...)
	planChanged, d := types.ListValueFrom(ctx, objectInfoObjectType, []OwnerModel{{ID: types.StringValue("2"), Type: types.StringValue("IDENTITY"), Name: types.StringUnknown()}})
	diags.Append(d...)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !equalIgnoringUnknown(planUnknownName, state) {
		t.Error("expected unknown nested value to be ignored")
	}
	if equalIgnoringUnknown(planChanged, state) {
		t.Error("expected changed id to be detected")
	}
	if equalIgnoringUnknown(types.ListNull(objectInfoObjectType), state) {
		t.Error("expected removed block to be detected")
	}
}

func TestAccessModelMetadataFillUnknownKeepsPlannedValues(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	values, d := types.ListValueFrom(ctx, accessModelMetadataValueObjectType(), []AccessModelMetadataValueModel{
		{Value: types.StringValue("public"), Name: types.StringUnknown(), Status: types.StringUnknown()},
	})
	diags.Append(d...)
	attrs, d := types.ListValueFrom(ctx, accessModelMetadataAttributeObjectType(), []AccessModelMetadataAttributeModel{{
		Key: types.StringValue("iscPrivacy"), Name: types.StringValue("Privacy"),
		Multiselect: types.BoolUnknown(), Status: types.StringValue("active"), Type: types.StringUnknown(), Description: types.StringUnknown(),
		ObjectTypes: types.ListNull(accessModelMetadataObjectTypeValueObjectType()), Values: values,
	}})
	diags.Append(d...)
	planned, d := types.ListValueFrom(ctx, accessModelMetadataObjectType(), []AccessModelMetadataModel{{Attributes: attrs}})
	diags.Append(d...)

	got := accessModelMetadataFillUnknown(ctx, planned, &AttributeDTOList{Attributes: []*AccessModelMetadataAttribute{{
		Key: "iscPrivacy", Name: "Privacy (API)", Multiselect: boolPtr(false), Status: "inactive", Type: "governance",
		ObjectTypes: []string{"all"}, Values: []*AccessModelMetadataValue{{Value: "public", Name: "Public", Status: "active"}},
	}}}, &diags)
	var models []AccessModelMetadataModel
	diags.Append(got.ElementsAs(ctx, &models, false)...)
	var gotAttrs []AccessModelMetadataAttributeModel
	diags.Append(models[0].Attributes.ElementsAs(ctx, &gotAttrs, false)...)
	var gotValues []AccessModelMetadataValueModel
	diags.Append(gotAttrs[0].Values.ElementsAs(ctx, &gotValues, false)...)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	a := gotAttrs[0]
	if a.Name.ValueString() != "Privacy" || a.Status.ValueString() != "active" || !a.ObjectTypes.IsNull() {
		t.Errorf("expected planned values to be kept, got %+v", a)
	}
	if a.Type.ValueString() != "governance" || a.Multiselect.IsUnknown() || gotValues[0].Name.ValueString() != "Public" {
		t.Errorf("expected unknown values to be filled, got %+v %+v", a, gotValues[0])
	}
}
