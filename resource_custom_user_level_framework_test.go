package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func customUserLevelTestOwner(id, ownerType, name types.String) types.List {
	return types.ListValueMust(objectInfoObjectType, []attr.Value{types.ObjectValueMust(objectInfoObjectType.AttrTypes, map[string]attr.Value{
		"id": id, "type": ownerType, "name": name,
	})})
}

func TestCustomUserLevelClient(t *testing.T) {
	var requests []string
	var gotBody string
	status := "DRAFT"
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-SailPoint-Experimental") != "true" {
			t.Errorf("missing experimental header on %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, r.Method+" "+r.URL.Path+" "+r.Header.Get("Content-Type"))
		gotBody = string(body)
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case strings.HasSuffix(r.URL.Path, "/publish"):
			status = customUserLevelStatusActive
			_, _ = w.Write([]byte(`{"userLevelId":"u-1","publish":true,"status":"ACTIVE"}`))
		default:
			fmt.Fprintf(w, `{"id":"u-1","name":"IAM","description":"d","owner":{"type":"IDENTITY","id":"i-1","name":"John"},
				"rightSets":[{"id":"idn:parent"},{"id":"idn:child","parentId":"idn:parent"}],"status":%q,"created":"c","modified":"m"}`, status)
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := CustomUserLevelModel{
		ID: types.StringUnknown(), Name: types.StringValue("IAM"), Description: types.StringValue("d"),
		Owner:     customUserLevelTestOwner(types.StringValue("i-1"), types.StringUnknown(), types.StringUnknown()),
		RightSets: oauthClientTestSet("idn:parent"), Publish: types.BoolValue(true),
		AssignedRightSets: types.SetUnknown(types.StringType), Status: types.StringUnknown(),
		Created: types.StringUnknown(), Modified: types.StringUnknown(),
	}
	created, err := client.CreateCustomUserLevel(ctx, &CustomUserLevelRequest{
		Name: "IAM", Description: "d", Owner: customUserLevelOwnerValue(ctx, plan.Owner, &diags),
		RightSets: oauthClientStringSetValue(ctx, plan.RightSets, &diags),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotBody != `{"name":"IAM","description":"d","owner":{"type":"IDENTITY","id":"i-1"},"rightSets":["idn:parent"]}` {
		t.Fatalf("unexpected create body %s", gotBody)
	}
	published, err := customUserLevelPublishIfRequested(ctx, client, plan, created)
	if err != nil || published.Status != customUserLevelStatusActive {
		t.Fatalf("unexpected publish result %+v (%v)", published, err)
	}
	if requests[1] != "POST /v2026/authorization/custom-user-levels/u-1/publish " || requests[2] != "GET /v2026/authorization/custom-user-levels/u-1 " {
		t.Fatalf("unexpected requests %v", requests)
	}
	customUserLevelResolveComputed(ctx, &plan, published, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	var owners []OwnerModel
	plan.Owner.ElementsAs(ctx, &owners, false)
	if plan.ID.ValueString() != "u-1" || plan.Status.ValueString() != "ACTIVE" || len(plan.AssignedRightSets.Elements()) != 2 ||
		len(plan.RightSets.Elements()) != 1 || owners[0].Name.ValueString() != "John" || owners[0].Type.ValueString() != "IDENTITY" {
		t.Fatalf("unexpected state %+v", plan)
	}

	// An active user level is not published again.
	requests = nil
	if _, err := customUserLevelPublishIfRequested(ctx, client, plan, published); err != nil || len(requests) != 0 {
		t.Fatalf("unexpected publish requests %v (%v)", requests, err)
	}
	if _, err := client.PatchCustomUserLevel(ctx, "u-1", []jsonPatchOp{{Op: "replace", Path: "/name", Value: "IAM2"}}); err != nil ||
		requests[0] != "PATCH /v2026/authorization/custom-user-levels/u-1 application/json-patch+json" {
		t.Fatalf("unexpected patch request %v (%v)", requests, err)
	}
	if err := client.DeleteCustomUserLevel(ctx, "u-1"); err != nil || requests[1] != "DELETE /v2026/authorization/custom-user-levels/u-1 " {
		t.Fatalf("unexpected delete request %v (%v)", requests, err)
	}
}

func TestCustomUserLevelRightSetsStateIgnoresInheritedChildren(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	parent := "idn:parent"
	assigned := []CustomUserLevelRightSet{{ID: "idn:parent"}, {ID: "idn:child", ParentID: &parent}}

	prior := oauthClientTestSet("idn:parent")
	if state := customUserLevelRightSetsState(ctx, prior, assigned, &diags); !state.Equal(prior) {
		t.Fatalf("expected configured right sets to be kept, got %s", state)
	}
	// A right set that is neither configured nor a child of a configured one is drift.
	unrelated := append(assigned, CustomUserLevelRightSet{ID: "idn:other"})
	if state := customUserLevelRightSetsState(ctx, prior, unrelated, &diags); len(state.Elements()) != 3 {
		t.Fatalf("expected drift to be detected, got %s", state)
	}
	// A configured right set that is missing is drift.
	if state := customUserLevelRightSetsState(ctx, oauthClientTestSet("idn:parent", "idn:missing"), assigned, &diags); len(state.Elements()) != 2 {
		t.Fatalf("expected drift to be detected, got %s", state)
	}
	if state := customUserLevelRightSetsState(ctx, types.SetNull(types.StringType), nil, &diags); !state.IsNull() {
		t.Fatalf("expected unset right sets to stay null, got %s", state)
	}
	// A configured parent that is only returned through its assigned children is covered.
	childrenOnly := []CustomUserLevelRightSet{{ID: "idn:child", ParentID: &parent}, {ID: "idn:child2", ParentID: &parent}}
	if state := customUserLevelRightSetsState(ctx, prior, childrenOnly, &diags); !state.Equal(prior) {
		t.Fatalf("expected a parent returned through its children to be kept, got %s", state)
	}
}

func TestCustomUserLevelReadKeepsConfiguredOwnerName(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	ownerList := func(id, name string) types.List {
		return types.ListValueMust(objectInfoObjectType, []attr.Value{types.ObjectValueMust(objectInfoObjectType.AttrTypes, map[string]attr.Value{
			"id": types.StringValue(id), "type": types.StringValue("IDENTITY"), "name": types.StringValue(name),
		})})
	}
	level := &CustomUserLevel{ID: "u-1", Name: "IAM", Owner: &CustomUserLevelOwner{Type: "IDENTITY", ID: "i-1", Name: "John Doe"}}
	data := CustomUserLevelModel{Owner: ownerList("i-1", "john.doe"), RightSets: types.SetNull(types.StringType)}
	setCustomUserLevelState(ctx, &data, level, &diags)
	if !data.Owner.Equal(ownerList("i-1", "john.doe")) {
		t.Errorf("configured owner name of an unchanged owner must be kept, got %s", data.Owner)
	}
	data.Owner = ownerList("i-2", "jane.doe")
	setCustomUserLevelState(ctx, &data, level, &diags)
	if !data.Owner.Equal(ownerList("i-1", "John Doe")) {
		t.Errorf("a changed owner must be read from the API, got %s", data.Owner)
	}
	data.Owner = types.ListNull(objectInfoObjectType)
	setCustomUserLevelState(ctx, &data, level, &diags)
	if !data.Owner.Equal(ownerList("i-1", "John Doe")) {
		t.Errorf("an imported owner must be read from the API, got %s", data.Owner)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestCustomUserLevelPatchOps(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := CustomUserLevelModel{
		Name: types.StringValue("IAM"), Description: types.StringValue("d"),
		Owner:     customUserLevelTestOwner(types.StringValue("i-1"), types.StringValue("IDENTITY"), types.StringValue("John")),
		RightSets: oauthClientTestSet("idn:parent"),
	}
	plan := state
	plan.Owner = customUserLevelTestOwner(types.StringValue("i-1"), types.StringUnknown(), types.StringUnknown())
	if ops := customUserLevelPatchOps(ctx, plan, state, &diags); len(ops) != 0 {
		t.Fatalf("expected no ops for unknown computed owner attributes, got %+v", ops)
	}
	plan.Owner = customUserLevelTestOwner(types.StringValue("i-2"), types.StringUnknown(), types.StringUnknown())
	plan.RightSets = types.SetNull(types.StringType)
	encoded, _ := json.Marshal(customUserLevelPatchOps(ctx, plan, state, &diags))
	want := `[{"op":"replace","path":"/owner","value":{"type":"IDENTITY","id":"i-2"}},{"op":"replace","path":"/rightSets","value":[]}]`
	if string(encoded) != want {
		t.Fatalf("unexpected patch ops %s", encoded)
	}
}

func TestAuthorizationRightSetsList(t *testing.T) {
	var queries []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-SailPoint-Experimental") != "true" || r.URL.Path != "/v2026/authorization/authorization-assignable-right-sets" {
			t.Errorf("unexpected request %s %v", r.URL.Path, r.Header)
		}
		queries = append(queries, r.URL.RawQuery)
		if r.URL.Query().Get("offset") != "0" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		items := make([]string, 0, authorizationRightSetsPageSize)
		items = append(items, `{"id":"idn:parent","name":"Parent","category":"identity","nestedConfig":{"ancestorId":"idn:parent","depth":0,"parentId":null,"childrenIds":["idn:child"]},
			"children":[{"id":"idn:child","name":"Child","description":"c","category":"identity","nestedConfig":{"ancestorId":"idn:parent","depth":1,"parentId":"idn:parent","childrenIds":[]},"children":[]}]}`)
		for i := 1; i < authorizationRightSetsPageSize; i++ {
			items = append(items, fmt.Sprintf(`{"id":"idn:r-%d","name":"R","category":"identity"}`, i))
		}
		_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	rightSets, err := client.ListAuthorizationRightSets(ctx, "identity")
	if err != nil || len(rightSets) != authorizationRightSetsPageSize {
		t.Fatalf("unexpected result %d (%v)", len(rightSets), err)
	}
	if len(queries) != 2 || queries[0] != "filters=category+eq+%22identity%22&limit=50&offset=0" {
		t.Fatalf("unexpected queries %v", queries)
	}

	var diags diag.Diagnostics
	list := authorizationRightSetsState(ctx, rightSets, &diags)
	var models []AuthorizationRightSetModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) != authorizationRightSetsPageSize+1 {
		t.Fatalf("unexpected flattened right sets %d (%v)", len(models), diags)
	}
	if models[0].ID.ValueString() != "idn:parent" || !models[0].ParentID.IsNull() || len(models[0].ChildrenIDs.Elements()) != 1 ||
		models[1].ID.ValueString() != "idn:child" || models[1].ParentID.ValueString() != "idn:parent" || models[1].Depth.ValueInt64() != 1 ||
		!models[2].AncestorID.IsNull() || !models[0].Description.IsNull() {
		t.Fatalf("unexpected flattened right sets %+v", models[:3])
	}
}
