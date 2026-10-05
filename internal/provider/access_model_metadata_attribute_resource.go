package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewAccessModelMetadataAttributeResource)
	registerDataSource(NewAccessModelMetadataAttributeDataSource)
}

// accessModelMetadataAttributeDefaultValueStatus is the status assumed for a value when the API
// response does not contain it.
const accessModelMetadataAttributeDefaultValueStatus = "active"

// AccessModelMetadataAttributeValue is a value of an access model metadata attribute.
type AccessModelMetadataAttributeValue struct {
	Value  string `json:"value"`
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

// AccessModelMetadataAttributeDTO is an attribute as returned by the
// /v2026/access-model-metadata/attributes API.
type AccessModelMetadataAttributeDTO struct {
	Key         string                              `json:"key,omitempty"`
	Name        string                              `json:"name"`
	Multiselect *bool                               `json:"multiselect,omitempty"`
	Status      string                              `json:"status,omitempty"`
	Type        string                              `json:"type,omitempty"`
	ObjectTypes []string                            `json:"objectTypes,omitempty"`
	Description string                              `json:"description,omitempty"`
	Values      []AccessModelMetadataAttributeValue `json:"values,omitempty"`
}

func (c *Client) GetAccessModelMetadataAttribute(ctx context.Context, key string) (*AccessModelMetadataAttributeDTO, error) {
	var attribute AccessModelMetadataAttributeDTO
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/access-model-metadata/attributes/%s", key), nil, &attribute); err != nil {
		return nil, err
	}
	return &attribute, nil
}

