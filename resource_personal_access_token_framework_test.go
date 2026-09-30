package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPersonalAccessTokenClient(t *testing.T) {
	var requests []string
	var gotBody, gotContentType string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		gotBody, gotContentType = string(body), r.Header.Get("Content-Type")
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("owner-id") == "me":
			_, _ = w.Write([]byte(`[{"id":"p-1","name":"mine","scope":["sp:scopes:all"],"owner":{"type":"IDENTITY","id":"i-1","name":"Me"},"created":"c"}]`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":"p-1","name":"mine","scope":["sp:scopes:all"],"owner":{"type":"IDENTITY","id":"i-1","name":"Me"},"created":"c"},
				{"id":"p-2","name":"other","scope":["sp:scopes:all"],"owner":{"type":"IDENTITY","id":"i-2","name":"Other"},"created":"c"}]`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			_, _ = w.Write([]byte(`{"id":"p-3","secret":"s3cr3t","name":"ci","scope":["sp:scopes:all"],"owner":{"type":"IDENTITY","id":"i-1","name":"Me"},
				"created":"c","accessTokenValiditySeconds":43200,"expirationDate":"2030-01-01T00:00:00.000Z"}`))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	// The token of the caller is found in the first list.
	token, err := client.GetPersonalAccessToken(ctx, "p-1")
	if err != nil || token.Name != "mine" || len(requests) != 1 || requests[0] != "GET /v2026/personal-access-tokens?owner-id=me" {
		t.Fatalf("unexpected lookup %+v %v (%v)", token, requests, err)
	}
	// Other tokens are searched in the tenant-wide list.
	requests = nil
	token, err = client.GetPersonalAccessToken(ctx, "p-2")
	if err != nil || token.Name != "other" || len(requests) != 2 || requests[1] != "GET /v2026/personal-access-tokens" {
		t.Fatalf("unexpected fallback lookup %+v %v (%v)", token, requests, err)
	}
	if _, err := client.GetPersonalAccessToken(ctx, "missing"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if token, err := client.GetPersonalAccessTokenByName(ctx, "me", "mine"); err != nil || token.ID != "p-1" {
		t.Fatalf("unexpected lookup by name %+v (%v)", token, err)
	}

	var diags diag.Diagnostics
	plan := PersonalAccessTokenModel{
		ID: types.StringUnknown(), Name: types.StringValue("ci"), Scope: types.SetUnknown(types.StringType),
		AccessTokenValiditySeconds: types.Int64Unknown(), ExpirationDate: types.StringValue("2030-01-01T00:00:00Z"),
		UserAwareTokenNeverExpires: types.BoolUnknown(), Owner: types.ListUnknown(objectInfoObjectType),
		Managed: types.BoolUnknown(), Secret: types.StringUnknown(), Created: types.StringUnknown(),
	}
	created, err := client.CreatePersonalAccessToken(ctx, personalAccessTokenFromModel(ctx, plan, &diags))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotBody != `{"name":"ci","expirationDate":"2030-01-01T00:00:00Z"}` {
		t.Fatalf("unexpected create body %s", gotBody)
	}
	personalAccessTokenResolveUnknowns(ctx, &plan, created, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if plan.Secret.ValueString() != "s3cr3t" || plan.ID.ValueString() != "p-3" || plan.AccessTokenValiditySeconds.ValueInt64() != 43200 ||
		plan.UserAwareTokenNeverExpires.ValueBool() || plan.ExpirationDate.ValueString() != "2030-01-01T00:00:00Z" || len(plan.Owner.Elements()) != 1 {
		t.Fatalf("unexpected state after create %+v", plan)
	}

	if _, err := client.PatchPersonalAccessToken(ctx, "p-3", []jsonPatchOp{{Op: "replace", Path: "/name", Value: "ci2"}}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotContentType != "application/json-patch+json" || requests[len(requests)-1] != "PATCH /v2026/personal-access-tokens/p-3" {
		t.Fatalf("unexpected patch request %v %s", requests, gotContentType)
	}
	if err := client.DeletePersonalAccessToken(ctx, "p-3"); err != nil || requests[len(requests)-1] != "DELETE /v2026/personal-access-tokens/p-3" {
		t.Fatalf("unexpected delete request %v (%v)", requests, err)
	}
}

func TestPersonalAccessTokenPatchOpsClearExpiration(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := PersonalAccessTokenModel{
		Name: types.StringValue("ci"), Scope: oauthClientTestSet("sp:scopes:all"),
		ExpirationDate: types.StringValue("2030-01-01T00:00:00Z"), UserAwareTokenNeverExpires: types.BoolValue(false),
	}
	plan := state
	plan.ExpirationDate = types.StringNull()
	plan.UserAwareTokenNeverExpires = types.BoolValue(true)
	encoded, _ := json.Marshal(personalAccessTokenPatchOps(ctx, plan, state, &diags))
	want := `[{"op":"replace","path":"/expirationDate","value":null},{"op":"replace","path":"/userAwareTokenNeverExpires","value":true}]`
	if string(encoded) != want {
		t.Fatalf("unexpected patch ops %s", encoded)
	}

	plan = state
	plan.Scope = oauthClientTestSet("sp:search:read")
	encoded, _ = json.Marshal(personalAccessTokenPatchOps(ctx, plan, state, &diags))
	if string(encoded) != `[{"op":"replace","path":"/scope","value":["sp:search:read"]}]` {
		t.Fatalf("unexpected patch ops %s", encoded)
	}
}

func TestPersonalAccessTokenStateKeepsEquivalentDate(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	expiration := "2030-01-01T00:00:00.000Z"
	neverExpires := false
	data := PersonalAccessTokenModel{
		ExpirationDate: types.StringValue("2030-01-01T01:00:00+01:00"), Scope: oauthClientTestSet("sp:scopes:all"),
		AccessTokenValiditySeconds: types.Int64Null(), Secret: types.StringValue("kept"),
	}
	setPersonalAccessTokenState(ctx, &data, &PersonalAccessToken{
		ID: "p-1", Name: "ci", Scope: []string{"sp:scopes:all"}, ExpirationDate: &expiration, UserAwareTokenNeverExpires: &neverExpires,
		Owner: &PersonalAccessTokenOwner{Type: "IDENTITY", ID: "i-1", Name: "Me"}, Created: "c",
	}, &diags)
	if data.ExpirationDate.ValueString() != "2030-01-01T01:00:00+01:00" || data.Secret.ValueString() != "kept" || !data.AccessTokenValiditySeconds.IsNull() {
		t.Fatalf("unexpected state %+v", data)
	}
	later := "2031-01-01T00:00:00Z"
	setPersonalAccessTokenState(ctx, &data, &PersonalAccessToken{ID: "p-1", Name: "ci", ExpirationDate: &later}, &diags)
	if data.ExpirationDate.ValueString() != later {
		t.Fatalf("expected changed expiration to be refreshed, got %s", data.ExpirationDate)
	}
}
