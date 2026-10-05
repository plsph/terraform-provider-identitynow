package provider

import "testing"

func TestSessionConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewSessionConfigResource, "/v2026/auth-org/session-config", `{"maxIdleTime": 15, "rememberMe": true, "maxSessionTime": 45}`)
}
