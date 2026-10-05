package provider

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSourceScheduleClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotContentType = r.Method, r.URL.Path, string(body), r.Header.Get("Content-Type")
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			fallthrough
		default:
			_, _ = w.Write([]byte(`{"type":"ACCOUNT_AGGREGATION","cronExpression":"0 0 12 * * ?"}`))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	if _, err := client.CreateSourceSchedule(ctx, "src-1", &SourceSchedule{Type: "ACCOUNT_AGGREGATION", CronExpression: "0 0 12 * * ?"}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/sources/src-1/schedules" || gotBody != `{"type":"ACCOUNT_AGGREGATION","cronExpression":"0 0 12 * * ?"}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}

	plan := SourceScheduleModel{SourceID: types.StringValue("src-1"), Type: types.StringValue("ACCOUNT_AGGREGATION"), CronExpression: types.StringValue("0 0 6 * * ?")}
	state := plan
	state.CronExpression = types.StringValue("0 0 12 * * ?")
	if _, err := client.PatchSourceSchedule(ctx, "src-1", "ACCOUNT_AGGREGATION", sourceSchedulePatch(plan, state)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/sources/src-1/schedules/ACCOUNT_AGGREGATION" || gotContentType != "application/json-patch+json" ||
		gotBody != `[{"op":"replace","path":"/cronExpression","value":"0 0 6 * * ?"}]` {
		t.Fatalf("unexpected patch request %s %s %s %s", gotMethod, gotPath, gotContentType, gotBody)
	}
	if ops := sourceSchedulePatch(plan, plan); len(ops) != 0 {
		t.Fatalf("expected no operations without changes, got %+v", ops)
	}

	schedule, err := client.GetSourceSchedule(ctx, "src-1", "ACCOUNT_AGGREGATION")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	read := SourceScheduleModel{SourceID: types.StringValue("src-1"), Type: types.StringValue("ACCOUNT_AGGREGATION")}
	setSourceScheduleState(&read, schedule)
	if read.ID.ValueString() != "src-1/ACCOUNT_AGGREGATION" || read.CronExpression.ValueString() != "0 0 12 * * ?" {
		t.Fatalf("unexpected state %+v", read)
	}
	if err := client.DeleteSourceSchedule(ctx, "src-1", "ACCOUNT_AGGREGATION"); err != nil || gotMethod != http.MethodDelete {
		t.Fatalf("unexpected delete request %s (%v)", gotMethod, err)
	}
}

func TestSourceScheduleTypeValidator(t *testing.T) {
	for value, wantErr := range map[string]bool{"ACCOUNT_AGGREGATION": false, "GROUP_AGGREGATION": false, "account_aggregation": true, "OTHER": true} {
		resp := &validator.StringResponse{}
		sourceScheduleTypeValidator{}.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("type"), ConfigValue: types.StringValue(value)}, resp)
		if resp.Diagnostics.HasError() != wantErr {
			t.Errorf("value %s: expected error %v, got %v", value, wantErr, resp.Diagnostics)
		}
	}
}
