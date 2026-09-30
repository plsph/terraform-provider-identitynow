package main

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLauncherClient(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"l-1","created":"c","modified":"m","owner":{"type":"IDENTITY","id":"i-1"},"name":"Onboard",
			"description":"d","type":"INTERACTIVE_PROCESS","disabled":false,"reference":{"type":"WORKFLOW","id":"w-1"},"config":"{\"workflowId\":\"w-1\"}"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := LauncherModel{
		ID: types.StringUnknown(), Name: types.StringValue("Onboard"), Description: types.StringValue("d"),
		Type: types.StringValue(launcherDefaultType), Disabled: types.BoolValue(false),
		Reference: types.ListValueMust(launcherReferenceObjectType, []attr.Value{types.ObjectValueMust(launcherReferenceObjectType.AttrTypes, map[string]attr.Value{
			"id": types.StringValue("w-1"), "type": types.StringUnknown(),
		})}),
		ConfigJSON: types.StringValue(`{ "workflowId": "w-1" }`), Owner: types.ListUnknown(objectInfoObjectType),
		Created: types.StringUnknown(), Modified: types.StringUnknown(),
	}
	launcher := launcherFromModel(ctx, &plan, &diags)
	created, err := client.CreateLauncher(ctx, launcher)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := `{"name":"Onboard","description":"d","type":"INTERACTIVE_PROCESS","disabled":false,"reference":{"type":"WORKFLOW","id":"w-1"},"config":"{ \"workflowId\": \"w-1\" }"}`
	if gotMethod != http.MethodPost || gotPath != "/v2026/launchers" || gotBody != want {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	launcherResolveComputed(ctx, &plan, created, &diags)
	var references []LauncherReferenceModel
	diags.Append(plan.Reference.ElementsAs(ctx, &references, false)...)
	if diags.HasError() || plan.ID.ValueString() != "l-1" || references[0].Type.ValueString() != "WORKFLOW" || len(plan.Owner.Elements()) != 1 ||
		plan.ConfigJSON.ValueString() != `{ "workflowId": "w-1" }` {
		t.Fatalf("unexpected state after create %+v (%v)", plan, diags)
	}

	if _, err := client.UpdateLauncher(ctx, "l-1", launcher); err != nil || gotMethod != http.MethodPut || gotPath != "/v2026/launchers/l-1" || gotBody != want {
		t.Fatalf("unexpected update request %s %s %s (%v)", gotMethod, gotPath, gotBody, err)
	}
	read, err := client.GetLauncher(ctx, "l-1")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	setLauncherState(ctx, &plan, read, &diags)
	if plan.ConfigJSON.ValueString() != `{ "workflowId": "w-1" }` {
		t.Fatalf("expected equivalent config to be kept, got %s", plan.ConfigJSON)
	}
	read.Config = `{"workflowId":"w-2"}`
	read.Reference = nil
	setLauncherState(ctx, &plan, read, &diags)
	if plan.ConfigJSON.ValueString() != `{"workflowId":"w-2"}` || !plan.Reference.IsNull() {
		t.Fatalf("expected drift to be detected, got %+v", plan)
	}
	if err := client.DeleteLauncher(ctx, "l-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/launchers/l-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}
