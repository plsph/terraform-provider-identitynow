package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewNonEmployeeSchemaAttributeResource)
	registerDataSource(NewNonEmployeeSchemaAttributeDataSource)
}

// NonEmployeeSchemaAttribute is a schema attribute of a non-employee source as returned by the
// /v2026/non-employee-sources/{sourceId}/schema-attributes API.
type NonEmployeeSchemaAttribute struct {
	ID            string `json:"id,omitempty"`
	System        bool   `json:"system,omitempty"`
	Created       string `json:"created,omitempty"`
	Modified      string `json:"modified,omitempty"`
	Type          string `json:"type"`
	Label         string `json:"label"`
	TechnicalName string `json:"technicalName"`
	HelpText      string `json:"helpText,omitempty"`
	Placeholder   string `json:"placeholder,omitempty"`
	Required      *bool  `json:"required,omitempty"`
}

func (c *Client) GetNonEmployeeSchemaAttribute(ctx context.Context, sourceID, id string) (*NonEmployeeSchemaAttribute, error) {
	var attribute NonEmployeeSchemaAttribute
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/non-employee-sources/%s/schema-attributes/%s", sourceID, id), nil, &attribute); err != nil {
		return nil, err
	}
	return &attribute, nil
}

// GetNonEmployeeSchemaAttributeByTechnicalName lists the schema attributes of a source (at most 18,
// not paginated) and returns the one with the given technical name.
func (c *Client) GetNonEmployeeSchemaAttributeByTechnicalName(ctx context.Context, sourceID, technicalName string) (*NonEmployeeSchemaAttribute, error) {
	var attributes []NonEmployeeSchemaAttribute
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/non-employee-sources/%s/schema-attributes", sourceID), nil, &attributes); err != nil {
		return nil, err
	}
	for i := range attributes {
		if attributes[i].TechnicalName == technicalName {
			return &attributes[i], nil
		}
	}
	return nil, &NotFoundError{fmt.Sprintf("non-employee schema attribute %q not found", technicalName)}
}

func (c *Client) CreateNonEmployeeSchemaAttribute(ctx context.Context, sourceID string, attribute *NonEmployeeSchemaAttribute) (*NonEmployeeSchemaAttribute, error) {
	var created NonEmployeeSchemaAttribute
	if err := c.doJSON(ctx, http.MethodPost, apiPath("/v2026/non-employee-sources/%s/schema-attributes", sourceID), attribute, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchNonEmployeeSchemaAttribute(ctx context.Context, sourceID, id string, ops []jsonPatchOp) (*NonEmployeeSchemaAttribute, error) {
	var updated NonEmployeeSchemaAttribute
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/non-employee-sources/%s/schema-attributes/%s", sourceID, id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteNonEmployeeSchemaAttribute(ctx context.Context, sourceID, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/non-employee-sources/%s/schema-attributes/%s", sourceID, id), nil, nil)
}

var _ resource.Resource = &NonEmployeeSchemaAttributeResource{}
var _ resource.ResourceWithImportState = &NonEmployeeSchemaAttributeResource{}

func NewNonEmployeeSchemaAttributeResource() resource.Resource {
	return &NonEmployeeSchemaAttributeResource{}
}

type NonEmployeeSchemaAttributeResource struct {
	client *Config
}

type NonEmployeeSchemaAttributeModel struct {
	ID                  types.String `tfsdk:"id"`
	NonEmployeeSourceID types.String `tfsdk:"non_employee_source_id"`
	TechnicalName       types.String `tfsdk:"technical_name"`
	Type                types.String `tfsdk:"type"`
	Label               types.String `tfsdk:"label"`
	HelpText            types.String `tfsdk:"help_text"`
	Placeholder         types.String `tfsdk:"placeholder"`
	Required            types.Bool   `tfsdk:"required"`
	System              types.Bool   `tfsdk:"system"`
	Created             types.String `tfsdk:"created"`
	Modified            types.String `tfsdk:"modified"`
}

func (r *NonEmployeeSchemaAttributeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_non_employee_schema_attribute"
}

func (r *NonEmployeeSchemaAttributeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom schema attribute of a non-employee source. A source has 8 mandatory attributes and up to 10 custom attributes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Schema attribute ID",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"non_employee_source_id": schema.StringAttribute{
				MarkdownDescription: "ID of the non-employee source (`identitynow_non_employee_source.id`). Changing it forces a new attribute.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"technical_name": schema.StringAttribute{
				MarkdownDescription: "Technical name of the attribute, unique per source. It cannot be changed, changing it forces a new attribute.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Attribute type. Only `TEXT` is supported for custom attributes, which is the default. Changing it forces a new attribute.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("TEXT"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{triggerSubscriptionOneOfValidator{values: []string{"TEXT", "DATE", "IDENTITY"}}},
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "Label displayed in the UI",
				Required:            true,
			},
			"help_text": schema.StringAttribute{
				MarkdownDescription: "Help text displayed in the UI",
				Optional:            true,
			},
			"placeholder": schema.StringAttribute{
				MarkdownDescription: "Hint text shown in the empty input field",
				Optional:            true,
			},
			"required": schema.BoolAttribute{
				MarkdownDescription: "Whether the attribute is required for all non-employees of the source. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"system": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a mandatory system attribute",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Creation date",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modified": schema.StringAttribute{
				MarkdownDescription: "Last modification date",
				Computed:            true,
			},
		},
	}
}

func (r *NonEmployeeSchemaAttributeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func nonEmployeeSchemaAttributeFromModel(data NonEmployeeSchemaAttributeModel) *NonEmployeeSchemaAttribute {
	return &NonEmployeeSchemaAttribute{
		Type:          data.Type.ValueString(),
		Label:         data.Label.ValueString(),
		TechnicalName: data.TechnicalName.ValueString(),
		HelpText:      data.HelpText.ValueString(),
		Placeholder:   data.Placeholder.ValueString(),
		Required:      boolPointer(data.Required),
	}
}

// setNonEmployeeSchemaAttributeState refreshes the model from the API, keeping null for unset
// optional attributes.
func setNonEmployeeSchemaAttributeState(data *NonEmployeeSchemaAttributeModel, attribute *NonEmployeeSchemaAttribute) {
	data.ID = types.StringValue(attribute.ID)
	data.TechnicalName = types.StringValue(attribute.TechnicalName)
	data.Type = types.StringValue(attribute.Type)
	data.Label = types.StringValue(attribute.Label)
	data.HelpText = optionalStringState(data.HelpText, attribute.HelpText)
	data.Placeholder = optionalStringState(data.Placeholder, attribute.Placeholder)
	data.Required = types.BoolValue(attribute.Required != nil && *attribute.Required)
	data.System = types.BoolValue(attribute.System)
	data.Created = types.StringValue(attribute.Created)
	data.Modified = types.StringValue(attribute.Modified)
}

// nonEmployeeSchemaAttributePatchOps returns replace operations for the patchable attributes that
// changed. Removed text values are replaced with an empty string.
func nonEmployeeSchemaAttributePatchOps(plan, state NonEmployeeSchemaAttributeModel) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.Label, state.Label, "/label", plan.Label.ValueString())
	b.replaceIfChanged(plan.HelpText, state.HelpText, "/helpText", plan.HelpText.ValueString())
	b.replaceIfChanged(plan.Placeholder, state.Placeholder, "/placeholder", plan.Placeholder.ValueString())
	b.replaceIfChanged(plan.Required, state.Required, "/required", plan.Required.ValueBool())
	return b.ops
}

func (r *NonEmployeeSchemaAttributeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NonEmployeeSchemaAttributeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateNonEmployeeSchemaAttribute(ctx, data.NonEmployeeSourceID.ValueString(), nonEmployeeSchemaAttributeFromModel(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create non-employee schema attribute: %s", err))
		return
	}
	data.ID = types.StringValue(created.ID)
	data.System = types.BoolValue(created.System)
	data.Created = types.StringValue(created.Created)
	data.Modified = types.StringValue(created.Modified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NonEmployeeSchemaAttributeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NonEmployeeSchemaAttributeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	attribute, err := client.GetNonEmployeeSchemaAttribute(ctx, data.NonEmployeeSourceID.ValueString(), data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read non-employee schema attribute: %s", err))
		return
	}
	setNonEmployeeSchemaAttributeState(&data, attribute)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NonEmployeeSchemaAttributeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NonEmployeeSchemaAttributeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Modified = state.Modified
	if ops := nonEmployeeSchemaAttributePatchOps(plan, state); len(ops) > 0 {
		client, err := r.client.IdentityNowClient(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		updated, err := client.PatchNonEmployeeSchemaAttribute(ctx, plan.NonEmployeeSourceID.ValueString(), plan.ID.ValueString(), ops)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update non-employee schema attribute: %s", err))
			return
		}
		plan.Modified = types.StringValue(updated.Modified)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NonEmployeeSchemaAttributeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NonEmployeeSchemaAttributeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteNonEmployeeSchemaAttribute(ctx, data.NonEmployeeSourceID.ValueString(), data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete non-employee schema attribute: %s", err))
	}
}

func (r *NonEmployeeSchemaAttributeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	sourceID, id, ok := strings.Cut(req.ID, "/")
	if !ok || sourceID == "" || id == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected <non_employee_source_id>/<id>, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("non_employee_source_id"), sourceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

var _ datasource.DataSource = &NonEmployeeSchemaAttributeDataSource{}
var _ datasource.DataSourceWithValidateConfig = &NonEmployeeSchemaAttributeDataSource{}

func NewNonEmployeeSchemaAttributeDataSource() datasource.DataSource {
	return &NonEmployeeSchemaAttributeDataSource{}
}

type NonEmployeeSchemaAttributeDataSource struct {
	client *Config
}

func (d *NonEmployeeSchemaAttributeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_non_employee_schema_attribute"
}

func (d *NonEmployeeSchemaAttributeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a schema attribute of a non-employee source by ID or technical name. Mandatory system attributes can be looked up too.",
		Attributes: map[string]dsschema.Attribute{
			"non_employee_source_id": dsschema.StringAttribute{
				MarkdownDescription: "ID of the non-employee source",
				Required:            true,
			},
			"id": dsschema.StringAttribute{
				MarkdownDescription: "Schema attribute ID. Exactly one of `id` or `technical_name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"technical_name": dsschema.StringAttribute{
				MarkdownDescription: "Technical name of the attribute. Exactly one of `id` or `technical_name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"type":        dsschema.StringAttribute{MarkdownDescription: "Attribute type: `TEXT`, `DATE` or `IDENTITY`", Computed: true},
			"label":       dsschema.StringAttribute{MarkdownDescription: "Label displayed in the UI", Computed: true},
			"help_text":   dsschema.StringAttribute{MarkdownDescription: "Help text displayed in the UI", Computed: true},
			"placeholder": dsschema.StringAttribute{MarkdownDescription: "Hint text shown in the empty input field", Computed: true},
			"required":    dsschema.BoolAttribute{MarkdownDescription: "Whether the attribute is required for all non-employees", Computed: true},
			"system":      dsschema.BoolAttribute{MarkdownDescription: "Whether this is a mandatory system attribute", Computed: true},
			"created":     dsschema.StringAttribute{MarkdownDescription: "Creation date", Computed: true},
			"modified":    dsschema.StringAttribute{MarkdownDescription: "Last modification date", Computed: true},
		},
	}
}

func (d *NonEmployeeSchemaAttributeDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "technical_name")
}

func (d *NonEmployeeSchemaAttributeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NonEmployeeSchemaAttributeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NonEmployeeSchemaAttributeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var attribute *NonEmployeeSchemaAttribute
	lookup := path.Root("id")
	if !data.ID.IsNull() {
		attribute, err = client.GetNonEmployeeSchemaAttribute(ctx, data.NonEmployeeSourceID.ValueString(), data.ID.ValueString())
	} else {
		lookup = path.Root("technical_name")
		attribute, err = client.GetNonEmployeeSchemaAttributeByTechnicalName(ctx, data.NonEmployeeSourceID.ValueString(), data.TechnicalName.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(lookup, "Non-employee schema attribute not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read non-employee schema attribute: %s", err))
		return
	}
	data.HelpText = types.StringValue(attribute.HelpText)
	data.Placeholder = types.StringValue(attribute.Placeholder)
	setNonEmployeeSchemaAttributeState(&data, attribute)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
