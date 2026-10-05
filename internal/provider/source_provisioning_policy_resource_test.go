package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSourceProvisioningPolicyClient(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"name":"Account","usageType":"CREATE","fields":[{"name":"email","type":"string"}]}`))
		default:
			_, _ = w.Write([]byte(`{"name":"Account","description":"d","usageType":"CREATE","fields":[{"name":"email","type":"string","isRequired":false,"transform":{}}]}`))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()

	data := SourceProvisioningPolicyModel{
		SourceID: types.StringValue("src-1"), UsageType: types.StringValue("CREATE"), Name: types.StringValue("Account"),
		Description: types.StringNull(), FieldsJSON: types.StringNull(),
	}
	if _, err := client.CreateSourceProvisioningPolicy(ctx, "src-1", sourceProvisioningPolicyFromModel(data)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/sources/src-1/provisioning-policies" || gotBody != `{"name":"Account","description":"","usageType":"CREATE","fields":[]}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}

	data.FieldsJSON = types.StringValue(`[{"name":"email","type":"string"}]`)
	if _, err := client.UpdateSourceProvisioningPolicy(ctx, "src-1", "CREATE", sourceProvisioningPolicyFromModel(data)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2026/sources/src-1/provisioning-policies/CREATE" || gotBody != `{"name":"Account","description":"","usageType":"CREATE","fields":[{"name":"email","type":"string"}]}` {
		t.Fatalf("unexpected update request %s %s %s", gotMethod, gotPath, gotBody)
	}

	policy, err := client.GetSourceProvisioningPolicy(ctx, "src-1", "CREATE")
	if err != nil || gotMethod != http.MethodGet || policy.Description != "d" {
		t.Fatalf("unexpected get %s %+v (%v)", gotMethod, policy, err)
	}
	if err := client.DeleteSourceProvisioningPolicy(ctx, "src-1", "CREATE"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/sources/src-1/provisioning-policies/CREATE" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestSourceProvisioningPolicyState(t *testing.T) {
	configured := `[{"name": "email", "type": "string", "transform": {"type": "identityAttribute", "attributes": {"name": "email"}}}]`
	data := SourceProvisioningPolicyModel{
		ID: types.StringValue("src-1/CREATE"), SourceID: types.StringValue("src-1"), UsageType: types.StringValue("CREATE"),
		Name: types.StringValue("Account"), Description: types.StringNull(), FieldsJSON: types.StringValue(configured),
	}
	setSourceProvisioningPolicyState(&data, &SourceProvisioningPolicy{
		Name:      "Account",
		UsageType: "CREATE",
		Fields:    json.RawMessage(`[{"name":"email","type":"string","isRequired":false,"isMultiValued":false,"attributes":{},"transform":{"attributes":{"name":"email"},"type":"identityAttribute"}}]`),
	})
	if !data.Description.IsNull() {
		t.Errorf("expected unset description to stay null, got %s", data.Description)
	}
	if data.FieldsJSON.ValueString() != configured {
		t.Errorf("expected configured fields to be kept when only API defaults differ, got %s", data.FieldsJSON)
	}
	if data.ID.ValueString() != "src-1/CREATE" {
		t.Errorf("unexpected id %s", data.ID)
	}

	// Drift is detected and the API value is stored without the default keys.
	setSourceProvisioningPolicyState(&data, &SourceProvisioningPolicy{
		Name:   "Account",
		Fields: json.RawMessage(`[{"name":"mail","type":"string","isRequired":false,"transform":{}}]`),
	})
	if data.FieldsJSON.ValueString() != `[{"name":"mail","type":"string"}]` {
		t.Errorf("expected drift to be detected, got %s", data.FieldsJSON)
	}

	// Empty fields map to null unless an empty array was configured.
	if got := sourceProvisioningPolicyFieldsState(types.StringNull(), json.RawMessage(`[]`)); !got.IsNull() {
		t.Errorf("expected null, got %s", got)
	}
	if got := sourceProvisioningPolicyFieldsState(types.StringValue(`[]`), nil); got.ValueString() != `[]` {
		t.Errorf("expected empty array to be kept, got %s", got)
	}
}
