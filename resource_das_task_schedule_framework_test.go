package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const dasTaskScheduleTestResponse = `{"scheduleTaskId":11,"scheduleTaskName":"Crawl File Share","taskTypeName":"Crawl",
"interval":1,"scheduleType":"Weekly","active":true,"startTime":1700000000,"endTime":null,"daysOfWeek":["Monday","Friday"],
"runAfterScheduleTaskId":null,"applicationId":42,"createdByDisplayName":"John Doe","nextRun":1700600000,"lastRun":null}`

func TestDasTaskScheduleClient(t *testing.T) {
	client, requests := serviceDeskIntegrationTestServer(t, func(r *http.Request) (int, string) {
		switch r.Method {
		case http.MethodPost:
			return http.StatusOK, "11"
		case http.MethodPut, http.MethodDelete:
			return http.StatusNoContent, ""
		}
		return http.StatusOK, dasTaskScheduleTestResponse
	})
	ctx := context.Background()
	var diags diag.Diagnostics
	request := dasTaskScheduleFromModel(ctx, DasTaskScheduleModel{
		TaskTypeName:     types.StringValue("Crawl"),
		ScheduleType:     types.StringValue("Weekly"),
		ScheduleTaskName: types.StringNull(),
		Interval:         types.Int64Value(1),
		StartTime:        types.Int64Unknown(),
		EndTime:          types.Int64Null(),
		DaysOfWeek:       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("Monday")}),
		Active:           types.BoolValue(true),
		ApplicationID:    types.Int64Value(42),
	}, &diags)
	id, err := client.CreateDasTaskSchedule(ctx, request)
	if err != nil || id != 11 {
		t.Fatalf("unexpected create result %d (%v)", id, err)
	}
	got := (*requests)[0]
	if got.Method != http.MethodPost || got.Path != "/v2026/das/tasks/schedules" {
		t.Fatalf("unexpected create request %+v", got)
	}
	serviceDeskIntegrationTestJSONEqual(t, got.Body, `{"taskTypeName":"Crawl","scheduleType":"Weekly","interval":1,"daysOfWeek":["Monday"],"active":true,"applicationId":42}`)
	if err := client.UpdateDasTaskSchedule(ctx, "11", request); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[1]; got.Method != http.MethodPut || got.Path != "/v2026/das/tasks/schedules/11" {
		t.Fatalf("unexpected update request %+v", got)
	}
	schedule, err := client.GetDasTaskSchedule(ctx, "11")
	if err != nil || schedule.ScheduleTaskName != "Crawl File Share" || *schedule.ApplicationID != 42 {
		t.Fatalf("unexpected schedule %+v (%v)", schedule, err)
	}
	if err := client.DeleteDasTaskSchedule(ctx, "11"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := (*requests)[3]; got.Method != http.MethodDelete || got.Path != "/v2026/das/tasks/schedules/11" {
		t.Fatalf("unexpected delete request %+v", got)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestDasTaskScheduleState(t *testing.T) {
	ctx := context.Background()
	var api DasTaskSchedule
	if err := json.Unmarshal([]byte(dasTaskScheduleTestResponse), &api); err != nil {
		t.Fatal(err)
	}
	days := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("Friday"), types.StringValue("Monday")})
	data := DasTaskScheduleModel{
		ID:                       types.StringValue("11"),
		TaskTypeName:             types.StringValue("Crawl"),
		ScheduleType:             types.StringValue("Weekly"),
		ScheduleTaskName:         types.StringNull(),
		Interval:                 types.Int64Value(1),
		StartTime:                types.Int64Unknown(),
		EndTime:                  types.Int64Null(),
		DaysOfWeek:               days,
		Active:                   types.BoolValue(true),
		RunAfterScheduleTaskID:   types.Int64Null(),
		ApplicationID:            types.Int64Value(42),
		RunAfterScheduleTaskName: types.StringUnknown(),
		CreatedByDisplayName:     types.StringUnknown(),
		NextRun:                  types.Int64Unknown(),
		LastRun:                  types.Int64Unknown(),
	}
	var diags diag.Diagnostics
	setDasTaskScheduleState(ctx, &data, &api, false, &diags)
	serviceDeskIntegrationTestAssertKnown(t, data)
	serviceDeskIntegrationTestSetState(t, NewDasTaskScheduleResource(), &data)
	if !data.ScheduleTaskName.IsNull() || data.StartTime.ValueInt64() != 1700000000 || !data.LastRun.IsNull() || !data.DaysOfWeek.Equal(days) {
		t.Errorf("unexpected state after create %+v", data)
	}

	// Read keeps the configured order and null optional values, and detects changed values.
	api.Interval = nil
	api.Active = false
	api.ScheduleTaskName = ""
	setDasTaskScheduleState(ctx, &data, &api, true, &diags)
	if !data.DaysOfWeek.Equal(days) || !data.EndTime.IsNull() || !data.RunAfterScheduleTaskID.IsNull() || data.Active.ValueBool() ||
		!data.Interval.Equal(types.Int64Value(0)) || !data.ScheduleTaskName.IsNull() {
		t.Errorf("unexpected state after refresh %+v", data)
	}
	// Unset schedule_task_name and interval stay null; a name set outside Terraform is detected.
	data.Interval = types.Int64Null()
	api.ScheduleTaskName = "Renamed"
	setDasTaskScheduleState(ctx, &data, &api, true, &diags)
	if !data.Interval.IsNull() || data.ScheduleTaskName.ValueString() != "Renamed" {
		t.Errorf("unexpected state after second refresh %+v", data)
	}
	if data.ID.ValueString() != "11" || diags.HasError() {
		t.Errorf("unexpected ID %s (%v)", data.ID, diags)
	}
}
