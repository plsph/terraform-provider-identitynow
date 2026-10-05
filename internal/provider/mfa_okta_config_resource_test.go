package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const mfaOktaConfigTestDoc = `{"mfaMethod": "okta-verify", "enabled": true, "host": "example.okta.com", "accessKey": "token", "identityAttribute": "email"}`

func TestMfaOktaConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewMfaOktaConfigResource, "/v2026/mfa/okta-verify/config", mfaOktaConfigTestDoc)
}

// When the API does not return the access key, it must be configured: the PUT would clear it.
func TestMfaOktaConfigRefusesToClearAccessKeyNotReturned(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewMfaOktaConfigResource, "/v2026/mfa/okta-verify/config", mfaOktaConfigTestDoc)
	h.api.hidden = []string{"accessKey"}
	imported := h.read(h.imported())
	resp := h.updateResponse(imported, map[string]attr.Value{"enabled": types.BoolValue(false)})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Detail(), "configure `access_key`") {
		t.Fatalf("expected an error naming access_key, got %v", resp.Diagnostics)
	}
	if writes := h.api.writes(); len(writes) != 0 {
		t.Fatalf("nothing must be sent, got %+v", writes)
	}
	configured := map[string]attr.Value{"enabled": types.BoolValue(false), "access_key": types.StringValue("new-token")}
	state := h.update(imported, configured)
	if writes := h.api.writes(); len(writes) != 1 || writes[0].Body.(map[string]interface{})["accessKey"] != "new-token" {
		t.Fatalf("unexpected requests %+v", writes)
	}
	h.checkValues("update", state, configured)
	if refreshed := h.read(state); !refreshed.Raw.Equal(state.Raw) {
		t.Fatalf("read shows drift:\n%s\n%s", state.Raw, refreshed.Raw)
	}
}
