package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const notificationTemplateTestResponse = `{"id":"nt-1","key":"cloud_manual_work_item_summary","name":"Work Items","medium":"EMAIL","locale":"en","subject":"You have work","header":null,"body":"Hello","footer":null,"from":"no-reply@example.com","replyTo":"","description":"Default description","created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z","slackTemplate":{"key":null,"text":"","isSubscription":false,"autoApprovalData":{"itemId":null}},"teamsTemplate":null}`

func TestNotificationTemplateClient(t *testing.T) {
	var gotMethod, gotPath, gotQuery, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotQuery, gotBody = r.Method, r.URL.Path, r.URL.Query().Get("filters"), string(body)
		switch {
		case r.URL.Path == "/v2026/notification-templates/bulk-delete":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/notification-templates":
			_, _ = w.Write([]byte(`[` + notificationTemplateTestResponse + `]`))
		default:
			_, _ = w.Write([]byte(notificationTemplateTestResponse))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := NotificationTemplateModel{
		ID:                types.StringUnknown(),
		Key:               types.StringValue("cloud_manual_work_item_summary"),
		Medium:            types.StringValue("EMAIL"),
		Locale:            types.StringValue("en"),
		Name:              types.StringUnknown(),
		Subject:           types.StringValue("You have work"),
		Header:            types.StringUnknown(),
		Body:              types.StringValue("Hello"),
		Footer:            types.StringUnknown(),
		From:              types.StringValue("no-reply@example.com"),
		ReplyTo:           types.StringUnknown(),
		Description:       types.StringUnknown(),
		SlackTemplateJSON: types.StringNull(),
		TeamsTemplateJSON: types.StringNull(),
		Created:           types.StringUnknown(),
		Modified:          types.StringUnknown(),
	}
	saved, err := client.SaveNotificationTemplate(ctx, notificationTemplateFromModel(plan, &diags))
	if err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	want := `{"key":"cloud_manual_work_item_summary","medium":"EMAIL","locale":"en","subject":"You have work","body":"Hello","from":"no-reply@example.com"}`
	if gotMethod != http.MethodPost || gotPath != "/v2026/notification-templates" || gotBody != want {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	resolveNotificationTemplateComputed(&plan, saved)
	if plan.ID.ValueString() != "nt-1" || plan.Name.ValueString() != "Work Items" || plan.Description.ValueString() != "Default description" ||
		plan.ReplyTo.ValueString() != "" || !plan.Header.IsNull() || !plan.Footer.IsNull() || plan.Created.IsUnknown() || plan.Modified.ValueString() != "2026-01-02T00:00:00Z" {
		t.Fatalf("expected unknown values to be resolved, got %+v", plan)
	}
	if plan.Subject.ValueString() != "You have work" || !plan.SlackTemplateJSON.IsNull() {
		t.Fatalf("expected planned values to be kept, got %+v", plan)
	}

	// Update sends the ID with the full template.
	plan.Subject = types.StringValue("Changed")
	if _, err := client.SaveNotificationTemplate(ctx, notificationTemplateFromModel(plan, &diags)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	var sent map[string]interface{}
	_ = json.Unmarshal([]byte(gotBody), &sent)
	if sent["id"] != "nt-1" || sent["subject"] != "Changed" || sent["description"] != "Default description" {
		t.Fatalf("unexpected update body %s", gotBody)
	}
	if _, ok := sent["header"]; ok {
		t.Fatalf("header must never be sent: %s", gotBody)
	}

	if _, err := client.GetNotificationTemplate(ctx, "nt-1"); err != nil || gotMethod != http.MethodGet || gotPath != "/v2026/notification-templates/nt-1" {
		t.Fatalf("unexpected get request %s %s (%v)", gotMethod, gotPath, err)
	}
	if _, err := client.GetNotificationTemplateByKey(ctx, "cloud_manual_work_item_summary", "EMAIL", "en"); err != nil ||
		gotQuery != `key eq "cloud_manual_work_item_summary" and medium eq "EMAIL" and locale eq "en"` {
		t.Fatalf("unexpected lookup %q (%v)", gotQuery, err)
	}
	if _, err := client.GetNotificationTemplateByKey(ctx, "cloud_manual_work_item_summary", "SLACK", "en"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}

	if err := client.DeleteNotificationTemplate(ctx, "cloud_manual_work_item_summary", "EMAIL", "en"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/notification-templates/bulk-delete" || gotBody != `[{"key":"cloud_manual_work_item_summary","medium":"EMAIL","locale":"en"}]` {
		t.Fatalf("unexpected delete request %s %s %s", gotMethod, gotPath, gotBody)
	}
}

func TestNotificationTemplateStateJSON(t *testing.T) {
	var api NotificationTemplate
	_ = json.Unmarshal([]byte(notificationTemplateTestResponse), &api)

	// A Slack template with only empty values maps to null when not configured.
	data := NotificationTemplateModel{SlackTemplateJSON: types.StringNull(), TeamsTemplateJSON: types.StringNull()}
	setNotificationTemplateState(&data, &api)
	if !data.SlackTemplateJSON.IsNull() || !data.TeamsTemplateJSON.IsNull() || !data.Header.IsNull() || data.Name.ValueString() != "Work Items" {
		t.Fatalf("unexpected state %+v", data)
	}

	// A configured value is kept when the API only adds empty defaults.
	api.SlackTemplate = map[string]interface{}{"text": "Hi", "blocks": `[{"type":"section"}]`, "isSubscription": false, "key": nil}
	data.SlackTemplateJSON = types.StringValue(`{"blocks": "[{\"type\":\"section\"}]", "text": "Hi"}`)
	prior := data.SlackTemplateJSON
	setNotificationTemplateState(&data, &api)
	if !data.SlackTemplateJSON.Equal(prior) {
		t.Fatalf("expected equivalent JSON to be kept, got %s", data.SlackTemplateJSON)
	}

	// A real change is detected.
	api.SlackTemplate["text"] = "Changed"
	setNotificationTemplateState(&data, &api)
	if data.SlackTemplateJSON.Equal(prior) {
		t.Fatalf("expected drift to be detected")
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(data.SlackTemplateJSON.ValueString()), &decoded); err != nil || decoded["text"] != "Changed" {
		t.Fatalf("unexpected JSON %s", data.SlackTemplateJSON)
	}
}

// notificationTemplateTestCreate runs Create with the configured values; the other optional and
// computed values are unknown in the plan, the Slack and Teams templates are null.
func notificationTemplateTestCreate(t *testing.T, serverURL string, configured map[string]attr.Value) *resource.CreateResponse {
	t.Helper()
	ctx := context.Background()
	r := NewNotificationTemplateResource().(*NotificationTemplateResource)
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: tenantSettingsTestProviderConfig(serverURL)}, &resource.ConfigureResponse{})
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	sch := schemaResp.Schema
	planned := map[string]attr.Value{"slack_template_json": types.StringNull(), "teams_template_json": types.StringNull()}
	for name, v := range configured {
		planned[name] = v
	}
	config := map[string]attr.Value{}
	for name, v := range configured {
		config[name] = v
	}
	req := resource.CreateRequest{
		Config: tfsdk.Config{Schema: sch, Raw: tenantSettingsTestRaw(t, sch, config, false)},
		Plan:   tfsdk.Plan{Schema: sch, Raw: tenantSettingsTestRaw(t, sch, planned, true)},
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sch, Raw: tenantSettingsTestRaw(t, sch, nil, false)}}
	r.Create(ctx, req, resp)
	return resp
}

