package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const campaignTemplateTestResponse = `{"id":"ct-1","name":"Managers","description":"Quarterly","created":"2026-01-01T00:00:00Z",
"modified":"2026-01-02T00:00:00Z","scheduled":false,"ownerRef":{"id":"o-1","type":"IDENTITY","name":"Owner","email":"o@example.com"},
"deadlineDuration":"P2W","campaign":{"id":null,"name":"Manager review","description":"Review","type":"MANAGER",
"emailNotificationEnabled":false,"status":null,"created":null,"sunsetCommentsRequired":true,"alerts":null}}`

func TestCampaignTemplateClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotContentType = r.Method, r.URL.Path, string(body), r.Header.Get("Content-Type")
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(campaignTemplateTestResponse))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	template := campaignTemplateFromModel(CampaignTemplateModel{
		Name:             types.StringValue("Managers"),
		Description:      types.StringValue("Quarterly"),
		DeadlineDuration: types.StringNull(),
		CampaignJSON:     types.StringValue(`{"name":"Manager review","description":"Review","type":"MANAGER"}`),
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	created, err := client.CreateCampaignTemplate(ctx, template)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/campaign-templates" ||
		gotBody != `{"name":"Managers","description":"Quarterly","campaign":{"description":"Review","name":"Manager review","type":"MANAGER"}}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	if created.ID != "ct-1" || created.OwnerRef == nil || created.OwnerRef.Name != "Owner" || *created.DeadlineDuration != "P2W" {
		t.Fatalf("unexpected decoded template %+v", created)
	}

	ops := []jsonPatchOp{{Op: "replace", Path: "/name", Value: "New"}}
	if _, err := client.PatchCampaignTemplate(ctx, "ct-1", ops); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/campaign-templates/ct-1" || gotContentType != "application/json-patch+json" ||
		gotBody != `[{"op":"replace","path":"/name","value":"New"}]` {
		t.Fatalf("unexpected patch request %s %s %s %s", gotMethod, gotPath, gotContentType, gotBody)
	}
	if _, err := client.GetCampaignTemplate(ctx, "ct-1"); err != nil || gotMethod != http.MethodGet {
		t.Fatalf("unexpected get request %s (%v)", gotMethod, err)
	}
	if err := client.DeleteCampaignTemplate(ctx, "ct-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/campaign-templates/ct-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestCampaignTemplateJSONSubsetState(t *testing.T) {
	var api interface{}
	_ = json.Unmarshal([]byte(`{"name":"n","type":"SEARCH","emailNotificationEnabled":false,
		"searchCampaignInfo":{"type":"IDENTITY","query":"*","reviewer":null,"accessConstraints":[{"type":"ROLE","ids":["r-1"],"operator":"SELECTED"}]}}`), &api)

	prior := types.StringValue(`{"type": "SEARCH", "name": "n", "searchCampaignInfo": {"query": "*", "type": "IDENTITY", "reviewer": null, "identityIds": [],
		"accessConstraints": [{"type": "ROLE", "operator": "SELECTED", "ids": ["r-1"]}]}}`)
	if got := jsonSubsetState(prior, api); !got.Equal(prior) {
		t.Fatalf("expected configured subset to be kept, got %s", got)
	}

	changed := types.StringValue(`{"name":"n","searchCampaignInfo":{"query":"other"}}`)
	got := jsonSubsetState(changed, api)
	if got.Equal(changed) || !jsonSemanticallyEqual([]byte(got.ValueString()), mustCampaignTemplateJSON(t, api)) {
		t.Fatalf("expected drift to be reported with the API value, got %s", got)
	}

	shorter := types.StringValue(`{"searchCampaignInfo":{"accessConstraints":[]}}`)
	if got := jsonSubsetState(shorter, api); got.Equal(shorter) {
		t.Fatalf("expected a different array length to be reported as drift")
	}

	if got := jsonSubsetState(types.StringNull(), map[string]interface{}{}); !got.IsNull() {
		t.Fatalf("expected empty API value to map to null, got %s", got)
	}
	emptyArray := types.StringValue(`[]`)
	if got := jsonSubsetState(emptyArray, []interface{}{}); !got.Equal(emptyArray) {
		t.Fatalf("expected configured empty array to be kept, got %s", got)
	}
	if got := jsonSubsetState(types.StringNull(), []interface{}{"a"}); got.ValueString() != `["a"]` {
		t.Fatalf("expected imported value to be encoded, got %s", got)
	}
}

func mustCampaignTemplateJSON(t *testing.T, value interface{}) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func campaignTemplateTestModel() CampaignTemplateModel {
	return CampaignTemplateModel{
		ID:               types.StringValue("ct-1"),
		Name:             types.StringValue("Managers"),
		Description:      types.StringValue("Quarterly"),
		DeadlineDuration: types.StringNull(),
		CampaignJSON:     types.StringValue(`{"name": "Manager review", "description": "Review", "type": "MANAGER"}`),
		Scheduled:        types.BoolValue(false),
		OwnerRef:         types.ListNull(objectInfoObjectType),
		Created:          types.StringValue("2026-01-01T00:00:00Z"),
		Modified:         types.StringValue("2026-01-01T00:00:00Z"),
	}
}

func TestCampaignTemplateReadState(t *testing.T) {
	ctx := context.Background()
	var template CampaignTemplate
	if err := json.Unmarshal([]byte(campaignTemplateTestResponse), &template); err != nil {
		t.Fatal(err)
	}
	var diags diag.Diagnostics

	data := campaignTemplateTestModel()
	prior := data.CampaignJSON
	template.DeadlineDuration = nil
	setCampaignTemplateState(ctx, &data, &template, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.CampaignJSON.Equal(prior) {
		t.Fatalf("expected campaign JSON with added defaults and read-only fields to be kept, got %s", data.CampaignJSON)
	}
	if !data.DeadlineDuration.IsNull() || data.Modified.ValueString() != "2026-01-02T00:00:00Z" {
		t.Fatalf("unexpected state %+v", data)
	}
	var owners []OwnerModel
	diags.Append(data.OwnerRef.ElementsAs(ctx, &owners, false)...)
	if len(owners) != 1 || owners[0].ID.ValueString() != "o-1" {
		t.Fatalf("unexpected owner %v", data.OwnerRef)
	}

	// Import: read-only campaign fields are not stored.
	imported := CampaignTemplateModel{ID: types.StringValue("ct-1"), DeadlineDuration: types.StringNull(), CampaignJSON: types.StringNull()}
	setCampaignTemplateState(ctx, &imported, &template, &diags)
	var campaign map[string]interface{}
	_ = json.Unmarshal([]byte(imported.CampaignJSON.ValueString()), &campaign)
	if _, ok := campaign["status"]; ok || campaign["type"] != "MANAGER" || campaign["sunsetCommentsRequired"] != true {
		t.Fatalf("unexpected imported campaign JSON %s", imported.CampaignJSON)
	}
}

func TestCampaignTemplateComputedKeepsPlannedValues(t *testing.T) {
	ctx := context.Background()
	var template CampaignTemplate
	_ = json.Unmarshal([]byte(campaignTemplateTestResponse), &template)
	data := campaignTemplateTestModel()
	data.ID, data.Created, data.Modified = types.StringUnknown(), types.StringUnknown(), types.StringUnknown()
	data.Scheduled, data.OwnerRef = types.BoolUnknown(), types.ListUnknown(objectInfoObjectType)
	planned := data.CampaignJSON
	var diags diag.Diagnostics
	setCampaignTemplateComputed(ctx, &data, &template, &diags)
	for name, value := range map[string]attr.Value{"id": data.ID, "created": data.Created, "modified": data.Modified, "scheduled": data.Scheduled, "owner_ref": data.OwnerRef} {
		if value.IsUnknown() {
			t.Fatalf("%s is unknown after apply", name)
		}
	}
	if !data.CampaignJSON.Equal(planned) || !data.DeadlineDuration.IsNull() {
		t.Fatalf("expected planned values to be kept, got %+v", data)
	}
}

func TestCampaignTemplatePatchOps(t *testing.T) {
	var diags diag.Diagnostics
	state := campaignTemplateTestModel()
	state.DeadlineDuration = types.StringValue("P2W")

	plan := campaignTemplateTestModel()
	plan.Modified = types.StringUnknown()
	plan.CampaignJSON = types.StringValue(`{"type":"MANAGER","name":"Manager review","description":"Review"}`)
	ops := campaignTemplatePatchOps(plan, state, &diags)
	if len(ops) != 1 || ops[0].Op != "remove" || ops[0].Path != "/deadlineDuration" {
		t.Fatalf("expected only a deadline removal, got %+v", ops)
	}

	plan.DeadlineDuration = types.StringValue("P2W")
	plan.Description = types.StringValue("Yearly")
	plan.CampaignJSON = types.StringValue(`{"name":"Manager review","description":"Review","type":"MANAGER","emailNotificationEnabled":true}`)
	ops = campaignTemplatePatchOps(plan, state, &diags)
	if len(ops) != 2 || ops[0].Path != "/description" || ops[1].Path != "/campaign" {
		t.Fatalf("unexpected ops %+v", ops)
	}
	if campaign, ok := ops[1].Value.(map[string]interface{}); !ok || campaign["emailNotificationEnabled"] != true {
		t.Fatalf("unexpected campaign value %+v", ops[1].Value)
	}
}
