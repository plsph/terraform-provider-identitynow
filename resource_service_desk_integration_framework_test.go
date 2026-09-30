package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// serviceDeskIntegrationTestRequest is a request recorded by a test server.
type serviceDeskIntegrationTestRequest struct {
	Method, Path, Query, Body, ContentType, Experimental string
}

// serviceDeskIntegrationTestServer records requests and answers them with respond.
func serviceDeskIntegrationTestServer(t *testing.T, respond func(r *http.Request) (int, string)) (*Client, *[]serviceDeskIntegrationTestRequest) {
	t.Helper()
	var requests []serviceDeskIntegrationTestRequest
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, serviceDeskIntegrationTestRequest{
			Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(body),
			ContentType: r.Header.Get("Content-Type"), Experimental: r.Header.Get("X-SailPoint-Experimental"),
		})
		status, response := respond(r)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	})
	return NewClient(context.Background(), server.URL, "id", "secret", 1000), &requests
}

// serviceDeskIntegrationTestJSONEqual reports whether two JSON documents are semantically equal.
func serviceDeskIntegrationTestJSONEqual(t *testing.T, got, want string) {
	t.Helper()
	if !jsonSemanticallyEqual([]byte(got), []byte(want)) {
		t.Fatalf("unexpected JSON\n got: %s\nwant: %s", got, want)
	}
}

// serviceDeskIntegrationTestAssertKnown fails when any attribute value of the model is unknown.
func serviceDeskIntegrationTestAssertKnown(t *testing.T, model interface{}) {
	t.Helper()
	var check func(name string, value attr.Value)
	check = func(name string, value attr.Value) {
		if value == nil {
			return
		}
		if value.IsUnknown() {
			t.Errorf("%s is unknown", name)
			return
		}
		switch v := value.(type) {
		case types.List:
			for _, element := range v.Elements() {
				check(name+"[]", element)
			}
		case types.Object:
			for key, attribute := range v.Attributes() {
				check(name+"."+key, attribute)
			}
		}
	}
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if v.Type().Field(i).Anonymous {
				walk(field)
				continue
			}
			if value, ok := field.Interface().(attr.Value); ok {
				check(v.Type().Field(i).Name, value)
			}
		}
	}
	walk(reflect.ValueOf(model))
}

// serviceDeskIntegrationTestSetState stores the model in a state built from the resource schema,
// which verifies that the model matches the schema.
func serviceDeskIntegrationTestSetState(t *testing.T, r resource.Resource, model interface{}) {
	t.Helper()
	ctx := context.Background()
	var resp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &resp)
	state := tfsdk.State{Schema: resp.Schema, Raw: tftypes.NewValue(resp.Schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatalf("unable to set state: %v", diags)
	}
}

// serviceDeskIntegrationTestSetDataSourceState stores the model in a data source state.
func serviceDeskIntegrationTestSetDataSourceState(t *testing.T, d datasource.DataSource, model interface{}) {
	t.Helper()
	ctx := context.Background()
	var resp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &resp)
	state := tfsdk.State{Schema: resp.Schema, Raw: tftypes.NewValue(resp.Schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatalf("unable to set data source state: %v", diags)
	}
}

func serviceDeskIntegrationTestRef(t *testing.T, id string, refType, name types.String) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(context.Background(), objectInfoObjectType, []OwnerModel{{ID: types.StringValue(id), Type: refType, Name: name}})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return list
}

const serviceDeskIntegrationTestResponse = `{"id":"sdi-1","name":"ServiceNow","description":"Tickets","type":"ServiceNowSDIM",
"created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z",
"ownerRef":{"type":"IDENTITY","id":"owner-1","name":"John Doe"},
"clusterRef":{"type":"CLUSTER","id":"cluster-1","name":"Cluster"},
"managedSources":["src-1"],
"provisioningConfig":{"universalManager":false,"managedResourceRefs":[{"type":"SOURCE","id":"src-1","name":"AD"}],"noProvisioningRequests":true},
"attributes":{"url":"https://example.service-now.com","username":"svc"},
"beforeProvisioningRule":{"type":"RULE","id":"rule-1","name":"Rule"}}`

