package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewMfaOktaConfigResource)
}

// mfaOktaConfigSpec describes /v2026/mfa/okta-verify/config (GET/PUT). Updates use putMerged. The
// access key is sensitive; when the API does not return it, the known value is kept. Since the PUT
// replaces the whole object, an update fails when the access key is neither configured nor
// returned by the API, instead of clearing it.
var mfaOktaConfigSpec = &tenantSettingsSpec{
	TypeName: "mfa_okta_config",
	ID:       "mfa-okta-config",
	Title:    "MFA Okta config",
	Description: "Manages the tenant-wide Okta Verify multi-factor authentication configuration. There is one configuration per tenant: " +
		"only the configured settings are managed, the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path: "/v2026/mfa/okta-verify/config",
	Mode: tenantSettingsPutMerged,
	Fields: []tenantSettingsField{
		{Name: "mfa_method", Path: []string{"mfaMethod"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "MFA method name (`okta-verify`)"},
		{Name: "enabled", Path: []string{"enabled"}, Kind: tenantSettingsBool,
			Description: "Whether the MFA method is enabled"},
		{Name: "host", Path: []string{"host"}, Kind: tenantSettingsString,
			Description: "Host name of the Okta organization, e.g. `example.okta.com`"},
		{Name: "access_key", Path: []string{"accessKey"}, Kind: tenantSettingsString, Sensitive: true, WriteOnly: true,
			Description: "Okta API token"},
		{Name: "identity_attribute", Path: []string{"identityAttribute"}, Kind: tenantSettingsString,
			Description: "Identity attribute that maps identities to Okta users, e.g. `email`"},
	},
}

func NewMfaOktaConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: mfaOktaConfigSpec}
}
