package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewRolePropagationConfigResource)
}

// rolePropagationConfigSpec describes /v2026/role-propagation-config (GET/PUT, experimental). The
// PUT request only has `enabled`, the response also has dates, so the body is built from the
// writable settings only.
var rolePropagationConfigSpec = &tenantSettingsSpec{
	TypeName: "role_propagation_config",
	ID:       "role-propagation-config",
	Title:    "Role propagation config",
	Description: "Manages whether role change propagation is enabled for the tenant. There is one configuration per tenant; " +
		"destroying the resource only removes it from Terraform state.",
	Path:         "/v2026/role-propagation-config",
	Experimental: true,
	Mode:         tenantSettingsPutBody,
	Fields: []tenantSettingsField{
		{Name: "enabled", Path: []string{"enabled"}, Kind: tenantSettingsBool,
			Description: "Whether the role change propagation process is enabled"},
		{Name: "enabled_date", Path: []string{"enabledDate"}, Kind: tenantSettingsString, ReadOnly: true,
			Description: "Time when role change propagation was last enabled"},
		{Name: "created_date", Path: []string{"createdDate"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "Time when the configuration was first created"},
		{Name: "modified_date", Path: []string{"modifiedDate"}, Kind: tenantSettingsString, ReadOnly: true,
			Description: "Time when the configuration was last updated"},
	},
}

func NewRolePropagationConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: rolePropagationConfigSpec}
}
