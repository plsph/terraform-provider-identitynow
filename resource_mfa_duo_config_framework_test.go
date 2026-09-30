package main

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const mfaDuoConfigTestDoc = `{"mfaMethod": "duo-web", "enabled": true, "host": "api-1.duosecurity.com", "accessKey": "key", "identityAttribute": "email", "configProperties": {"ikey": "Q123", "skey": "S456"}}`

func TestMfaDuoConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewMfaDuoConfigResource, "/v2026/mfa/duo-web/config", mfaDuoConfigTestDoc)
}

// Secrets are sensitive, and the known values are kept when the API does not return them.
func TestMfaDuoConfigKeepsSecretsNotReturned(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewMfaDuoConfigResource, "/v2026/mfa/duo-web/config", mfaDuoConfigTestDoc)
	for _, name := range []string{"access_key", "config_properties_json"} {
		if !h.schema.Attributes[name].(interface{ IsSensitive() bool }).IsSensitive() {
			t.Fatalf("%s must be sensitive", name)
		}
	}
	h.api.hidden = []string{"accessKey"}
	configured := map[string]attr.Value{
		"access_key":             types.StringValue("s3cr3t"),
		"config_properties_json": types.StringValue(`{"skey": "s3cr3t", "ikey": "Q123"}`),
	}
	state := h.create(configured)
	h.checkValues("create", state, configured)
	// The API stops returning the secret key of the config properties.
	h.api.doc["configProperties"] = map[string]interface{}{"ikey": "Q123"}
	refreshed := h.read(state)
	h.checkValues("read", refreshed, configured)

	// Import (no prior values) leaves secrets that are not returned null.
	imported := h.read(h.imported())
	values := h.values(imported)
	if !values["access_key"].IsNull() || values["enabled"].IsNull() {
		t.Fatalf("unexpected imported state %v", values)
	}
}

// The PUT replaces the whole configuration: when the API does not return a secret that is not
// configured, the update fails instead of sending the configuration without it.
func TestMfaDuoConfigRefusesToClearSecretsNotReturned(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewMfaDuoConfigResource, "/v2026/mfa/duo-web/config", mfaDuoConfigTestDoc)
	h.api.hidden = []string{"accessKey"}
	h.api.doc["configProperties"] = map[string]interface{}{"ikey": "Q123"}
	imported := h.read(h.imported())

	resp := h.updateResponse(imported, map[string]attr.Value{"enabled": types.BoolValue(false)})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Detail(), "configure `access_key` and the `skey` key of `config_properties_json`") {
		t.Fatalf("expected an error naming the secrets, got %v", resp.Diagnostics)
	}
	if writes := h.api.writes(); len(writes) != 0 {
		t.Fatalf("nothing must be sent, got %+v", writes)
	}

	// A configured JSON object without the secret key is merged into the current object, which
	// does not have it either.
	resp = h.updateResponse(imported, map[string]attr.Value{
		"access_key":             types.StringValue("s3cr3t"),
		"config_properties_json": types.StringValue(`{"ikey": "Q789"}`),
	})
	if !resp.Diagnostics.HasError() || strings.Contains(resp.Diagnostics[0].Detail(), "access_key") || !strings.Contains(resp.Diagnostics[0].Detail(), "`skey`") {
		t.Fatalf("expected an error naming skey only, got %v", resp.Diagnostics)
	}

	// With the secrets configured the update is sent.
	configured := map[string]attr.Value{
		"enabled":                types.BoolValue(false),
		"access_key":             types.StringValue("s3cr3t"),
		"config_properties_json": types.StringValue(`{"skey": "S456"}`),
	}
	state := h.update(imported, configured)
	writes := h.api.writes()
	if len(writes) != 1 {
		t.Fatalf("expected one PUT, got %+v", writes)
	}
	body := writes[0].Body.(map[string]interface{})
	if body["accessKey"] != "s3cr3t" || !tenantSettingsTestJSONEqual(body["configProperties"], map[string]interface{}{"ikey": "Q123", "skey": "S456"}) {
		t.Fatalf("unexpected body %v", body)
	}
	h.checkValues("update", state, configured)

	// When the API returns the secrets, they do not have to be configured.
	h.api.hidden = nil
	h.api.requests = nil
	h.update(h.read(h.imported()), map[string]attr.Value{"enabled": types.BoolValue(true)})
	if writes := h.api.writes(); len(writes) != 1 || writes[0].Body.(map[string]interface{})["accessKey"] != "s3cr3t" {
		t.Fatalf("expected the returned secret to be sent back, got %+v", writes)
	}
}
