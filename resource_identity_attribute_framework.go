package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewIdentityAttributeResource)
	registerDataSource(NewIdentityAttributeDataSource)
}

// IdentityAttribute is an identity attribute as returned by the /v2026/identity-attributes API.
type IdentityAttribute struct {
	Name        string          `json:"name"`
	DisplayName *string         `json:"displayName,omitempty"`
	Standard    *bool           `json:"standard,omitempty"`
	Type        *string         `json:"type,omitempty"`
	Multi       *bool           `json:"multi,omitempty"`
	Searchable  *bool           `json:"searchable,omitempty"`
	System      *bool           `json:"system,omitempty"`
	Sources     json.RawMessage `json:"sources,omitempty"`
}

func (c *Client) GetIdentityAttribute(ctx context.Context, name string) (*IdentityAttribute, error) {
	var attribute IdentityAttribute
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/identity-attributes/%s", name), nil, &attribute); err != nil {
		return nil, err
	}
	return &attribute, nil
}

func (c *Client) CreateIdentityAttribute(ctx context.Context, attribute *IdentityAttribute) (*IdentityAttribute, error) {
	var created IdentityAttribute
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/identity-attributes", attribute, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateIdentityAttribute(ctx context.Context, name string, attribute *IdentityAttribute) (*IdentityAttribute, error) {
	var updated IdentityAttribute
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/identity-attributes/%s", name), attribute, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteIdentityAttribute(ctx context.Context, name string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/identity-attributes/%s", name), nil, nil)
}

var _ resource.Resource = &IdentityAttributeResource{}
var _ resource.ResourceWithImportState = &IdentityAttributeResource{}

func NewIdentityAttributeResource() resource.Resource {
	return &IdentityAttributeResource{}
}

type IdentityAttributeResource struct {
	client *Config
}

type IdentityAttributeModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Standard    types.Bool   `tfsdk:"standard"`
	Type        types.String `tfsdk:"type"`
	Multi       types.Bool   `tfsdk:"multi"`
	Searchable  types.Bool   `tfsdk:"searchable"`
	System      types.Bool   `tfsdk:"system"`
	SourcesJSON types.String `tfsdk:"sources_json"`
}

func (r *IdentityAttributeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_attribute"
}

