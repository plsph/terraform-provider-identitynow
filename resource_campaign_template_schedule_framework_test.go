package main

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func campaignTemplateScheduleTestSelector(t *testing.T, selectorType string, values []string, interval types.Int64) types.List {
	t.Helper()
	ctx := context.Background()
	valueList, diags := types.ListValueFrom(ctx, types.StringType, values)
	list, d := types.ListValueFrom(ctx, campaignTemplateScheduleSelectorObjectType, []CampaignTemplateScheduleSelectorModel{{
		Type: types.StringValue(selectorType), Values: valueList, Interval: interval,
	}})
	diags.Append(d...)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return list
}

func TestCampaignTemplateScheduleClient(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"type":"WEEKLY","days":{"type":"LIST","values":["1"],"interval":null},"hours":{"type":"LIST","values":["8"]},"expiration":null,"timeZoneId":"Europe/Warsaw"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	data := CampaignTemplateScheduleModel{
		CampaignTemplateID: types.StringValue("ct-1"),
		Type:               types.StringValue("WEEKLY"),
		Months:             types.ListNull(campaignTemplateScheduleSelectorObjectType),
		Days:               campaignTemplateScheduleTestSelector(t, "LIST", []string{"1"}, types.Int64Null()),
		Hours:              campaignTemplateScheduleTestSelector(t, "LIST", []string{"8"}, types.Int64Value(3)),
		Expiration:         types.StringNull(),
		TimeZoneID:         types.StringUnknown(),
	}
	schedule := campaignTemplateScheduleFromModel(ctx, data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if err := client.SetCampaignTemplateSchedule(ctx, "ct-1", schedule); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2026/campaign-templates/ct-1/schedule" ||
		gotBody != `{"type":"WEEKLY","days":{"type":"LIST","values":["1"]},"hours":{"type":"LIST","values":["8"],"interval":3}}` {
		t.Fatalf("unexpected set request %s %s %s", gotMethod, gotPath, gotBody)
	}

	read, err := client.GetCampaignTemplateSchedule(ctx, "ct-1")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	setCampaignTemplateScheduleState(ctx, &data, read, &diags)
	if diags.HasError() || data.TimeZoneID.ValueString() != "Europe/Warsaw" || !data.Months.IsNull() || !data.Expiration.IsNull() {
		t.Fatalf("unexpected state %+v (%v)", data, diags)
	}
	var days []CampaignTemplateScheduleSelectorModel
	diags.Append(data.Days.ElementsAs(ctx, &days, false)...)
	if len(days) != 1 || !days[0].Interval.IsNull() {
		t.Fatalf("expected unset interval to stay null, got %+v", days)
	}
	var hours []CampaignTemplateScheduleSelectorModel
	diags.Append(data.Hours.ElementsAs(ctx, &hours, false)...)
	if len(hours) != 1 || hours[0].Interval.ValueInt64() != 0 {
		t.Fatalf("expected removed interval to be reported as drift, got %+v", hours)
	}

	if err := client.DeleteCampaignTemplateSchedule(ctx, "ct-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/campaign-templates/ct-1/schedule" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestCampaignTemplateScheduleTimeState(t *testing.T) {
	prior := types.StringValue("2027-01-01T00:00:00Z")
	same := "2027-01-01T01:00:00+01:00"
	if got := campaignTemplateScheduleTimeState(prior, &same); !got.Equal(prior) {
		t.Fatalf("expected the same instant to keep the prior value, got %s", got)
	}
	other := "2027-02-01T00:00:00Z"
	if got := campaignTemplateScheduleTimeState(prior, &other); got.ValueString() != other {
		t.Fatalf("expected a different instant to be reported, got %s", got)
	}
	if got := campaignTemplateScheduleTimeState(types.StringNull(), nil); !got.IsNull() {
		t.Fatalf("expected unset expiration to stay null, got %s", got)
	}
}
