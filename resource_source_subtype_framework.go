package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewSourceSubtypeResource)
	registerDataSource(NewSourceSubtypeDataSource)
}

// SourceSubtype is a machine account subtype as used by the experimental /v2026/source-subtypes API.
type SourceSubtype struct {
	ID            string                  `json:"id,omitempty"`
	SourceID      string                  `json:"sourceId,omitempty"`
	TechnicalName string                  `json:"technicalName,omitempty"`
	DisplayName   string                  `json:"displayName"`
	Description   string                  `json:"description"`
	Type          string                  `json:"type,omitempty"`
	Created       string                  `json:"created,omitempty"`
	Modified      string                  `json:"modified,omitempty"`
	SystemManaged *bool                   `json:"systemManaged,omitempty"`
	Source        *SourceSubtypeSourceRef `json:"source,omitempty"`
}

// SourceSubtypeSourceRef is the source reference of a subtype.
type SourceSubtypeSourceRef struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// sourceSubtypeSourceID returns the source ID of a subtype, from sourceId or the source reference.
func sourceSubtypeSourceID(subtype *SourceSubtype) string {
	if subtype.SourceID == "" && subtype.Source != nil {
		return subtype.Source.ID
	}
	return subtype.SourceID
}

func (c *Client) GetSourceSubtype(ctx context.Context, id string) (*SourceSubtype, error) {
	var subtype SourceSubtype
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/source-subtypes/%s", id), nil, &subtype, withExperimental()); err != nil {
		return nil, err
	}
	return &subtype, nil
}

// GetSourceSubtypeByTechnicalName returns the subtype of a source with the given technical name.
// The list is filtered by source server side and matched by technical name client side.
func (c *Client) GetSourceSubtypeByTechnicalName(ctx context.Context, sourceID, technicalName string) (*SourceSubtype, error) {
	subtypes, err := listAllPages[SourceSubtype](ctx, c, "/v2026/source-subtypes", url.Values{"filters": {eqFilter("source.id", sourceID)}}, withExperimental())
	if err != nil {
		return nil, err
	}
	for i := range subtypes {
		if subtypes[i].TechnicalName == technicalName && sourceSubtypeSourceID(&subtypes[i]) == sourceID {
			return &subtypes[i], nil
		}
	}
	return nil, &NotFoundError{fmt.Sprintf("subtype %q of source %q not found", technicalName, sourceID)}
}

func (c *Client) CreateSourceSubtype(ctx context.Context, subtype *SourceSubtype) (*SourceSubtype, error) {
	var created SourceSubtype
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/source-subtypes", subtype, &created, withExperimental()); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchSourceSubtype(ctx context.Context, id string, ops []jsonPatchOp) (*SourceSubtype, error) {
	var updated SourceSubtype
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/source-subtypes/%s", id), ops, &updated, withExperimental(), withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteSourceSubtype(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/source-subtypes/%s", id), nil, nil, withExperimental())
}

var _ resource.Resource = &SourceSubtypeResource{}
var _ resource.ResourceWithImportState = &SourceSubtypeResource{}

func NewSourceSubtypeResource() resource.Resource {
	return &SourceSubtypeResource{}
}

type SourceSubtypeResource struct {
	client *Config
}

type SourceSubtypeModel struct {
	ID            types.String `tfsdk:"id"`
	SourceID      types.String `tfsdk:"source_id"`
	TechnicalName types.String `tfsdk:"technical_name"`
	DisplayName   types.String `tfsdk:"display_name"`
	Description   types.String `tfsdk:"description"`
	Type          types.String `tfsdk:"type"`
	SystemManaged types.Bool   `tfsdk:"system_managed"`
	Created       types.String `tfsdk:"created"`
	Modified      types.String `tfsdk:"modified"`
}

func (r *SourceSubtypeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_subtype"
}

func (r *SourceSubtypeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a machine account subtype of a source. Uses an experimental API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Subtype ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source_id": schema.StringAttribute{
				MarkdownDescription: "ID of the source the subtype belongs to. Changing it forces a new subtype.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"technical_name": schema.StringAttribute{
				MarkdownDescription: "Technical name of the subtype. Changing it forces a new subtype.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the subtype.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the subtype.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of the subtype, `MACHINE` or unset. Changing it forces a new subtype.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"system_managed": schema.BoolAttribute{
				MarkdownDescription: "Whether the subtype is managed by the system.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Creation date.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modified": schema.StringAttribute{
				MarkdownDescription: "Last modification date.",
				Computed:            true,
			},
		},
	}
}

