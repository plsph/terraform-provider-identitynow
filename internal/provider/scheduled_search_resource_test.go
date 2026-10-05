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

const scheduledSearchTestResponse = `{"id":"sch-1","owner":{"type":"IDENTITY","id":"o-1"},"ownerId":"o-1","name":"Daily","description":null,
"savedSearchId":"ss-1","created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z",
"schedule":{"type":"DAILY","hours":{"type":"LIST","values":["9"],"interval":null},"days":null,"months":null,"expiration":null,"timeZoneId":"UTC"},
"recipients":[{"type":"IDENTITY","id":"i-2"},{"type":"IDENTITY","id":"i-1"}],"enabled":true,"emailEmptyResults":false,"displayQueryDetails":false}`

func scheduledSearchTestModel(t *testing.T) ScheduledSearchModel {
	recipients, diags := types.ListValueFrom(context.Background(), scheduledSearchRefObjectType, []ScheduledSearchRefModel{
		{Type: types.StringValue("IDENTITY"), ID: types.StringValue("i-1")},
		{Type: types.StringValue("IDENTITY"), ID: types.StringValue("i-2")},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return ScheduledSearchModel{
		ID:                  types.StringValue("sch-1"),
		Name:                types.StringValue("Daily"),
		Description:         types.StringNull(),
		SavedSearchID:       types.StringValue("ss-1"),
		ScheduleJSON:        types.StringValue(`{"type":"DAILY","hours":{"type":"LIST","values":["9"]}}`),
		Recipient:           recipients,
		Enabled:             types.BoolValue(true),
		EmailEmptyResults:   types.BoolValue(false),
		DisplayQueryDetails: types.BoolValue(false),
		Owner:               types.ListNull(scheduledSearchRefObjectType),
		Created:             types.StringValue("2026-01-01T00:00:00Z"),
		Modified:            types.StringValue("2026-01-02T00:00:00Z"),
	}
}

func TestScheduledSearchClient(t *testing.T) {
	var gotMethods []string
	var putBody, postBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethods = append(gotMethods, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodPost:
			postBody = string(body)
		case http.MethodPut:
			putBody = string(body)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(scheduledSearchTestResponse))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := scheduledSearchTestModel(t)
	plan.ID, plan.Owner, plan.Created, plan.Modified = types.StringUnknown(), types.ListUnknown(scheduledSearchRefObjectType), types.StringUnknown(), types.StringUnknown()
	plan.EmailEmptyResults, plan.DisplayQueryDetails = types.BoolUnknown(), types.BoolUnknown()
	created, err := client.CreateScheduledSearch(ctx, scheduledSearchFromModel(ctx, plan, &diags))
	if err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	if postBody != `{"name":"Daily","savedSearchId":"ss-1","schedule":{"hours":{"type":"LIST","values":["9"]},"type":"DAILY"},"recipients":[{"type":"IDENTITY","id":"i-1"},{"type":"IDENTITY","id":"i-2"}],"enabled":true}` {
		t.Fatalf("unexpected create body %s", postBody)
	}
	setScheduledSearchComputed(ctx, &plan, created, &diags)
	if plan.ID.ValueString() != "sch-1" || plan.Owner.IsUnknown() || len(plan.Owner.Elements()) != 1 || plan.EmailEmptyResults.IsUnknown() || plan.DisplayQueryDetails.IsUnknown() {
		t.Fatalf("unexpected state after create %+v", plan)
	}

	plan.Name = types.StringNull()
	if _, err := client.UpdateScheduledSearch(ctx, "sch-1", scheduledSearchManagedFields(ctx, plan, &diags)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	var sent map[string]interface{}
	_ = json.Unmarshal([]byte(putBody), &sent)
	if owner, ok := sent["owner"].(map[string]interface{}); !ok || owner["id"] != "o-1" || sent["name"] != nil || sent["displayQueryDetails"] != false {
		t.Fatalf("unexpected merged PUT body %s", putBody)
	}
	if err := client.DeleteScheduledSearch(ctx, "sch-1"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := `["POST /v2026/scheduled-searches","GET /v2026/scheduled-searches/sch-1","PUT /v2026/scheduled-searches/sch-1","DELETE /v2026/scheduled-searches/sch-1"]`
	if encoded, _ := json.Marshal(gotMethods); string(encoded) != want {
		t.Fatalf("unexpected requests %s", encoded)
	}
}

func TestScheduledSearchReadState(t *testing.T) {
	ctx := context.Background()
	var search ScheduledSearch
	if err := json.Unmarshal([]byte(scheduledSearchTestResponse), &search); err != nil {
		t.Fatal(err)
	}
	var diags diag.Diagnostics
	data := scheduledSearchTestModel(t)
	prior := data
	setScheduledSearchState(ctx, &data, &search, &diags)
	if diags.HasError() || !data.ScheduleJSON.Equal(prior.ScheduleJSON) || !data.Recipient.Equal(prior.Recipient) || !data.Description.IsNull() {
		t.Fatalf("expected refreshed state to match the configuration, got %+v (%v)", data, diags)
	}
	search.Schedule["hours"] = map[string]interface{}{"type": "LIST", "values": []interface{}{"10"}}
	setScheduledSearchState(ctx, &data, &search, &diags)
	if data.ScheduleJSON.Equal(prior.ScheduleJSON) {
		t.Fatalf("expected schedule drift to be reported")
	}
}
