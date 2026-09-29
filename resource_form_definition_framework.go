package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &FormDefinitionResource{}
var _ resource.ResourceWithImportState = &FormDefinitionResource{}
var _ resource.ResourceWithValidateConfig = &FormDefinitionResource{}

func NewFormDefinitionResource() resource.Resource {
	return &FormDefinitionResource{}
}

type FormDefinitionResource struct {
	client *Config
}

type FormDefinitionResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Owner              types.List   `tfsdk:"owner"`
	FormInput          types.List   `tfsdk:"form_input"`
	FormElementsJSON   types.String `tfsdk:"form_elements_json"`
	FormConditionsJSON types.String `tfsdk:"form_conditions_json"`
	UsedBy             types.List   `tfsdk:"used_by"`
	Created            types.String `tfsdk:"created"`
	Modified           types.String `tfsdk:"modified"`
}

type FormDefinitionInputModel struct {
	ID          types.String `tfsdk:"id"`
	Type        types.String `tfsdk:"type"`
	Label       types.String `tfsdk:"label"`
	Description types.String `tfsdk:"description"`
}

type FormUsedByModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	Name types.String `tfsdk:"name"`
}

var formOwnerObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":   types.StringType,
	"type": types.StringType,
	"name": types.StringType,
}}

var formInputObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":          types.StringType,
	"type":        types.StringType,
	"label":       types.StringType,
	"description": types.StringType,
}}

var formUsedByObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":   types.StringType,
	"type": types.StringType,
	"name": types.StringType,
}}

func (r *FormDefinitionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_form_definition"
}

func (r *FormDefinitionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SailPoint custom form definition.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Form definition ID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Form definition name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Form definition description",
				Optional:            true,
			},
			"form_elements_json": schema.StringAttribute{
				MarkdownDescription: "List of nested form elements as a JSON array. Root elements must be of type `SECTION`.",
				Optional:            true,
				Validators:          []validator.String{jsonArrayStringValidator{}},
			},
			"form_conditions_json": schema.StringAttribute{
				MarkdownDescription: "Conditional logic that dynamically modifies the form, as a JSON array.",
				Optional:            true,
				Validators:          []validator.String{jsonArrayStringValidator{}},
			},
			"used_by": schema.ListNestedAttribute{
				MarkdownDescription: "Systems currently using the form definition",
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "The date and time the form definition was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"modified": schema.StringAttribute{
				MarkdownDescription: "The date and time the form definition was modified",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"owner": schema.ListNestedBlock{
				MarkdownDescription: "Form definition owner. Exactly one owner is required.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Owner identity ID",
							Required:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Owner type (IDENTITY)",
							Required:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Owner name",
							Optional:            true,
						},
					},
				},
			},
			"form_input": schema.ListNestedBlock{
				MarkdownDescription: "Form inputs required when creating a form instance",
				PlanModifiers:       []planmodifier.List{formInputIDsFromState{}},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Form input identifier, assigned by the API",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Form input type (STRING or ARRAY)",
							Required:            true,
						},
						"label": schema.StringAttribute{
							MarkdownDescription: "Form input name",
							Optional:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Form input description",
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

func (r *FormDefinitionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var owner types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("owner"), &owner)...)
	if resp.Diagnostics.HasError() || owner.IsUnknown() {
		return
	}
	if len(owner.Elements()) != 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("owner"),
			"Invalid owner configuration",
			fmt.Sprintf("Exactly one owner block is required, got %d.", len(owner.Elements())),
		)
	}
}

func (r *FormDefinitionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *Config, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *FormDefinitionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FormDefinitionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form := formDefinitionFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating Form Definition", map[string]interface{}{"name": form.Name})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	newForm, err := client.CreateFormDefinition(ctx, form)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create form definition: %s", err))
		return
	}

	data.ID = types.StringValue(newForm.ID)
	data.FormInput = formInputWithIDs(ctx, data.FormInput, newForm.FormInput, &resp.Diagnostics)
	data.UsedBy = formUsedByState(ctx, newForm.UsedBy, &resp.Diagnostics)
	data.Created = types.StringValue(newForm.Created)
	data.Modified = types.StringValue(newForm.Modified)

	tflog.Trace(ctx, "created a form definition resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FormDefinitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FormDefinitionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Form Definition", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	form, err := client.GetFormDefinition(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read form definition: %s", err))
		return
	}

	setFormDefinitionState(ctx, &data, form, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FormDefinitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FormDefinitionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating Form Definition", map[string]interface{}{"id": data.ID.ValueString()})

	form := formDefinitionFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	updatedForm, err := client.UpdateFormDefinition(ctx, data.ID.ValueString(), formDefinitionPatches(form))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update form definition: %s", err))
		return
	}

	// used_by and created keep their planned (prior state) values, Read refreshes them.
	data.FormInput = formInputWithIDs(ctx, data.FormInput, updatedForm.FormInput, &resp.Diagnostics)
	data.Modified = types.StringValue(updatedForm.Modified)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FormDefinitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FormDefinitionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Form Definition", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	err = client.DeleteFormDefinition(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete form definition: %s", err))
		return
	}
}

