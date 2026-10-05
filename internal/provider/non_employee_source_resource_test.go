package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type nonEmployeeSourceTestPrivate map[string][]byte

func (p nonEmployeeSourceTestPrivate) GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics) {
	return p[key], nil
}

func nonEmployeeSourceTestModel(t *testing.T) NonEmployeeSourceModel {
	t.Helper()
	ctx := context.Background()
	owner, d := types.ListValueFrom(ctx, nonEmployeeSourceOwnerObjectType, []NonEmployeeSourceOwnerModel{{ID: types.StringValue("owner-1")}})
	approvers, d2 := types.ListValueFrom(ctx, types.StringType, []string{"a-1", "a-2"})
	managers, d3 := types.SetValueFrom(ctx, types.StringType, []string{"m-1"})
	if d.HasError() || d2.HasError() || d3.HasError() {
		t.Fatalf("unexpected diagnostics")
	}
	return NonEmployeeSourceModel{
		ID:                  types.StringValue("nes-1"),
		SourceID:            types.StringValue("src-1"),
		CloudExternalID:     types.StringValue("99"),
		Name:                types.StringValue("Contractors"),
		Description:         types.StringValue("External contractors"),
		Owner:               owner,
		ManagementWorkgroup: types.StringValue("wg-1"),
		Approvers:           approvers,
		AccountManagers:     managers,
		Created:             types.StringValue("2026-01-01T00:00:00Z"),
		Modified:            types.StringValue("2026-01-01T00:00:00Z"),
	}
}

func TestNonEmployeeSourceClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotContentType = r.Method, r.URL.Path, string(body), r.Header.Get("Content-Type")
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/non-employee-sources":
			_, _ = w.Write([]byte(`[{"id":"nes-1","sourceId":"src-1","name":"Contractors","description":"d"}]`))
		default:
			_, _ = w.Write([]byte(`{"id":"nes-1","sourceId":"src-1","name":"Contractors","description":"External contractors","approvers":[{"type":"IDENTITY","id":"a-1"},{"type":"GOVERNANCE_GROUP","id":"a-2"}],"accountManagers":[{"type":"IDENTITY","id":"m-1"}],"cloudExternalId":"99","created":"2026-01-01T00:00:00Z","modified":"2026-01-01T00:00:00Z"}`))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := nonEmployeeSourceTestModel(t)
	created, err := client.CreateNonEmployeeSource(ctx, nonEmployeeSourceFromModel(ctx, plan, &diags))
	want := `{"name":"Contractors","description":"External contractors","owner":{"id":"owner-1"},"managementWorkgroup":"wg-1","approvers":[{"id":"a-1"},{"id":"a-2"}],"accountManagers":[{"id":"m-1"}]}`
	if err != nil || diags.HasError() || gotMethod != http.MethodPost || gotPath != "/v2026/non-employee-sources" || gotBody != want {
		t.Fatalf("unexpected create request %s %s %s (%v %v)", gotMethod, gotPath, gotBody, err, diags)
	}
	if created.CloudExternalID != "99" || created.SourceID != "src-1" {
		t.Fatalf("unexpected create response %+v", created)
	}

	if _, err := client.GetNonEmployeeSource(ctx, "nes-1"); err != nil || gotMethod != http.MethodGet || gotPath != "/v2026/non-employee-sources/nes-1" {
		t.Fatalf("unexpected get request %s %s (%v)", gotMethod, gotPath, err)
	}
	if found, err := client.GetNonEmployeeSourceByName(ctx, "Contractors"); err != nil || found.ID != "nes-1" {
		t.Fatalf("unexpected lookup by name %+v (%v)", found, err)
	}

	if _, err := client.PatchNonEmployeeSource(ctx, "nes-1", []jsonPatchOp{{Op: "replace", Path: "/name", Value: "New"}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/non-employee-sources/nes-1" || gotContentType != "application/json-patch+json" {
		t.Fatalf("unexpected patch request %s %s %s", gotMethod, gotPath, gotContentType)
	}
	if err := client.DeleteNonEmployeeSource(ctx, "nes-1"); err != nil || gotMethod != http.MethodDelete {
		t.Fatalf("unexpected delete request %s (%v)", gotMethod, err)
	}
}

func TestNonEmployeeSourceStateKeepsWriteOnlyFields(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	data := nonEmployeeSourceTestModel(t)
	api := &NonEmployeeSource{ID: "nes-1", SourceID: "src-1", Name: "Renamed", Description: "External contractors",
		Approvers: []NonEmployeeSourceIdentityRef{{ID: "a-2", Type: "IDENTITY"}}, Created: "c", Modified: "m"}
	setNonEmployeeSourceState(ctx, &data, api, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if data.Name.ValueString() != "Renamed" || len(data.Approvers.Elements()) != 1 || data.Modified.ValueString() != "m" {
		t.Errorf("expected API values to be refreshed, got %+v", data)
	}
	if len(data.Owner.Elements()) != 1 || data.ManagementWorkgroup.ValueString() != "wg-1" || data.CloudExternalID.ValueString() != "99" {
		t.Errorf("expected fields the API does not return to be kept, got %+v", data)
	}
	if data.AccountManagers.IsNull() || len(data.AccountManagers.Elements()) != 0 {
		t.Errorf("expected configured account managers to become an empty set, got %s", data.AccountManagers)
	}

	unset := NonEmployeeSourceModel{Approvers: types.ListNull(types.StringType), AccountManagers: types.SetNull(types.StringType), Owner: types.ListNull(nonEmployeeSourceOwnerObjectType)}
	setNonEmployeeSourceState(ctx, &unset, api, &diags)
	if !unset.AccountManagers.IsNull() || !unset.Owner.IsNull() || !unset.ManagementWorkgroup.IsNull() {
		t.Errorf("expected unset attributes to stay null, got %+v", unset)
	}
}

func TestNonEmployeeSourcePatchOps(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := nonEmployeeSourceTestModel(t)
	plan := nonEmployeeSourceTestModel(t)
	if ops := nonEmployeeSourcePatchOps(ctx, plan, state, &diags); len(ops) != 0 {
		t.Fatalf("expected no operations, got %+v", ops)
	}
	plan.Description = types.StringValue("Changed")
	plan.AccountManagers = types.SetNull(types.StringType)
	encoded, _ := json.Marshal(nonEmployeeSourcePatchOps(ctx, plan, state, &diags))
	want := `[{"op":"replace","path":"/description","value":"Changed"},{"op":"replace","path":"/accountManagers","value":[]}]`
	if string(encoded) != want || diags.HasError() {
		t.Fatalf("unexpected operations\n got %s\nwant %s", encoded, want)
	}
}

func TestNonEmployeeSourceWriteOnlyApplied(t *testing.T) {
	ctx := context.Background()
	if nonEmployeeSourceWriteOnlyApplied(ctx, nil) || nonEmployeeSourceWriteOnlyApplied(ctx, nonEmployeeSourceTestPrivate{}) {
		t.Fatalf("expected imported resources to adopt the configuration")
	}
	if !nonEmployeeSourceWriteOnlyApplied(ctx, nonEmployeeSourceTestPrivate{nonEmployeeSourceWriteOnlyKey: []byte("true")}) {
		t.Fatalf("expected resources created by Terraform to require replacement")
	}
}
