package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const managedClientTestResponse = `{"id":"client-1","clientId":"cid-1","clusterId":"mc-1","description":"","name":"VA-cid-1",
"status":"NORMAL","type":"VA","clusterType":"idn","secret":"api-key","createdAt":"2026-01-01T00:00:00Z",
"updatedAt":"2026-01-02T00:00:00Z","ipAddress":null,"provisionStatus":"PROVISIONED"}`

func TestManagedClientClient(t *testing.T) {
	client, requests := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		if r.Method == http.MethodDelete {
			return http.StatusNoContent, ""
		}
		return http.StatusOK, managedClientTestResponse
	})
	ctx := context.Background()
	created, err := client.CreateManagedClient(ctx, &ManagedClientRequest{ClusterID: "mc-1", Description: stringPointer(types.StringValue("VA 1"))})
	if err != nil || created.Secret != "api-key" || created.ClientID != "cid-1" {
		t.Fatalf("unexpected create result %+v (%v)", created, err)
	}
	if got := (*requests)[0]; got.Method != http.MethodPost || got.Path != "/v2026/managed-clients" || got.Body != `{"clusterId":"mc-1","description":"VA 1"}` {
		t.Fatalf("unexpected create request %+v", got)
	}
	if _, err := client.PatchManagedClient(ctx, "client-1", []jsonPatchOp{{Op: "replace", Path: "/name", Value: "VA 1"}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[1]; got.Method != http.MethodPatch || got.Path != "/v2026/managed-clients/client-1" || got.ContentType != "application/json-patch+json" {
		t.Fatalf("unexpected patch request %+v", got)
	}
	if err := client.DeleteManagedClient(ctx, "client-1"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[2]; got.Method != http.MethodDelete || got.Path != "/v2026/managed-clients/client-1" {
		t.Fatalf("unexpected delete request %+v", got)
	}
}

func TestManagedClientState(t *testing.T) {
	var api ManagedClient
	if err := json.Unmarshal([]byte(managedClientTestResponse), &api); err != nil {
		t.Fatal(err)
	}
	unknown := types.StringUnknown()
	data := ManagedClientModel{
		ID: unknown, ClusterID: types.StringValue("mc-1"), Name: unknown, Description: types.StringNull(), Type: unknown,
		ClientID: unknown, Secret: unknown, Status: unknown, ClusterType: unknown, AlertKey: unknown, APIGatewayBaseURL: unknown,
		IPAddress: unknown, LastSeen: unknown, SinceLastSeen: unknown, VaDownloadURL: unknown, VaVersion: unknown,
		ProvisionStatus: unknown, CreatedAt: unknown, UpdatedAt: unknown,
	}
	setManagedClientState(&data, &api, false)
	serviceDeskIntegrationTestAssertKnown(t, data)
	serviceDeskIntegrationTestSetState(t, NewManagedClientResource(), &data)
	if data.Name.ValueString() != "VA-cid-1" || data.Type.ValueString() != "VA" || data.Secret.ValueString() != "api-key" || !data.Description.IsNull() || !data.IPAddress.IsNull() {
		t.Errorf("unexpected state after create %+v", data)
	}

	// The secret is only returned once; later reads keep it and an empty description stays null.
	api.Secret = ""
	api.Name = "Renamed"
	setManagedClientState(&data, &api, true)
	if data.Secret.ValueString() != "api-key" || !data.Description.IsNull() || data.Name.ValueString() != "Renamed" {
		t.Errorf("unexpected state after refresh %+v", data)
	}

	state := data
	plan := data
	plan.Description = types.StringValue("VA 1")
	ops := managedClientPatchOps(plan, state)
	if len(ops) != 1 || ops[0].Path != "/description" || ops[0].Value != "VA 1" {
		t.Fatalf("unexpected ops %+v", ops)
	}
	ops = managedClientPatchOps(state, plan)
	if len(ops) != 1 || ops[0].Value != "" {
		t.Fatalf("a removed description must be cleared, got %+v", ops)
	}

	// An update keeps the known planned id, client_id and created_at.
	updated := plan
	updated.UpdatedAt = types.StringUnknown()
	other := api
	other.ID, other.ClientID, other.CreatedAt = "other", "other-cid", "2030-01-01T00:00:00Z"
	setManagedClientState(&updated, &other, false)
	if !updated.ID.Equal(plan.ID) || !updated.ClientID.Equal(plan.ClientID) || !updated.CreatedAt.Equal(plan.CreatedAt) || updated.UpdatedAt.IsUnknown() {
		t.Errorf("known planned values must be kept, got %+v", updated)
	}

	ds := managedClientDataSourceState(&api)
	serviceDeskIntegrationTestSetDataSourceState(t, NewManagedClientDataSource(), &ds)
	if ds.ClientID.ValueString() != "cid-1" {
		t.Errorf("unexpected data source state %+v", ds)
	}
}
