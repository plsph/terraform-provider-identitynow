package main

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestCustomPasswordInstructionClient(t *testing.T) {
	var gotMethod, gotURI, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-SailPoint-Experimental") != "true" {
			t.Errorf("missing experimental header on %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotURI, gotBody = r.Method, r.URL.RequestURI(), string(body)
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"pageId":"reset-password:enter-password","locale":"en","pageContent":"See the policy"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	if _, err := client.CreateCustomPasswordInstruction(ctx, &CustomPasswordInstruction{
		PageID: "reset-password:enter-password", PageContent: "See the policy", Locale: "en",
	}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotURI != "/v2026/custom-password-instructions" ||
		gotBody != `{"pageId":"reset-password:enter-password","pageContent":"See the policy","locale":"en"}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotURI, gotBody)
	}
	instruction, err := client.GetCustomPasswordInstruction(ctx, "reset-password:enter-password", "en")
	if err != nil || gotURI != "/v2026/custom-password-instructions/reset-password:enter-password?locale=en" {
		t.Fatalf("unexpected get request %s (%v)", gotURI, err)
	}
	data := CustomPasswordInstructionModel{PageID: types.StringValue("reset-password:enter-password"), Locale: types.StringValue("en")}
	setCustomPasswordInstructionState(&data, instruction)
	if data.ID.ValueString() != "reset-password:enter-password/en" || data.PageContent.ValueString() != "See the policy" {
		t.Fatalf("unexpected state %+v", data)
	}
	sanitized := &CustomPasswordInstruction{PageID: "reset-password:enter-password", Locale: "en", PageContent: "See <b>the</b> policy"}
	configured := CustomPasswordInstructionModel{
		PageID: types.StringValue("reset-password:enter-password"), Locale: types.StringValue("en"),
		PageContent: types.StringValue("See <b onclick=\"x()\">the</b> policy"),
	}
	setCustomPasswordInstructionResourceState(&configured, sanitized)
	if configured.PageContent.ValueString() != "See <b onclick=\"x()\">the</b> policy" || configured.ID.ValueString() != "reset-password:enter-password/en" {
		t.Fatalf("expected the configured page content to be kept, got %+v", configured)
	}
	imported := CustomPasswordInstructionModel{
		PageID: types.StringValue("reset-password:enter-password"), Locale: types.StringValue("en"), PageContent: types.StringNull(),
	}
	setCustomPasswordInstructionResourceState(&imported, sanitized)
	if imported.PageContent.ValueString() != "See <b>the</b> policy" {
		t.Fatalf("expected the API page content after import, got %+v", imported)
	}
	if err := client.DeleteCustomPasswordInstruction(ctx, "reset-password:enter-password", "en"); err != nil || gotMethod != http.MethodDelete ||
		gotURI != "/v2026/custom-password-instructions/reset-password:enter-password?locale=en" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotURI, err)
	}
}

func TestCustomPasswordInstructionImportState(t *testing.T) {
	ctx := context.Background()
	r := NewCustomPasswordInstructionResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	for importID, want := range map[string][2]string{
		"mfa:select":                     {"mfa:select", "default"},
		"reset-password:finish/de-DE":    {"reset-password:finish", "de-DE"},
		"unlock-account:enter-username/": {"", ""},
	} {
		resp := &resource.ImportStateResponse{State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		}}
		r.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{ID: importID}, resp)
		if want[0] == "" {
			if !resp.Diagnostics.HasError() {
				t.Errorf("expected an error for import ID %q", importID)
			}
			continue
		}
		var data CustomPasswordInstructionModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &data)...)
		if resp.Diagnostics.HasError() || data.PageID.ValueString() != want[0] || data.Locale.ValueString() != want[1] ||
			data.ID.ValueString() != want[0]+"/"+want[1] {
			t.Errorf("unexpected import of %q: %+v (%v)", importID, data, resp.Diagnostics)
		}
	}
}
