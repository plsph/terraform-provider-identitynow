package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const orgConfigTestDoc = `{
	"orgName": "acme-solar",
	"timeZone": "America/Toronto",
	"lcsChangeHonorsSourceEnableFeature": false,
	"armCustomerId": null,
	"armSapSystemIdMappings": null,
	"armAuth": null,
	"armDb": null,
	"armSsoUrl": null,
	"iaiEnableCertificationRecommendations": true,
	"sodReportConfigs": [{"columnName": "SOD Business Name", "required": true, "included": true, "order": 1}]
}`

func TestOrgConfigLifecycle(t *testing.T) {
	tenantSettingsTestLifecycle(t, NewOrgConfigResource, "/v2026/org-config", orgConfigTestDoc)
}

func TestOrgConfigPatchesOnlyTimeZone(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewOrgConfigResource, "/v2026/org-config", orgConfigTestDoc)
	state := h.create(map[string]attr.Value{"time_zone": types.StringValue("Europe/Warsaw")})
	writes := h.api.writes()
	if len(writes) != 1 || !tenantSettingsTestJSONEqual(writes[0].Body, []interface{}{map[string]interface{}{"op": "replace", "path": "/timeZone", "value": "Europe/Warsaw"}}) {
		t.Fatalf("unexpected requests %+v", writes)
	}
	values := h.values(state)
	if values["org_name"].(types.String).ValueString() != "acme-solar" || !values["iai_enable_certification_recommendations"].Equal(types.BoolValue(true)) {
		t.Fatalf("unconfigured settings must be read from the API, got %v", values)
	}
	if values["sod_report_configs_json"].IsNull() || !values["arm_db"].IsNull() {
		t.Fatalf("unexpected JSON or null settings %v", values)
	}
}

func TestOrgConfigDataSource(t *testing.T) {
	api := &tenantSettingsTestAPI{t: t, path: "/v2026/org-config"}
	if err := json.Unmarshal([]byte(orgConfigTestDoc), &api.doc); err != nil {
		t.Fatal(err)
	}
	server := newTestServer(t, api.handler)
	ctx := context.Background()
	d := NewOrgConfigDataSource().(*tenantSettingsDataSource)
	d.Configure(ctx, datasource.ConfigureRequest{ProviderData: tenantSettingsTestProviderConfig(server.URL)}, &datasource.ConfigureResponse{})
	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx)
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}, resp)
	tenantSettingsTestNoErrors(t, "read", resp.Diagnostics)
	var timeZone, id types.String
	resp.State.GetAttribute(ctx, path.Root("time_zone"), &timeZone)
	resp.State.GetAttribute(ctx, path.Root("id"), &id)
	if timeZone.ValueString() != "America/Toronto" || id.ValueString() != "org-config" || !resp.State.Raw.IsFullyKnown() {
		t.Fatalf("unexpected state %s", resp.State.Raw)
	}
}

// JSON Patch replace fails for a missing member: settings the API omits are added.
func TestOrgConfigAddsMissingSettings(t *testing.T) {
	h := newTenantSettingsTestHarness(t, NewOrgConfigResource, "/v2026/org-config", `{"orgName": "acme-solar", "timeZone": "America/Toronto", "armDb": null}`)
	h.create(map[string]attr.Value{
		"time_zone":       types.StringValue("Europe/Warsaw"),
		"arm_db":          types.StringValue("db"),
		"arm_customer_id": types.StringValue("c-1"),
	})
	writes := h.api.writes()
	want := []interface{}{
		map[string]interface{}{"op": "replace", "path": "/timeZone", "value": "Europe/Warsaw"},
		map[string]interface{}{"op": "add", "path": "/armCustomerId", "value": "c-1"},
		map[string]interface{}{"op": "replace", "path": "/armDb", "value": "db"},
	}
	if len(writes) != 1 || !tenantSettingsTestJSONEqual(writes[0].Body, want) {
		t.Fatalf("unexpected requests %+v", writes)
	}
}
