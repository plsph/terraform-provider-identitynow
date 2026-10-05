package provider

import "testing"

func TestLockoutConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewLockoutConfigResource, "/v2026/auth-org/lockout-config", `{"maximumAttempts": 5, "lockoutDuration": 15, "lockoutWindow": 5}`)
}
