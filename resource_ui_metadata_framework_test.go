package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const uiMetadataTestDoc = `{"iframeWhiteList": "http://example.com", "usernameLabel": "Email", "usernameEmptyText": null}`

func TestUiMetadataLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewUiMetadataResource, "/v2026/ui-metadata/tenant", uiMetadataTestDoc)
}

// Unconfigured settings are sent with their current value, so the full-replacement PUT keeps them.
func TestUiMetadataKeepsUnconfiguredSettings(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewUiMetadataResource, "/v2026/ui-metadata/tenant", uiMetadataTestDoc)
	state := h.create(map[string]attr.Value{"username_label": types.StringValue("Work email")})
	writes := h.api.writes()
	want := map[string]interface{}{"iframeWhiteList": "http://example.com", "usernameLabel": "Work email", "usernameEmptyText": nil}
	if len(writes) != 1 || !tenantSettingsTestJSONEqual(writes[0].Body, want) {
		t.Fatalf("unexpected requests %+v", writes)
	}
	values := h.values(state)
	if !values["username_empty_text"].IsNull() || values["iframe_white_list"].(types.String).ValueString() != "http://example.com" {
		t.Fatalf("unexpected state %v", values)
	}
}
