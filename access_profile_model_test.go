package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

func boolPtr(v bool) *bool { return &v }
