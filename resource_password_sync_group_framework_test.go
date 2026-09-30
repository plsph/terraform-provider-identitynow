package main

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPasswordSyncGroupClient(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/password-sync-groups":
			_, _ = w.Write([]byte(`[{"id":"g-0","name":"other"},{"id":"g-1","name":"Sync","passwordPolicyId":"pp-1","sourceIds":["b","a"]}]`))
		default:
			_, _ = w.Write([]byte(`{"id":"g-1","name":"Sync","passwordPolicyId":"pp-1","sourceIds":["b","a"],"created":"c","modified":"m"}`))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	sourceIDs, _ := types.SetValueFrom(ctx, types.StringType, []string{"b", "a"})
	data := PasswordSyncGroupModel{ID: types.StringUnknown(), Name: types.StringValue("Sync"), PasswordPolicyID: types.StringValue("pp-1"), SourceIDs: sourceIDs,
		Created: types.StringUnknown(), Modified: types.StringUnknown()}
	created, err := client.CreatePasswordSyncGroup(ctx, passwordSyncGroupFromModel(ctx, data, &diags))
	if err != nil || gotMethod != http.MethodPost || gotPath != "/v2026/password-sync-groups" || gotBody != `{"name":"Sync","passwordPolicyId":"pp-1","sourceIds":["a","b"]}` {
		t.Fatalf("unexpected create request %s %s %s (%v)", gotMethod, gotPath, gotBody, err)
	}
	applyPasswordSyncGroupResponse(&data, created)
	if data.ID.ValueString() != "g-1" || data.Created.ValueString() != "c" || data.Modified.ValueString() != "m" || !data.SourceIDs.Equal(sourceIDs) {
		t.Fatalf("unexpected state after create %+v", data)
	}

	// Unset optional values are cleared with the full PUT request.
	cleared := PasswordSyncGroupModel{Name: types.StringValue("Sync"), PasswordPolicyID: types.StringNull(), SourceIDs: types.SetNull(types.StringType)}
	group := passwordSyncGroupFromModel(ctx, cleared, &diags)
	group.ID = "g-1"
	if _, err := client.UpdatePasswordSyncGroup(ctx, "g-1", group); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2026/password-sync-groups/g-1" || gotBody != `{"id":"g-1","name":"Sync","passwordPolicyId":null,"sourceIds":[]}` {
		t.Fatalf("unexpected update request %s %s %s", gotMethod, gotPath, gotBody)
	}

	found, err := client.GetPasswordSyncGroupByName(ctx, "Sync")
	if err != nil || found.ID != "g-1" {
		t.Fatalf("unexpected lookup result %+v (%v)", found, err)
	}
	if err := client.DeletePasswordSyncGroup(ctx, "g-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/password-sync-groups/g-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestPasswordSyncGroupStateKeepsUnsetOptionalValues(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	data := PasswordSyncGroupModel{ID: types.StringValue("g-1"), PasswordPolicyID: types.StringNull(), SourceIDs: types.SetNull(types.StringType)}
	setPasswordSyncGroupState(ctx, &data, &PasswordSyncGroup{ID: "g-1", Name: "Sync"}, &diags)
	if diags.HasError() || !data.PasswordPolicyID.IsNull() || !data.SourceIDs.IsNull() || !data.Created.IsNull() {
		t.Fatalf("expected unset values to stay null, got %+v (%v)", data, diags)
	}
	policyID := "pp-2"
	setPasswordSyncGroupState(ctx, &data, &PasswordSyncGroup{ID: "g-1", Name: "Sync", PasswordPolicyID: &policyID, SourceIDs: []string{"a"}}, &diags)
	if data.PasswordPolicyID.ValueString() != "pp-2" || len(data.SourceIDs.Elements()) != 1 {
		t.Fatalf("expected drift to be detected, got %+v", data)
	}
}
