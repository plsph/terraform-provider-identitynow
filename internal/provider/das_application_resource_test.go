package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const dasApplicationTestItem = `{"id":42,"name":"File Share","description":"Finance share","type":"Windows File Server",
"tags":[{"id":7,"name":"finance"}],"testConnectionStatus":"Success","testConnectionDate":1700000000000,
"rcClusterId":"rc-1","dcClusterId":null,"pcClusterId":"pc-1"}`

func TestDasApplicationClientCreateFindsNewApplication(t *testing.T) {
	created := false
	client, requests := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		switch {
		case r.Method == http.MethodPost:
			created = true
			return http.StatusNoContent, ""
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/das/applications":
			// An older application with the same name exists before and after the creation.
			older := strings.Replace(dasApplicationTestItem, `"id":42`, `"id":41`, 1)
			if created {
				return http.StatusOK, "[" + older + "," + dasApplicationTestItem + "]"
			}
			return http.StatusOK, "[" + older + "]"
		case r.Method == http.MethodPut || r.Method == http.MethodDelete:
			return http.StatusNoContent, ""
		}
		return http.StatusOK, dasApplicationTestItem
	})
	ctx := context.Background()
	var diags diag.Diagnostics
	request := dasApplicationFromModel(ctx, DasApplicationModel{
		Name:                           types.StringValue("File Share"),
		ApplicationType:                types.Int64Value(8),
		Description:                    types.StringNull(),
		Tag:                            types.ListValueMust(types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.Int64Type, "value": types.StringType}}, []attr.Value{types.ObjectValueMust(map[string]attr.Type{"key": types.Int64Type, "value": types.StringType}, map[string]attr.Value{"key": types.Int64Value(7), "value": types.StringValue("finance")})}),
		IdentityCollectorID:            types.Int64Value(3),
		ApplicationCrawlerSettingsJSON: types.StringValue(`{"isEnabled":true,"clusterId":"rc-1"}`),
		ExecuteNow:                     types.BoolValue(false),
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	application, err := client.CreateDasApplication(ctx, request)
	if err != nil || application.ID != 42 {
		t.Fatalf("unexpected create result %+v (%v)", application, err)
	}
	if got := (*requests)[0]; got.Method != http.MethodGet || !strings.Contains(got.Query, "filters=appName+eq+%22File+Share%22") {
		t.Fatalf("unexpected lookup request %+v", got)
	}
	got := (*requests)[1]
	if got.Method != http.MethodPost || got.Path != "/v2026/das/applications" {
		t.Fatalf("unexpected create request %+v", got)
	}
	serviceDeskIntegrationTestJSONEqual(t, got.Body, `{"applicationType":8,"name":"File Share","tags":[{"key":7,"value":"finance"}],
		"identityCollectorId":3,"applicationCrawlerSettings":{"isEnabled":true,"clusterId":"rc-1"},"executeNow":false}`)

	if err := client.UpdateDasApplication(ctx, "42", request); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[3]; got.Method != http.MethodPut || got.Path != "/v2026/das/applications/42" {
		t.Fatalf("unexpected update request %+v", got)
	}
	if err := client.DeleteDasApplication(ctx, "42"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[4]; got.Method != http.MethodDelete || got.Path != "/v2026/das/applications/42" {
		t.Fatalf("unexpected delete request %+v", got)
	}
}

func TestDasApplicationClientCreateFailsWhenNotIdentifiable(t *testing.T) {
	client, _ := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		if r.Method == http.MethodPost {
			return http.StatusNoContent, ""
		}
		return http.StatusOK, "[]"
	})
	if _, err := client.CreateDasApplication(context.Background(), &DasApplicationRequest{Name: "Missing", ApplicationType: 8}); err == nil || !strings.Contains(err.Error(), "0 new applications") {
		t.Fatalf("expected an error, got %v", err)
	}
}

