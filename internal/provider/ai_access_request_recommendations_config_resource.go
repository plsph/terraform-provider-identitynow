package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewAiAccessRequestRecommendationsConfigResource)
}

// aiAccessRequestRecommendationsConfigSpec describes /v2026/ai-access-request-recommendations/config
// (GET/PUT, experimental).
var aiAccessRequestRecommendationsConfigSpec = &tenantSettingsSpec{
	TypeName: "ai_access_request_recommendations_config",
	ID:       "ai-access-request-recommendations-config",
	Title:    "AI access request recommendations config",
	Description: "Manages the tenant-wide configuration of AI access request recommendations. There is one configuration per tenant: " +
		"only the configured settings are managed, the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path:         "/v2026/ai-access-request-recommendations/config",
	Experimental: true,
	Mode:         tenantSettingsPutMerged,
	Fields: []tenantSettingsField{
		{Name: "score_threshold", Path: []string{"scoreThreshold"}, Kind: tenantSettingsFloat64,
			Description: "Value the internal calculations must exceed to recommend access"},
		{Name: "start_date_attribute", Path: []string{"startDateAttribute"}, Kind: tenantSettingsString,
			Description: "Identity attribute with the start date of identities"},
		{Name: "restriction_attribute", Path: []string{"restrictionAttribute"}, Kind: tenantSettingsString,
			Description: "Identity attribute recommendations are restricted to"},
		{Name: "use_restriction_attribute", Path: []string{"useRestrictionAttribute"}, Kind: tenantSettingsBool,
			Description: "Whether only `restriction_attribute` is used to make recommendations"},
		{Name: "mover_attribute", Path: []string{"moverAttribute"}, Kind: tenantSettingsString,
			Description: "Identity attribute that tells whether an identity is a mover"},
		{Name: "joiner_attribute", Path: []string{"joinerAttribute"}, Kind: tenantSettingsString,
			Description: "Identity attribute that tells whether an identity is a joiner"},
	},
}

func NewAiAccessRequestRecommendationsConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: aiAccessRequestRecommendationsConfigSpec}
}
