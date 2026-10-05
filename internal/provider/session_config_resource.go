package provider

import "github.com/hashicorp/terraform-plugin-framework/resource"

func init() {
	registerResource(NewSessionConfigResource)
}

// sessionConfigSpec describes /v2026/auth-org/session-config (GET/PATCH).
var sessionConfigSpec = &tenantSettingsSpec{
	TypeName: "session_config",
	ID:       "session-config",
	Title:    "Session config",
	Description: "Manages the tenant-wide user session configuration. There is one configuration per tenant: " +
		"only the configured settings are managed, the other settings keep their current value. " +
		"Destroying the resource only removes it from Terraform state.",
	Path: "/v2026/auth-org/session-config",
	Mode: tenantSettingsPatch,
	Fields: []tenantSettingsField{
		{Name: "max_idle_time", Path: []string{"maxIdleTime"}, Kind: tenantSettingsInt64,
			Description: "Maximum time in minutes a session can be idle"},
		{Name: "max_session_time", Path: []string{"maxSessionTime"}, Kind: tenantSettingsInt64,
			Description: "Maximum session time in minutes"},
		{Name: "remember_me", Path: []string{"rememberMe"}, Kind: tenantSettingsBool,
			Description: "Whether 'remember me' is enabled on the login page"},
	},
}

func NewSessionConfigResource() resource.Resource {
	return &tenantSettingsResource{spec: sessionConfigSpec}
}