// Create refuses to take over an existing custom template, and a new template starts from the
// default template for the attributes that are not configured.
func TestNotificationTemplateCreate(t *testing.T) {
	existing := false
	var saved map[string]interface{}
	var requests []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path+" "+r.URL.Query().Get("filters"))
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/notification-templates":
			if existing {
				_, _ = w.Write([]byte(`[` + notificationTemplateTestResponse + `]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/notification-template-defaults":
			_, _ = w.Write([]byte(`[{"key":"cloud_manual_work_item_summary","name":"Work Items","medium":"EMAIL","locale":"en","subject":"Default subject","body":"Default body","from":"$__global.emailFromAddress","replyTo":"$__global.emailReplyToAddress","description":"Default description"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2026/notification-templates":
			body, _ := io.ReadAll(r.Body)
			saved = map[string]interface{}{}
			_ = json.Unmarshal(body, &saved)
			saved["id"], saved["created"], saved["modified"] = "nt-1", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"
			_ = json.NewEncoder(w).Encode(saved)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	configured := map[string]attr.Value{
		"key":     types.StringValue("cloud_manual_work_item_summary"),
		"medium":  types.StringValue("EMAIL"),
		"locale":  types.StringValue("en"),
		"subject": types.StringValue("Custom subject"),
	}

	resp := notificationTemplateTestCreate(t, server.URL, configured)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics %v", resp.Diagnostics)
	}
	filter := `key eq "cloud_manual_work_item_summary" and medium eq "EMAIL" and locale eq "en"`
	want := []string{"GET /v2026/notification-templates " + filter, "GET /v2026/notification-template-defaults " + filter, "POST /v2026/notification-templates "}
	if strings.Join(requests, "|") != strings.Join(want, "|") {
		t.Fatalf("unexpected requests %v", requests)
	}
	if saved["subject"] != "Custom subject" || saved["body"] != "Default body" || saved["from"] != "$__global.emailFromAddress" || saved["name"] != "Work Items" || saved["description"] != "Default description" {
		t.Fatalf("unconfigured values must be taken from the default template, got %v", saved)
	}
	if !resp.State.Raw.IsFullyKnown() {
		t.Fatalf("state has unknown values: %s", resp.State.Raw)
	}
	var data NotificationTemplateModel
	resp.Diagnostics.Append(resp.State.Get(context.Background(), &data)...)
	if data.ID.ValueString() != "nt-1" || data.Subject.ValueString() != "Custom subject" || data.Body.ValueString() != "Default body" || data.ReplyTo.ValueString() != "$__global.emailReplyToAddress" {
		t.Fatalf("unexpected state %+v", data)
	}

	// An existing custom template is not overwritten.
	existing, saved, requests = true, nil, nil
	resp = notificationTemplateTestCreate(t, server.URL, configured)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Detail(), "terraform import identitynow_notification_template.<name> nt-1") {
		t.Fatalf("expected an already exists error, got %v", resp.Diagnostics)
	}
	if saved != nil || len(requests) != 1 {
		t.Fatalf("nothing must be saved, got %v %v", requests, saved)
	}
}

func TestNotificationTemplateFillFromDefault(t *testing.T) {
	data := NotificationTemplateModel{
		Name: types.StringUnknown(), Subject: types.StringValue("Configured"), Body: types.StringUnknown(),
		From: types.StringUnknown(), ReplyTo: types.StringValue(""), Description: types.StringUnknown(),
	}
	notificationTemplateFillFromDefault(&data, &NotificationTemplate{Name: "N", Subject: "S", Body: "B", From: "F", ReplyTo: "R", Description: "D"})
	if data.Name.ValueString() != "N" || data.Subject.ValueString() != "Configured" || data.Body.ValueString() != "B" ||
		data.From.ValueString() != "F" || data.ReplyTo.ValueString() != "" || data.Description.ValueString() != "D" {
		t.Fatalf("unexpected model %+v", data)
	}
}
