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

func triggerSubscriptionTestHTTPConfig(t *testing.T, model TriggerSubscriptionHTTPConfigModel) types.List {
	t.Helper()
	list, d := types.ListValueFrom(context.Background(), triggerSubscriptionHTTPConfigObjectType, []TriggerSubscriptionHTTPConfigModel{model})
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}
	return list
}

func triggerSubscriptionTestModel(t *testing.T) TriggerSubscriptionResourceModel {
	return TriggerSubscriptionResourceModel{
		ID:               types.StringValue("s-1"),
		Name:             types.StringValue("Identity created"),
		Description:      types.StringNull(),
		TriggerID:        types.StringValue("idn:identity-created"),
		TriggerName:      types.StringValue("Identity Created"),
		Type:             types.StringValue("HTTP"),
		ResponseDeadline: types.StringValue("PT1H"),
		HTTPConfig: triggerSubscriptionTestHTTPConfig(t, TriggerSubscriptionHTTPConfigModel{
			URL:                    types.StringValue("https://example.com/hook"),
			HTTPDispatchMode:       types.StringValue("SYNC"),
			HTTPAuthenticationType: types.StringValue("BASIC_AUTH"),
			BasicAuthUserName:      types.StringValue("user"),
			BasicAuthPassword:      types.StringValue("secret"),
			BearerToken:            types.StringNull(),
		}),
		EventBridgeConfig:  types.ListNull(triggerSubscriptionEventBridgeConfigObjectType),
		WorkflowConfigJSON: types.StringNull(),
		Enabled:            types.BoolValue(true),
		Filter:             types.StringNull(),
	}
}

