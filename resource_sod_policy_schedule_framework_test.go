package main

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSodPolicyScheduleClientAndState(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"name":"Weekly","description":"","created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z",
			"schedule":{"type":"WEEKLY","days":{"type":"LIST","values":["1"],"interval":null},"hours":{"type":"LIST","values":["8"],"interval":null},"expiration":null,"timeZoneId":"UTC"},
			"recipients":[{"type":"IDENTITY","id":"i-2","name":"Two"},{"type":"IDENTITY","id":"i-1","name":"One"}],
			"emailEmptyResults":false,"creatorId":"c-1","modifierId":"m-1"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	recipients, d := types.ListValueFrom(ctx, objectInfoObjectType, []OwnerModel{
		{ID: types.StringValue("i-1"), Type: types.StringValue("IDENTITY"), Name: types.StringNull()},
		{ID: types.StringValue("i-2"), Type: types.StringValue("IDENTITY"), Name: types.StringValue("Two")},
	})
	diags.Append(d...)
	data := SodPolicyScheduleModel{
		ID:                types.StringUnknown(),
		PolicyID:          types.StringValue("sp-1"),
		Name:              types.StringValue("Weekly"),
		Description:       types.StringNull(),
		ScheduleJSON:      types.StringValue(`{"type":"WEEKLY","days":{"type":"LIST","values":["1"]},"hours":{"type":"LIST","values":["8"]}}`),
		Recipient:         recipients,
		EmailEmptyResults: types.BoolUnknown(),
		CreatorID:         types.StringUnknown(),
		ModifierID:        types.StringUnknown(),
		Created:           types.StringUnknown(),
		Modified:          types.StringUnknown(),
	}
	planned := data
	updated, err := client.SetSodPolicySchedule(ctx, "sp-1", sodPolicyScheduleFromModel(ctx, data, &diags))
	if err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2026/sod-policies/sp-1/schedule" ||
		gotBody != `{"name":"Weekly","schedule":{"days":{"type":"LIST","values":["1"]},"hours":{"type":"LIST","values":["8"]},"type":"WEEKLY"},"recipients":[{"type":"IDENTITY","id":"i-1"},{"type":"IDENTITY","id":"i-2","name":"Two"}]}` {
		t.Fatalf("unexpected set request %s %s %s", gotMethod, gotPath, gotBody)
	}
	setSodPolicyScheduleComputed(&data, updated)
	if data.ID.ValueString() != "sp-1" || data.EmailEmptyResults.IsUnknown() || data.CreatorID.ValueString() != "c-1" || !data.ScheduleJSON.Equal(planned.ScheduleJSON) {
		t.Fatalf("unexpected state after apply %+v", data)
	}

	read, err := client.GetSodPolicySchedule(ctx, "sp-1")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	setSodPolicyScheduleState(ctx, &data, read, &diags)
	if diags.HasError() || !data.ScheduleJSON.Equal(planned.ScheduleJSON) || !data.Description.IsNull() || !data.Recipient.Equal(planned.Recipient) {
		t.Fatalf("expected refreshed state to match the configuration, got %+v (%v)", data, diags)
	}

	if err := client.DeleteSodPolicySchedule(ctx, "sp-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/sod-policies/sp-1/schedule" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}
