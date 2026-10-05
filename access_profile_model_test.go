package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAccessProfileStateIncludesAdvancedFields(t *testing.T) {
	ap := &AccessProfile{
		Name:        "Example",
		Description: "Example profile",
		Segments:    []string{"seg-1", "seg-2"},
		AccessModelMetadata: &AttributeDTOList{Attributes: []*AccessModelMetadataAttribute{{
			Key:         "iscPrivacy",
			Name:        "Privacy",
			Multiselect: boolPtr(false),
			Status:      "active",
			Type:        "governance",
			ObjectTypes: []string{"all"},
			Description: "Specifies privacy level",
			Values: []*AccessModelMetadataValue{{
				Value:  "public",
				Name:   "Public",
				Status: "active",
			}},
		}}},
		RevocationRequestConfig: &AccessProfileRevocationRequestConfig{ApprovalSchemes: []*ApprovalSchemes{{
			ApproverType: "GOVERNANCE_GROUP",
			ApproverId:   "group-1",
		}}},
		AdditionalOwners: []*AdditionalOwnerRef{{
			Type: "IDENTITY",
			ID:   "identity-1",
			Name: "Support",
		}},
		ProvisioningCriteria: &ProvisioningCriteriaLevel1{
			Operation: "EQUALS",
			Attribute: "email",
			Value:     "support@example.com",
		},
	}

	data := AccessProfileResourceModel{}
	var diags diag.Diagnostics
	r := &AccessProfileResource{}
	r.setStateFromAPI(context.Background(), &data, ap, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if data.Segments.IsNull() || data.Segments.IsUnknown() || len(data.Segments.Elements()) != 2 {
		t.Fatalf("expected segments to be populated, got %#v", data.Segments)
	}
	if data.AccessModelMetadata.IsNull() || data.AccessModelMetadata.IsUnknown() {
		t.Fatal("expected access_model_metadata to be populated")
	}
	if data.RevocationRequestConfig.IsNull() || data.RevocationRequestConfig.IsUnknown() {
		t.Fatal("expected revocation_request_config to be populated")
	}
	if data.AdditionalOwners.IsNull() || data.AdditionalOwners.IsUnknown() {
		t.Fatal("expected additional_owners to be populated")
	}
	if data.ProvisioningCriteria.IsNull() || data.ProvisioningCriteria.IsUnknown() {
		t.Fatal("expected provisioning_criteria to be populated")
	}
}

func TestAccessProfileStateOmitsEmptyRevocationConfig(t *testing.T) {
	ap := &AccessProfile{
		Name:                    "Example",
		Description:             "Example profile",
		RevocationRequestConfig: &AccessProfileRevocationRequestConfig{},
	}

	data := AccessProfileResourceModel{}
	var diags diag.Diagnostics
	r := &AccessProfileResource{}
	r.setStateFromAPI(context.Background(), &data, ap, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if data.RevocationRequestConfig.IsNull() || data.RevocationRequestConfig.IsUnknown() {
		t.Fatalf("expected empty revocation config to be represented as an empty list, got %#v", data.RevocationRequestConfig)
	}
	if len(data.RevocationRequestConfig.Elements()) != 0 {
		t.Fatalf("expected no revocation config blocks, got %#v", data.RevocationRequestConfig)
	}
}

func TestAccessProfileAccessRequestConfigFormDefinitionRoundTrip(t *testing.T) {
	ctx := context.Background()
	data := AccessProfileResourceModel{
		AccessRequestConfig: types.ListNull(types.ObjectType{}),
		Segments:            types.SetNull(types.StringType),
	}
	var diags diag.Diagnostics
	r := &AccessProfileResource{}
	r.setStateFromAPI(ctx, &data, &AccessProfile{
		Name:                "Example",
		AccessRequestConfig: &AccessRequestConfigList{FormDefinitionId: "form-1"},
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(data.AccessRequestConfig.Elements()) != 1 {
		t.Fatalf("expected config with only a form definition to be kept, got %s", data.AccessRequestConfig)
	}
	ap := r.apiFromModel(ctx, data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if ap.AccessRequestConfig == nil || ap.AccessRequestConfig.FormDefinitionId != "form-1" {
		t.Fatalf("expected form definition form-1, got %+v", ap.AccessRequestConfig)
	}

	r.setStateFromAPI(ctx, &data, &AccessProfile{
		Name:                "Example",
		AccessRequestConfig: &AccessRequestConfigList{CommentsRequired: true},
	}, &diags)
	var models []AccessRequestConfigModel
	diags.Append(data.AccessRequestConfig.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !models[0].FormDefinitionID.IsNull() {
		t.Fatalf("expected unset form definition to be null, got %s", models[0].FormDefinitionID)
	}
}

func boolPtr(v bool) *bool { return &v }

func TestAccessProfilePatchesOnlyChangedFields(t *testing.T) {
	state := AccessProfileResourceModel{
		Name:                    types.StringValue("Old"),
		Description:             types.StringValue("Same"),
		Owner:                   types.ListNull(types.ObjectType{}),
		Source:                  types.ListNull(types.ObjectType{}),
		Entitlements:            types.ListNull(types.ObjectType{}),
		AccessRequestConfig:     types.ListNull(types.ObjectType{}),
		RevocationRequestConfig: types.ListNull(types.ObjectType{}),
		Segments:                types.SetNull(types.StringType),
		AccessModelMetadata:     types.ListNull(types.ObjectType{}),
		ProvisioningCriteria:    types.ListNull(types.ObjectType{}),
		AdditionalOwners:        types.ListNull(types.ObjectType{}),
		Enabled:                 types.BoolValue(true),
		Requestable:             types.BoolValue(true),
	}
	plan := state
	plan.Name = types.StringValue("New")
	plan.Requestable = types.BoolValue(false)

	patches := accessProfilePatches(plan, state, &AccessProfile{Name: "New", Enabled: boolPtr(true), Requestable: boolPtr(false)})
	if len(patches) != 2 || patches[0].Path != "/name" || patches[1].Path != "/requestable" {
		var paths []string
		for _, p := range patches {
			paths = append(paths, p.Path)
		}
		t.Fatalf("expected /name and /requestable patches, got %v", paths)
	}
}

func TestAccessProfileUnknownBoolsAreNotSent(t *testing.T) {
	data := AccessProfileResourceModel{
		Name:                    types.StringValue("Example"),
		Description:             types.StringValue("Example"),
		Owner:                   types.ListNull(types.ObjectType{}),
		Source:                  types.ListNull(types.ObjectType{}),
		Entitlements:            types.ListNull(types.ObjectType{}),
		AccessRequestConfig:     types.ListNull(types.ObjectType{}),
		RevocationRequestConfig: types.ListNull(types.ObjectType{}),
		Segments:                types.SetNull(types.StringType),
		AccessModelMetadata:     types.ListNull(types.ObjectType{}),
		ProvisioningCriteria:    types.ListNull(types.ObjectType{}),
		AdditionalOwners:        types.ListNull(types.ObjectType{}),
		Enabled:                 types.BoolUnknown(),
		Requestable:             types.BoolUnknown(),
	}
	var diags diag.Diagnostics
	ap := (&AccessProfileResource{}).apiFromModel(context.Background(), data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if ap.Enabled != nil || ap.Requestable != nil {
		t.Fatalf("expected unknown booleans to be omitted, got enabled=%v requestable=%v", ap.Enabled, ap.Requestable)
	}
	if got := computedBoolFromAPI(types.BoolUnknown(), boolPtr(true)); !got.ValueBool() {
		t.Errorf("expected API value for unknown planned bool")
	}
	if got := computedBoolFromAPI(types.BoolValue(false), boolPtr(true)); got.ValueBool() {
		t.Errorf("expected planned value to be kept")
	}
}

func TestAccessProfileProvisioningCriteriaRoundTrip(t *testing.T) {
	ctx := context.Background()
	criteria := &ProvisioningCriteriaLevel1{
		Operation: "OR",
		Children: []*ProvisioningCriteriaLevel2{{
			Operation: "AND",
			Children: []*ProvisioningCriteriaLevel3{
				{Operation: "EQUALS", Attribute: "department", Value: "IT"},
			},
		}},
	}
	var diags diag.Diagnostics
	state := provisioningCriteriaAPIToState(ctx, criteria, &diags)
	got := provisioningCriteriaModelToAPI(ctx, state, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got.Operation != "OR" || len(got.Children) != 1 || len(got.Children[0].Children) != 1 || got.Children[0].Children[0].Value != "IT" {
		t.Fatalf("unexpected criteria after round trip: %+v", got)
	}

	var models []ProvisioningCriteriaModel
	diags.Append(state.ElementsAs(ctx, &models, false)...)
	if !models[0].Attribute.IsNull() || !models[0].Value.IsNull() {
		t.Fatalf("expected empty attribute and value to be null, got %+v", models[0])
	}
}

func TestAccessProfileStateKeepsUnsetSegmentsNull(t *testing.T) {
	data := AccessProfileResourceModel{Segments: types.SetNull(types.StringType)}
	var diags diag.Diagnostics
	(&AccessProfileResource{}).setStateFromAPI(context.Background(), &data, &AccessProfile{Name: "Example"}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.Segments.IsNull() {
		t.Fatalf("expected segments to stay null, got %s", data.Segments)
	}
}

// Regression test: the API returns segments in a different order than configured, which must not
// show up as a diff in the plan.
func TestAccessProfileSegmentsIgnoreOrder(t *testing.T) {
	ctx := context.Background()
	prior, _ := types.SetValueFrom(ctx, types.StringType, []string{"seg-a", "seg-b", "seg-c"})
	data := AccessProfileResourceModel{Segments: prior}
	var diags diag.Diagnostics
	(&AccessProfileResource{}).setStateFromAPI(ctx, &data, &AccessProfile{Name: "Example", Segments: []string{"seg-c", "seg-a", "seg-b"}}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.Segments.Equal(prior) {
		t.Fatalf("expected segments in another order to equal the prior value, got %s", data.Segments)
	}
	state := AccessProfileResourceModel{Segments: data.Segments}
	plan := AccessProfileResourceModel{Segments: prior}
	for _, p := range accessProfilePatches(plan, state, &AccessProfile{}) {
		if p.Path == "/segments" {
			t.Fatalf("expected no segments patch when only the order differs")
		}
	}
}

// Regression test: removing additional owners that exist in IdentityNow (e.g. added in the UI)
// must send an empty /additionalOwners patch, so the result matches the planned zero blocks.
func TestAccessProfilePatchesClearAdditionalOwners(t *testing.T) {
	ownerType := types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "id": types.StringType, "name": types.StringType}}
	owner := types.ObjectValueMust(ownerType.AttrTypes, map[string]attr.Value{
		"type": types.StringValue("IDENTITY"),
		"id":   types.StringValue("identity-1"),
		"name": types.StringNull(),
	})
	state := AccessProfileResourceModel{
		Name:                    types.StringValue("GNC-Read Only"),
		Description:             types.StringValue("Same"),
		Owner:                   types.ListNull(types.ObjectType{}),
		Source:                  types.ListNull(types.ObjectType{}),
		Entitlements:            types.ListNull(types.ObjectType{}),
		AccessRequestConfig:     types.ListNull(types.ObjectType{}),
		RevocationRequestConfig: types.ListNull(types.ObjectType{}),
		Segments:                types.SetNull(types.StringType),
		AccessModelMetadata:     types.ListNull(types.ObjectType{}),
		ProvisioningCriteria:    types.ListNull(types.ObjectType{}),
		AdditionalOwners:        types.ListValueMust(ownerType, []attr.Value{owner}),
		Enabled:                 types.BoolValue(true),
		Requestable:             types.BoolValue(true),
	}
	plan := state
	plan.AdditionalOwners = types.ListValueMust(ownerType, []attr.Value{})

	var diags diag.Diagnostics
	ap := (&AccessProfileResource{}).apiFromModel(context.Background(), plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	patches := accessProfilePatches(plan, state, ap)
	if len(patches) != 1 || patches[0].Path != "/additionalOwners" {
		t.Fatalf("expected a single /additionalOwners patch, got %+v", patches)
	}
	if owners, ok := patches[0].Value.([]*AdditionalOwnerRef); !ok || owners == nil || len(owners) != 0 {
		t.Fatalf("expected an empty additional owners list, got %#v", patches[0].Value)
	}
}
