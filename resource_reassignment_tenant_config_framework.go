package main

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewReassignmentTenantConfigResource)
}

// reassignmentTenantConfigSpec describes /v2026/reassignment-configurations/tenant-config (GET/PUT,
// experimental). The PUT request only has `configDetails`, the response also has audit details,
// so the body is built from the writable settings only.
var reassignmentTenantConfigSpec = &tenantSettingsSpec{
	TypeName: "reassignment_tenant_config",
	ID:       "reassignment-tenant-config",
	Title:    "Reassignment tenant config",
	Description: "Manages the tenant-wide work reassignment configuration. There is one configuration per tenant; " +
		"destroying the resource only removes it from Terraform state.",
	Path:         "/v2026/reassignment-configurations/tenant-config",
	Experimental: true,
	Mode:         tenantSettingsPutBody,
	Fields: []tenantSettingsField{
		{Name: "disabled", Path: []string{"configDetails", "disabled"}, Kind: tenantSettingsBool,
			Description: "Whether reassignment configurations are disabled for the tenant"},
		{Name: "created", Path: []string{"auditDetails", "created"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "Creation date of the configuration"},
		{Name: "created_by_id", Path: []string{"auditDetails", "createdBy", "id"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "ID of the identity that created the configuration"},
		{Name: "created_by_name", Path: []string{"auditDetails", "createdBy", "name"}, Kind: tenantSettingsString, ReadOnly: true, Stable: true,
			Description: "Name of the identity that created the configuration"},
		{Name: "modified", Path: []string{"auditDetails", "modified"}, Kind: tenantSettingsString, ReadOnly: true,
			Description: "Last modification date of the configuration"},
		{Name: "modified_by_id", Path: []string{"auditDetails", "modifiedBy", "id"}, Kind: tenantSettingsString, ReadOnly: true,
			Description: "ID of the identity that last modified the configuration"},
		{Name: "modified_by_name", Path: []string{"auditDetails", "modifiedBy", "name"}, Kind: tenantSettingsString, ReadOnly: true,
			Description: "Name of the identity that last modified the configuration"},
	},
}

func NewReassignmentTenantConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: reassignmentTenantConfigSpec}
}
