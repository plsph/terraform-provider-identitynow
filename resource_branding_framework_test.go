package main

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBrandingClientSendsMultipartForm(t *testing.T) {
	withFastRetries(t)
	var gotMethod, gotPath string
	var gotFields map[string][]string
	var gotFiles int
	attempts := 0
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data; boundary=") {
				t.Errorf("unexpected content type %q", r.Header.Get("Content-Type"))
			}
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("invalid multipart body: %s", err)
			}
			gotFields, gotFiles = r.MultipartForm.Value, len(r.MultipartForm.File)
			// The body can be sent again when the request is retried.
			attempts++
			if attempts == 1 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"name":"corp","productName":"Corp IAM","actionButtonColor":"0074D9","activeLinkColor":"011E69",
			"navigationColor":"011E69","emailFromAddress":null,"standardLogoURL":"https://logo","loginInformationalMessage":null}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	plan := BrandingModel{
		ID: types.StringUnknown(), Name: types.StringValue("corp"), ProductName: types.StringValue("Corp IAM"),
		ActionButtonColor: types.StringValue("0074D9"), ActiveLinkColor: types.StringNull(), NavigationColor: types.StringNull(),
		EmailFromAddress: types.StringNull(), LoginInformationalMessage: types.StringNull(), StandardLogoURL: types.StringUnknown(),
	}
	created, err := client.CreateBranding(ctx, brandingFromModel(plan, nil))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/brandings" || gotFiles != 0 || len(gotFields) != 3 ||
		gotFields["name"][0] != "corp" || gotFields["productName"][0] != "Corp IAM" || gotFields["actionButtonColor"][0] != "0074D9" {
		t.Fatalf("unexpected create request %s %s %v", gotMethod, gotPath, gotFields)
	}
	brandingResolveComputed(&plan, created)
	if plan.ID.ValueString() != "corp" || !plan.ActiveLinkColor.IsNull() || !plan.EmailFromAddress.IsNull() ||
		plan.StandardLogoURL.ValueString() != "https://logo" || plan.ActionButtonColor.ValueString() != "0074D9" {
		t.Fatalf("unexpected state after create %+v", plan)
	}

	// An update sends the configured values, clears removed values and does not send unset ones.
	attempts = 1
	prior := plan
	prior.LoginInformationalMessage = types.StringValue("Welcome")
	plan.NavigationColor = types.StringValue("011E69")
	if _, err := client.UpdateBranding(ctx, "corp", brandingFromModel(plan, &prior)); err != nil || gotMethod != http.MethodPut || gotPath != "/v2026/brandings/corp" {
		t.Fatalf("unexpected update request %s %s (%v)", gotMethod, gotPath, err)
	}
	if _, ok := gotFields["emailFromAddress"]; ok || gotFields["navigationColor"][0] != "011E69" ||
		len(gotFields["loginInformationalMessage"]) != 1 || gotFields["loginInformationalMessage"][0] != "" || gotFiles != 0 {
		t.Fatalf("unexpected update fields %v", gotFields)
	}
	branding, err := client.GetBranding(ctx, "corp")
	if err != nil || gotMethod != http.MethodGet {
		t.Fatalf("unexpected get request %s (%v)", gotMethod, err)
	}
	read := BrandingModel{ID: types.StringValue("corp"), ProductName: types.StringValue("Corp IAM")}
	setBrandingState(&read, branding)
	if read.Name.ValueString() != "corp" || !read.LoginInformationalMessage.IsNull() || read.NavigationColor.ValueString() != "011E69" {
		t.Fatalf("unexpected refreshed state %+v", read)
	}
	// An unset value stays null when the API returns an empty value, and a set value is refreshed.
	empty := ""
	branding.ActionButtonColor = &empty
	read = BrandingModel{ID: types.StringValue("corp"), ProductName: types.StringValue("Corp IAM"), EmailFromAddress: types.StringValue("x@example.com")}
	setBrandingState(&read, branding)
	if !read.ActionButtonColor.IsNull() || read.EmailFromAddress.ValueString() != "" {
		t.Fatalf("unexpected refreshed state %+v", read)
	}
	if err := client.DeleteBranding(ctx, "corp"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/brandings/corp" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}
