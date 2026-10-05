package provider

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const networkConfigTestDoc = `{"range": ["1.3.7.2", "255.255.255.252/30"], "geolocation": ["CA", "FR"], "whitelisted": true}`

func TestNetworkConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewNetworkConfigResource, "/v2026/auth-org/network-config", networkConfigTestDoc)
}

// Without a network configuration GET returns 404: Create uses POST with the configured settings,
// and Read removes the resource from state.
func TestNetworkConfigCreatesWithPost(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewNetworkConfigResource, "/v2026/auth-org/network-config", `{}`)
	h.api.missing = true
	configured := map[string]attr.Value{"geolocation": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("PL")})}
	state := h.create(configured)
	writes := h.api.writes()
	if len(writes) != 1 || writes[0].Method != http.MethodPost || !tenantSettingsTestJSONEqual(writes[0].Body, map[string]interface{}{"geolocation": []interface{}{"PL"}}) {
		t.Fatalf("unexpected requests %+v", writes)
	}
	values := h.values(state)
	if !values["geolocation"].Equal(configured["geolocation"]) || !values["range"].IsNull() || !values["whitelisted"].Equal(types.BoolValue(false)) {
		t.Fatalf("unexpected state %v", values)
	}

	h.api.missing = true
	if removed := h.read(state); !removed.Raw.IsNull() {
		t.Fatalf("read of a missing configuration must remove the resource, got %s", removed.Raw)
	}
}

// Reordered ranges are the same setting and do not show a diff.
func TestNetworkConfigKeepsOrderOfRanges(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewNetworkConfigResource, "/v2026/auth-org/network-config", networkConfigTestDoc)
	configured := map[string]attr.Value{"range": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("255.255.255.252/30"), types.StringValue("1.3.7.2")})}
	state := h.create(configured)
	h.api.doc["range"] = []interface{}{"1.3.7.2", "255.255.255.252/30"}
	if refreshed := h.read(state); !refreshed.Raw.Equal(state.Raw) {
		t.Fatalf("read shows drift:\n%s\n%s", state.Raw, refreshed.Raw)
	}
}
