package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func nonEmployeeSchemaAttributeTestModel() NonEmployeeSchemaAttributeModel {
	return NonEmployeeSchemaAttributeModel{
		ID:                  types.StringValue("attr-1"),
		NonEmployeeSourceID: types.StringValue("nes-1"),
		TechnicalName:       types.StringValue("costCenter"),
		Type:                types.StringValue("TEXT"),
		Label:               types.StringValue("Cost center"),
		HelpText:            types.StringValue("Enter the cost center"),
		Placeholder:         types.StringNull(),
		Required:            types.BoolValue(false),
		System:              types.BoolValue(false),
		Created:             types.StringValue("c"),
		Modified:            types.StringValue("m"),
	}
}

func TestNonEmployeeSchemaAttributeClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotContentType = r.Method, r.URL.Path, string(body), r.Header.Get("Content-Type")
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/non-employee-sources/nes-1/schema-attributes":
			_, _ = w.Write([]byte(`[{"id":"sys-1","system":true,"type":"TEXT","label":"First name","technicalName":"firstName"},{"id":"attr-1","type":"TEXT","label":"Cost center","technicalName":"costCenter"}]`))
		default:
			_, _ = w.Write([]byte(`{"id":"attr-1","system":false,"type":"TEXT","label":"Cost center","technicalName":"costCenter","helpText":"Enter the cost center","required":false,"created":"c","modified":"m"}`))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	if _, err := client.CreateNonEmployeeSchemaAttribute(ctx, "nes-1", nonEmployeeSchemaAttributeFromModel(nonEmployeeSchemaAttributeTestModel())); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := `{"type":"TEXT","label":"Cost center","technicalName":"costCenter","helpText":"Enter the cost center","required":false}`
	if gotMethod != http.MethodPost || gotPath != "/v2026/non-employee-sources/nes-1/schema-attributes" || gotBody != want {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	if _, err := client.GetNonEmployeeSchemaAttribute(ctx, "nes-1", "attr-1"); err != nil || gotPath != "/v2026/non-employee-sources/nes-1/schema-attributes/attr-1" {
		t.Fatalf("unexpected get request %s (%v)", gotPath, err)
	}
	if found, err := client.GetNonEmployeeSchemaAttributeByTechnicalName(ctx, "nes-1", "firstName"); err != nil || found.ID != "sys-1" {
		t.Fatalf("unexpected lookup %+v (%v)", found, err)
	}
	if _, err := client.GetNonEmployeeSchemaAttributeByTechnicalName(ctx, "nes-1", "missing"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := client.PatchNonEmployeeSchemaAttribute(ctx, "nes-1", "attr-1", []jsonPatchOp{{Op: "replace", Path: "/label", Value: "x"}}); err != nil ||
		gotMethod != http.MethodPatch || gotContentType != "application/json-patch+json" {
		t.Fatalf("unexpected patch request %s %s (%v)", gotMethod, gotContentType, err)
	}
	if err := client.DeleteNonEmployeeSchemaAttribute(ctx, "nes-1", "attr-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/non-employee-sources/nes-1/schema-attributes/attr-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestNonEmployeeSchemaAttributeStateAndPatch(t *testing.T) {
	data := nonEmployeeSchemaAttributeTestModel()
	required := true
	setNonEmployeeSchemaAttributeState(&data, &NonEmployeeSchemaAttribute{ID: "attr-1", Type: "TEXT", Label: "Cost center", TechnicalName: "costCenter", Required: &required, Modified: "m2"})
	if !data.Placeholder.IsNull() || data.HelpText.ValueString() != "" || !data.Required.ValueBool() || data.Modified.ValueString() != "m2" {
		t.Fatalf("unexpected state %+v", data)
	}

	state := nonEmployeeSchemaAttributeTestModel()
	plan := nonEmployeeSchemaAttributeTestModel()
	if ops := nonEmployeeSchemaAttributePatchOps(plan, state); len(ops) != 0 {
		t.Fatalf("expected no operations, got %+v", ops)
	}
	plan.HelpText = types.StringNull()
	plan.Required = types.BoolValue(true)
	encoded, _ := json.Marshal(nonEmployeeSchemaAttributePatchOps(plan, state))
	want := `[{"op":"replace","path":"/helpText","value":""},{"op":"replace","path":"/required","value":true}]`
	if string(encoded) != want {
		t.Fatalf("unexpected operations\n got %s\nwant %s", encoded, want)
	}
}
