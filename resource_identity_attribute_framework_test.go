package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const identityAttributeTestResponse = `{"name":"costCenter","displayName":"Cost Center","standard":false,"type":"string","multi":false,"searchable":true,"system":false,
"sources":[{"type":"rule","properties":{"ruleType":"IdentityAttribute","ruleName":"Cost Center"}}]}`

func TestIdentityAttributeClient(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(identityAttributeTestResponse))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	plan := IdentityAttributeModel{
		Name:        types.StringValue("costCenter"),
		DisplayName: types.StringValue("Cost Center"),
		Standard:    types.BoolUnknown(),
		Type:        types.StringUnknown(),
		Multi:       types.BoolUnknown(),
		Searchable:  types.BoolValue(true),
		System:      types.BoolUnknown(),
		SourcesJSON: types.StringNull(),
	}
	created, err := client.CreateIdentityAttribute(ctx, identityAttributeFromModel(plan))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/identity-attributes" ||
		gotBody != `{"name":"costCenter","displayName":"Cost Center","searchable":true,"sources":[]}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	if created.Name != "costCenter" || *created.Type != "string" {
		t.Fatalf("unexpected response %+v", created)
	}

	// Updates send the full object from the plan.
	identityAttributeResolveApply(&plan, created)
	plan.SourcesJSON = types.StringValue(`[{"type":"rule","properties":{"ruleName":"Cost Center"}}]`)
	if _, err := client.UpdateIdentityAttribute(ctx, "costCenter", identityAttributeFromModel(plan)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2026/identity-attributes/costCenter" ||
		gotBody != `{"name":"costCenter","displayName":"Cost Center","standard":false,"type":"string","multi":false,"searchable":true,"system":false,"sources":[{"type":"rule","properties":{"ruleName":"Cost Center"}}]}` {
		t.Fatalf("unexpected update request %s %s %s", gotMethod, gotPath, gotBody)
	}

	if err := client.DeleteIdentityAttribute(ctx, "costCenter"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/identity-attributes/costCenter" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestIdentityAttributeState(t *testing.T) {
	var api IdentityAttribute
	if err := json.Unmarshal([]byte(identityAttributeTestResponse), &api); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Apply keeps planned values and resolves unknowns.
	plan := IdentityAttributeModel{
		Name: types.StringValue("costCenter"), DisplayName: types.StringValue("Planned"), Standard: types.BoolUnknown(),
		Type: types.StringUnknown(), Multi: types.BoolValue(false), Searchable: types.BoolUnknown(), System: types.BoolUnknown(),
		SourcesJSON: types.StringNull(),
	}
	identityAttributeResolveApply(&plan, &api)
	if plan.ID.ValueString() != "costCenter" || plan.DisplayName.ValueString() != "Planned" || plan.Type.ValueString() != "string" ||
		!plan.Searchable.ValueBool() || plan.Standard.IsUnknown() || plan.System.IsUnknown() || !plan.SourcesJSON.IsNull() {
		t.Fatalf("unexpected state after apply %+v", plan)
	}

	// Read keeps equivalent JSON.
	data := IdentityAttributeModel{SourcesJSON: types.StringValue(`[{"properties": {"ruleName": "Cost Center", "ruleType": "IdentityAttribute"}, "type": "rule"}]`)}
	prior := data.SourcesJSON
	setIdentityAttributeState(&data, &api)
	if !data.SourcesJSON.Equal(prior) || data.ID.ValueString() != "costCenter" {
		t.Fatalf("expected equivalent JSON to be kept, got %s", data.SourcesJSON)
	}

	// Read keeps null for unset sources when the API has none.
	empty := IdentityAttributeModel{SourcesJSON: types.StringNull()}
	api.Sources = json.RawMessage(`[]`)
	setIdentityAttributeState(&empty, &api)
	if !empty.SourcesJSON.IsNull() {
		t.Fatalf("expected sources to stay null, got %s", empty.SourcesJSON)
	}
}