func (c *Client) CreateAccessModelMetadataAttribute(ctx context.Context, attribute *AccessModelMetadataAttributeDTO) (*AccessModelMetadataAttributeDTO, error) {
	var created AccessModelMetadataAttributeDTO
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/access-model-metadata/attributes", attribute, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateAccessModelMetadataAttribute(ctx context.Context, key string, ops []jsonPatchOp) (*AccessModelMetadataAttributeDTO, error) {
	var updated AccessModelMetadataAttributeDTO
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/access-model-metadata/attributes/%s", key), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

var _ resource.Resource = &AccessModelMetadataAttributeResource{}
var _ resource.ResourceWithImportState = &AccessModelMetadataAttributeResource{}
var _ resource.ResourceWithModifyPlan = &AccessModelMetadataAttributeResource{}

func NewAccessModelMetadataAttributeResource() resource.Resource {
	return &AccessModelMetadataAttributeResource{}
}

type AccessModelMetadataAttributeResource struct {
	client *Config
}

type AccessModelMetadataAttributeResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Key         types.String `tfsdk:"key"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	ObjectTypes types.List   `tfsdk:"object_types"`
	Description types.String `tfsdk:"description"`
	Multiselect types.Bool   `tfsdk:"multiselect"`
	Status      types.String `tfsdk:"status"`
	Values      types.List   `tfsdk:"values"`
}

type AccessModelMetadataAttributeValueModel struct {
	Value  types.String `tfsdk:"value"`
	Name   types.String `tfsdk:"name"`
	Status types.String `tfsdk:"status"`
}

var accessModelMetadataAttributeValueObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"value":  types.StringType,
	"name":   types.StringType,
	"status": types.StringType,
}}

func (r *AccessModelMetadataAttributeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_model_metadata_attribute"
}

func (r *AccessModelMetadataAttributeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutableString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: description,
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an access model metadata attribute. The API has no delete operation: destroying the resource only removes it from Terraform state. For the same reason `key`, `type`, `status` and `object_types` cannot be changed after creation: the plan fails instead of replacing the attribute.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Attribute ID, the same as `key`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key":    immutableString("Unique technical name of the attribute. The API derives it from `name` when not set. It cannot be changed after creation."),
			"type":   immutableString("Type of the attribute, `custom` or `governance`. The API sets a default when not set. It cannot be changed after creation."),
			"status": immutableString("Status of the attribute, e.g. `active`. The API sets a default when not set. It cannot be changed after creation."),
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the attribute.",
				Required:            true,
			},
			"object_types": schema.ListAttribute{
				MarkdownDescription: "Object types the attribute values can be applied to, `all` or `entitlement`. The API sets a default when not set. It cannot be changed after creation.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the attribute.",
				Optional:            true,
			},
			"multiselect": schema.BoolAttribute{
				MarkdownDescription: "Whether an object can have multiple values of the attribute. The API defaults it to `false`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"values": schema.ListNestedBlock{
				MarkdownDescription: "Allowed values of the attribute, one block per value. Updates replace the whole list of values.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							MarkdownDescription: "Unique technical name of the value.",
							Required:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Display name of the value.",
							Required:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Status of the value, e.g. `active`. The API sets a default when not set.",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (r *AccessModelMetadataAttributeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// accessModelMetadataAttributeValues converts the value blocks to API values. Unknown statuses are
// omitted so the API applies its default.
func accessModelMetadataAttributeValues(ctx context.Context, list types.List, diags *diag.Diagnostics) []AccessModelMetadataAttributeValue {
	values := []AccessModelMetadataAttributeValue{}
	if list.IsNull() || list.IsUnknown() {
		return values
	}
	var models []AccessModelMetadataAttributeValueModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	for _, m := range models {
		value := AccessModelMetadataAttributeValue{Value: m.Value.ValueString(), Name: m.Name.ValueString()}
		if !m.Status.IsNull() && !m.Status.IsUnknown() {
			value.Status = m.Status.ValueString()
		}
		values = append(values, value)
	}
	return values
}

func accessModelMetadataAttributeFromModel(ctx context.Context, data AccessModelMetadataAttributeResourceModel, diags *diag.Diagnostics) *AccessModelMetadataAttributeDTO {
	attribute := &AccessModelMetadataAttributeDTO{
		Key:         data.Key.ValueString(),
		Name:        data.Name.ValueString(),
		Multiselect: boolPointer(data.Multiselect),
		Status:      data.Status.ValueString(),
		Type:        data.Type.ValueString(),
		Description: data.Description.ValueString(),
		Values:      accessModelMetadataAttributeValues(ctx, data.Values, diags),
	}
	if !data.ObjectTypes.IsNull() && !data.ObjectTypes.IsUnknown() {
		diags.Append(data.ObjectTypes.ElementsAs(ctx, &attribute.ObjectTypes, false)...)
	}
	return attribute
}

// accessModelMetadataAttributeValuesState converts API values to the block list, ordered like prior.
func accessModelMetadataAttributeValuesState(ctx context.Context, prior types.List, values []AccessModelMetadataAttributeValue, diags *diag.Diagnostics) types.List {
	if len(values) == 0 {
		return types.ListNull(accessModelMetadataAttributeValueObjectType)
	}
	var priorValues []AccessModelMetadataAttributeValue
	if !prior.IsNull() && !prior.IsUnknown() {
		priorValues = accessModelMetadataAttributeValues(ctx, prior, diags)
	}
	ordered := orderByPriorIDs(values, priorValues, func(v AccessModelMetadataAttributeValue) string { return v.Value })
	models := make([]AccessModelMetadataAttributeValueModel, 0, len(ordered))
	for _, v := range ordered {
		models = append(models, AccessModelMetadataAttributeValueModel{
			Value: types.StringValue(v.Value), Name: types.StringValue(v.Name), Status: types.StringValue(v.Status),
		})
	}
	list, d := types.ListValueFrom(ctx, accessModelMetadataAttributeValueObjectType, models)
	diags.Append(d...)
	return list
}

// accessModelMetadataAttributeResolveValues keeps the planned values and resolves unknown
// statuses from the API response.
func accessModelMetadataAttributeResolveValues(ctx context.Context, planned types.List, api []AccessModelMetadataAttributeValue, diags *diag.Diagnostics) types.List {
	if planned.IsNull() || planned.IsUnknown() {
		return planned
	}
	statuses := map[string]string{}
	for _, v := range api {
		statuses[v.Value] = v.Status
	}
	var models []AccessModelMetadataAttributeValueModel
	diags.Append(planned.ElementsAs(ctx, &models, false)...)
	for i := range models {
		if models[i].Status.IsUnknown() {
			status := statuses[models[i].Value.ValueString()]
			if status == "" {
				status = accessModelMetadataAttributeDefaultValueStatus
			}
			models[i].Status = types.StringValue(status)
		}
	}
	list, d := types.ListValueFrom(ctx, accessModelMetadataAttributeValueObjectType, models)
	diags.Append(d...)
	return list
}

func accessModelMetadataAttributeObjectTypesState(ctx context.Context, objectTypes []string, diags *diag.Diagnostics) types.List {
	list, d := types.ListValueFrom(ctx, types.StringType, append([]string{}, objectTypes...))
	diags.Append(d...)
	return list
}

// accessModelMetadataAttributeResolveApply keeps the planned values after apply and resolves the
// unknown ones from the API response.
func accessModelMetadataAttributeResolveApply(ctx context.Context, data *AccessModelMetadataAttributeResourceModel, attribute *AccessModelMetadataAttributeDTO, diags *diag.Diagnostics) {
	data.Key = computedStringFromAPI(data.Key, attribute.Key)
	data.ID = types.StringValue(data.Key.ValueString())
	data.Type = computedStringFromAPI(data.Type, attribute.Type)
	data.Status = computedStringFromAPI(data.Status, attribute.Status)
	data.Multiselect = computedBoolFromAPI(data.Multiselect, attribute.Multiselect)
	if data.ObjectTypes.IsUnknown() {
		data.ObjectTypes = accessModelMetadataAttributeObjectTypesState(ctx, attribute.ObjectTypes, diags)
	}
	data.Values = accessModelMetadataAttributeResolveValues(ctx, data.Values, attribute.Values, diags)
}

// setAccessModelMetadataAttributeState refreshes the model from the API.
func setAccessModelMetadataAttributeState(ctx context.Context, data *AccessModelMetadataAttributeResourceModel, attribute *AccessModelMetadataAttributeDTO, diags *diag.Diagnostics) {
	data.ID = types.StringValue(attribute.Key)
	data.Key = types.StringValue(attribute.Key)
	data.Name = types.StringValue(attribute.Name)
	data.Type = types.StringValue(attribute.Type)
	data.Status = types.StringValue(attribute.Status)
	data.Description = optionalStringState(data.Description, attribute.Description)
	data.Multiselect = types.BoolValue(attribute.Multiselect != nil && *attribute.Multiselect)
	data.ObjectTypes = accessModelMetadataAttributeObjectTypesState(ctx, attribute.ObjectTypes, diags)
	data.Values = accessModelMetadataAttributeValuesState(ctx, data.Values, attribute.Values, diags)
}

// accessModelMetadataAttributeImmutableChanges returns the attributes that cannot be updated
// (key, type, status and object_types) whose known planned value differs from the state. The API
// has no delete operation, so the attribute cannot be replaced either: re-creating it with the
// same key fails because the old attribute still exists.
func accessModelMetadataAttributeImmutableChanges(plan, state AccessModelMetadataAttributeResourceModel) []string {
	var changed []string
	fields := []struct {
		name           string
		planned, prior attr.Value
	}{
		{"key", plan.Key, state.Key},
		{"type", plan.Type, state.Type},
		{"status", plan.Status, state.Status},
		{"object_types", plan.ObjectTypes, state.ObjectTypes},
	}
	for _, f := range fields {
		if f.planned.IsUnknown() || f.prior.IsNull() || f.prior.IsUnknown() || f.planned.Equal(f.prior) {
			continue
		}
		changed = append(changed, f.name)
	}
	return changed
}

// accessModelMetadataAttributeImmutableError adds an error for each attribute that cannot be changed.
func accessModelMetadataAttributeImmutableError(diags *diag.Diagnostics, changed []string) {
	for _, name := range changed {
		diags.AddAttributeError(path.Root(name), "Attribute cannot be changed",
			fmt.Sprintf("The %s of an access model metadata attribute cannot be changed after creation, and the attribute cannot be replaced because the IdentityNow API cannot delete it. "+
				"Revert the change, or change the attribute in the IdentityNow UI and update the configuration, or create a new attribute with another key in a new resource.", name))
	}
}

// ModifyPlan rejects changes of the attributes that can neither be updated nor replaced.
func (r *AccessModelMetadataAttributeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return // destroy or create
	}
	var plan, state AccessModelMetadataAttributeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accessModelMetadataAttributeImmutableError(&resp.Diagnostics, accessModelMetadataAttributeImmutableChanges(plan, state))
}

// accessModelMetadataAttributePatches returns JSON Patch operations for the changed patchable
// fields: name, description, multiselect and values.
func accessModelMetadataAttributePatches(ctx context.Context, plan, state AccessModelMetadataAttributeResourceModel, diags *diag.Diagnostics) []jsonPatchOp {
	b := &patchBuilder{}
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	if !plan.Description.Equal(state.Description) {
		b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/description", Value: plan.Description.ValueString()})
	}
	b.replaceIfChanged(plan.Multiselect, state.Multiselect, "/multiselect", plan.Multiselect.ValueBool())
	b.replaceIfChanged(plan.Values, state.Values, "/values", accessModelMetadataAttributeValues(ctx, plan.Values, diags))
	return b.ops
}

func (r *AccessModelMetadataAttributeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccessModelMetadataAttributeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := accessModelMetadataAttributeFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateAccessModelMetadataAttribute(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create access model metadata attribute: %s", err))
		return
	}
	accessModelMetadataAttributeResolveApply(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessModelMetadataAttributeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccessModelMetadataAttributeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	attribute, err := client.GetAccessModelMetadataAttribute(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read access model metadata attribute: %s", err))
		return
	}
	setAccessModelMetadataAttributeState(ctx, &data, attribute, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessModelMetadataAttributeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state AccessModelMetadataAttributeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Values that were unknown during the plan are checked again.
	accessModelMetadataAttributeImmutableError(&resp.Diagnostics, accessModelMetadataAttributeImmutableChanges(data, state))
	ops := accessModelMetadataAttributePatches(ctx, data, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var updated *AccessModelMetadataAttributeDTO
	if len(ops) > 0 {
		updated, err = client.UpdateAccessModelMetadataAttribute(ctx, data.ID.ValueString(), ops)
	} else {
		updated, err = client.GetAccessModelMetadataAttribute(ctx, data.ID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update access model metadata attribute: %s", err))
		return
	}
	accessModelMetadataAttributeResolveApply(ctx, &data, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete only removes the attribute from state, the API has no delete operation.
func (r *AccessModelMetadataAttributeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Access model metadata attribute left in place",
		"The IdentityNow API does not support deleting access model metadata attributes. The resource was removed from Terraform state and the attribute and its values were left unchanged in IdentityNow.",
	)
}

func (r *AccessModelMetadataAttributeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), req.ID)...)
}

var _ datasource.DataSource = &AccessModelMetadataAttributeDataSource{}

func NewAccessModelMetadataAttributeDataSource() datasource.DataSource {
	return &AccessModelMetadataAttributeDataSource{}
}

type AccessModelMetadataAttributeDataSource struct {
	client *Config
}

func (d *AccessModelMetadataAttributeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_model_metadata_attribute"
}

func (d *AccessModelMetadataAttributeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up an access model metadata attribute by key.",
		Attributes: map[string]dsschema.Attribute{
			"id":           dsschema.StringAttribute{MarkdownDescription: "Attribute ID, the same as `key`.", Computed: true},
			"key":          dsschema.StringAttribute{MarkdownDescription: "Technical name of the attribute.", Required: true},
			"name":         dsschema.StringAttribute{MarkdownDescription: "Display name of the attribute.", Computed: true},
			"type":         dsschema.StringAttribute{MarkdownDescription: "Type of the attribute.", Computed: true},
			"status":       dsschema.StringAttribute{MarkdownDescription: "Status of the attribute.", Computed: true},
			"object_types": dsschema.ListAttribute{MarkdownDescription: "Object types the attribute values can be applied to.", Computed: true, ElementType: types.StringType},
			"description":  dsschema.StringAttribute{MarkdownDescription: "Description of the attribute.", Computed: true},
			"multiselect":  dsschema.BoolAttribute{MarkdownDescription: "Whether an object can have multiple values of the attribute.", Computed: true},
			"values": dsschema.ListNestedAttribute{
				MarkdownDescription: "Allowed values of the attribute, each with `value`, `name` and `status`.",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"value":  dsschema.StringAttribute{MarkdownDescription: "Technical name of the value.", Computed: true},
						"name":   dsschema.StringAttribute{MarkdownDescription: "Display name of the value.", Computed: true},
						"status": dsschema.StringAttribute{MarkdownDescription: "Status of the value.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *AccessModelMetadataAttributeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *Config, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *AccessModelMetadataAttributeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccessModelMetadataAttributeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	attribute, err := client.GetAccessModelMetadataAttribute(ctx, data.Key.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("key"), "Access model metadata attribute not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read access model metadata attribute: %s", err))
		return
	}
	data.Description = types.StringValue("")
	setAccessModelMetadataAttributeState(ctx, &data, attribute, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
