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

func TestParameterClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotContentType = r.Method, r.URL.Path, string(body), r.Header.Get("Content-Type")
		if r.Method == http.MethodDelete {
			return
		}
		_, _ = w.Write([]byte(`{"id":"p-1","ownerId":"i-1","name":"db","type":"password","primaryField":"username",
			"publicFields":{"username":"svc"},"lastModifiedAt":"t1","privateFieldsLastModifiedAt":"t1"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	created, err := client.CreateParameter(ctx, &ParameterRequest{
		OwnerID: "i-1", Name: "db", Type: "password", PublicFields: map[string]interface{}{"username": "svc"}, PrivateFields: "jwe",
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := `{"ownerId":"i-1","name":"db","type":"password","publicFields":{"username":"svc"},"privateFields":"jwe"}`
	if gotMethod != http.MethodPost || gotPath != "/v2026/parameter-storage/parameters" || gotBody != want {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	data := ParameterModel{
		ID: types.StringUnknown(), Name: types.StringValue("db"), Description: types.StringNull(), Type: types.StringValue("password"),
		OwnerID: types.StringValue("i-1"), PublicFieldsJSON: types.StringValue(`{ "username": "svc" }`), PrivateFields: types.StringValue("jwe"),
		PrimaryField: types.StringUnknown(), LastModifiedAt: types.StringUnknown(), PrivateFieldsLastModifiedAt: types.StringUnknown(),
	}
	parameterResolveComputed(&data, created)
	if data.ID.ValueString() != "p-1" || data.PrimaryField.ValueString() != "username" || data.PrivateFields.ValueString() != "jwe" ||
		data.PublicFieldsJSON.ValueString() != `{ "username": "svc" }` || data.PrivateFieldsLastModifiedAt.ValueString() != "t1" {
		t.Fatalf("unexpected state after create %+v", data)
	}

	if _, err := client.PatchParameter(ctx, "p-1", []jsonPatchOp{{Op: "replace", Path: "/name", Value: "db2"}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/parameter-storage/parameters/p-1" || gotContentType != "application/json-patch+json" {
		t.Fatalf("unexpected patch request %s %s %s", gotMethod, gotPath, gotContentType)
	}
	if err := client.DeleteParameter(ctx, "p-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/parameter-storage/parameters/p-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestParameterStateKeepsPrivateFieldsUntilChangedOutside(t *testing.T) {
	data := ParameterModel{
		Description: types.StringNull(), PublicFieldsJSON: types.StringValue(`{"username": "svc"}`),
		PrivateFields: types.StringValue("jwe"), PrivateFieldsLastModifiedAt: types.StringValue("t1"),
	}
	parameter := &Parameter{ID: "p-1", OwnerID: "i-1", Name: "db", Type: "password", PublicFields: map[string]interface{}{"username": "svc"},
		LastModifiedAt: "t2", PrivateFieldsLastModifiedAt: "t1"}
	setParameterState(&data, parameter)
	if data.PrivateFields.ValueString() != "jwe" || !data.Description.IsNull() || data.PublicFieldsJSON.ValueString() != `{"username": "svc"}` {
		t.Fatalf("unexpected refreshed state %+v", data)
	}
	parameter.PrivateFieldsLastModifiedAt = "t3"
	setParameterState(&data, parameter)
	if !data.PrivateFields.IsNull() || data.PrivateFieldsLastModifiedAt.ValueString() != "t3" {
		t.Fatalf("expected private fields changed outside Terraform to be cleared, got %+v", data)
	}
}

func TestParameterPatchOps(t *testing.T) {
	state := ParameterModel{
		Name: types.StringValue("db"), Description: types.StringValue("d"), OwnerID: types.StringValue("i-1"),
		PublicFieldsJSON: types.StringValue(`{"username":"svc"}`), PrivateFields: types.StringValue("jwe"),
	}
	plan := state
	plan.Description = types.StringNull()
	plan.PublicFieldsJSON = types.StringNull()
	plan.PrivateFields = types.StringValue("jwe2")
	ops, err := parameterPatchOps(plan, state)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	encoded, _ := json.Marshal(ops)
	want := `[{"op":"remove","path":"/description"},{"op":"replace","path":"/publicFields","value":{}},{"op":"replace","path":"/privateFields","value":"jwe2"}]`
	if string(encoded) != want {
		t.Fatalf("unexpected patch ops %s", encoded)
	}
	// Removing the private fields from the configuration does not change them.
	plan = state
	plan.PrivateFields = types.StringNull()
	if ops, _ := parameterPatchOps(plan, state); len(ops) != 0 {
		t.Fatalf("expected no ops, got %+v", ops)
	}
}

func TestParameterTimestampsEqual(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"2026-01-01T10:00:00Z", "2026-01-01T10:00:00.000Z", true},
		{"2026-01-01T10:00:00Z", "2026-01-01T12:00:00+02:00", true},
		{"2026-01-01T10:00:00.123456Z", "2026-01-01T10:00:00.123456", true},
		{"2026-01-01T10:00:00Z", "2026-01-01T10:00:01Z", false},
		{"t1", "t1", true},
		{"t1", "t2", false},
	} {
		if got := parameterTimestampsEqual(tc.a, tc.b); got != tc.want {
			t.Errorf("parameterTimestampsEqual(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}

	data := ParameterModel{
		Description: types.StringNull(), PublicFieldsJSON: types.StringNull(),
		PrivateFields: types.StringValue("jwe"), PrivateFieldsLastModifiedAt: types.StringValue("2026-01-01T10:00:00Z"),
	}
	setParameterState(&data, &Parameter{ID: "p-1", PrivateFieldsLastModifiedAt: "2026-01-01T10:00:00.000Z"})
	if data.PrivateFields.ValueString() != "jwe" || data.PrivateFieldsLastModifiedAt.ValueString() != "2026-01-01T10:00:00Z" {
		t.Errorf("an equal timestamp in another format must keep the private fields, got %+v", data)
	}
}

func TestParameterReadBack(t *testing.T) {
	var gotMethod, gotPath string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"id":"p-1","ownerId":"i-1","name":"db","type":"password","lastModifiedAt":"2026-01-01T10:00:00.000Z","privateFieldsLastModifiedAt":"2026-01-01T10:00:00.000Z"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	var diags diag.Diagnostics
	parameter := parameterReadBack(context.Background(), client, &Parameter{ID: "p-1", PrivateFieldsLastModifiedAt: "2026-01-01T10:00:00Z"}, &diags)
	if diags.HasError() || gotMethod != http.MethodGet || gotPath != "/v2026/parameter-storage/parameters/p-1" ||
		parameter.PrivateFieldsLastModifiedAt != "2026-01-01T10:00:00.000Z" {
		t.Fatalf("unexpected read back %s %s %+v (%v)", gotMethod, gotPath, parameter, diags)
	}
}