func (r *FormDefinitionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// formDefinitionFromModel converts the Terraform model to the API FormDefinition struct.
func formDefinitionFromModel(ctx context.Context, data FormDefinitionResourceModel, diags *diag.Diagnostics) *FormDefinition {
	form := &FormDefinition{
		Name:           data.Name.ValueString(),
		Description:    data.Description.ValueString(),
		FormElements:   formDefinitionJSONValue(data.FormElementsJSON),
		FormConditions: formDefinitionJSONValue(data.FormConditionsJSON),
	}

	var owners []OwnerModel
	diags.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if len(owners) > 0 {
		form.Owner = &FormOwner{
			ID:   owners[0].ID.ValueString(),
			Type: owners[0].Type.ValueString(),
			Name: owners[0].Name.ValueString(),
		}
	}

	if !data.FormInput.IsNull() && !data.FormInput.IsUnknown() {
		var inputs []FormDefinitionInputModel
		diags.Append(data.FormInput.ElementsAs(ctx, &inputs, false)...)
		for _, input := range inputs {
			// ID is unknown for new inputs, ValueString returns "" so it is omitted and the API assigns one.
			form.FormInput = append(form.FormInput, &FormDefinitionInput{
				ID:          input.ID.ValueString(),
				Type:        input.Type.ValueString(),
				Label:       input.Label.ValueString(),
				Description: input.Description.ValueString(),
			})
		}
	}

	return form
}

// formDefinitionPatches builds the JSON patch for a full update of the form definition.
func formDefinitionPatches(form *FormDefinition) []*UpdateFormDefinition {
	formInput := form.FormInput
	if formInput == nil {
		formInput = []*FormDefinitionInput{}
	}
	formElements := form.FormElements
	if formElements == nil {
		formElements = json.RawMessage("[]")
	}
	formConditions := form.FormConditions
	if formConditions == nil {
		formConditions = json.RawMessage("[]")
	}
	return []*UpdateFormDefinition{
		{Op: "replace", Path: "/name", Value: form.Name},
		{Op: "replace", Path: "/description", Value: form.Description},
		{Op: "replace", Path: "/owner", Value: form.Owner},
		{Op: "replace", Path: "/formInput", Value: formInput},
		{Op: "replace", Path: "/formElements", Value: formElements},
		{Op: "replace", Path: "/formConditions", Value: formConditions},
	}
}

// setFormDefinitionState maps an API form definition onto the model, keeping values from
// prior state where the API response is semantically equivalent.
func setFormDefinitionState(ctx context.Context, data *FormDefinitionResourceModel, form *FormDefinition, diags *diag.Diagnostics) {
	data.ID = types.StringValue(form.ID)
	data.Name = types.StringValue(form.Name)
	if form.Description != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(form.Description)
	}
	data.Owner = formOwnerState(ctx, form.Owner, data.Owner, diags)
	data.FormInput = formInputState(ctx, form.FormInput, data.FormInput, diags)
	data.FormElementsJSON = formDefinitionJSONState(data.FormElementsJSON, form.FormElements)
	data.FormConditionsJSON = formDefinitionJSONState(data.FormConditionsJSON, form.FormConditions)
	data.UsedBy = formUsedByState(ctx, form.UsedBy, diags)
	data.Created = types.StringValue(form.Created)
	data.Modified = types.StringValue(form.Modified)
}

// formOwnerState maps the API owner to state. The owner name is optional in configuration, so it
// stays null when the prior state has no name, to avoid a diff against the name the API returns.
func formOwnerState(ctx context.Context, owner *FormOwner, prior types.List, diags *diag.Diagnostics) types.List {
	if owner == nil {
		return types.ListNull(formOwnerObjectType)
	}
	name := stringValueOrNull(owner.Name)
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorOwners []OwnerModel
		diags.Append(prior.ElementsAs(ctx, &priorOwners, false)...)
		if len(priorOwners) > 0 && priorOwners[0].Name.IsNull() {
			name = types.StringNull()
		}
	}
	owners, d := types.ListValueFrom(ctx, formOwnerObjectType, []OwnerModel{{
		ID:   types.StringValue(owner.ID),
		Type: types.StringValue(owner.Type),
		Name: name,
	}})
	diags.Append(d...)
	return owners
}

