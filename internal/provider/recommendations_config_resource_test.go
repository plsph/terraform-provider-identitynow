package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const recommendationsConfigTestDoc = `{"recommenderFeatures": ["jobTitle", "department"], "peerGroupPercentageThreshold": 0.5, "runAutoSelectOnce": false, "onlyTuneThreshold": false}`

func TestRecommendationsConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewRecommendationsConfigResource, "/v2026/recommendations/config", recommendationsConfigTestDoc)
}

// run_auto_select_once and only_tune_threshold are one-shot requests that the pipeline resets:
// the reset is not drift, and an unchanged request is not sent again.
func TestRecommendationsConfigOneShotFlags(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewRecommendationsConfigResource, "/v2026/recommendations/config", recommendationsConfigTestDoc)
	configured := map[string]attr.Value{"run_auto_select_once": types.BoolValue(true), "peer_group_percentage_threshold": types.Float64Value(0.5)}
	state := h.create(configured)
	if writes := h.api.writes(); len(writes) != 1 || writes[0].Body.(map[string]interface{})["runAutoSelectOnce"] != true {
		t.Fatalf("create must send the request, got %+v", writes)
	}

	// The pipeline runs and resets the flag.
	h.api.doc["runAutoSelectOnce"] = false
	refreshed := h.read(state)
	if !refreshed.Raw.Equal(state.Raw) {
		t.Fatalf("the reset must not show as drift:\n%s\n%s", state.Raw, refreshed.Raw)
	}

	// Another change does not send the unchanged request again.
	h.api.requests = nil
	updated := h.update(refreshed, map[string]attr.Value{"run_auto_select_once": types.BoolValue(true), "peer_group_percentage_threshold": types.Float64Value(0.75)})
	writes := h.api.writes()
	if len(writes) != 1 || writes[0].Body.(map[string]interface{})["runAutoSelectOnce"] != false || writes[0].Body.(map[string]interface{})["peerGroupPercentageThreshold"] != 0.75 {
		t.Fatalf("unexpected requests %+v", writes)
	}
	h.checkValues("update", updated, map[string]attr.Value{"run_auto_select_once": types.BoolValue(true)})

	// Changing the request sends it.
	h.api.requests = nil
	h.update(updated, map[string]attr.Value{"run_auto_select_once": types.BoolValue(false), "only_tune_threshold": types.BoolValue(true)})
	writes = h.api.writes()
	if len(writes) != 1 || writes[0].Body.(map[string]interface{})["runAutoSelectOnce"] != false || writes[0].Body.(map[string]interface{})["onlyTuneThreshold"] != true {
		t.Fatalf("unexpected requests %+v", writes)
	}

	// Import shows the API value.
	if values := h.values(h.read(h.imported())); !values["only_tune_threshold"].Equal(types.BoolValue(true)) {
		t.Fatalf("unexpected imported values %v", values)
	}
}
