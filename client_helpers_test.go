package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestListAllPagesAndFindByName(t *testing.T) {
	var queries []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		if r.URL.Query().Get("offset") == "0" {
			items := make([]string, listPageSize)
			for i := range items {
				items[i] = fmt.Sprintf(`{"id":"%d","name":"n%d"}`, i, i)
			}
			_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
			return
		}
		_, _ = w.Write([]byte(`[{"id":"x","name":"wanted"}]`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)

	tag, err := client.GetTagByName(context.Background(), `wanted`)
	if err != nil || tag.ID != "x" {
		t.Fatalf("expected tag from the second page, got %+v (%v)", tag, err)
	}
	if len(queries) != 2 || !strings.Contains(queries[1], "offset=250") || !strings.Contains(queries[0], "filters=name+eq+%22wanted%22") {
		t.Fatalf("unexpected queries %v", queries)
	}
	if _, err := client.GetTagByName(context.Background(), "missing"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestPutMergedKeepsUnmanagedFields(t *testing.T) {
	var put map[string]interface{}
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"id":"1","name":"old","unmanaged":{"keep":true}}`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &put)
		_, _ = w.Write(body)
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)

	var out map[string]interface{}
	if err := client.putMerged(context.Background(), "/v2026/things/1", map[string]interface{}{"name": "new"}, &out); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if put["name"] != "new" || put["unmanaged"] == nil || put["id"] != "1" {
		t.Fatalf("unexpected PUT body %v", put)
	}
}

func TestPatchBuilder(t *testing.T) {
	var b patchBuilder
	b.replaceIfChanged(types.StringValue("a"), types.StringValue("a"), "/same", "a")
	b.replaceIfChanged(types.StringValue("b"), types.StringValue("a"), "/name", "b")
	b.replaceIfChanged(types.StringUnknown(), types.StringValue("a"), "/computed", nil)
	b.replaceOrRemoveIfChanged(types.StringNull(), types.StringValue("a"), "/description", nil)
	b.replaceOrRemoveIfChanged(types.StringValue("x"), types.StringNull(), "/owner", "x")
	body, _ := json.Marshal(b.ops)
	want := `[{"op":"replace","path":"/name","value":"b"},{"op":"remove","path":"/description"},{"op":"replace","path":"/owner","value":"x"}]`
	if string(body) != want {
		t.Fatalf("got %s, want %s", body, want)
	}
	// false and empty values are sent, only a nil value is omitted
	body, _ = json.Marshal([]jsonPatchOp{{Op: "replace", Path: "/enabled", Value: false}, {Op: "replace", Path: "/description", Value: ""}})
	if string(body) != `[{"op":"replace","path":"/enabled","value":false},{"op":"replace","path":"/description","value":""}]` {
		t.Fatalf("unexpected encoding %s", body)
	}
}

func TestOptionalStateHelpers(t *testing.T) {
	if !optionalStringState(types.StringNull(), "").IsNull() {
		t.Error("expected empty API string to keep null")
	}
	if optionalStringState(types.StringValue(""), "").ValueString() != "" || optionalStringState(types.StringNull(), "x").ValueString() != "x" {
		t.Error("unexpected string state")
	}
	f := false
	if !optionalBoolState(types.BoolNull(), &f).IsNull() || optionalBoolState(types.BoolValue(false), &f).IsNull() {
		t.Error("unexpected bool state")
	}
	var zero int64
	if !optionalInt64State(types.Int64Null(), &zero).IsNull() || optionalInt64State(types.Int64Value(0), nil).ValueInt64() != 0 {
		t.Error("unexpected int64 state")
	}
}
