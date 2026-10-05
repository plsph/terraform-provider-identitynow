package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func init() {
	registerResource(NewServiceProviderConfigResource)
}

const (
	serviceProviderConfigIdpRole = "SAML_IDP"
	serviceProviderConfigSpRole  = "SAML_SP"
)

// serviceProviderConfigSpec describes /v2026/auth-org/service-provider-config (GET/PATCH), the
// SAML configuration with IdentityNow as the service provider. The API object has a
// federationProtocolDetails list with an identity provider (IdP, role SAML_IDP) element and a
// service provider (SP, role SAML_SP) element. serviceProviderConfigView exposes them as the `idp`
// and `sp` objects the field paths refer to. The SP element cannot be changed, its settings are
// read only. Changed IdP settings are sent as one replace operation of the whole IdP element with
// the configured settings applied, see serviceProviderConfigPatchOps.
var serviceProviderConfigSpec = &tenantSettingsSpec{
	TypeName: "service_provider_config",
	ID:       "service-provider-config",
	Title:    "Service provider config",
	Description: "Manages the tenant-wide SAML single sign-on configuration, in which IdentityNow is the service provider of an external identity provider. " +
		"There is one configuration per tenant: only the configured settings are managed, the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path:     "/v2026/auth-org/service-provider-config",
	Mode:     tenantSettingsPatch,
	View:     serviceProviderConfigView,
	PatchOps: serviceProviderConfigPatchOps,
	Fields: []tenantSettingsField{
		{Name: "enabled", Path: []string{"enabled"}, Kind: tenantSettingsBool,
			Description: "Whether SAML authentication is enabled"},
		{Name: "bypass_idp", Path: []string{"bypassIdp"}, Kind: tenantSettingsBool,
			Description: "Whether basic login with the `prompt=true` parameter is allowed, e.g. while debugging the SAML setup. When disabled, only org admins with MFA can bypass the identity provider"},
		{Name: "saml_configuration_valid", Path: []string{"samlConfigurationValid"}, Kind: tenantSettingsBool, ReadOnly: true,
			Description: "Whether the SAML configuration is valid"},
		{Name: "idp_entity_id", Path: []string{"idp", "entityId"}, Kind: tenantSettingsString,
			Description: "Entity ID of the identity provider"},
		{Name: "idp_binding", Path: []string{"idp", "binding"}, Kind: tenantSettingsString,
			Description: "SAML binding of the identity provider, e.g. `urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST`"},
		{Name: "idp_authn_context", Path: []string{"idp", "authnContext"}, Kind: tenantSettingsString,
			Description: "SAML authentication context, e.g. `urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport`"},
		{Name: "idp_include_authn_context", Path: []string{"idp", "includeAuthnContext"}, Kind: tenantSettingsBool,
			Description: "Whether the configured authentication context is used instead of the default"},
		{Name: "idp_name_id", Path: []string{"idp", "nameId"}, Kind: tenantSettingsString,
			Description: "Name ID format, e.g. `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`"},
		{Name: "idp_mapping_attribute", Path: []string{"idp", "mappingAttribute"}, Kind: tenantSettingsString,
			Description: "Identity attribute that is matched with the SAML name ID, e.g. `email`"},
		{Name: "idp_login_url_post", Path: []string{"idp", "loginUrlPost"}, Kind: tenantSettingsString,
			Description: "Identity provider login URL for the HTTP-POST binding"},
		{Name: "idp_login_url_redirect", Path: []string{"idp", "loginUrlRedirect"}, Kind: tenantSettingsString,
			Description: "Identity provider login URL for the HTTP-Redirect binding"},
		{Name: "idp_logout_url", Path: []string{"idp", "logoutUrl"}, Kind: tenantSettingsString,
			Description: "Identity provider logout URL"},
		{Name: "idp_cert", Path: []string{"idp", "cert"}, Kind: tenantSettingsString, Sensitive: true,
			Description: "Base64-encoded signing certificate of the identity provider"},
		{Name: "idp_certificate_name", Path: []string{"idp", "certificateName"}, Kind: tenantSettingsString, ReadOnly: true,
			Description: "Subject name of the identity provider certificate"},
		{Name: "idp_certificate_expiration_date", Path: []string{"idp", "certificateExpirationDate"}, Kind: tenantSettingsString, ReadOnly: true,
			Description: "Expiration date of the identity provider certificate"},
		{Name: "idp_jit_enabled", Path: []string{"idp", "jitConfiguration", "enabled"}, Kind: tenantSettingsBool,
			Description: "Whether just-in-time provisioning of identities is enabled. It requires `idp_jit_source_id` and `idp_jit_source_attribute_mappings` with `firstName`, `lastName` and `email`"},
		{Name: "idp_jit_source_id", Path: []string{"idp", "jitConfiguration", "sourceId"}, Kind: tenantSettingsString,
			Description: "ID of the source that just-in-time provisioned accounts are created on"},
		{Name: "idp_jit_source_attribute_mappings", Path: []string{"idp", "jitConfiguration", "sourceAttributeMappings"}, Kind: tenantSettingsStringMap,
			Description: "Map of identity profile attribute names to SAML assertion attribute names for just-in-time provisioning"},
		{Name: "sp_entity_id", Path: []string{"sp", "entityId"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "Entity ID of IdentityNow as the service provider"},
		{Name: "sp_alias", Path: []string{"sp", "alias"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "Alias of the service provider"},
		{Name: "sp_callback_url", Path: []string{"sp", "callbackUrl"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "Assertion consumer (callback) URL of the service provider, to be configured in the identity provider"},
		{Name: "sp_legacy_acs_url", Path: []string{"sp", "legacyAcsUrl"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "Legacy assertion consumer service URL of the service provider"},
	},
}

func NewServiceProviderConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: serviceProviderConfigSpec}
}

// serviceProviderConfigDetails returns the federation protocol detail elements and the index of
// the element with the given role, or -1.
func serviceProviderConfigDetails(doc map[string]interface{}, role string) ([]interface{}, int) {
	details, _ := doc["federationProtocolDetails"].([]interface{})
	for i, d := range details {
		if m, ok := d.(map[string]interface{}); ok && m["role"] == role {
			return details, i
		}
	}
	return details, -1
}

// serviceProviderConfigView adds the IdP and SP elements of the federation protocol details as
// the `idp` and `sp` objects.
func serviceProviderConfigView(doc map[string]interface{}) map[string]interface{} {
	view := make(map[string]interface{}, len(doc)+2)
	for k, v := range doc {
		view[k] = v
	}
	if details, i := serviceProviderConfigDetails(doc, serviceProviderConfigIdpRole); i >= 0 {
		view["idp"] = details[i]
	}
	if details, i := serviceProviderConfigDetails(doc, serviceProviderConfigSpRole); i >= 0 {
		view["sp"] = details[i]
	}
	return view
}

// serviceProviderConfigPatchOps replaces top-level settings directly. Changed IdP settings are
// applied to the current IdP element, which is replaced as a whole, so the API validates a
// complete IdP element (and JIT configuration). Without an IdP element, one is added at index 0
// (the API documents index 0 as the IdP element).
func serviceProviderConfigPatchOps(ctx context.Context, c *Client, spec *tenantSettingsSpec, changes []tenantSettingsChange) ([]jsonPatchOp, error) {
	var ops []jsonPatchOp
	var idpChanges []tenantSettingsChange
	for _, ch := range changes {
		if ch.Field.Path[0] == "idp" {
			idpChanges = append(idpChanges, ch)
			continue
		}
		ops = append(ops, jsonPatchOp{Op: "replace", Path: tenantSettingsPointer(ch.Field.Path), Value: ch.Value})
	}
	if len(idpChanges) == 0 {
		return ops, nil
	}
	current, err := c.tenantSettingsGet(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("reading current identity provider settings: %w", err)
	}
	details, i := serviceProviderConfigDetails(current, serviceProviderConfigIdpRole)
	element := map[string]interface{}{"role": serviceProviderConfigIdpRole}
	if i >= 0 {
		element = details[i].(map[string]interface{})
	}
	for _, ch := range idpChanges {
		if ch.Field.Name == "idp_cert" {
			// Derived from the certificate by the API.
			delete(element, "certificateName")
			delete(element, "certificateExpirationDate")
		}
		element = tenantSettingsSetAt(element, ch.Field.Path[1:], ch.Value, false).(map[string]interface{})
	}
	if i >= 0 {
		return append(ops, jsonPatchOp{Op: "replace", Path: fmt.Sprintf("/federationProtocolDetails/%d", i), Value: element}), nil
	}
	return append(ops, jsonPatchOp{Op: "add", Path: "/federationProtocolDetails/0", Value: element}), nil
}
