package main

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const simIntegrationTestResponse = `{"id":"sim-1","name":"ServiceNow SIM","created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z",
"description":"Tickets","type":"ServiceNow Service Desk","sources":["src-1","src-2"],"cluster":"cluster-1",
"statusMap":{"closed_complete":"Committed"},"request":{"description":"SailPoint Access Request"},
"attributes":{"uid":"svc"},"beforeProvisioningRule":{"type":"RULE","id":"rule-1","name":"Rule"}}`

func TestSimIntegrationClientSendsExperimentalHeader(t *testing.T) {
	client, requests := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		if r.Method == http.MethodDelete {
			return http.StatusNoContent, ""
		}
		return http.StatusOK, simIntegrationTestResponse
	})
	ctx := context.Background()
	var diags diag.Diagnostics
	integration := simIntegrationFromModel(ctx, SimIntegrationModel{
		Name:                   types.StringValue("ServiceNow SIM"),
		Description:            types.StringNull(),
		Type:                   types.StringUnknown(),
		Sources:                types.ListValueMust(types.StringType, []attr.Value{types.StringValue("src-1")}),
		Cluster:                types.StringValue("cluster-1"),
		StatusMapJSON:          types.StringValue(`{"closed_complete":"Committed"}`),
		RequestJSON:            types.StringNull(),
		AttributesJSON:         types.StringValue(`{"uid":"svc","password":"s3cret"}`),
		BeforeProvisioningRule: types.ListNull(objectInfoObjectType),
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if _, err := client.CreateSimIntegration(ctx, integration); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if _, err := client.UpdateSimIntegration(ctx, "sim-1", integration); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if _, err := client.GetSimIntegration(ctx, "sim-1"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err := client.DeleteSimIntegration(ctx, "sim-1"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := []struct{ method, path string }{
		{http.MethodPost, "/v2026/sim-integrations"},
		{http.MethodPut, "/v2026/sim-integrations/sim-1"},
		{http.MethodGet, "/v2026/sim-integrations/sim-1"},
		{http.MethodDelete, "/v2026/sim-integrations/sim-1"},
	}
	for i, w := range want {
		got := (*requests)[i]
		if got.Method != w.method || got.Path != w.path || got.Experimental != "true" {
			t.Errorf("request %d: unexpected %+v", i, got)
		}
	}
	serviceDeskIntegrationTestJSONEqual(t, (*requests)[0].Body, `{"name":"ServiceNow SIM","sources":["src-1"],"cluster":"cluster-1",
		"statusMap":{"closed_complete":"Committed"},"attributes":{"uid":"svc","password":"s3cret"}}`)
}

func TestSimIntegrationResponseShapes(t *testing.T) {
	// The spec documents the response with the service desk integration schema and shows the
	// attributes as an encoded string; both are mapped onto the SIM fields.
	client, _ := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `{"id":"sim-1","name":"SIM","managedSources":["src-1"],"clusterRef":{"type":"CLUSTER","id":"cluster-1"},"attributes":"{\"uid\":\"svc\"}"}`
	})
	integration, err := client.GetSimIntegration(context.Background(), "sim-1")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(integration.Sources) != 1 || integration.Cluster != "cluster-1" || integration.Attributes["uid"] != "svc" {
		t.Fatalf("unexpected integration %+v", integration)
	}
}

func TestSimIntegrationState(t *testing.T) {
	ctx := context.Background()
	client, _ := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		return http.StatusOK, simIntegrationTestResponse
	})
	api, err := client.GetSimIntegration(ctx, "sim-1")
	if err != nil {
		t.Fatal(err)
	}
	planned := SimIntegrationModel{
		ID:                     types.StringUnknown(),
		Name:                   types.StringValue("ServiceNow SIM"),
		Description:            types.StringValue("Tickets"),
		Type:                   types.StringUnknown(),
		Sources:                types.ListValueMust(types.StringType, []attr.Value{types.StringValue("src-2"), types.StringValue("src-1")}),
		Cluster:                types.StringValue("cluster-1"),
		StatusMapJSON:          types.StringValue(`{"closed_complete": "Committed"}`),
		RequestJSON:            types.StringNull(),
		AttributesJSON:         types.StringValue(`{"uid":"svc","password":"s3cret"}`),
		BeforeProvisioningRule: serviceDeskIntegrationTestRef(t, "rule-1", types.StringUnknown(), types.StringUnknown()),
		Created:                types.StringUnknown(),
		Modified:               types.StringUnknown(),
	}
	data := planned
	var diags diag.Diagnostics
	setSimIntegrationState(ctx, &data, api, false, &diags)
	serviceDeskIntegrationTestAssertKnown(t, data)
	serviceDeskIntegrationTestSetState(t, NewSimIntegrationResource(), &data)
	if data.Type.ValueString() != "ServiceNow Service Desk" || !data.RequestJSON.IsNull() || !data.Sources.Equal(planned.Sources) {
		t.Errorf("unexpected state after apply %+v", data)
	}

	// Read keeps the prior order, equivalent JSON and the omitted password, and detects new values.
	setSimIntegrationState(ctx, &data, api, true, &diags)
	if !data.Sources.Equal(planned.Sources) || !data.StatusMapJSON.Equal(planned.StatusMapJSON) || !data.AttributesJSON.Equal(planned.AttributesJSON) {
		t.Errorf("unexpected state after refresh %+v", data)
	}
	if !data.RequestJSON.IsNull() {
		t.Errorf("unset optional request_json must stay null, got %s", data.RequestJSON)
	}
	// Keys added by the API do not cause a diff; changed configured values are detected.
	data.StatusMapJSON = types.StringValue(`{}`)
	api.StatusMap["closed_cancelled"] = "Failed"
	setSimIntegrationState(ctx, &data, api, true, &diags)
	if data.StatusMapJSON.ValueString() != `{}` {
		t.Errorf("keys added by the API must not cause a diff, got %s", data.StatusMapJSON)
	}
	data.StatusMapJSON = types.StringValue(`{"closed_complete":"Failed"}`)
	setSimIntegrationState(ctx, &data, api, true, &diags)
	if !strings.Contains(data.StatusMapJSON.ValueString(), "Committed") {
		t.Errorf("changed status map must be detected, got %s", data.StatusMapJSON)
	}
	// A configured rule name that differs from the API display name is kept.
	data.BeforeProvisioningRule = serviceDeskIntegrationTestRef(t, "rule-1", types.StringValue("RULE"), types.StringValue("My rule"))
	setSimIntegrationState(ctx, &data, api, true, &diags)
	var rules []OwnerModel
	diags.Append(data.BeforeProvisioningRule.ElementsAs(ctx, &rules, false)...)
	if len(rules) != 1 || rules[0].Name.ValueString() != "My rule" {
		t.Errorf("configured rule name must be kept, got %+v", rules)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	// Update keeps the known id and created of the plan.
	updated := data
	updated.Modified = types.StringUnknown()
	apiUpdated := *api
	apiUpdated.ID, apiUpdated.Created = "other", "2030-01-01T00:00:00Z"
	setSimIntegrationState(ctx, &updated, &apiUpdated, false, &diags)
	if updated.ID.ValueString() != "sim-1" || updated.Created.ValueString() != "2026-01-01T00:00:00Z" {
		t.Errorf("known planned id and created must be kept, got %s %s", updated.ID, updated.Created)
	}

	// Import and the data source have no prior configuration and read the JSON attributes from the API.
	ds := SimIntegrationModel{ID: types.StringValue("sim-1"), Sources: types.ListNull(types.StringType), BeforeProvisioningRule: types.ListNull(objectInfoObjectType)}
	setSimIntegrationState(ctx, &ds, api, true, &diags)
	serviceDeskIntegrationTestSetDataSourceState(t, NewSimIntegrationDataSource(), &ds)
	if !strings.Contains(ds.RequestJSON.ValueString(), "SailPoint Access Request") || ds.AttributesJSON.IsNull() || ds.StatusMapJSON.IsNull() {
		t.Errorf("imported JSON attributes must come from the API, got %+v", ds)
	}
}
