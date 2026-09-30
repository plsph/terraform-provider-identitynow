package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const accessRequestConfigTestDoc = `{
	"approvalsMustBeExternal": false,
	"reauthorizationEnabled": false,
	"govGroupVisibilityEnabled": true,
	"requestOnBehalfOfConfig": {"allowRequestOnBehalfOfAnyoneByAnyone": false, "allowRequestOnBehalfOfEmployeeByManager": true},
	"entitlementRequestConfig": {
		"accessRequestConfig": {"approvalSchemes": [{"approverType": "MANAGER", "approverId": null}], "requestCommentRequired": true, "denialCommentRequired": false},
		"revocationRequestConfig": {"approvalSchemes": []}
	}
}`

func TestAccessRequestConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewAccessRequestConfigResource, "/v2026/access-request-config", accessRequestConfigTestDoc)
}

// Configuring one nested setting keeps the sibling settings and the unconfigured keys of the
// entitlement request configuration.
func TestAccessRequestConfigMergesNestedSettings(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewAccessRequestConfigResource, "/v2026/access-request-config", accessRequestConfigTestDoc)
	configured := map[string]attr.Value{
		"request_on_behalf_of_anyone_by_anyone": types.BoolValue(true),
		"entitlement_request_config_json":       types.StringValue(`{"accessRequestConfig": {"denialCommentRequired": true}}`),
	}
	state := h.create(configured)
	writes := h.api.writes()
	if len(writes) != 1 {
		t.Fatalf("expected one PUT, got %+v", writes)
	}
	body := writes[0].Body
	want := map[string]interface{}{
		"requestOnBehalfOfConfig": map[string]interface{}{"allowRequestOnBehalfOfAnyoneByAnyone": true, "allowRequestOnBehalfOfEmployeeByManager": true},
		"accessRequestConfig": map[string]interface{}{
			"approvalSchemes":        []interface{}{map[string]interface{}{"approverType": "MANAGER", "approverId": nil}},
			"requestCommentRequired": true,
			"denialCommentRequired":  true,
		},
	}
	if got, _ := tenantSettingsLookup(body, []string{"requestOnBehalfOfConfig"}); !tenantSettingsTestJSONEqual(got, want["requestOnBehalfOfConfig"]) {
		t.Fatalf("unexpected requestOnBehalfOfConfig %v", got)
	}
	if got, _ := tenantSettingsLookup(body, []string{"entitlementRequestConfig", "accessRequestConfig"}); !tenantSettingsTestJSONEqual(got, want["accessRequestConfig"]) {
		t.Fatalf("unexpected accessRequestConfig %v", got)
	}
	if got, _ := tenantSettingsLookup(body, []string{"govGroupVisibilityEnabled"}); got != true {
		t.Fatalf("unconfigured settings must keep their value, got %v", got)
	}
	h.checkValues("create", state, configured)
	values := h.values(state)
	if !values["gov_group_visibility_enabled"].Equal(types.BoolValue(true)) || !values["request_on_behalf_of_employee_by_manager"].Equal(types.BoolValue(true)) {
		t.Fatalf("unconfigured settings must be read from the API, got %v", values)
	}
	if refreshed := h.read(state); !refreshed.Raw.Equal(state.Raw) {
		t.Fatalf("read shows drift:\n%s\n%s", state.Raw, refreshed.Raw)
	}
}

// The API adds keys to the approval scheme elements (approverId: null) and omits false values:
// neither is drift of the configured entitlement request configuration.
func TestAccessRequestConfigEntitlementConfigNoDriftFromAPIDefaults(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewAccessRequestConfigResource, "/v2026/access-request-config", accessRequestConfigTestDoc)
	configured := map[string]attr.Value{
		"entitlement_request_config_json": types.StringValue(`{"accessRequestConfig": {"approvalSchemes": [{"approverType": "MANAGER"}, {"approverType": "GOVERNANCE_GROUP", "approverId": "g-1"}], "denialCommentRequired": false}}`),
	}
	state := h.create(configured)
	h.api.doc["entitlementRequestConfig"] = map[string]interface{}{
		"accessRequestConfig": map[string]interface{}{
			"approvalSchemes": []interface{}{
				map[string]interface{}{"approverType": "MANAGER", "approverId": nil},
				map[string]interface{}{"approverType": "GOVERNANCE_GROUP", "approverId": "g-1"},
			},
			"requestCommentRequired": true,
		},
	}
	refreshed := h.read(state)
	if !refreshed.Raw.Equal(state.Raw) {
		t.Fatalf("read shows drift:\n%s\n%s", state.Raw, refreshed.Raw)
	}

	// A changed approver is drift.
	h.api.doc["entitlementRequestConfig"].(map[string]interface{})["accessRequestConfig"].(map[string]interface{})["approvalSchemes"] = []interface{}{
		map[string]interface{}{"approverType": "MANAGER", "approverId": nil},
		map[string]interface{}{"approverType": "GOVERNANCE_GROUP", "approverId": "g-2"},
	}
	values := h.values(h.read(state))
	want := `{"accessRequestConfig":{"approvalSchemes":[{"approverType":"MANAGER"},{"approverId":"g-2","approverType":"GOVERNANCE_GROUP"}],"denialCommentRequired":false}}`
	if got := values["entitlement_request_config_json"].(types.String).ValueString(); !jsonSemanticallyEqual([]byte(got), []byte(want)) {
		t.Fatalf("expected the changed approver to be shown, got %s", got)
	}
}
