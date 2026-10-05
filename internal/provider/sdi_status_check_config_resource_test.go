package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const sdiStatusCheckConfigTestDoc = `{"provisioningStatusCheckIntervalMinutes": "30", "provisioningMaxStatusCheckDays": "2"}`

func TestSdiStatusCheckConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewSdiStatusCheckConfigResource, "/v2026/service-desk-integrations/status-check-configuration", sdiStatusCheckConfigTestDoc)
}

// The API represents the numbers as strings.
func TestSdiStatusCheckConfigSendsStrings(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewSdiStatusCheckConfigResource, "/v2026/service-desk-integrations/status-check-configuration", sdiStatusCheckConfigTestDoc)
	state := h.create(map[string]attr.Value{"provisioning_max_status_check_days": types.Int64Value(5)})
	writes := h.api.writes()
	want := map[string]interface{}{"provisioningStatusCheckIntervalMinutes": "30", "provisioningMaxStatusCheckDays": "5", tenantSettingsTestUnmanagedKey: "keep"}
	if len(writes) != 1 || !tenantSettingsTestJSONEqual(writes[0].Body, want) {
		t.Fatalf("unexpected requests %+v", writes)
	}
	if v := h.values(state)["provisioning_status_check_interval_minutes"]; !v.Equal(types.Int64Value(30)) {
		t.Fatalf("unexpected interval %s", v)
	}
}
