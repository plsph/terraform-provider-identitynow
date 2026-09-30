package main

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewUiMetadataResource)
}

// uiMetadataSpec describes /v2026/ui-metadata/tenant (GET/PUT, experimental). The request and
// response schemas are different objects, so the body is built from the writable settings only.
var uiMetadataSpec = &tenantSettingsSpec{
	TypeName: "ui_metadata",
	ID:       "ui-metadata",
	Title:    "UI metadata",
	Description: "Manages the tenant-wide user interface metadata (login page labels and iframe allow list). There is one configuration per tenant: " +
		"only the configured settings are managed, the other settings keep their current value. " +
		"Changes can take up to 5 minutes to take effect. Destroying the resource only removes it from Terraform state.",
	Path:         "/v2026/ui-metadata/tenant",
	Experimental: true,
	Mode:         tenantSettingsPutBody,
	Fields: []tenantSettingsField{
		{Name: "iframe_white_list", Path: []string{"iframeWhiteList"}, Kind: tenantSettingsString,
			Description: "Space-separated domains that may embed the non-authenticated parts of IdentityNow (e.g. password reset) in an iframe"},
		{Name: "username_label", Path: []string{"usernameLabel"}, Kind: tenantSettingsString,
			Description: "Label of the username field on the login page"},
		{Name: "username_empty_text", Path: []string{"usernameEmptyText"}, Kind: tenantSettingsString,
			Description: "Placeholder text of the username field on the login page"},
	},
}

func NewUiMetadataResource() resource.Resource {
	return &tenantSettingsResource{spec: uiMetadataSpec}
}