func (r *SourceSubtypeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// setSourceSubtypeState maps an API subtype onto the model. With refresh, all values are taken
// from the API; otherwise known planned values are kept and only unknown values are resolved.
func setSourceSubtypeState(data *SourceSubtypeModel, subtype *SourceSubtype, refresh bool) {
	if refresh {
		data.ID = types.StringValue(subtype.ID)
		if sourceID := sourceSubtypeSourceID(subtype); sourceID != "" {
			data.SourceID = types.StringValue(sourceID)
		}
		data.TechnicalName = types.StringValue(subtype.TechnicalName)
		data.DisplayName = types.StringValue(subtype.DisplayName)
		data.Description = types.StringValue(subtype.Description)
		data.Type = stringValueOrNull(subtype.Type)
		data.SystemManaged = types.BoolValue(subtype.SystemManaged != nil && *subtype.SystemManaged)
		data.Created = stringValueOrNull(subtype.Created)
		data.Modified = stringValueOrNull(subtype.Modified)
		return
	}
	if data.ID.IsUnknown() {
		data.ID = types.StringValue(subtype.ID)
	}
	if data.Type.IsUnknown() {
		data.Type = stringValueOrNull(subtype.Type)
	}
	data.SystemManaged = computedBoolFromAPI(data.SystemManaged, subtype.SystemManaged)
	if data.Created.IsUnknown() {
		data.Created = stringValueOrNull(subtype.Created)
	}
	data.Modified = stringValueOrNull(subtype.Modified)
}

// sourceSubtypePatch returns the JSON Patch operations for the changed patchable attributes.
func sourceSubtypePatch(plan, state SourceSubtypeModel) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.DisplayName, state.DisplayName, "/displayName", plan.DisplayName.ValueString())
	b.replaceIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	return b.ops
}

func (r *SourceSubtypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SourceSubtypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateSourceSubtype(ctx, &SourceSubtype{
		SourceID:      data.SourceID.ValueString(),
		TechnicalName: data.TechnicalName.ValueString(),
		DisplayName:   data.DisplayName.ValueString(),
		Description:   data.Description.ValueString(),
		Type:          data.Type.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create source subtype: %s", err))
		return
	}
	setSourceSubtypeState(&data, created, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceSubtypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SourceSubtypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	subtype, err := client.GetSourceSubtype(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source subtype: %s", err))
		return
	}
	setSourceSubtypeState(&data, subtype, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceSubtypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state SourceSubtypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := sourceSubtypePatch(data, state)
	if len(ops) == 0 {
		data.Modified = state.Modified
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.PatchSourceSubtype(ctx, data.ID.ValueString(), ops)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update source subtype: %s", err))
		return
	}
	setSourceSubtypeState(&data, updated, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceSubtypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SourceSubtypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteSourceSubtype(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete source subtype: %s", err))
	}
}

func (r *SourceSubtypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &SourceSubtypeDataSource{}
var _ datasource.DataSourceWithValidateConfig = &SourceSubtypeDataSource{}

func NewSourceSubtypeDataSource() datasource.DataSource {
	return &SourceSubtypeDataSource{}
}

type SourceSubtypeDataSource struct {
	client *Config
}

func (d *SourceSubtypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_subtype"
}

func (d *SourceSubtypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a machine account subtype by ID, or by source ID and technical name. Uses an experimental API.",
		Attributes: map[string]dsschema.Attribute{
			"id":             dsschema.StringAttribute{MarkdownDescription: "Subtype ID. Set either `id`, or `source_id` and `technical_name`.", Optional: true, Computed: true},
			"source_id":      dsschema.StringAttribute{MarkdownDescription: "ID of the source. Required together with `technical_name` when `id` is not set.", Optional: true, Computed: true},
			"technical_name": dsschema.StringAttribute{MarkdownDescription: "Technical name of the subtype. Required together with `source_id` when `id` is not set.", Optional: true, Computed: true},
			"display_name":   dsschema.StringAttribute{MarkdownDescription: "Display name of the subtype.", Computed: true},
			"description":    dsschema.StringAttribute{MarkdownDescription: "Description of the subtype.", Computed: true},
			"type":           dsschema.StringAttribute{MarkdownDescription: "Type of the subtype, `MACHINE` or null.", Computed: true},
			"system_managed": dsschema.BoolAttribute{MarkdownDescription: "Whether the subtype is managed by the system.", Computed: true},
			"created":        dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"modified":       dsschema.StringAttribute{MarkdownDescription: "Last modification date.", Computed: true},
		},
	}
}

func (d *SourceSubtypeDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var data SourceSubtypeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	byID := !data.ID.IsNull()
	byName := !data.SourceID.IsNull() || !data.TechnicalName.IsNull()
	if byID == byName || (byName && (data.SourceID.IsNull() || data.TechnicalName.IsNull())) {
		resp.Diagnostics.AddError("Invalid configuration", "Set either id, or both source_id and technical_name.")
	}
}

func (d *SourceSubtypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourceSubtypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SourceSubtypeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var subtype *SourceSubtype
	if !data.ID.IsNull() {
		subtype, err = client.GetSourceSubtype(ctx, data.ID.ValueString())
	} else {
		subtype, err = client.GetSourceSubtypeByTechnicalName(ctx, data.SourceID.ValueString(), data.TechnicalName.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Source subtype not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source subtype: %s", err))
		return
	}
	setSourceSubtypeState(&data, subtype, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
