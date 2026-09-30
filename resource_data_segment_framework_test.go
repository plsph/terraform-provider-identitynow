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

const dataSegmentTestResponse = `{"id":"ds-1","name":"EMEA","created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z",
"description":"","scopes":[{"scope":"ENTITLEMENT","visibility":"ALL","scopeFilter":null,"scopeSelection":[]}],
"memberSelection":[],"memberFilter":{"expression":{"operator":"EQUALS","attribute":"region","value":{"type":"STRING","value":"EMEA"},"children":null}},
"membership":"FILTER","enabled":true,"published":false}`

func dataSegmentTestModel() DataSegmentModel {
	return DataSegmentModel{
		ID:               types.StringValue("ds-1"),
		Name:             types.StringValue("EMEA"),
		Description:      types.StringNull(),
		Membership:       types.StringValue("FILTER"),
		MemberFilterJSON: types.StringValue(`{"expression":{"operator":"EQUALS","attribute":"region","value":{"type":"STRING","value":"EMEA"}}}`),
		MemberSelection:  types.ListNull(dataSegmentSelectionObjectType),
		ScopesJSON:       types.StringValue(`[{"scope":"ENTITLEMENT","visibility":"ALL"}]`),
		Enabled:          types.BoolValue(true),
		Publish:          types.BoolValue(false),
		Published:        types.BoolValue(false),
		Created:          types.StringValue("2026-01-01T00:00:00Z"),
		Modified:         types.StringValue("2026-01-02T00:00:00Z"),
	}
}

func TestDataSegmentClient(t *testing.T) {
	var requests []string
	var bodies []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Header.Get("X-SailPoint-Experimental") != "true" {
			t.Errorf("missing experimental header on %s %s", r.Method, r.URL.Path)
		}
		requests = append(requests, r.Method+" "+r.URL.RequestURI()+" "+r.Header.Get("Content-Type"))
		bodies = append(bodies, string(body))
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
			return
		case http.MethodPost:
			if strings.Contains(r.URL.Path, "ds-1") {
				return
			}
		}
		_, _ = w.Write([]byte(dataSegmentTestResponse))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := dataSegmentTestModel()
	plan.ID, plan.Enabled, plan.Published, plan.Created, plan.Modified = types.StringUnknown(), types.BoolUnknown(), types.BoolUnknown(), types.StringUnknown(), types.StringUnknown()
	plan.Publish = types.BoolValue(true)
	created, err := client.CreateDataSegment(ctx, dataSegmentFromModel(ctx, plan, &diags))
	if err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	if bodies[0] != `{"name":"EMEA","scopes":[{"scope":"ENTITLEMENT","visibility":"ALL"}],"memberFilter":{"expression":{"attribute":"region","operator":"EQUALS","value":{"type":"STRING","value":"EMEA"}}},"membership":"FILTER"}` {
		t.Fatalf("unexpected create body %s", bodies[0])
	}
	(&DataSegmentResource{}).publish(ctx, client, &plan, created, &diags)
	if diags.HasError() || plan.ID.ValueString() != "ds-1" || plan.Enabled.IsUnknown() || plan.Published.IsUnknown() || plan.Modified.IsUnknown() {
		t.Fatalf("unexpected state after create %+v (%v)", plan, diags)
	}
	if bodies[1] != `["ds-1"]` {
		t.Fatalf("unexpected publish body %s", bodies[1])
	}

	if _, err := client.PatchDataSegment(ctx, "ds-1", []jsonPatchOp{{Op: "replace", Path: "/enabled", Value: false}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := client.DeleteDataSegment(ctx, "ds-1", true); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []string{
		"POST /v2026/data-segments application/json",
		"POST /v2026/data-segments/ds-1?publishAll=false application/json",
		"GET /v2026/data-segments/ds-1 ",
		"PATCH /v2026/data-segments/ds-1 application/json-patch+json",
		"DELETE /v2026/data-segments/ds-1?published=true ",
	}
	if strings.Join(requests, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected requests\n%s", strings.Join(requests, "\n"))
	}
}

func TestDataSegmentReadStateAndPatch(t *testing.T) {
	ctx := context.Background()
	var segment DataSegment
	if err := json.Unmarshal([]byte(dataSegmentTestResponse), &segment); err != nil {
		t.Fatal(err)
	}
	var diags diag.Diagnostics
	data := dataSegmentTestModel()
	prior := data
	setDataSegmentState(ctx, &data, &segment, &diags)
	if diags.HasError() || !data.MemberFilterJSON.Equal(prior.MemberFilterJSON) || !data.ScopesJSON.Equal(prior.ScopesJSON) ||
		!data.Description.IsNull() || !data.MemberSelection.IsNull() {
		t.Fatalf("expected refreshed state to match the configuration, got %+v (%v)", data, diags)
	}

	plan := dataSegmentTestModel()
	plan.Modified, plan.Published = types.StringUnknown(), types.BoolUnknown()
	plan.MemberFilterJSON = types.StringNull()
	plan.Membership = types.StringValue("SELECTION")
	plan.MemberSelection, _ = types.ListValueFrom(ctx, dataSegmentSelectionObjectType, []DataSegmentSelectionModel{{Type: types.StringValue("IDENTITY"), ID: types.StringValue("i-1")}})
	ops := dataSegmentPatchOps(ctx, plan, prior, &diags)
	encoded, _ := json.Marshal(ops)
	if string(encoded) != `[{"op":"replace","path":"/membership","value":"SELECTION"},{"op":"replace","path":"/memberSelection","value":[{"type":"IDENTITY","id":"i-1"}]},{"op":"remove","path":"/memberFilter"}]` {
		t.Fatalf("unexpected ops %s", encoded)
	}

	// After an update, values kept from state are not replaced by the response.
	updated := dataSegmentTestModel()
	updated.Created, updated.Modified, updated.Published = types.StringValue("2025-12-31T00:00:00Z"), types.StringUnknown(), types.BoolUnknown()
	setDataSegmentComputed(&updated, &segment)
	if updated.Created.ValueString() != "2025-12-31T00:00:00Z" || updated.Modified.ValueString() != segment.Modified || updated.Published.IsUnknown() {
		t.Fatalf("unexpected state after update %+v", updated)
	}

	plan = dataSegmentTestModel()
	plan.ScopesJSON = types.StringNull()
	plan.Publish = types.BoolValue(true)
	encoded, _ = json.Marshal(dataSegmentPatchOps(ctx, plan, prior, &diags))
	if string(encoded) != `[{"op":"replace","path":"/scopes","value":[]}]` {
		t.Fatalf("unexpected ops %s", encoded)
	}
}
