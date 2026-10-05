package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewAccessRequestConfigResource)
}

// accessRequestConfigSpec describes /v2026/access-request-config (GET/PUT). The PUT replaces the
// whole configuration, so updates use putMerged: settings that are not configured keep their
// current value. The request-on-behalf-of settings are typed; the entitlement request
// configuration (approval schemes, durations) is a JSON object merged key by key.
var accessRequestConfigSpec = &tenantSettingsSpec{
	TypeName: "access_request_config",
	ID:       "access-request-config",
	Title:    "Access request config",
	Description: "Manages the tenant-wide access request configuration. There is one configuration per tenant: " +
		"only the configured settings are managed, the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path: "/v2026/access-request-config",
	Mode: tenantSettingsPutMerged,
	Fields: []tenantSettingsField{
		{Name: "approvals_must_be_external", Path: []string{"approvalsMustBeExternal"}, Kind: tenantSettingsBool,
			Description: "Whether approvals must be processed by an external system. This blocks Request Center access requests for users who are not org admins"},
		{Name: "reauthorization_enabled", Path: []string{"reauthorizationEnabled"}, Kind: tenantSettingsBool,
			Description: "Whether reauthorization is enforced for appropriately configured access items"},
		{Name: "gov_group_visibility_enabled", Path: []string{"govGroupVisibilityEnabled"}, Kind: tenantSettingsBool,
			Description: "Whether requesters and requested-for users can see the names of governance group members when a request awaits the group's approval"},
		{Name: "request_on_behalf_of_anyone_by_anyone", Path: []string{"requestOnBehalfOfConfig", "allowRequestOnBehalfOfAnyoneByAnyone"}, Kind: tenantSettingsBool,
			Description: "Whether anyone can request access for anyone"},
		{Name: "request_on_behalf_of_employee_by_manager", Path: []string{"requestOnBehalfOfConfig", "allowRequestOnBehalfOfEmployeeByManager"}, Kind: tenantSettingsBool,
			Description: "Whether managers can request access for their direct reports"},
		{Name: "entitlement_request_config_json", Path: []string{"entitlementRequestConfig"}, Kind: tenantSettingsJSONObject,
			Description: "Default entitlement request configuration with `accessRequestConfig` and `revocationRequestConfig` objects (`approvalSchemes`, `requestCommentRequired`, `denialCommentRequired`, `reauthorizationRequired`, `requireEndDate`, `maxPermittedAccessDuration`)"},
	},
}

func NewAccessRequestConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: accessRequestConfigSpec}
}