func TestDasApplicationState(t *testing.T) {
	ctx := context.Background()
	client, _ := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		return http.StatusOK, dasApplicationTestItem
	})
	api, err := client.GetDasApplication(ctx, "42")
	if err != nil {
		t.Fatal(err)
	}
	settings := types.StringValue(`{"isEnabled": true}`)
	unknown := types.StringUnknown()
	data := DasApplicationModel{
		ID: unknown, Name: types.StringValue("File Share"), ApplicationType: types.Int64Value(8), Description: types.StringNull(),
		Tag:                            types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.Int64Type, "value": types.StringType}}),
		ApplicationCrawlerSettingsJSON: settings, ExecuteNow: types.BoolValue(true),
		Type: unknown, TestConnectionStatus: unknown, TestConnectionDate: types.Int64Unknown(),
		RcClusterID: unknown, DcClusterID: unknown, PcClusterID: unknown,
	}
	setDasApplicationState(&data, api, false)
	serviceDeskIntegrationTestAssertKnown(t, data)
	serviceDeskIntegrationTestSetState(t, NewDasApplicationResource(), &data)
	if data.ID.ValueString() != "42" || !data.Description.IsNull() || !data.DcClusterID.IsNull() || data.TestConnectionDate.ValueInt64() != 1700000000000 {
		t.Errorf("unexpected state after create %+v", data)
	}

	// Read refreshes name and description and keeps the settings, which the API does not return.
	setDasApplicationState(&data, api, true)
	if !data.ApplicationCrawlerSettingsJSON.Equal(settings) || data.Description.ValueString() != "Finance share" || !data.ExecuteNow.ValueBool() {
		t.Errorf("unexpected state after refresh %+v", data)
	}

	var diags diag.Diagnostics
	ds := dasApplicationDataSourceState(ctx, api, &diags)
	serviceDeskIntegrationTestSetDataSourceState(t, NewDasApplicationDataSource(), &ds)
	if diags.HasError() || len(ds.Tags.Elements()) != 1 || ds.Type.ValueString() != "Windows File Server" {
		t.Errorf("unexpected data source state %+v (%v)", ds, diags)
	}
}

func TestDasApplicationUnmanagedWarning(t *testing.T) {
	tagType := types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.Int64Type, "value": types.StringType}}
	settings := types.StringValue(`{"isEnabled":true}`)
	plan := DasApplicationModel{
		ApplicationCrawlerSettingsJSON: settings, PermissionCollectorSettingsJSON: settings,
		DataClassificationSettingsJSON: types.StringNull(), ActivityConfigurationSettingsJSON: types.StringNull(),
		Tag: types.ListNull(tagType),
	}
	prior := plan
	prior.ActivityConfigurationSettingsJSON = settings
	warning := dasApplicationUnmanagedWarning(plan, prior)
	if !strings.Contains(warning, "`data_classification_settings_json`, `tag`") || strings.Contains(warning, "activity") || strings.Contains(warning, "crawler") {
		t.Errorf("unexpected warning %q", warning)
	}
	plan.DataClassificationSettingsJSON, prior.DataClassificationSettingsJSON = settings, settings
	plan.Tag = types.ListValueMust(tagType, []attr.Value{types.ObjectValueMust(tagType.AttrTypes, map[string]attr.Value{
		"key": types.Int64Value(1), "value": types.StringNull(),
	})})
	if warning := dasApplicationUnmanagedWarning(plan, prior); warning != "" {
		t.Errorf("expected no warning when everything is configured or removed on purpose, got %q", warning)
	}
}

func TestDasApplicationTypeRequiresReplace(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		state types.Int64
		want  bool
	}{
		{types.Int64Value(8), true},
		{types.Int64Null(), false},
	} {
		resp := &int64planmodifier.RequiresReplaceIfFuncResponse{}
		dasApplicationTypeRequiresReplace(ctx, planmodifier.Int64Request{StateValue: tc.state, PlanValue: types.Int64Value(9)}, resp)
		if resp.RequiresReplace != tc.want {
			t.Errorf("state %s: expected RequiresReplace %v", tc.state, tc.want)
		}
	}
}
