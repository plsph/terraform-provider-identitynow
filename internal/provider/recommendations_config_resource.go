package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewRecommendationsConfigResource)
}

// recommendationsConfigSpec describes /v2026/recommendations/config (GET/PUT, experimental).
// run_auto_select_once and only_tune_threshold are one-shot requests that the recommendation
// pipeline resets after its next run; they are Trigger settings, so the reset is not drift and an
// unchanged value is not sent again.
var recommendationsConfigSpec = &tenantSettingsSpec{
	TypeName: "recommendations_config",
	ID:       "recommendations-config",
	Title:    "Recommendations config",
	Description: "Manages the tenant-wide configuration of certification recommendations. There is one configuration per tenant: " +
		"only the configured settings are managed, the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path:         "/v2026/recommendations/config",
	Experimental: true,
	Mode:         tenantSettingsPutMerged,
	Fields: []tenantSettingsField{
		{Name: "recommender_features", Path: []string{"recommenderFeatures"}, Kind: tenantSettingsStringList,
			Description: "Identity attributes used to calculate certification recommendations"},
		{Name: "peer_group_percentage_threshold", Path: []string{"peerGroupPercentageThreshold"}, Kind: tenantSettingsFloat64,
			Description: "Fraction between 0 and 1 that the recommendation calculation must exceed to recommend approval"},
		{Name: "run_auto_select_once", Path: []string{"runAutoSelectOnce"}, Kind: tenantSettingsBool, Trigger: true,
			Description: "Whether the next pipeline run selects new attributes and threshold values automatically"},
		{Name: "only_tune_threshold", Path: []string{"onlyTuneThreshold"}, Kind: tenantSettingsBool, Trigger: true,
			Description: "Whether the next pipeline run selects new threshold values automatically"},
	},
}

func NewRecommendationsConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: recommendationsConfigSpec}
}
