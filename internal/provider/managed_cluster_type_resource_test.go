package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestManagedClusterTypeClient(t *testing.T) {
	client, requests := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		if r.Method == http.MethodDelete {
			return http.StatusNoContent, ""
		}
		return http.StatusOK, `{"id":"mct-1","type":"custom","pod":"stg01","org":"acme","managedProcessIds":["p-1"]}`
	})
	ctx := context.Background()
	created, err := client.CreateManagedClusterType(ctx, &ManagedClusterType{Type: "custom", Pod: "stg01", Org: "acme", ManagedProcessIDs: []string{"p-1"}})
	if err != nil || created.ID != "mct-1" {
		t.Fatalf("unexpected create result %+v (%v)", created, err)
	}
	if got := (*requests)[0]; got.Method != http.MethodPost || got.Path != "/v2026/managed-cluster-types" || got.Body != `{"type":"custom","pod":"stg01","org":"acme","managedProcessIds":["p-1"]}` {
		t.Fatalf("unexpected create request %+v", got)
	}
	if _, err := client.PatchManagedClusterType(ctx, "mct-1", []jsonPatchOp{{Op: "replace", Path: "/pod", Value: "stg02"}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[1]; got.Method != http.MethodPatch || got.Path != "/v2026/managed-cluster-types/mct-1" || got.ContentType != "application/json-patch+json" || got.Body != `[{"op":"replace","path":"/pod","value":"stg02"}]` {
		t.Fatalf("unexpected patch request %+v", got)
	}
	if err := client.DeleteManagedClusterType(ctx, "mct-1"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[2]; got.Method != http.MethodDelete || got.Path != "/v2026/managed-cluster-types/mct-1" {
		t.Fatalf("unexpected delete request %+v", got)
	}
}

func TestManagedClusterTypeStateAndPatches(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	data := ManagedClusterTypeModel{ManagedProcessIDs: types.ListNull(types.StringType)}
	setManagedClusterTypeState(ctx, &data, &ManagedClusterType{ID: "mct-1", Type: "custom", Pod: "stg01", Org: "acme"}, &diags)
	serviceDeskIntegrationTestSetState(t, NewManagedClusterTypeResource(), &data)
	if !data.ManagedProcessIDs.IsNull() {
		t.Errorf("unset process IDs must stay null, got %s", data.ManagedProcessIDs)
	}

	plan := data
	plan.ManagedProcessIDs = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("p-1")})
	ops := managedClusterTypePatchOps(ctx, plan, data, &diags)
	if len(ops) != 1 || ops[0].Path != "/managedProcessIds" {
		t.Fatalf("unexpected ops %+v", ops)
	}
	ops = managedClusterTypePatchOps(ctx, data, plan, &diags)
	if len(ops) != 1 || len(ops[0].Value.([]string)) != 0 {
		t.Fatalf("removed process IDs must be replaced with an empty list, got %+v", ops)
	}
	if ops := managedClusterTypePatchOps(ctx, data, data, &diags); len(ops) != 0 {
		t.Fatalf("expected no ops, got %+v", ops)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}
