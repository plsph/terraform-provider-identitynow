package main

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewMfaDuoConfigResource)
}

// mfaDuoConfigSpec describes /v2026/mfa/duo-web/config (GET/PUT). Updates use putMerged. The
// access key and the config properties (secret key, integration key) are sensitive; when the API
// does not return them, the known values are kept. Since the PUT replaces the whole object, an
// update fails when the access key or the `skey` config property is neither configured nor
// returned by the API, instead of clearing it.
var mfaDuoConfigSpec = &tenantSettingsSpec{
	TypeName: "mfa_duo_config",
	ID:       "mfa-duo-config",
	Title:    "MFA Duo config",
	Description: "Manages the tenant-wide Duo Web multi-factor authentication configuration. There is one configuration per tenant: " +
		"only the configured settings are managed, the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path: "/v2026/mfa/duo-web/config",
	Mode: tenantSettingsPutMerged,
	Fields: []tenantSettingsField{
		{Name: "mfa_method", Path: []string{"mfaMethod"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "MFA method name (`duo-web`)"},
		{Name: "enabled", Path: []string{"enabled"}, Kind: tenantSettingsBool,
			Description: "Whether the MFA method is enabled"},
		{Name: "host", Path: []string{"host"}, Kind: tenantSettingsString,
			Description: "Host name or IP address of the Duo API"},
		{Name: "access_key", Path: []string{"accessKey"}, Kind: tenantSettingsString, Sensitive: true, WriteOnly: true,
			Description: "Secret key for authenticating requests to Duo. It must be configured when the API does not return it, because the update replaces the whole configuration and would clear it"},
		{Name: "identity_attribute", Path: []string{"identityAttribute"}, Kind: tenantSettingsString,
			Description: "Identity attribute that maps identities to Duo users, e.g. `email`"},
		{Name: "config_properties_json", Path: []string{"configProperties"}, Kind: tenantSettingsJSONObject, Sensitive: true, WriteOnly: true,
			WriteOnlyKeys: []string{"skey"},
			Description:   "Additional Duo properties, e.g. `skey` and `ikey`. The `skey` key must be configured when the API does not return it, because the update replaces the whole configuration and would clear it"},
	},
}

func NewMfaDuoConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: mfaDuoConfigSpec}
}