func (r *IdentityAttributeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	optionalComputedBool := func(description string) schema.BoolAttribute {
		return schema.BoolAttribute{
			MarkdownDescription: description,
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an identity attribute.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identity attribute ID, the same as `name`",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Technical name of the identity attribute. It identifies the attribute, changing it forces a new identity attribute.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Business-friendly name of the identity attribute. The API fills it in when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"standard": optionalComputedBool("Whether the attribute is a standard (default) attribute. Standard attributes cannot be deleted."),
			"type": schema.StringAttribute{
				MarkdownDescription: "Identity attribute type, e.g. `string`. The API fills it in when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"multi":      optionalComputedBool("Whether the attribute is multi-valued"),
			"searchable": optionalComputedBool("Whether the attribute is searchable. Searchable attributes must not be `standard` or `multi`."),
			"system": schema.BoolAttribute{
				MarkdownDescription: "Whether the attribute is a system attribute that has no source and is not configurable",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sources_json": schema.StringAttribute{
				MarkdownDescription: "Sources the attribute value is derived from, as a JSON array of objects with `type` (e.g. `rule`) and `properties`.",
				Optional:            true,
				Validators:          []validator.String{jsonArrayStringValidator{}},
			},
		},
	}
}

func (r *IdentityAttributeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// identityAttributeFromModel builds the request body from the plan. Unknown values are omitted, so
// the API applies its defaults on create. For updates the plan has every value, since unset
// optional and computed attributes keep their state value.
func identityAttributeFromModel(data IdentityAttributeModel) *IdentityAttribute {
	attribute := &IdentityAttribute{
		Name:        data.Name.ValueString(),
		DisplayName: stringPointer(data.DisplayName),
		Standard:    boolPointer(data.Standard),
		Type:        stringPointer(data.Type),
		Multi:       boolPointer(data.Multi),
		Searchable:  boolPointer(data.Searchable),
		System:      boolPointer(data.System),
		Sources:     json.RawMessage("[]"),
	}
	if !data.SourcesJSON.IsNull() && !data.SourcesJSON.IsUnknown() && data.SourcesJSON.ValueString() != "" {
		attribute.Sources = json.RawMessage(data.SourcesJSON.ValueString())
	}
	return attribute
}

func identityAttributeBool(value *bool) bool {
	return value != nil && *value
}

func identityAttributeString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// identityAttributeResolveApply keeps the planned values and resolves unknowns from the API.
func identityAttributeResolveApply(data *IdentityAttributeModel, attribute *IdentityAttribute) {
	data.ID = types.StringValue(data.Name.ValueString())
	data.DisplayName = computedStringFromAPI(data.DisplayName, identityAttributeString(attribute.DisplayName))
	data.Type = computedStringFromAPI(data.Type, identityAttributeString(attribute.Type))
	data.Standard = computedBoolFromAPI(data.Standard, attribute.Standard)
	data.Multi = computedBoolFromAPI(data.Multi, attribute.Multi)
	data.Searchable = computedBoolFromAPI(data.Searchable, attribute.Searchable)
	data.System = computedBoolFromAPI(data.System, attribute.System)
}

// setIdentityAttributeState refreshes the model from the API.
func setIdentityAttributeState(data *IdentityAttributeModel, attribute *IdentityAttribute) {
	data.ID = types.StringValue(attribute.Name)
	data.Name = types.StringValue(attribute.Name)
	data.DisplayName = types.StringValue(identityAttributeString(attribute.DisplayName))
	data.Type = types.StringValue(identityAttributeString(attribute.Type))
	data.Standard = types.BoolValue(identityAttributeBool(attribute.Standard))
	data.Multi = types.BoolValue(identityAttributeBool(attribute.Multi))
	data.Searchable = types.BoolValue(identityAttributeBool(attribute.Searchable))
	data.System = types.BoolValue(identityAttributeBool(attribute.System))
	data.SourcesJSON = formDefinitionJSONState(data.SourcesJSON, attribute.Sources)
}

func (r *IdentityAttributeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IdentityAttributeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateIdentityAttribute(ctx, identityAttributeFromModel(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create identity attribute: %s", err))
		return
	}
	identityAttributeResolveApply(&data, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityAttributeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IdentityAttributeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	attribute, err := client.GetIdentityAttribute(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read identity attribute: %s", err))
		return
	}
	setIdentityAttributeState(&data, attribute)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityAttributeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IdentityAttributeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// Every field of the attribute is modeled, so the full object from the plan is sent.
	updated, err := client.UpdateIdentityAttribute(ctx, data.Name.ValueString(), identityAttributeFromModel(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update identity attribute: %s", err))
		return
	}
	identityAttributeResolveApply(&data, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityAttributeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IdentityAttributeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteIdentityAttribute(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete identity attribute (standard and system attributes cannot be deleted): %s", err))
	}
}

func (r *IdentityAttributeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

var _ datasource.DataSource = &IdentityAttributeDataSource{}

func NewIdentityAttributeDataSource() datasource.DataSource {
	return &IdentityAttributeDataSource{}
}

type IdentityAttributeDataSource struct {
	client *Config
}

func (d *IdentityAttributeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_attribute"
}

func (d *IdentityAttributeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up an identity attribute by its technical name.",
		Attributes: map[string]dsschema.Attribute{
			"id":           dsschema.StringAttribute{MarkdownDescription: "Identity attribute ID, the same as `name`", Computed: true},
			"name":         dsschema.StringAttribute{MarkdownDescription: "Technical name of the identity attribute", Required: true},
			"display_name": dsschema.StringAttribute{MarkdownDescription: "Business-friendly name of the identity attribute", Computed: true},
			"standard":     dsschema.BoolAttribute{MarkdownDescription: "Whether the attribute is a standard attribute", Computed: true},
			"type":         dsschema.StringAttribute{MarkdownDescription: "Identity attribute type", Computed: true},
			"multi":        dsschema.BoolAttribute{MarkdownDescription: "Whether the attribute is multi-valued", Computed: true},
			"searchable":   dsschema.BoolAttribute{MarkdownDescription: "Whether the attribute is searchable", Computed: true},
			"system":       dsschema.BoolAttribute{MarkdownDescription: "Whether the attribute is a system attribute", Computed: true},
			"sources_json": dsschema.StringAttribute{MarkdownDescription: "Sources of the attribute value as a JSON array", Computed: true},
		},
	}
}

func (d *IdentityAttributeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IdentityAttributeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IdentityAttributeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	attribute, err := client.GetIdentityAttribute(ctx, data.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Identity attribute not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read identity attribute: %s", err))
		return
	}
	setIdentityAttributeState(&data, attribute)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
