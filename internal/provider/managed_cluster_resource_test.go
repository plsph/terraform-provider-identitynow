package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const managedClusterTestResponse = `{"id":"mc-1","name":"Cluster","pod":"stg01","org":"acme","type":"idn",
"configuration":{"gmtOffset":"-5","clusterExternalId":"ext-1","debug":null},"description":"VA cluster",
"clientType":"VA","ccgVersion":"77.0.0","pinnedConfig":false,"operational":true,"status":"NORMAL",
"publicKey":null,"alertKey":"","clientIds":["client-1"],"serviceCount":2,"createdAt":"2026-01-01T00:00:00Z",
"updatedAt":"2026-01-02T00:00:00Z","consolidatedHealthIndicatorsStatus":"NORMAL"}`

func managedClusterTestMap(t *testing.T, values map[string]string) types.Map {
	t.Helper()
	elements := map[string]attr.Value{}
	for k, v := range values {
		elements[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elements)
}

func TestManagedClusterClient(t *testing.T) {
	client, requests := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		switch {
		case r.Method == http.MethodDelete:
			return http.StatusNoContent, ""
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/managed-clusters":
			return http.StatusOK, "[" + managedClusterTestResponse + "]"
		}
		return http.StatusOK, managedClusterTestResponse
	})
	ctx := context.Background()
	description := "VA cluster"
	if _, err := client.CreateManagedCluster(ctx, &ManagedClusterRequest{Name: "Cluster", Type: "idn", Configuration: map[string]string{"gmtOffset": "-5"}, Description: &description}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	got := (*requests)[0]
	if got.Method != http.MethodPost || got.Path != "/v2026/managed-clusters" {
		t.Fatalf("unexpected create request %+v", got)
	}
	serviceDeskIntegrationTestJSONEqual(t, got.Body, `{"name":"Cluster","type":"idn","configuration":{"gmtOffset":"-5"},"description":"VA cluster"}`)

	updated, err := client.PatchManagedCluster(ctx, "mc-1", []jsonPatchOp{{Op: "replace", Path: "/name", Value: "Cluster"}})
	if err != nil || updated.Status != "NORMAL" || updated.ServiceCount != 2 {
		t.Fatalf("unexpected patch result %+v (%v)", updated, err)
	}
	if got := (*requests)[1]; got.Method != http.MethodPatch || got.Path != "/v2026/managed-clusters/mc-1" || got.ContentType != "application/json-patch+json" || got.Body != `[{"op":"replace","path":"/name","value":"Cluster"}]` {
		t.Fatalf("unexpected patch request %+v", got)
	}
	if _, err := client.GetManagedClusterByName(ctx, "Cluster"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := client.DeleteManagedCluster(ctx, "mc-1", false); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := client.DeleteManagedCluster(ctx, "mc-1", true); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[3]; got.Method != http.MethodDelete || got.Query != "" {
		t.Fatalf("unexpected delete request %+v", got)
	}
	if got := (*requests)[4]; got.Method != http.MethodDelete || got.Path != "/v2026/managed-clusters/mc-1" || got.Query != "removeClients=true" {
		t.Fatalf("unexpected delete request %+v", got)
	}
}

func TestManagedClusterStateAndPatches(t *testing.T) {
	ctx := context.Background()
	var api ManagedCluster
	if err := json.Unmarshal([]byte(managedClusterTestResponse), &api); err != nil {
		t.Fatal(err)
	}
	var diags diag.Diagnostics

	// Create keeps the planned configuration and resolves the computed attributes.
	var data ManagedClusterModel
	data.ID = types.StringUnknown()
	data.Name = types.StringValue("Cluster")
	data.Type = types.StringUnknown()
	data.Description = types.StringUnknown()
	data.Configuration = managedClusterTestMap(t, map[string]string{"gmtOffset": "-5"})
	data.RemoveClientsOnDestroy = types.BoolValue(false)
	setManagedClusterState(ctx, &data, &api, false, &diags)
	serviceDeskIntegrationTestAssertKnown(t, data)
	serviceDeskIntegrationTestSetState(t, NewManagedClusterResource(), &data)
	if data.Type.ValueString() != "idn" || data.Description.ValueString() != "VA cluster" || len(data.Configuration.Elements()) != 1 || !data.PublicKey.IsNull() {
		t.Errorf("unexpected state after create %+v", data)
	}

	// Read only refreshes the configured configuration keys.
	api.Configuration["gmtOffset"] = managedClusterTestStringPtr("-6")
	setManagedClusterState(ctx, &data, &api, true, &diags)
	if !data.Configuration.Equal(managedClusterTestMap(t, map[string]string{"gmtOffset": "-6"})) {
		t.Errorf("unexpected configuration after refresh %s", data.Configuration)
	}

	// Patches contain only the changed attributes, and the configuration is merged.
	state := data
	plan := data
	plan.Configuration = managedClusterTestMap(t, map[string]string{"timezone": "UTC"})
	ops := managedClusterPatchOps(ctx, plan, state, &api, &diags)
	if len(ops) != 1 || ops[0].Path != "/configuration" {
		t.Fatalf("unexpected ops %+v", ops)
	}
	merged := ops[0].Value.(map[string]*string)
	if _, ok := merged["gmtOffset"]; ok || *merged["timezone"] != "UTC" || *merged["clusterExternalId"] != "ext-1" {
		t.Errorf("unexpected merged configuration %+v", merged)
	}
	if _, ok := merged["debug"]; !ok {
		t.Errorf("unmanaged keys must be kept")
	}
	plan = state
	plan.Name = types.StringValue("Renamed")
	ops = managedClusterPatchOps(ctx, plan, state, &api, &diags)
	if len(ops) != 1 || ops[0].Path != "/name" || ops[0].Value != "Renamed" {
		t.Fatalf("unexpected ops %+v", ops)
	}
	if ops := managedClusterPatchOps(ctx, state, state, &api, &diags); len(ops) != 0 {
		t.Fatalf("expected no ops, got %+v", ops)
	}

	// Import has no configured keys, so the configuration stays null.
	// An update keeps the known planned id, pod, org and created_at.
	updated := state
	updated.UpdatedAt = types.StringUnknown()
	other := api
	other.ID, other.Pod, other.Org, other.CreatedAt = "other", "other-pod", "other-org", "2030-01-01T00:00:00Z"
	setManagedClusterState(ctx, &updated, &other, false, &diags)
	if !updated.ID.Equal(state.ID) || !updated.Pod.Equal(state.Pod) || !updated.Org.Equal(state.Org) || !updated.CreatedAt.Equal(state.CreatedAt) || updated.UpdatedAt.IsUnknown() {
		t.Errorf("known planned values must be kept, got %+v", updated)
	}

	var imported ManagedClusterModel
	imported.ID = types.StringValue("mc-1")
	setManagedClusterState(ctx, &imported, &api, true, &diags)
	serviceDeskIntegrationTestSetState(t, NewManagedClusterResource(), &imported)
	if !imported.Configuration.IsNull() || !imported.RemoveClientsOnDestroy.Equal(types.BoolValue(false)) {
		t.Errorf("unexpected imported state %+v", imported)
	}

	var ds ManagedClusterComputedModel
	setManagedClusterDataSourceState(ctx, &ds, &api, &diags)
	serviceDeskIntegrationTestSetDataSourceState(t, NewManagedClusterDataSource(), &ds)
	if len(ds.Configuration.Elements()) != 3 {
		t.Errorf("data source must return the full configuration, got %s", ds.Configuration)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func managedClusterTestStringPtr(v string) *string { return &v }
