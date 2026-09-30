package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func identityProfileTestRef(t *testing.T, id string, typ, name types.String) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(context.Background(), objectInfoObjectType, []OwnerModel{{ID: types.StringValue(id), Type: typ, Name: name}})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return list
}

const identityProfileTestResponse = `{"id":"ip-1","name":"Employees","description":null,"priority":10,
"owner":{"type":"IDENTITY","id":"owner-1","name":"John Doe"},
"authoritativeSource":{"type":"SOURCE","id":"src-1","name":"HR"},
"identityRefreshRequired":true,"identityCount":3,
"identityAttributeConfig":{"enabled":true,"attributeTransforms":[{"identityAttributeName":"email","transformDefinition":{"type":"accountAttribute","attributes":{"sourceName":"HR","attributeName":"mail"}}}]},
"identityExceptionReportReference":{"taskResultId":"task-1","reportName":"Exceptions"},
"hasTimeBasedAttr":false,"created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z"}`

func TestIdentityProfileClient(t *testing.T) {
	var gotMethod, gotPath, gotQuery, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotQuery, gotBody, gotContentType = r.Method, r.URL.Path, r.URL.Query().Get("filters"), string(body), r.Header.Get("Content-Type")
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"id":"task-2","name":"Identity Profile Delete","completionStatus":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/identity-profiles":
			_, _ = w.Write([]byte("[" + identityProfileTestResponse + "]"))
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(identityProfileTestResponse))
		default:
			_, _ = w.Write([]byte(identityProfileTestResponse))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := IdentityProfileModel{
		Name:                           types.StringValue("Employees"),
		Description:                    types.StringNull(),
		Priority:                       types.Int64Unknown(),
		Owner:                          types.ListNull(objectInfoObjectType),
		AuthoritativeSource:            identityProfileTestRef(t, "src-1", types.StringUnknown(), types.StringUnknown()),
		IdentityAttributeConfigEnabled: types.BoolValue(true),
		AttributeConfigJSON:            types.StringUnknown(),
	}
	created, err := client.CreateIdentityProfile(ctx, identityProfileFromModel(ctx, plan, &diags))
	if err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/identity-profiles" ||
		gotBody != `{"name":"Employees","authoritativeSource":{"type":"SOURCE","id":"src-1"},"identityAttributeConfig":{"enabled":true}}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	if created.ID != "ip-1" || *created.IdentityCount != 3 || created.Owner.Name != "John Doe" {
		t.Fatalf("unexpected response %+v", created)
	}

	if _, err := client.UpdateIdentityProfile(ctx, "ip-1", []jsonPatchOp{{Op: "replace", Path: "/name", Value: "New"}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/identity-profiles/ip-1" || gotContentType != "application/json-patch+json" ||
		gotBody != `[{"op":"replace","path":"/name","value":"New"}]` {
		t.Fatalf("unexpected update request %s %s %s %s", gotMethod, gotPath, gotContentType, gotBody)
	}

	if _, err := client.GetIdentityProfileByName(ctx, "Employees"); err != nil || gotQuery != `name eq "Employees"` {
		t.Fatalf("unexpected lookup by name %q (%v)", gotQuery, err)
	}

	// The delete is asynchronous: 202 with a task reference is a success.
	if err := client.DeleteIdentityProfile(ctx, "ip-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/identity-profiles/ip-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func identityProfileTestAPI(t *testing.T) *IdentityProfile {
	t.Helper()
	var profile IdentityProfile
	if err := json.Unmarshal([]byte(identityProfileTestResponse), &profile); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	return &profile
}

func TestIdentityProfileApplyKeepsPlanAndResolvesUnknowns(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	data := IdentityProfileModel{
		Name:                           types.StringValue("Employees"),
		Description:                    types.StringNull(),
		Priority:                       types.Int64Unknown(),
		Owner:                          identityProfileTestRef(t, "owner-1", types.StringUnknown(), types.StringUnknown()),
		AuthoritativeSource:            identityProfileTestRef(t, "src-1", types.StringValue("SOURCE"), types.StringUnknown()),
		IdentityAttributeConfigEnabled: types.BoolUnknown(),
		AttributeConfigJSON:            types.StringUnknown(),
	}
	identityProfileResolveApply(ctx, &data, identityProfileTestAPI(t), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if data.ID.ValueString() != "ip-1" || data.Priority.ValueInt64() != 10 || !data.IdentityAttributeConfigEnabled.ValueBool() ||
		!data.IdentityRefreshRequired.ValueBool() || data.IdentityCount.ValueInt64() != 3 || data.ExceptionReportTaskResultID.ValueString() != "task-1" {
		t.Fatalf("unexpected computed values %+v", data)
	}
	if data.AttributeConfigJSON.IsUnknown() || !strings.Contains(data.AttributeConfigJSON.ValueString(), `"identityAttributeName":"email"`) {
		t.Fatalf("expected transforms from the API, got %s", data.AttributeConfigJSON)
	}
	var owners []OwnerModel
	data.Owner.ElementsAs(ctx, &owners, false)
	if owners[0].Type.ValueString() != "IDENTITY" || owners[0].Name.ValueString() != "John Doe" {
		t.Fatalf("expected owner type and name from the API, got %+v", owners)
	}
	if !data.Description.IsNull() {
		t.Fatalf("expected description to stay null")
	}

	// Planned values are kept even if the response differs.
	planned := IdentityProfileModel{
		Name:                           types.StringValue("Employees"),
		Description:                    types.StringValue("Planned"),
		Priority:                       types.Int64Value(5),
		Owner:                          types.ListNull(objectInfoObjectType),
		AuthoritativeSource:            identityProfileTestRef(t, "src-1", types.StringValue("SOURCE"), types.StringValue("HR")),
		IdentityAttributeConfigEnabled: types.BoolValue(false),
		AttributeConfigJSON:            types.StringValue(`[]`),
	}
	identityProfileResolveApply(ctx, &planned, identityProfileTestAPI(t), &diags)
	if planned.Priority.ValueInt64() != 5 || planned.IdentityAttributeConfigEnabled.ValueBool() || planned.AttributeConfigJSON.ValueString() != `[]` ||
		!planned.Owner.IsNull() || planned.Description.ValueString() != "Planned" {
		t.Fatalf("expected planned values to be kept, got %+v", planned)
	}
}

func TestIdentityProfileReadKeepsNullAndEquivalentJSON(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	data := IdentityProfileModel{
		Description:         types.StringNull(),
		AttributeConfigJSON: types.StringValue(`[{"transformDefinition": {"attributes": {"attributeName": "mail", "sourceName": "HR"}, "type": "accountAttribute"}, "identityAttributeName": "email"}]`),
	}
	prior := data.AttributeConfigJSON
	setIdentityProfileState(ctx, &data, identityProfileTestAPI(t), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.Description.IsNull() {
		t.Fatalf("expected description to stay null, got %s", data.Description)
	}
	if !data.AttributeConfigJSON.Equal(prior) {
		t.Fatalf("expected equivalent JSON to be kept, got %s", data.AttributeConfigJSON)
	}
	if data.Name.ValueString() != "Employees" || data.Priority.ValueInt64() != 10 || len(data.Owner.Elements()) != 1 || len(data.AuthoritativeSource.Elements()) != 1 {
		t.Fatalf("unexpected state %+v", data)
	}

	noOwner := identityProfileTestAPI(t)
	noOwner.Owner = nil
	setIdentityProfileState(ctx, &data, noOwner, &diags)
	if !data.Owner.IsNull() {
		t.Fatalf("expected null owner, got %s", data.Owner)
	}
}

func TestIdentityProfilePatches(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := IdentityProfileModel{
		Name:                           types.StringValue("Employees"),
		Description:                    types.StringValue("Old"),
		Priority:                       types.Int64Value(10),
		Owner:                          identityProfileTestRef(t, "owner-1", types.StringValue("IDENTITY"), types.StringValue("John Doe")),
		AuthoritativeSource:            identityProfileTestRef(t, "src-1", types.StringValue("SOURCE"), types.StringValue("HR")),
		IdentityAttributeConfigEnabled: types.BoolValue(true),
		AttributeConfigJSON:            types.StringValue(`[{"identityAttributeName":"email"}]`),
	}

	// Unknown nested computed values and reformatted JSON are not changes.
	plan := state
	plan.Owner = identityProfileTestRef(t, "owner-1", types.StringUnknown(), types.StringUnknown())
	plan.AttributeConfigJSON = types.StringValue(`[ { "identityAttributeName": "email" } ]`)
	if batches := identityProfilePatches(ctx, plan, state, &diags); len(batches) != 0 {
		t.Fatalf("expected no patches, got %+v", batches)
	}

	plan.Description = types.StringNull()
	plan.Priority = types.Int64Value(20)
	encoded, _ := json.Marshal(identityProfilePatches(ctx, plan, state, &diags))
	if string(encoded) != `[[{"op":"remove","path":"/description"},{"op":"replace","path":"/priority","value":20}]]` {
		t.Fatalf("unexpected patches %s", encoded)
	}

	// The authoritative source and the attribute configuration are sent in separate requests.
	plan = state
	plan.AuthoritativeSource = identityProfileTestRef(t, "src-2", types.StringUnknown(), types.StringUnknown())
	plan.IdentityAttributeConfigEnabled = types.BoolValue(false)
	encoded, _ = json.Marshal(identityProfilePatches(ctx, plan, state, &diags))
	if string(encoded) != `[[{"op":"replace","path":"/authoritativeSource","value":{"type":"SOURCE","id":"src-2"}}],`+
		`[{"op":"replace","path":"/identityAttributeConfig","value":{"attributeTransforms":[{"identityAttributeName":"email"}],"enabled":false}}]]` {
		t.Fatalf("unexpected patches %s", encoded)
	}

	// Without a source change the attribute configuration is part of the single request.
	plan = state
	plan.Name = types.StringValue("Staff")
	plan.Owner = types.ListNull(objectInfoObjectType)
	plan.AttributeConfigJSON = types.StringValue(`[]`)
	encoded, _ = json.Marshal(identityProfilePatches(ctx, plan, state, &diags))
	if string(encoded) != `[[{"op":"replace","path":"/name","value":"Staff"},{"op":"remove","path":"/owner"},`+
		`{"op":"replace","path":"/identityAttributeConfig","value":{"attributeTransforms":[],"enabled":true}}]]` {
		t.Fatalf("unexpected patches %s", encoded)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}
