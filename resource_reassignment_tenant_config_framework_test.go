package main

import "testing"

func TestReassignmentTenantConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewReassignmentTenantConfigResource, "/v2026/reassignment-configurations/tenant-config", `{"auditDetails": {"created": "2025-01-01T00:00:00Z", "createdBy": {"id": "id-1", "name": "Admin"}, "modified": "2026-01-01T00:00:00Z", "modifiedBy": {"id": "id-2", "name": "Other"}}, "configDetails": {"disabled": false}}`)
}
