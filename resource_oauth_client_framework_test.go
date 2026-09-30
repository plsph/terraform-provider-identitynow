package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func oauthClientTestSet(values ...string) types.Set {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, elements)
}

func TestOauthClientClient(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody, gotContentType = r.Method, r.URL.Path, string(body), r.Header.Get("Content-Type")
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"id":"c-1","secret":"s3cr3t","name":"ci","description":"CI","accessTokenValiditySeconds":750,
			"refreshTokenValiditySeconds":86400,"grantTypes":["CLIENT_CREDENTIALS"],"accessType":"OFFLINE","type":"CONFIDENTIAL",
			"internal":false,"enabled":true,"strongAuthSupported":false,"claimsSupported":false,"scope":["sp:scopes:all"],
			"created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z"}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := OauthClientModel{
		Name: types.StringValue("ci"), Description: types.StringValue("CI"),
		BusinessName: types.StringNull(), HomepageURL: types.StringNull(),
		AccessTokenValiditySeconds: types.Int64Value(750), RefreshTokenValiditySeconds: types.Int64Unknown(),
		RedirectURIs: types.SetNull(types.StringType), GrantTypes: oauthClientTestSet("CLIENT_CREDENTIALS"),
		AccessType: types.StringValue("OFFLINE"), Type: types.StringUnknown(), Internal: types.BoolUnknown(),
		Enabled: types.BoolValue(true), StrongAuthSupported: types.BoolUnknown(), ClaimsSupported: types.BoolUnknown(),
		Scope: types.SetUnknown(types.StringType), ID: types.StringUnknown(), Secret: types.StringUnknown(),
		Created: types.StringUnknown(), Modified: types.StringUnknown(),
	}
	created, err := client.CreateOauthClient(ctx, oauthClientFromModel(ctx, plan, &diags))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := `{"name":"ci","description":"CI","accessTokenValiditySeconds":750,"grantTypes":["CLIENT_CREDENTIALS"],"accessType":"OFFLINE","enabled":true}`
	if gotMethod != http.MethodPost || gotPath != "/v2026/oauth-clients" || gotBody != want {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}

	oauthClientResolveUnknowns(ctx, &plan, created, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if plan.ID.ValueString() != "c-1" || plan.Secret.ValueString() != "s3cr3t" || plan.Type.ValueString() != "CONFIDENTIAL" ||
		plan.RefreshTokenValiditySeconds.ValueInt64() != 86400 || plan.Internal.ValueBool() || len(plan.Scope.Elements()) != 1 ||
		!plan.BusinessName.IsNull() || !plan.RedirectURIs.IsNull() || plan.Modified.ValueString() != "2026-01-02T00:00:00Z" {
		t.Fatalf("unexpected state after create %+v", plan)
	}

	ops := []jsonPatchOp{{Op: "replace", Path: "/name", Value: "ci2"}}
	if _, err := client.PatchOauthClient(ctx, "c-1", ops); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/v2026/oauth-clients/c-1" || gotContentType != "application/json-patch+json" ||
		gotBody != `[{"op":"replace","path":"/name","value":"ci2"}]` {
		t.Fatalf("unexpected patch request %s %s %s %s", gotMethod, gotPath, gotContentType, gotBody)
	}
	if err := client.DeleteOauthClient(ctx, "c-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/oauth-clients/c-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestOauthClientReadKeepsNullOptionalsAndSecret(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	var client OauthClient
	_ = json.Unmarshal([]byte(`{"id":"c-1","name":"ci","description":"CI","businessName":null,"homepageUrl":"","accessTokenValiditySeconds":750,
		"refreshTokenValiditySeconds":86400,"redirectUris":[],"grantTypes":["REFRESH_TOKEN","AUTHORIZATION_CODE"],"accessType":"ONLINE",
		"type":"PUBLIC","internal":false,"enabled":false,"strongAuthSupported":true,"claimsSupported":false,"scope":["sp:scopes:all"],
		"created":"c","modified":"m"}`), &client)
	data := OauthClientModel{
		BusinessName: types.StringNull(), HomepageURL: types.StringNull(), RedirectURIs: types.SetNull(types.StringType),
		RefreshTokenValiditySeconds: types.Int64Value(86400), Secret: types.StringValue("kept"),
		GrantTypes: oauthClientTestSet("AUTHORIZATION_CODE", "REFRESH_TOKEN"), Scope: oauthClientTestSet("sp:scopes:all"),
	}
	setOauthClientState(ctx, &data, &client, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.BusinessName.IsNull() || !data.HomepageURL.IsNull() || !data.RedirectURIs.IsNull() {
		t.Fatalf("expected unset optional attributes to stay null, got %+v", data)
	}
	if data.Secret.ValueString() != "kept" || data.Enabled.ValueBool() || !data.StrongAuthSupported.ValueBool() || len(data.GrantTypes.Elements()) != 2 {
		t.Fatalf("unexpected refreshed state %+v", data)
	}
}

func TestOauthClientPatchOpsOnlyChangedFields(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := OauthClientModel{
		Name: types.StringValue("ci"), Description: types.StringValue("CI"), BusinessName: types.StringValue("acme"),
		HomepageURL: types.StringNull(), AccessTokenValiditySeconds: types.Int64Value(750), RefreshTokenValiditySeconds: types.Int64Value(86400),
		RedirectURIs: oauthClientTestSet("https://a"), GrantTypes: oauthClientTestSet("CLIENT_CREDENTIALS"), AccessType: types.StringValue("OFFLINE"),
		Enabled: types.BoolValue(true), StrongAuthSupported: types.BoolValue(false), ClaimsSupported: types.BoolValue(false),
	}
	plan := state
	plan.Description = types.StringValue("CI pipeline")
	plan.BusinessName = types.StringNull()
	plan.RedirectURIs = types.SetNull(types.StringType)
	plan.Enabled = types.BoolValue(false)
	plan.StrongAuthSupported = types.BoolUnknown()
	encoded, _ := json.Marshal(oauthClientPatchOps(ctx, plan, state, &diags))
	want := `[{"op":"replace","path":"/description","value":"CI pipeline"},{"op":"remove","path":"/businessName"},{"op":"replace","path":"/redirectUris","value":[]},{"op":"replace","path":"/enabled","value":false}]`
	if string(encoded) != want {
		t.Fatalf("unexpected patch ops %s", encoded)
	}
	if ops := oauthClientPatchOps(ctx, state, state, &diags); len(ops) != 0 {
		t.Fatalf("expected no ops without changes, got %+v", ops)
	}
}
