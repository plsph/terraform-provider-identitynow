package main

import "testing"

func TestAiAccessRequestRecommendationsConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewAiAccessRequestRecommendationsConfigResource, "/v2026/ai-access-request-recommendations/config", `{"scoreThreshold": 0.5, "startDateAttribute": "startDate", "restrictionAttribute": "location", "moverAttribute": "isMover", "joinerAttribute": "isJoiner", "useRestrictionAttribute": true}`)
}