// formInputState maps the API form inputs to state. When there are no inputs, an empty prior list
// is kept as-is, so an absent block in configuration does not flip between null and empty.
func formInputState(ctx context.Context, inputs []*FormDefinitionInput, prior types.List, diags *diag.Diagnostics) types.List {
	if len(inputs) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
			return prior
		}
		return types.ListNull(formInputObjectType)
	}
	models := make([]FormDefinitionInputModel, len(inputs))
	for i, input := range inputs {
		models[i] = FormDefinitionInputModel{
			ID:          types.StringValue(input.ID),
			Type:        types.StringValue(input.Type),
			Label:       stringValueOrNull(input.Label),
			Description: stringValueOrNull(input.Description),
		}
	}
	list, d := types.ListValueFrom(ctx, formInputObjectType, models)
	diags.Append(d...)
	return list
}

// formInputIDsFromState keeps the IDs of existing form inputs in the plan. Inputs are matched to
// prior state by label and type, so inserting or removing an input does not shift IDs between inputs.
// Inputs without a match keep an unknown ID, which the API assigns.
type formInputIDsFromState struct{}

func (m formInputIDsFromState) Description(ctx context.Context) string {
	return "Keeps the IDs of form inputs that match prior state by label and type."
}

func (m formInputIDsFromState) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m formInputIDsFromState) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	var prior, planned []FormDefinitionInputModel
	resp.Diagnostics.Append(req.StateValue.ElementsAs(ctx, &prior, false)...)
	resp.Diagnostics.Append(req.PlanValue.ElementsAs(ctx, &planned, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	used := make([]bool, len(prior))
	for i := range planned {
		if !planned[i].ID.IsUnknown() && !planned[i].ID.IsNull() {
			continue
		}
		planned[i].ID = types.StringUnknown()
		for j := range prior {
			if !used[j] && prior[j].Label.Equal(planned[i].Label) && prior[j].Type.Equal(planned[i].Type) {
				planned[i].ID = prior[j].ID
				used[j] = true
				break
			}
		}
	}
	list, d := types.ListValueFrom(ctx, formInputObjectType, planned)
	resp.Diagnostics.Append(d...)
	resp.PlanValue = list
}

// formInputWithIDs fills the API-assigned IDs into the planned form inputs, matching them by position.
// Other planned values are kept as-is so the state matches the plan after apply.
func formInputWithIDs(ctx context.Context, planned types.List, inputs []*FormDefinitionInput, diags *diag.Diagnostics) types.List {
	if planned.IsNull() || planned.IsUnknown() {
		return planned
	}
	var models []FormDefinitionInputModel
	diags.Append(planned.ElementsAs(ctx, &models, false)...)
	for i := range models {
		if !models[i].ID.IsUnknown() {
			continue
		}
		models[i].ID = types.StringNull()
		if i < len(inputs) {
			models[i].ID = stringValueOrNull(inputs[i].ID)
		}
	}
	list, d := types.ListValueFrom(ctx, formInputObjectType, models)
	diags.Append(d...)
	return list
}

func formUsedByState(ctx context.Context, usedBy []*FormUsedBy, diags *diag.Diagnostics) types.List {
	models := make([]FormUsedByModel, len(usedBy))
	for i, u := range usedBy {
		models[i] = FormUsedByModel{
			ID:   types.StringValue(u.ID),
			Type: types.StringValue(u.Type),
			Name: types.StringValue(u.Name),
		}
	}
	list, d := types.ListValueFrom(ctx, formUsedByObjectType, models)
	diags.Append(d...)
	return list
}

// formDefinitionJSONValue returns the configured JSON string as a raw message for the API.
// The value is already checked to be a JSON array by jsonArrayStringValidator.
func formDefinitionJSONValue(value types.String) json.RawMessage {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	return json.RawMessage(value.ValueString())
}

// formDefinitionJSONState returns the state value for a JSON array attribute. The prior value is
// kept when it is semantically equal to the API value, so formatting and key order do not cause diffs.
// An empty or missing API value maps to null unless the prior value is also an empty array.
func formDefinitionJSONState(prior types.String, raw json.RawMessage) types.String {
	priorSet := !prior.IsNull() && !prior.IsUnknown()
	if isEmptyJSONArray(raw) {
		if priorSet && isEmptyJSONArray(json.RawMessage(prior.ValueString())) {
			return prior
		}
		return types.StringNull()
	}
	if priorSet && jsonSemanticallyEqual([]byte(prior.ValueString()), raw) {
		return prior
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return types.StringValue(string(raw))
	}
	normalized, err := json.Marshal(value)
	if err != nil {
		return types.StringValue(string(raw))
	}
	return types.StringValue(string(normalized))
}

func isEmptyJSONArray(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	if value == nil {
		return true
	}
	list, ok := value.([]interface{})
	return ok && len(list) == 0
}
