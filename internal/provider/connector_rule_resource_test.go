package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func connectorRuleTestSourceCode(t *testing.T, version, script string) types.List {
	t.Helper()
	list, d := types.ListValueFrom(context.Background(), connectorRuleSourceCodeObjectType, []ConnectorRuleSourceCodeModel{{
		Version: types.StringValue(version), Script: types.StringValue(script),
	}})
	if d.HasError() {
		t.Fatalf("unexpected diagnostics: %v", d)
	}
	return list
}

func TestConnectorRuleClient(t *testing.T) {
	const ruleJSON = `{"id":"rule-1","name":"Before Create","type":"ConnectorBeforeCreate","sourceCode":{"version":"1.0","script":"return plan;"},"attributes":{},"created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z"}`
	var gotMethod, gotPath, gotBody string
	var queries []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v2026/connector-rules":
			queries = append(queries, r.URL.RawQuery)
			if r.URL.Query().Get("offset") == "0" {
				// A full page of 50 rules, so a second page is requested.
				rules := make([]string, 0, 50)
				for i := 0; i < 49; i++ {
					rules = append(rules, fmt.Sprintf(`{"id":"r-%d","name":"rule %d","type":"BuildMap","sourceCode":{"version":"1.0","script":"x"}}`, i, i))
				}
				rules = append(rules, ruleJSON)
				_, _ = w.Write([]byte("[" + strings.Join(rules, ",") + "]"))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(ruleJSON))
		default:
			_, _ = w.Write([]byte(ruleJSON))
		}
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	data := ConnectorRuleModel{
		Name: types.StringValue("Before Create"), Description: types.StringNull(), Type: types.StringValue("ConnectorBeforeCreate"),
		SignatureJSON: types.StringUnknown(), SourceCode: connectorRuleTestSourceCode(t, "1.0", "return plan;"), AttributesJSON: types.StringNull(),
	}
	rule := connectorRuleFromModel(ctx, data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if _, err := client.CreateConnectorRule(ctx, rule); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2026/connector-rules" ||
		gotBody != `{"name":"Before Create","description":null,"type":"ConnectorBeforeCreate","sourceCode":{"version":"1.0","script":"return plan;"},"attributes":null}` {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}

	// The update request contains the ID and an unmanaged signature from the prior state.
	data.ID = types.StringValue("rule-1")
	data.SignatureJSON = types.StringValue(`{"input":[]}`)
	data.AttributesJSON = types.StringValue(`{"k":"v"}`)
	rule = connectorRuleFromModel(ctx, data, &diags)
	rule.ID = "rule-1"
	if _, err := client.UpdateConnectorRule(ctx, "rule-1", rule); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2026/connector-rules/rule-1" ||
		gotBody != `{"id":"rule-1","name":"Before Create","description":null,"type":"ConnectorBeforeCreate","signature":{"input":[]},"sourceCode":{"version":"1.0","script":"return plan;"},"attributes":{"k":"v"}}` {
		t.Fatalf("unexpected update request %s %s %s", gotMethod, gotPath, gotBody)
	}

	found, err := client.GetConnectorRuleByName(ctx, "Before Create")
	if err != nil || found.ID != "rule-1" {
		t.Fatalf("unexpected lookup result %+v (%v)", found, err)
	}
	if len(queries) != 2 || queries[0] != "limit=50&offset=0" || queries[1] != "limit=50&offset=50" {
		t.Fatalf("expected two pages of 50, got %v", queries)
	}
	if _, err := client.GetConnectorRuleByName(ctx, "missing"); !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
	if err := client.DeleteConnectorRule(ctx, "rule-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/connector-rules/rule-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}

func TestConnectorRuleState(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	data := ConnectorRuleModel{
		ID: types.StringValue("rule-1"), Name: types.StringValue("r"), Description: types.StringNull(), Type: types.StringValue("BuildMap"),
		SignatureJSON: types.StringValue(`{"input": [{"name": "col", "type": "Map"}]}`), SourceCode: connectorRuleTestSourceCode(t, "1.0", "return map;\n"),
		AttributesJSON: types.StringNull(),
	}
	prior := data
	setConnectorRuleState(ctx, &data, &ConnectorRule{
		ID: "rule-1", Name: "r", Type: "BuildMap", Signature: json.RawMessage(`{"input":[{"type":"Map","name":"col"}]}`),
		SourceCode: ConnectorRuleSourceCode{Version: "1.0", Script: "return map;"}, Attributes: map[string]interface{}{},
		Created: "2026-01-01T00:00:00Z",
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !data.Description.IsNull() || !data.AttributesJSON.IsNull() || !data.SignatureJSON.Equal(prior.SignatureJSON) || !data.SourceCode.Equal(prior.SourceCode) {
		t.Fatalf("expected equivalent values to be kept, got %+v", data)
	}
	if !data.Modified.IsNull() || data.Created.ValueString() != "2026-01-01T00:00:00Z" {
		t.Fatalf("unexpected dates %s %s", data.Created, data.Modified)
	}

	// A changed script is detected.
	setConnectorRuleState(ctx, &data, &ConnectorRule{ID: "rule-1", Name: "r", Type: "BuildMap", SourceCode: ConnectorRuleSourceCode{Version: "1.0", Script: "return null;"}}, &diags)
	if data.SourceCode.Equal(prior.SourceCode) || !data.SignatureJSON.IsNull() {
		t.Fatalf("expected drift to be detected, got %+v", data)
	}

	// Create resolves unknown values and keeps planned ones.
	planned := prior
	planned.ID, planned.SignatureJSON, planned.Created, planned.Modified = types.StringUnknown(), types.StringUnknown(), types.StringUnknown(), types.StringUnknown()
	applyConnectorRuleResponse(&planned, &ConnectorRule{ID: "rule-2", Signature: json.RawMessage(`{"input":[]}`), Created: "c", Modified: "m"})
	if planned.ID.ValueString() != "rule-2" || planned.SignatureJSON.ValueString() != `{"input":[]}` || planned.Created.ValueString() != "c" || planned.Modified.ValueString() != "m" {
		t.Fatalf("unexpected state after create %+v", planned)
	}
}