func TestTriggerSubscriptionClient(t *testing.T) {
	var gotMethod, gotPath, gotQuery, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotQuery, gotBody, gotContentType = r.Method, r.URL.Path, r.URL.Query().Get("filters"), string(body), r.Header.Get("Content-Type")
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":"other","name":"x","triggerId":"t","type":"HTTP"},{"id":"s-1","name":"Identity created","triggerId":"idn:identity-created","triggerName":"Identity Created","type":"HTTP","responseDeadline":"PT1H","enabled":false,"httpConfig":{"url":"https://example.com/hook","httpDispatchMode":"SYNC","httpAuthenticationType":"BASIC_AUTH","basicAuthConfig":{"userName":"user","password":null}}}]`))
		default:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"s-1","name":"Identity created","triggerId":"idn:identity-created","triggerName":"Identity Created","type":"HTTP","responseDeadline":"PT1H","enabled":true}`))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := triggerSubscriptionTestModel(t)
	plan.ID = types.StringUnknown()
	plan.TriggerName = types.StringUnknown()
	plan.ResponseDeadline = types.StringUnknown()
	created, err := client.CreateTriggerSubscription(ctx, triggerSubscriptionFromModel(ctx, plan, &diags))
	if err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	wantBody := `{"name":"Identity created","triggerId":"idn:identity-created","type":"HTTP","httpConfig":{"url":"https://example.com/hook","httpDispatchMode":"SYNC","httpAuthenticationType":"BASIC_AUTH","basicAuthConfig":{"userName":"user","password":"secret"}},"enabled":true}`
	if gotMethod != http.MethodPost || gotPath != "/v2026/trigger-subscriptions" || gotBody != wantBody {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	if created.ID != "s-1" || created.TriggerName != "Identity Created" {
		t.Fatalf("unexpected create response %+v", created)
	}

	read, err := client.GetTriggerSubscription(ctx, "s-1")
	if err != nil || read.ID != "s-1" || gotQuery != `id eq "s-1"` {
		t.Fatalf("unexpected read %+v %q (%v)", read, gotQuery, err)
	}
	if _, err := client.GetTriggerSubscription(ctx, "missing"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}

	ops := []jsonPatchOp{{Op: "replace", Path: "/name", Value: "New"}}
	if _, err := client.PatchTriggerSubscription(ctx, "s-1", ops); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/trigger-subscriptions/s-1" || gotContentType != "application/json-patch+json" || gotBody != `[{"op":"replace","path":"/name","value":"New"}]` {
		t.Fatalf("unexpected patch request %s %s %s %s", gotMethod, gotPath, gotContentType, gotBody)
	}
	if err := client.DeleteTriggerSubscription(ctx, "s-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/trigger-subscriptions/s-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestTriggerSubscriptionStateKeepsSecretsAndNulls(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	data := triggerSubscriptionTestModel(t)
	data.WorkflowConfigJSON = types.StringValue(`{"workflowId": "w-1"}`)
	var api TriggerSubscription
	_ = json.Unmarshal([]byte(`{"id":"s-1","name":"Identity created","triggerId":"idn:identity-created","triggerName":"Identity Created","type":"HTTP","responseDeadline":"PT1H","enabled":false,
		"workflowConfig":{"workflowId":"w-1"},
		"httpConfig":{"url":"https://example.com/other","httpDispatchMode":"SYNC","httpAuthenticationType":"BASIC_AUTH","basicAuthConfig":{"userName":"user","password":null},"bearerTokenAuthConfig":null}}`), &api)
	setTriggerSubscriptionState(ctx, &data, &api, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.Description.IsNull() || !data.Filter.IsNull() || !data.EventBridgeConfig.IsNull() {
		t.Errorf("expected unset optional attributes to stay null: %s %s %s", data.Description, data.Filter, data.EventBridgeConfig)
	}
	if data.WorkflowConfigJSON.ValueString() != `{"workflowId": "w-1"}` {
		t.Errorf("expected equivalent JSON to be kept, got %s", data.WorkflowConfigJSON)
	}
	if data.Enabled.ValueBool() {
		t.Errorf("expected enabled to be refreshed")
	}
	var models []TriggerSubscriptionHTTPConfigModel
	diags.Append(data.HTTPConfig.ElementsAs(ctx, &models, false)...)
	if len(models) != 1 || models[0].URL.ValueString() != "https://example.com/other" {
		t.Fatalf("expected the URL to be refreshed, got %+v", models)
	}
	if models[0].BasicAuthPassword.ValueString() != "secret" || !models[0].BearerToken.IsNull() || models[0].BasicAuthUserName.ValueString() != "user" {
		t.Errorf("expected secrets to be kept from state, got %+v", models[0])
	}

	// Enabled defaults to true when the API omits it; a missing HTTP config clears the block.
	api.Enabled, api.HTTPConfig = nil, nil
	setTriggerSubscriptionState(ctx, &data, &api, &diags)
	if !data.Enabled.ValueBool() || !data.HTTPConfig.IsNull() {
		t.Errorf("unexpected state %s %s", data.Enabled, data.HTTPConfig)
	}
}

func TestTriggerSubscriptionPatchOpsOnlyChangedFields(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := triggerSubscriptionTestModel(t)
	state.Filter = types.StringValue("$.x")
	plan := triggerSubscriptionTestModel(t)
	if ops := triggerSubscriptionPatchOps(ctx, plan, triggerSubscriptionTestModel(t), &diags); len(ops) != 0 {
		t.Fatalf("expected no operations, got %+v", ops)
	}

	plan.Name = types.StringValue("Renamed")
	plan.Description = types.StringNull()
	plan.Filter = types.StringNull()
	plan.ResponseDeadline = types.StringUnknown()
	plan.HTTPConfig = triggerSubscriptionTestHTTPConfig(t, TriggerSubscriptionHTTPConfigModel{
		URL:                    types.StringValue("https://example.com/hook"),
		HTTPDispatchMode:       types.StringValue("SYNC"),
		HTTPAuthenticationType: types.StringValue("BEARER_TOKEN"),
		BasicAuthUserName:      types.StringNull(),
		BasicAuthPassword:      types.StringNull(),
		BearerToken:            types.StringValue("token"),
	})
	ops := triggerSubscriptionPatchOps(ctx, plan, state, &diags)
	encoded, _ := json.Marshal(ops)
	want := `[{"op":"replace","path":"/name","value":"Renamed"},{"op":"remove","path":"/filter"},{"op":"replace","path":"/httpConfig","value":{"url":"https://example.com/hook","httpDispatchMode":"SYNC","httpAuthenticationType":"BEARER_TOKEN","bearerTokenAuthConfig":{"bearerToken":"token"}}}]`
	if string(encoded) != want {
		t.Fatalf("unexpected operations\n got %s\nwant %s", encoded, want)
	}

	// Switching to EventBridge removes the HTTP config.
	bridge, _ := types.ListValueFrom(ctx, triggerSubscriptionEventBridgeConfigObjectType, []TriggerSubscriptionEventBridgeConfigModel{{AWSAccount: types.StringValue("123456789012"), AWSRegion: types.StringValue("us-east-1")}})
	plan = triggerSubscriptionTestModel(t)
	plan.Type = types.StringValue("EVENTBRIDGE")
	plan.HTTPConfig = types.ListNull(triggerSubscriptionHTTPConfigObjectType)
	plan.EventBridgeConfig = bridge
	encoded, _ = json.Marshal(triggerSubscriptionPatchOps(ctx, plan, triggerSubscriptionTestModel(t), &diags))
	want = `[{"op":"replace","path":"/type","value":"EVENTBRIDGE"},{"op":"remove","path":"/httpConfig"},{"op":"add","path":"/eventBridgeConfig","value":{"awsAccount":"123456789012","awsRegion":"us-east-1"}}]`
	if string(encoded) != want || diags.HasError() {
		t.Fatalf("unexpected operations\n got %s\nwant %s (%v)", encoded, want, diags)
	}

	// Members that are null in the state may be missing from the API object: they are added.
	plan = triggerSubscriptionTestModel(t)
	plan.Description = types.StringValue("New")
	plan.Filter = types.StringValue("$.x")
	plan.WorkflowConfigJSON = types.StringValue(`{"workflowId":"w-1"}`)
	encoded, _ = json.Marshal(triggerSubscriptionPatchOps(ctx, plan, triggerSubscriptionTestModel(t), &diags))
	want = `[{"op":"add","path":"/description","value":"New"},{"op":"add","path":"/filter","value":"$.x"},{"op":"add","path":"/workflowConfig","value":{"workflowId":"w-1"}}]`
	if string(encoded) != want || diags.HasError() {
		t.Fatalf("unexpected operations\n got %s\nwant %s (%v)", encoded, want, diags)
	}
}

func TestTriggerDataSourceClient(t *testing.T) {
	var gotPath, gotQuery string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.Query().Get("filters")
		_, _ = w.Write([]byte(`[{"id":"idn:identity-created","name":"Identity Created","type":"FIRE_AND_FORGET","description":"d","inputSchema":"{\"type\":\"object\"}","exampleInput":{"identity":{"id":"i-1"}}}]`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	trigger, err := client.GetTriggerDefinition(ctx, "idn:identity-created")
	if err != nil || gotPath != "/v2026/triggers" || gotQuery != `id eq "idn:identity-created"` {
		t.Fatalf("unexpected request %s %q (%v)", gotPath, gotQuery, err)
	}
	var data TriggerDataSourceModel
	setTriggerDataSourceState(&data, trigger)
	if data.InputSchema.ValueString() != `{"type":"object"}` || data.ExampleInputJSON.ValueString() != `{"identity":{"id":"i-1"}}` || data.Type.ValueString() != "FIRE_AND_FORGET" {
		t.Fatalf("unexpected state %+v", data)
	}

	if _, err := client.GetTriggerDefinitionByName(ctx, "Identity Created"); err != nil || gotQuery != "" {
		t.Fatalf("unexpected lookup by name %q (%v)", gotQuery, err)
	}
	if _, err := client.GetTriggerDefinitionByName(ctx, "Missing"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
