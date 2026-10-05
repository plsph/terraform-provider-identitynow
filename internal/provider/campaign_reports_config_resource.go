package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewCampaignReportsConfigResource)
}

// campaignReportsConfigSpec describes /v2026/campaigns/reports-configuration (GET/PUT).
var campaignReportsConfigSpec = &tenantSettingsSpec{
	TypeName: "campaign_reports_config",
	ID:       "campaign-reports-config",
	Title:    "Campaign reports config",
	Description: "Manages the tenant-wide certification campaign reports configuration. There is one configuration per tenant; " +
		"destroying the resource only removes it from Terraform state.",
	Path: "/v2026/campaigns/reports-configuration",
	Mode: tenantSettingsPutMerged,
	Fields: []tenantSettingsField{
		{Name: "identity_attribute_columns", Path: []string{"identityAttributeColumns"}, Kind: tenantSettingsStringList,
			Description: "Identity attributes added as custom columns to campaign reports"},
	},
}

func NewCampaignReportsConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: campaignReportsConfigSpec}
}
