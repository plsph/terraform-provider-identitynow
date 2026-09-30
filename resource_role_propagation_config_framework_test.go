package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const rolePropagationConfigTestDoc = `{"enabled": false, "enabledDate": "2026-01-01T00:00:00Z", "createdDate": "2025-01-01T00:00:00Z", "modifiedDate": "2026-01-01T00:00:00Z"}`

func TestRolePropagationConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewRolePropagationConfigResource, "/v2026/role-propagation-config", rolePropagationConfigTestDoc)
}

// The PUT body only has the writable `enabled` setting, not the dates of the response.
func TestRolePropagationConfigSendsOnlyRequestFields(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewRolePropagationConfigResource, "/v2026/role-propagation-config", rolePropagationConfigTestDoc)
	state := h.create(map[string]attr.Value{"enabled": types.BoolValue(true)})
	writes := h.api.writes()
	if len(writes) != 1 || !tenantSettingsTestJSONEqual(writes[0].Body, map[string]interface{}{"enabled": true}) || writes[0].Experimental != "true" {
		t.Fatalf("unexpected requests %+v", writes)
	}
	if v := h.values(state)["created_date"]; v.(types.String).ValueString() != "2025-01-01T00:00:00Z" {
		t.Fatalf("unexpected created_date %s", v)
	}
}
