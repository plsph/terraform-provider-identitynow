package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewSdiStatusCheckConfigResource)
}

// sdiStatusCheckConfigSpec describes /v2026/service-desk-integrations/status-check-configuration
// (GET/PUT). The API represents both numbers as strings.
var sdiStatusCheckConfigSpec = &tenantSettingsSpec{
	TypeName: "sdi_status_check_config",
	ID:       "sdi-status-check-config",
	Title:    "SDI status check config",
	Description: "Manages the tenant-wide status check configuration of queued service desk integration tickets. There is one configuration per tenant: " +
		"only the configured settings are managed, the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path: "/v2026/service-desk-integrations/status-check-configuration",
	Mode: tenantSettingsPutMerged,
	Fields: []tenantSettingsField{
		{Name: "provisioning_status_check_interval_minutes", Path: []string{"provisioningStatusCheckIntervalMinutes"}, Kind: tenantSettingsInt64String,
			Description: "Interval in minutes between status checks"},
		{Name: "provisioning_max_status_check_days", Path: []string{"provisioningMaxStatusCheckDays"}, Kind: tenantSettingsInt64String,
			Description: "Maximum number of days the status is checked"},
	},
}

func NewSdiStatusCheckConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: sdiStatusCheckConfigSpec}
}
