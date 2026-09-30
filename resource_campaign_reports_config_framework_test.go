package main

import "testing"

func TestCampaignReportsConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewCampaignReportsConfigResource, "/v2026/campaigns/reports-configuration", `{"identityAttributeColumns": ["firstname", "lastname"]}`)
}
