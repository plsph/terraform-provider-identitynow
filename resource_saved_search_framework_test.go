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

const savedSearchTestResponse = `{"id":"ss-1","owner":{"type":"IDENTITY","id":"o-1"},"ownerId":"o-1","public":false,"name":"Disabled accounts",
"description":null,"created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z","indices":["identities"],
"columns":{"identity":[{"field":"displayName","header":"Display Name"}]},"query":"@accounts(disabled:true)","fields":[],
"orderBy":null,"sort":["displayName"],"filters":{"source.name":{"type":"TERMS","terms":["HR"],"exclude":false,"range":null}}}`

func savedSearchTestModel(t *testing.T) SavedSearchModel {
	ctx := context.Background()
	indices, diags := types.ListValueFrom(ctx, types.StringType, []string{"identities"})
	sort, d := types.ListValueFrom(ctx, types.StringType, []string{"displayName"})
	diags.Append(d...)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return SavedSearchModel{
		ID:          types.StringValue("ss-1"),
		Name:        types.StringValue("Disabled accounts"),
		Description: types.StringNull(),
		Public:      types.BoolValue(false),
		Indices:     indices,
		Query:       types.StringValue("@accounts(disabled:true)"),
		Fields:      types.ListNull(types.StringType),
		OrderBy:     types.MapNull(savedSearchOrderByType),
		Sort:        sort,
		FiltersJSON: types.StringValue(`{"source.name":{"type":"TERMS","terms":["HR"]}}`),
		ColumnsJSON: types.StringValue(`{"identity":[{"field":"displayName","header":"Display Name"}]}`),
		OwnerID:     types.StringValue("o-1"),
		Created:     types.StringValue("2026-01-01T00:00:00Z"),
		Modified:    types.StringValue("2026-01-02T00:00:00Z"),
	}
}

func TestSavedSearchClient(t *testing.T) {
	var gotMethods []string
	var putBody, postBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethods = append(gotMethods, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodPost:
			postBody = string(body)
		case http.MethodPut:
			putBody = string(body)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(savedSearchTestResponse))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := savedSearchTestModel(t)
	plan.ID, plan.Public, plan.OwnerID, plan.Created, plan.Modified = types.StringUnknown(), types.BoolUnknown(), types.StringUnknown(), types.StringUnknown(), types.StringUnknown()
	created, err := client.CreateSavedSearch(ctx, savedSearchFromModel(ctx, plan, &diags))
	if err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	if postBody != `{"name":"Disabled accounts","indices":["identities"],"columns":{"identity":[{"field":"displayName","header":"Display Name"}]},"query":"@accounts(disabled:true)","sort":["displayName"],"filters":{"source.name":{"terms":["HR"],"type":"TERMS"}}}` {
		t.Fatalf("unexpected create body %s", postBody)
	}
	setSavedSearchComputed(&plan, created)
	if plan.ID.ValueString() != "ss-1" || plan.OwnerID.ValueString() != "o-1" || plan.Public.IsUnknown() || !plan.Fields.IsNull() {
		t.Fatalf("unexpected state after create %+v", plan)
	}

	plan.Sort = types.ListNull(types.StringType)
	if _, err := client.UpdateSavedSearch(ctx, "ss-1", savedSearchManagedFields(ctx, plan, &diags)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	var sent map[string]interface{}
	_ = json.Unmarshal([]byte(putBody), &sent)
	if owner, ok := sent["owner"].(map[string]interface{}); !ok || owner["id"] != "o-1" {
		t.Fatalf("expected the owner to be kept, got %s", putBody)
	}
	if value, ok := sent["sort"]; !ok || value != nil {
		t.Fatalf("expected removed sort to be cleared, got %s", putBody)
	}
	if err := client.DeleteSavedSearch(ctx, "ss-1"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := `["POST /v2026/saved-searches","GET /v2026/saved-searches/ss-1","PUT /v2026/saved-searches/ss-1","DELETE /v2026/saved-searches/ss-1"]`
	if encoded, _ := json.Marshal(gotMethods); string(encoded) != want {
		t.Fatalf("unexpected requests %s", encoded)
	}
}

func TestSavedSearchReadState(t *testing.T) {
	ctx := context.Background()
	var search SavedSearch
	if err := json.Unmarshal([]byte(savedSearchTestResponse), &search); err != nil {
		t.Fatal(err)
	}
	var diags diag.Diagnostics
	data := savedSearchTestModel(t)
	prior := data
	setSavedSearchState(ctx, &data, &search, &diags)
	if diags.HasError() || !data.FiltersJSON.Equal(prior.FiltersJSON) || !data.ColumnsJSON.Equal(prior.ColumnsJSON) ||
		!data.Fields.IsNull() || !data.OrderBy.IsNull() || !data.Description.IsNull() || !data.Sort.Equal(prior.Sort) {
		t.Fatalf("expected refreshed state to match the configuration, got %+v (%v)", data, diags)
	}

	search.OrderBy = map[string][]string{"identity": {"lastName"}}
	setSavedSearchState(ctx, &data, &search, &diags)
	if data.OrderBy.IsNull() || len(data.OrderBy.Elements()) != 1 {
		t.Fatalf("expected order_by drift to be reported, got %s", data.OrderBy)
	}
}
