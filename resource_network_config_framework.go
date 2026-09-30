package main

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewNetworkConfigResource)
}

// networkConfigSpec describes /v2026/auth-org/network-config. GET returns 404 while the tenant has
// no network configuration; Create then uses POST, otherwise it and Update use JSON Patch of the
// configured settings.
var networkConfigSpec = &tenantSettingsSpec{
	TypeName: "network_config",
	ID:       "network-config",
	Title:    "Network config",
	Description: "Manages the tenant-wide network (IP range and geolocation) authentication restrictions. There is one configuration per tenant: " +
		"it is created when it does not exist yet, only the configured settings are managed and the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path:           "/v2026/auth-org/network-config",
	Mode:           tenantSettingsPatch,
	CreateWithPost: true,
	Fields: []tenantSettingsField{
		{Name: "range", Path: []string{"range"}, Kind: tenantSettingsStringList,
			Description: "IP addresses or subnets (CIDR), e.g. `10.0.0.0/8`"},
		{Name: "geolocation", Path: []string{"geolocation"}, Kind: tenantSettingsStringList,
			Description: "Two-letter uppercase country codes, e.g. `PL`"},
		{Name: "whitelisted", Path: []string{"whitelisted"}, Kind: tenantSettingsBool,
			Description: "Whether the IP ranges and countries are allowed (`true`) or blocked (`false`)"},
	},
}

func NewNetworkConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: networkConfigSpec}
}