func TestServiceDeskIntegrationClient(t *testing.T) {
	client, requests := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		switch {
		case r.Method == http.MethodDelete:
			return http.StatusNoContent, ""
		case r.URL.Path == "/v2026/service-desk-integrations/types":
			return http.StatusOK, `[{"name":"ServiceNow","type":"ServiceNowSDIM","scriptName":"servicenow"}]`
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/service-desk-integrations":
			return http.StatusOK, "[" + serviceDeskIntegrationTestResponse + "]"
		}
		return http.StatusOK, serviceDeskIntegrationTestResponse
	})
	ctx := context.Background()
	var diags diag.Diagnostics
	integration := serviceDeskIntegrationFromModel(ctx, ServiceDeskIntegrationModel{
		Name:                   types.StringValue("ServiceNow"),
		Description:            types.StringValue("Tickets"),
		Type:                   types.StringValue("ServiceNowSDIM"),
		OwnerRef:               serviceDeskIntegrationTestRef(t, "owner-1", types.StringUnknown(), types.StringUnknown()),
		ClusterRef:             types.ListNull(objectInfoObjectType),
		ManagedSources:         types.ListUnknown(types.StringType),
		ProvisioningConfigJSON: types.StringValue(`{"noProvisioningRequests":true}`),
		AttributesJSON:         types.StringValue(`{"url":"https://example.service-now.com","password":"s3cret"}`),
		BeforeProvisioningRule: serviceDeskIntegrationTestRef(t, "rule-1", types.StringNull(), types.StringNull()),
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	created, err := client.CreateServiceDeskIntegration(ctx, integration)
	if err != nil || created.ID != "sdi-1" || created.OwnerRef.Name != "John Doe" {
		t.Fatalf("unexpected create result %+v (%v)", created, err)
	}
	got := (*requests)[0]
	if got.Method != http.MethodPost || got.Path != "/v2026/service-desk-integrations" || got.Experimental != "" {
		t.Fatalf("unexpected create request %+v", got)
	}
	serviceDeskIntegrationTestJSONEqual(t, got.Body, `{"name":"ServiceNow","description":"Tickets","type":"ServiceNowSDIM",
		"ownerRef":{"type":"IDENTITY","id":"owner-1"},
		"provisioningConfig":{"noProvisioningRequests":true},
		"attributes":{"url":"https://example.service-now.com","password":"s3cret"},
		"beforeProvisioningRule":{"type":"RULE","id":"rule-1"}}`)

	if _, err := client.UpdateServiceDeskIntegration(ctx, "sdi-1", integration); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[1]; got.Method != http.MethodPut || got.Path != "/v2026/service-desk-integrations/sdi-1" || !strings.Contains(got.Body, `"password":"s3cret"`) {
		t.Fatalf("unexpected update request %+v", got)
	}
	if _, err := client.GetServiceDeskIntegrationByName(ctx, "ServiceNow"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[2]; got.Method != http.MethodGet || !strings.Contains(got.Query, "filters=name+eq+%22ServiceNow%22") {
		t.Fatalf("unexpected lookup request %+v", got)
	}
	list, err := client.ListServiceDeskIntegrationTypes(ctx)
	if err != nil || len(list) != 1 || list[0].ScriptName != "servicenow" {
		t.Fatalf("unexpected types %+v (%v)", list, err)
	}
	if err := client.DeleteServiceDeskIntegration(ctx, "sdi-1"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[4]; got.Method != http.MethodDelete || got.Path != "/v2026/service-desk-integrations/sdi-1" {
		t.Fatalf("unexpected delete request %+v", got)
	}
}

func TestServiceDeskIntegrationApplyKeepsPlanAndResolvesUnknowns(t *testing.T) {
	ctx := context.Background()
	var api ServiceDeskIntegration
	if err := json.Unmarshal([]byte(serviceDeskIntegrationTestResponse), &api); err != nil {
		t.Fatal(err)
	}
	data := ServiceDeskIntegrationModel{
		ID:                     types.StringUnknown(),
		Name:                   types.StringValue("ServiceNow"),
		Description:            types.StringValue("Tickets"),
		Type:                   types.StringValue("ServiceNowSDIM"),
		OwnerRef:               serviceDeskIntegrationTestRef(t, "owner-1", types.StringUnknown(), types.StringUnknown()),
		ClusterRef:             types.ListNull(objectInfoObjectType),
		ManagedSources:         types.ListUnknown(types.StringType),
		ProvisioningConfigJSON: types.StringUnknown(),
		AttributesJSON:         types.StringValue(`{"url": "https://example.service-now.com", "password": "s3cret"}`),
		BeforeProvisioningRule: serviceDeskIntegrationTestRef(t, "rule-1", types.StringValue("RULE"), types.StringUnknown()),
		Created:                types.StringUnknown(),
		Modified:               types.StringUnknown(),
	}
	planned := data
	var diags diag.Diagnostics
	setServiceDeskIntegrationState(ctx, &data, &api, false, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	serviceDeskIntegrationTestAssertKnown(t, data)
	serviceDeskIntegrationTestSetState(t, NewServiceDeskIntegrationResource(), &data)
	if !data.AttributesJSON.Equal(planned.AttributesJSON) {
		t.Errorf("planned attributes were overwritten: %s", data.AttributesJSON)
	}
	if !data.ClusterRef.IsNull() {
		t.Errorf("unset cluster_ref must stay null, got %s", data.ClusterRef)
	}
	var owners []OwnerModel
	data.OwnerRef.ElementsAs(ctx, &owners, false)
	if owners[0].Type.ValueString() != "IDENTITY" || owners[0].Name.ValueString() != "John Doe" {
		t.Errorf("unexpected owner %+v", owners[0])
	}
	if data.ManagedSources.String() != `["src-1"]` || !strings.Contains(data.ProvisioningConfigJSON.ValueString(), "managedResourceRefs") {
		t.Errorf("unexpected computed values %s %s", data.ManagedSources, data.ProvisioningConfigJSON)
	}
}

func TestServiceDeskIntegrationReadKeepsOmittedSecretsAndDetectsDrift(t *testing.T) {
	ctx := context.Background()
	var api ServiceDeskIntegration
	if err := json.Unmarshal([]byte(serviceDeskIntegrationTestResponse), &api); err != nil {
		t.Fatal(err)
	}
	prior := types.StringValue(`{"url":"https://example.service-now.com","password":"s3cret"}`)
	data := ServiceDeskIntegrationModel{
		OwnerRef:               types.ListNull(objectInfoObjectType),
		ClusterRef:             types.ListNull(objectInfoObjectType),
		BeforeProvisioningRule: types.ListNull(objectInfoObjectType),
		ManagedSources:         types.ListNull(types.StringType),
		ProvisioningConfigJSON: types.StringValue(`{"noProvisioningRequests":true}`),
		AttributesJSON:         prior,
	}
	var diags diag.Diagnostics
	setServiceDeskIntegrationState(ctx, &data, &api, true, &diags)
	if !data.AttributesJSON.Equal(prior) {
		t.Errorf("attributes without the omitted password must keep the prior value, got %s", data.AttributesJSON)
	}
	if data.ProvisioningConfigJSON.ValueString() != `{"noProvisioningRequests":true}` {
		t.Errorf("keys added by the API must not cause a diff, got %s", data.ProvisioningConfigJSON)
	}
	if data.OwnerRef.IsNull() || data.ManagedSources.IsNull() {
		t.Errorf("refresh must read references from the API")
	}

	api.Attributes["url"] = "https://changed.service-now.com"
	setServiceDeskIntegrationState(ctx, &data, &api, true, &diags)
	if data.AttributesJSON.Equal(prior) || !strings.Contains(data.AttributesJSON.ValueString(), "changed") {
		t.Errorf("changed attributes must be detected as drift, got %s", data.AttributesJSON)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestServiceDeskIntegrationDataSources(t *testing.T) {
	ctx := context.Background()
	var api ServiceDeskIntegration
	if err := json.Unmarshal([]byte(serviceDeskIntegrationTestResponse), &api); err != nil {
		t.Fatal(err)
	}
	data := ServiceDeskIntegrationModel{
		ID:                     types.StringNull(),
		Name:                   types.StringValue("ServiceNow"),
		OwnerRef:               types.ListNull(objectInfoObjectType),
		ClusterRef:             types.ListNull(objectInfoObjectType),
		BeforeProvisioningRule: types.ListNull(objectInfoObjectType),
		ManagedSources:         types.ListNull(types.StringType),
		ProvisioningConfigJSON: types.StringNull(),
		AttributesJSON:         types.StringNull(),
	}
	var diags diag.Diagnostics
	setServiceDeskIntegrationState(ctx, &data, &api, true, &diags)
	serviceDeskIntegrationTestAssertKnown(t, data)
	serviceDeskIntegrationTestSetDataSourceState(t, NewServiceDeskIntegrationDataSource(), &data)
	if data.ID.ValueString() != "sdi-1" || !strings.Contains(data.ProvisioningConfigJSON.ValueString(), "universalManager") {
		t.Errorf("unexpected data source state %+v", data)
	}

	typesData := serviceDeskIntegrationTypesState([]ServiceDeskIntegrationType{{Name: "ServiceNow", Type: "ServiceNowSDIM", ScriptName: "servicenow"}})
	serviceDeskIntegrationTestSetDataSourceState(t, NewServiceDeskIntegrationTypesDataSource(), &typesData)
	if len(typesData.Types) != 1 || typesData.Types[0].ScriptName.ValueString() != "servicenow" {
		t.Errorf("unexpected types state %+v", typesData)
	}
}

func TestServiceDeskIntegrationUpdateKeepsKnownComputedValues(t *testing.T) {
	ctx := context.Background()
	var api ServiceDeskIntegration
	if err := json.Unmarshal([]byte(serviceDeskIntegrationTestResponse), &api); err != nil {
		t.Fatal(err)
	}
	api.ID, api.Created = "other", "2030-01-01T00:00:00Z"
	data := ServiceDeskIntegrationModel{
		ID:                     types.StringValue("sdi-1"),
		Name:                   types.StringValue("ServiceNow"),
		Description:            types.StringValue("Tickets"),
		Type:                   types.StringValue("ServiceNowSDIM"),
		OwnerRef:               types.ListNull(objectInfoObjectType),
		ClusterRef:             types.ListNull(objectInfoObjectType),
		BeforeProvisioningRule: types.ListNull(objectInfoObjectType),
		ManagedSources:         types.ListNull(types.StringType),
		ProvisioningConfigJSON: types.StringNull(),
		AttributesJSON:         types.StringValue(`{}`),
		Created:                types.StringValue("2026-01-01T00:00:00Z"),
		Modified:               types.StringUnknown(),
	}
	var diags diag.Diagnostics
	setServiceDeskIntegrationState(ctx, &data, &api, false, &diags)
	if data.ID.ValueString() != "sdi-1" || data.Created.ValueString() != "2026-01-01T00:00:00Z" || data.Modified.ValueString() != "2026-01-02T00:00:00Z" {
		t.Errorf("known planned id and created must be kept, got %s %s %s", data.ID, data.Created, data.Modified)
	}
}

func TestServiceDeskIntegrationReadKeepsConfiguredReferenceName(t *testing.T) {
	ctx := context.Background()
	api := &ServiceDeskIntegrationRef{Type: "IDENTITY", ID: "owner-1", Name: "John Doe"}
	var diags diag.Diagnostics
	prior := serviceDeskIntegrationTestRef(t, "owner-1", types.StringValue("IDENTITY"), types.StringValue("john.doe"))
	if got := serviceDeskIntegrationRefState(ctx, prior, api, "IDENTITY", true, &diags); !got.Equal(prior) {
		t.Errorf("configured name of an unchanged reference must be kept, got %s", got)
	}
	changed := serviceDeskIntegrationTestRef(t, "owner-2", types.StringValue("IDENTITY"), types.StringValue("jane.doe"))
	want := serviceDeskIntegrationTestRef(t, "owner-1", types.StringValue("IDENTITY"), types.StringValue("John Doe"))
	if got := serviceDeskIntegrationRefState(ctx, changed, api, "IDENTITY", true, &diags); !got.Equal(want) {
		t.Errorf("a changed reference must be read from the API, got %s", got)
	}
	if got := serviceDeskIntegrationRefState(ctx, types.ListNull(objectInfoObjectType), api, "IDENTITY", true, &diags); !got.Equal(want) {
		t.Errorf("an imported reference must be read from the API, got %s", got)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestServiceDeskIntegrationDoesNotSendUniversalManager(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	integration := serviceDeskIntegrationFromModel(ctx, ServiceDeskIntegrationModel{
		Name:                   types.StringValue("ServiceNow"),
		Description:            types.StringValue("Tickets"),
		Type:                   types.StringValue("ServiceNowSDIM"),
		OwnerRef:               types.ListNull(objectInfoObjectType),
		ClusterRef:             types.ListNull(objectInfoObjectType),
		BeforeProvisioningRule: types.ListNull(objectInfoObjectType),
		ManagedSources:         types.ListNull(types.StringType),
		ProvisioningConfigJSON: types.StringValue(`{"universalManager":false,"noProvisioningRequests":true}`),
		AttributesJSON:         types.StringValue(`{}`),
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	encoded, _ := json.Marshal(integration)
	if strings.Contains(string(encoded), "universalManager") || !strings.Contains(string(encoded), `"noProvisioningRequests":true`) {
		t.Errorf("unexpected request body %s", encoded)
	}
}

func TestServiceDeskIntegrationSubsetJSONState(t *testing.T) {
	prior := types.StringValue(`{"a": 1, "secret": "x"}`)
	if got := serviceDeskIntegrationSubsetJSONState(prior, map[string]interface{}{"a": 1, "extra": true}); !got.Equal(prior) {
		t.Errorf("expected prior to be kept, got %s", got)
	}
	if got := serviceDeskIntegrationSubsetJSONState(prior, map[string]interface{}{"a": 2}); got.ValueString() != `{"a":2}` {
		t.Errorf("expected API value, got %s", got)
	}
	if got := serviceDeskIntegrationSubsetJSONState(types.StringNull(), nil); !got.IsNull() {
		t.Errorf("expected null, got %s", got)
	}
}
