package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestFormDefinitionClient(t *testing.T) {
	const formJSON = `{"id":"form-1","name":"my form","owner":{"type":"IDENTITY","id":"owner-1","name":"Owner"},"formElements":[{"id":"1","elementType":"SECTION"}],"created":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z"}`

	var gotMethod, gotPath, gotQuery, gotBody string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotQuery, gotBody = r.Method, r.URL.Path, r.URL.Query().Get("filters"), string(body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v2026/form-definitions/missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"detailCode":"404 Not found","messages":[{"text":"not found"}]}`))
		case r.Method == "DELETE":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == "GET" && r.URL.Path == "/v2026/form-definitions":
			_, _ = w.Write([]byte(`{"count":2,"results":[{"id":"form-0","name":"my form 2"},` + formJSON + `]}`))
		case r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(formJSON))
		default:
			_, _ = w.Write([]byte(formJSON))
		}
	})

	ctx := context.Background()
	client := NewClient(ctx, server.URL, "id", "secret", 100)

	t.Run("create sends form elements verbatim", func(t *testing.T) {
		form, err := client.CreateFormDefinition(ctx, &FormDefinition{
			Name:         "my form",
			Owner:        &FormOwner{Type: "IDENTITY", ID: "owner-1"},
			FormElements: json.RawMessage(`[{"id":"1","elementType":"SECTION"}]`),
		})
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if gotMethod != "POST" || gotPath != "/v2026/form-definitions" {
			t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
		}
		want := `{"name":"my form","owner":{"type":"IDENTITY","id":"owner-1"},"formElements":[{"id":"1","elementType":"SECTION"}]}`
		if gotBody != want {
			t.Fatalf("unexpected body:\n got %s\nwant %s", gotBody, want)
		}
		if form.ID != "form-1" || form.Created == "" {
			t.Fatalf("unexpected response %+v", form)
		}
	})

	t.Run("get by name filters and matches exactly", func(t *testing.T) {
		form, err := client.GetFormDefinitionByName(ctx, `my "form"`)
		if !isNotFound(err) {
			t.Fatalf("expected not found error, got %v (%+v)", err, form)
		}
		if gotQuery != `name eq "my \"form\""` {
			t.Fatalf("unexpected filter %q", gotQuery)
		}

		form, err = client.GetFormDefinitionByName(ctx, "my form")
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if form.ID != "form-1" {
			t.Fatalf("expected exact match form-1, got %s", form.ID)
		}
	})

	t.Run("update sends json patch", func(t *testing.T) {
		_, err := client.UpdateFormDefinition(ctx, "form-1", []*UpdateFormDefinition{{Op: "add", Path: "/name", Value: "new"}})
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if gotMethod != "PATCH" || gotPath != "/v2026/form-definitions/form-1" || gotBody != `[{"op":"add","path":"/name","value":"new"}]` {
			t.Fatalf("unexpected request %s %s %s", gotMethod, gotPath, gotBody)
		}
	})

	t.Run("delete handles no content", func(t *testing.T) {
		if err := client.DeleteFormDefinition(ctx, "form-1"); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if gotMethod != "DELETE" || gotPath != "/v2026/form-definitions/form-1" {
			t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
		}
	})

	t.Run("get missing returns not found", func(t *testing.T) {
		_, err := client.GetFormDefinition(ctx, "missing")
		if !isNotFound(err) {
			t.Fatalf("expected not found error, got %v", err)
		}
	})
}

func TestFormDefinitionPatchesUseReplaceAndEmptyDefaults(t *testing.T) {
	patches := formDefinitionPatches(&FormDefinition{Name: "form", Owner: &FormOwner{Type: "IDENTITY", ID: "owner-1"}})
	body, err := json.Marshal(patches)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	want := `[{"op":"replace","path":"/name","value":"form"},` +
		`{"op":"replace","path":"/description","value":""},` +
		`{"op":"replace","path":"/owner","value":{"type":"IDENTITY","id":"owner-1"}},` +
		`{"op":"replace","path":"/formInput","value":[]},` +
		`{"op":"replace","path":"/formElements","value":[]},` +
		`{"op":"replace","path":"/formConditions","value":[]}]`
	if string(body) != want {
		t.Fatalf("unexpected patch:\n got %s\nwant %s", body, want)
	}
}

func TestFormInputIDsAssignedByAPI(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	planned, d := types.ListValueFrom(ctx, formInputObjectType, []FormDefinitionInputModel{
		{ID: types.StringValue("existing-id"), Type: types.StringValue("STRING"), Label: types.StringValue("first"), Description: types.StringNull()},
		{ID: types.StringUnknown(), Type: types.StringValue("ARRAY"), Label: types.StringValue("second"), Description: types.StringValue("")},
	})
	diags.Append(d...)

	var model FormDefinitionResourceModel
	model.Owner = types.ListNull(formOwnerObjectType)
	model.FormInput = planned
	form := formDefinitionFromModel(ctx, model, &diags)
	body, err := json.Marshal(form.FormInput)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if want := `[{"id":"existing-id","type":"STRING","label":"first"},{"type":"ARRAY","label":"second"}]`; string(body) != want {
		t.Fatalf("unexpected form input request:\n got %s\nwant %s", body, want)
	}

	got := formInputWithIDs(ctx, planned, []*FormDefinitionInput{
		{ID: "existing-id", Type: "STRING", Label: "first"},
		{ID: "new-id", Type: "ARRAY", Label: "second"},
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	var inputs []FormDefinitionInputModel
	diags.Append(got.ElementsAs(ctx, &inputs, false)...)
	if len(inputs) != 2 || inputs[0].ID.ValueString() != "existing-id" || inputs[1].ID.ValueString() != "new-id" {
		t.Fatalf("unexpected ids %+v", inputs)
	}
	if inputs[1].Description.IsNull() || inputs[1].Description.ValueString() != "" {
		t.Errorf("expected planned description to be kept, got %s", inputs[1].Description)
	}
}

func TestFormDefinitionJSONState(t *testing.T) {
	tests := []struct {
		name  string
		prior types.String
		raw   string
		want  types.String
	}{
		{"keeps prior when semantically equal", types.StringValue(`[ {"b":1, "a":"x"} ]`), `[{"a":"x","b":1}]`, types.StringValue(`[ {"b":1, "a":"x"} ]`)},
		{"normalizes when different", types.StringValue(`[{"a":"x"}]`), `[{"b":2,"a":"y"}]`, types.StringValue(`[{"a":"y","b":2}]`)},
		{"missing value is null", types.StringNull(), ``, types.StringNull()},
		{"empty array is null", types.StringNull(), `[]`, types.StringNull()},
		{"keeps configured empty array", types.StringValue(`[]`), `null`, types.StringValue(`[]`)},
		{"import normalizes", types.StringNull(), `[{"z":true,"a":null}]`, types.StringValue(`[{"a":null,"z":true}]`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formDefinitionJSONState(tt.prior, json.RawMessage(tt.raw))
			if !got.Equal(tt.want) {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestJSONArrayStringValidator(t *testing.T) {
	for value, wantErr := range map[string]bool{
		`[]`:             false,
		`[{"id":"1"}]`:   false,
		`{"id":"1"}`:     true,
		`null`:           true,
		`[{"id":`:        true,
		`"not an array"`: true,
	} {
		req := validator.StringRequest{Path: path.Root("form_elements_json"), ConfigValue: types.StringValue(value)}
		resp := &validator.StringResponse{}
		jsonArrayStringValidator{}.ValidateString(context.Background(), req, resp)
		if resp.Diagnostics.HasError() != wantErr {
			t.Errorf("value %s: expected error %v, got %v", value, wantErr, resp.Diagnostics)
		}
	}
}

func TestSetFormDefinitionStatePreservesUnsetOptionalValues(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	prior, d := types.ListValueFrom(ctx, formOwnerObjectType, []OwnerModel{{
		ID: types.StringValue("owner-1"), Type: types.StringValue("IDENTITY"), Name: types.StringNull(),
	}})
	diags.Append(d...)
	emptyInputs, d := types.ListValue(formInputObjectType, []attr.Value{})
	diags.Append(d...)

	data := FormDefinitionResourceModel{
		Description:        types.StringNull(),
		Owner:              prior,
		FormInput:          emptyInputs,
		FormElementsJSON:   types.StringValue(`[{"elementType":"SECTION","id":"1"}]`),
		FormConditionsJSON: types.StringNull(),
	}
	setFormDefinitionState(ctx, &data, &FormDefinition{
		ID:           "form-1",
		Name:         "my form",
		Owner:        &FormOwner{Type: "IDENTITY", ID: "owner-1", Name: "Owner Name"},
		FormElements: json.RawMessage(`[{"id":"1","elementType":"SECTION"}]`),
		UsedBy:       []*FormUsedBy{{Type: "WORKFLOW", ID: "wf-1", Name: "My Workflow"}},
		Created:      "2026-01-01T00:00:00Z",
		Modified:     "2026-01-02T00:00:00Z",
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !data.Description.IsNull() {
		t.Errorf("expected description to stay null, got %s", data.Description)
	}
	var owners []OwnerModel
	diags.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if len(owners) != 1 || owners[0].ID.ValueString() != "owner-1" || !owners[0].Name.IsNull() {
		t.Errorf("expected owner name to stay null, got %+v", owners)
	}
	if data.FormInput.IsNull() || len(data.FormInput.Elements()) != 0 {
		t.Errorf("expected empty form input list to be kept, got %s", data.FormInput)
	}
	if data.FormElementsJSON.ValueString() != `[{"elementType":"SECTION","id":"1"}]` {
		t.Errorf("expected configured form elements JSON to be kept, got %s", data.FormElementsJSON)
	}
	if !data.FormConditionsJSON.IsNull() {
		t.Errorf("expected form conditions to be null, got %s", data.FormConditionsJSON)
	}
	var usedBy []FormUsedByModel
	diags.Append(data.UsedBy.ElementsAs(ctx, &usedBy, false)...)
	if len(usedBy) != 1 || usedBy[0].Type.ValueString() != "WORKFLOW" {
		t.Errorf("unexpected used_by %+v", usedBy)
	}

	// On import there is no prior state, so the owner name comes from the API.
	imported := FormDefinitionResourceModel{Owner: types.ListNull(formOwnerObjectType), FormInput: types.ListNull(formInputObjectType)}
	setFormDefinitionState(ctx, &imported, &FormDefinition{
		Owner:     &FormOwner{Type: "IDENTITY", ID: "owner-1", Name: "Owner Name"},
		FormInput: []*FormDefinitionInput{{ID: "input-1", Type: "STRING", Label: "input1"}},
	}, &diags)
	owners = nil
	diags.Append(imported.Owner.ElementsAs(ctx, &owners, false)...)
	if len(owners) != 1 || owners[0].Name.ValueString() != "Owner Name" {
		t.Errorf("expected imported owner name, got %+v", owners)
	}
	var inputs []FormDefinitionInputModel
	diags.Append(imported.FormInput.ElementsAs(ctx, &inputs, false)...)
	if len(inputs) != 1 || inputs[0].Label.ValueString() != "input1" || !inputs[0].Description.IsNull() {
		t.Errorf("unexpected form inputs %+v", inputs)
	}
}

func TestFormDefinitionSchemas(t *testing.T) {
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("schema diagnostic: %s: %s", d.Summary, d.Detail)
	}
	if _, ok := resp.ResourceSchemas["identitynow_form_definition"]; !ok {
		t.Error("identitynow_form_definition resource is not registered")
	}
	if _, ok := resp.DataSourceSchemas["identitynow_form_definition"]; !ok {
		t.Error("identitynow_form_definition data source is not registered")
	}
}

func TestFormInputIDsFromStateMatchesByLabelAndType(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	prior, d := types.ListValueFrom(ctx, formInputObjectType, []FormDefinitionInputModel{
		{ID: types.StringValue("id-a"), Type: types.StringValue("STRING"), Label: types.StringValue("a"), Description: types.StringNull()},
		{ID: types.StringValue("id-b"), Type: types.StringValue("STRING"), Label: types.StringValue("b"), Description: types.StringNull()},
	})
	diags.Append(d...)
	// A new input "new" is inserted before "b", so matching by position would give it b's ID.
	planned, d := types.ListValueFrom(ctx, formInputObjectType, []FormDefinitionInputModel{
		{ID: types.StringUnknown(), Type: types.StringValue("STRING"), Label: types.StringValue("a"), Description: types.StringNull()},
		{ID: types.StringUnknown(), Type: types.StringValue("STRING"), Label: types.StringValue("new"), Description: types.StringNull()},
		{ID: types.StringUnknown(), Type: types.StringValue("STRING"), Label: types.StringValue("b"), Description: types.StringNull()},
	})
	diags.Append(d...)

	req := planmodifier.ListRequest{StateValue: prior, PlanValue: planned}
	resp := &planmodifier.ListResponse{PlanValue: planned}
	formInputIDsFromState{}.PlanModifyList(ctx, req, resp)
	diags.Append(resp.Diagnostics...)

	var got []FormDefinitionInputModel
	diags.Append(resp.PlanValue.ElementsAs(ctx, &got, false)...)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got[0].ID.ValueString() != "id-a" || !got[1].ID.IsUnknown() || got[2].ID.ValueString() != "id-b" {
		t.Fatalf("unexpected ids: %s, %s, %s", got[0].ID, got[1].ID, got[2].ID)
	}
}
