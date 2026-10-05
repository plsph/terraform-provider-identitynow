package provider

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const privilegeCriteriaTestGroups = `[{"operator":"OR","criteriaItems":[{"targetType":"group","property":"displayName","operator":"CONTAINS","values":["admin"]}]}]`

func TestPrivilegeCriteriaClientAndState(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, string(body)
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"id":"pc-1","sourceId":"s-1","type":"CUSTOM","operator":"AND","privilegeLevel":"HIGH",
			"groups":[{"operator":"OR","criteriaItems":[{"targetType":"group","property":"displayName","operator":"CONTAINS","values":["admin"],"ignoreCase":false}]}]}`))
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	ctx := context.Background()
	var diags diag.Diagnostics

	data := PrivilegeCriteriaModel{
		ID:             types.StringUnknown(),
		SourceID:       types.StringValue("s-1"),
		Type:           types.StringUnknown(),
		Operator:       types.StringValue("AND"),
		GroupsJSON:     types.StringValue(privilegeCriteriaTestGroups),
		PrivilegeLevel: types.StringValue("HIGH"),
	}
	if _, err := client.CreatePrivilegeCriteria(ctx, privilegeCriteriaFromModel(data, &diags)); err != nil || diags.HasError() {
		t.Fatalf("unexpected error: %v %v", err, diags)
	}
	wantCreate := `{"sourceId":"s-1","type":"CUSTOM","operator":"AND","groups":[{"criteriaItems":[{"operator":"CONTAINS","property":"displayName","targetType":"group","values":["admin"]}],"operator":"OR"}],"privilegeLevel":"HIGH"}`
	if gotMethod != http.MethodPost || gotPath != "/v2026/criteria/privilege" || gotBody != wantCreate {
		t.Fatalf("unexpected create request %s %s %s", gotMethod, gotPath, gotBody)
	}

	data.ID, data.Type = types.StringValue("pc-1"), types.StringValue("CUSTOM")
	if _, err := client.UpdatePrivilegeCriteria(ctx, "pc-1", privilegeCriteriaFromModel(data, &diags)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2026/criteria/privilege/pc-1" || gotBody != `{"id":"pc-1",`+wantCreate[1:] {
		t.Fatalf("unexpected update request %s %s %s", gotMethod, gotPath, gotBody)
	}

	read, err := client.GetPrivilegeCriteria(ctx, "pc-1")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	prior := data.GroupsJSON
	setPrivilegeCriteriaState(&data, read)
	if !data.GroupsJSON.Equal(prior) {
		t.Fatalf("expected groups with the default ignoreCase to keep the configured value, got %s", data.GroupsJSON)
	}
	read.Groups = []interface{}{}
	setPrivilegeCriteriaState(&data, read)
	if !data.GroupsJSON.IsNull() {
		t.Fatalf("expected removed groups to be reported as drift, got %s", data.GroupsJSON)
	}

	if err := client.DeletePrivilegeCriteria(ctx, "pc-1"); err != nil || gotMethod != http.MethodDelete || gotPath != "/v2026/criteria/privilege/pc-1" {
		t.Fatalf("unexpected delete request %s %s (%v)", gotMethod, gotPath, err)
	}
}
