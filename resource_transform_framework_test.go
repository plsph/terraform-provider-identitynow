package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTransformClient(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"id":"t-1","name":"lower","type":"lower","attributes":{"input":{"type":"static"}},"internal":false}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	transform, err := transformFromModel(TransformResourceModel{
		Name: types.StringValue("lower"), Type: types.StringValue("lower"), AttributesJSON: types.StringNull(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if _, err := client.CreateTransform(ctx, transform); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/transforms" || gotBody != `{"name":"lower","type":"lower","attributes":{}}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}
	if _, err := client.UpdateTransform(ctx, "t-1", transform); err != nil || gotMethod != http.MethodPut || gotPath != "/v2026/transforms/t-1" {
		t.Fatalf("unexpected update request %s %s (%v)", gotMethod, gotPath, err)
	}
	if err := client.DeleteTransform(ctx, "t-1"); err != nil || gotMethod != http.MethodDelete {
		t.Fatalf("unexpected delete request %s (%v)", gotMethod, err)
	}
}

func TestTransformStateKeepsEquivalentJSON(t *testing.T) {
	var attributes map[string]interface{}
	_ = json.Unmarshal([]byte(`{"table":{"US":"United States"},"input":{"type":"static"}}`), &attributes)
	data := TransformResourceModel{AttributesJSON: types.StringValue(`{"input": {"type": "static"}, "table": {"US": "United States"}}`)}
	prior := data.AttributesJSON
	setTransformState(&data, &Transform{ID: "t-1", Name: "lookup", Type: "lookup", Attributes: attributes})
	if !data.AttributesJSON.Equal(prior) {
		t.Fatalf("expected equivalent JSON to be kept, got %s", data.AttributesJSON)
	}

	empty := TransformResourceModel{AttributesJSON: types.StringNull()}
	setTransformState(&empty, &Transform{ID: "t-2", Name: "uuid", Type: "uuid", Attributes: map[string]interface{}{}})
	if !empty.AttributesJSON.IsNull() {
		t.Fatalf("expected empty attributes to stay null, got %s", empty.AttributesJSON)
	}
}
