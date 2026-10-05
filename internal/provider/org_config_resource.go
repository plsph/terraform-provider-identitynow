package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func init() {
	registerResource(NewOrgConfigResource)
	registerDataSource(NewOrgConfigDataSource)
}

// orgConfigSpec describes /v2026/org-config (GET/PATCH). Updates send JSON Patch replace
// operations for the configured settings that changed.
var orgConfigSpec = &tenantSettingsSpec{
	TypeName: "org_config",
	ID:       "org-config",
	Title:    "Org config",
	Description: "Manages the tenant-wide organization configuration, e.g. the time zone. There is one configuration per tenant: " +
		"only the configured settings are managed, the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path: "/v2026/org-config",
	Mode: tenantSettingsPatch,
	Fields: []tenantSettingsField{
		{Name: "org_name", Path: []string{"orgName"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "Name of the org"},
		{Name: "time_zone", Path: []string{"timeZone"}, Kind: tenantSettingsString,
			Description: "Time zone of the org, e.g. `Europe/Warsaw`. It determines when scheduled tasks run. Valid values are returned by the `identitynow_valid_time_zones` data source"},
		{Name: "lcs_change_honors_source_enable_feature", Path: []string{"lcsChangeHonorsSourceEnableFeature"}, Kind: tenantSettingsBool,
			Description: "Whether the LCS_CHANGE_HONORS_SOURCE_ENABLE_FEATURE flag is enabled"},
		{Name: "iai_enable_certification_recommendations", Path: []string{"iaiEnableCertificationRecommendations"}, Kind: tenantSettingsBool,
			Description: "Whether AI certification recommendations are enabled"},
		{Name: "arm_customer_id", Path: []string{"armCustomerId"}, Kind: tenantSettingsString,
			Description: "Access Risk Management (ARM) customer ID"},
		{Name: "arm_sap_system_id_mappings", Path: []string{"armSapSystemIdMappings"}, Kind: tenantSettingsString,
			Description: "ARM mappings of IdentityNow source IDs to ARM system IDs, as returned by the API"},
		{Name: "arm_auth", Path: []string{"armAuth"}, Kind: tenantSettingsString, Sensitive: true, WriteOnly: true,
			Description: "ARM authentication string"},
		{Name: "arm_db", Path: []string{"armDb"}, Kind: tenantSettingsString,
			Description: "ARM database name"},
		{Name: "arm_sso_url", Path: []string{"armSsoUrl"}, Kind: tenantSettingsString,
			Description: "ARM SSO URL"},
		{Name: "sod_report_configs_json", Path: []string{"sodReportConfigs"}, Kind: tenantSettingsJSONArray,
			Description: "Separation of duties report columns, objects with `columnName`, `required`, `included` and `order`. Required columns cannot be changed"},
	},
}

func NewOrgConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: orgConfigSpec}
}

func NewOrgConfigDataSource() datasource.DataSource {
	return &tenantSettingsDataSource{spec: orgConfigSpec, typeName: "org_config", description: "Reads the tenant-wide organization configuration."}
}
