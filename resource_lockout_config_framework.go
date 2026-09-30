package main

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewLockoutConfigResource)
}

// lockoutConfigSpec describes /v2026/auth-org/lockout-config (GET/PATCH).
var lockoutConfigSpec = &tenantSettingsSpec{
	TypeName: "lockout_config",
	ID:       "lockout-config",
	Title:    "Lockout config",
	Description: "Manages the tenant-wide authentication lockout configuration. There is one configuration per tenant: " +
		"only the configured settings are managed, the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path: "/v2026/auth-org/lockout-config",
	Mode: tenantSettingsPatch,
	Fields: []tenantSettingsField{
		{Name: "maximum_attempts", Path: []string{"maximumAttempts"}, Kind: tenantSettingsInt64,
			Description: "Maximum number of failed authentication attempts before the user is locked out"},
		{Name: "lockout_duration", Path: []string{"lockoutDuration"}, Kind: tenantSettingsInt64,
			Description: "Time in minutes a user is locked out"},
		{Name: "lockout_window", Path: []string{"lockoutWindow"}, Kind: tenantSettingsInt64,
			Description: "Rolling window in minutes in which failed attempts count towards the maximum"},
	},
}

func NewLockoutConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: lockoutConfigSpec}
}
