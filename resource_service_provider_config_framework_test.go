package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const serviceProviderConfigTestDoc = `{
	"enabled": false,
	"bypassIdp": false,
	"samlConfigurationValid": true,
	"federationProtocolDetails": [
		{"role": "SAML_SP", "entityId": "https://acme.identitysoon.com/sp", "alias": "acme-sp", "callbackUrl": "https://acme.login.sailpoint.com/saml/SSO/alias/acme-sp"},
		{"role": "SAML_IDP", "entityId": "http://www.okta.com/exk", "cert": "MIID", "mappingAttribute": "email", "nameId": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			"jitConfiguration": {"enabled": false, "sourceId": "src-1", "sourceAttributeMappings": {"firstName": "okta.firstName"}},
			"certificateName": "CN=okta", "certificateExpirationDate": "Thu May 26 21:31:59 GMT 2033"}
	]
}`

func TestServiceProviderConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewServiceProviderConfigResource, "/v2026/auth-org/service-provider-config", serviceProviderConfigTestDoc)
}

// Changed identity provider settings replace the IdP element (found by its role) as a whole with
// the current values of the other IdP settings; the SP element is not changed.
func TestServiceProviderConfigReplacesIdpElement(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewServiceProviderConfigResource, "/v2026/auth-org/service-provider-config", serviceProviderConfigTestDoc)
	if !h.schema.Attributes["idp_cert"].(interface{ IsSensitive() bool }).IsSensitive() {
		t.Fatal("idp_cert must be sensitive")
	}
	configured := map[string]attr.Value{
		"enabled":            types.BoolValue(true),
		"idp_jit_enabled":    types.BoolValue(true),
		"idp_login_url_post": types.StringValue("https://okta.example.com/sso/saml"),
	}
	state := h.create(configured)
	writes := h.api.writes()
	if len(writes) != 1 {
		t.Fatalf("unexpected requests %+v", writes)
	}
	want := []interface{}{
		map[string]interface{}{"op": "replace", "path": "/enabled", "value": true},
		map[string]interface{}{"op": "replace", "path": "/federationProtocolDetails/1", "value": map[string]interface{}{
			"role": "SAML_IDP", "entityId": "http://www.okta.com/exk", "cert": "MIID", "mappingAttribute": "email",
			"nameId":           "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			"loginUrlPost":     "https://okta.example.com/sso/saml",
			"jitConfiguration": map[string]interface{}{"enabled": true, "sourceId": "src-1", "sourceAttributeMappings": map[string]interface{}{"firstName": "okta.firstName"}},
			"certificateName":  "CN=okta", "certificateExpirationDate": "Thu May 26 21:31:59 GMT 2033",
		}},
	}
	if !tenantSettingsTestJSONEqual(writes[0].Body, want) {
		t.Fatalf("unexpected patch %v", writes[0].Body)
	}
	values := h.values(state)
	if values["sp_alias"].(types.String).ValueString() != "acme-sp" || values["idp_entity_id"].(types.String).ValueString() != "http://www.okta.com/exk" ||
		values["idp_certificate_name"].(types.String).ValueString() != "CN=okta" || !values["saml_configuration_valid"].Equal(types.BoolValue(true)) {
		t.Fatalf("unexpected state %v", values)
	}
	if !values["idp_jit_source_attribute_mappings"].Equal(types.MapValueMust(types.StringType, map[string]attr.Value{"firstName": types.StringValue("okta.firstName")})) {
		t.Fatalf("unexpected mappings %s", values["idp_jit_source_attribute_mappings"])
	}
	if refreshed := h.read(state); !refreshed.Raw.Equal(state.Raw) {
		t.Fatalf("read shows drift:\n%s\n%s", state.Raw, refreshed.Raw)
	}
}
