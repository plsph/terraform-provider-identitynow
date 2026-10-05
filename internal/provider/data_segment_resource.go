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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewDataSegmentResource)
	registerDataSource(NewDataSegmentDataSource)
}

// DataSegment is a data access segment as returned by the experimental /v2026/data-segments API.
type DataSegment struct {
	ID              string                 `json:"id,omitempty"`
	Name            string                 `json:"name"`
	Created         string                 `json:"created,omitempty"`
	Modified        string                 `json:"modified,omitempty"`
	Description     string                 `json:"description,omitempty"`
	Scopes          []interface{}          `json:"scopes,omitempty"`
	MemberSelection []DataSegmentSelection `json:"memberSelection,omitempty"`
	MemberFilter    map[string]interface{} `json:"memberFilter,omitempty"`
	Membership      string                 `json:"membership,omitempty"`
	Enabled         *bool                  `json:"enabled,omitempty"`
	Published       *bool                  `json:"published,omitempty"`
}

// DataSegmentSelection is a typed reference to an identity selected as segment member.
type DataSegmentSelection struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id"`
}

func (c *Client) GetDataSegment(ctx context.Context, id string) (*DataSegment, error) {
	var segment DataSegment
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/data-segments/%s", id), nil, &segment, withExperimental()); err != nil {
		return nil, err
	}
	return &segment, nil
}

func (c *Client) CreateDataSegment(ctx context.Context, segment *DataSegment) (*DataSegment, error) {
	var created DataSegment
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/data-segments", segment, &created, withExperimental()); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchDataSegment(ctx context.Context, id string, ops []jsonPatchOp) (*DataSegment, error) {
	var updated DataSegment
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/data-segments/%s", id), ops, &updated, withExperimental(), withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

// PublishDataSegment publishes the pending changes of one segment, so its segmentation is applied.
// publishAll=false restricts publishing to the IDs in the body; the API default publishes all segments.
func (c *Client) PublishDataSegment(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, apiPath("/v2026/data-segments/%s", id)+"?publishAll=false", []string{id}, nil, withExperimental())
}

// DeleteDataSegment deletes the published or the unpublished version of a segment.
func (c *Client) DeleteDataSegment(ctx context.Context, id string, published bool) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/data-segments/%s", id)+fmt.Sprintf("?published=%t", published), nil, nil, withExperimental())
}

var _ resource.Resource = &DataSegmentResource{}
var _ resource.ResourceWithImportState = &DataSegmentResource{}

func NewDataSegmentResource() resource.Resource {
	return &DataSegmentResource{}
}

type DataSegmentResource struct {
	client *Config
}

type DataSegmentModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Membership       types.String `tfsdk:"membership"`
	MemberFilterJSON types.String `tfsdk:"member_filter_json"`
	MemberSelection  types.List   `tfsdk:"member_selection"`
	ScopesJSON       types.String `tfsdk:"scopes_json"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Publish          types.Bool   `tfsdk:"publish"`
	Published        types.Bool   `tfsdk:"published"`
	Created          types.String `tfsdk:"created"`
	Modified         types.String `tfsdk:"modified"`
}

type DataSegmentSelectionModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
}

var dataSegmentSelectionObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"type": types.StringType,
	"id":   types.StringType,
}}

func (r *DataSegmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_segment"
}

func (r *DataSegmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a data access segment, which limits the entitlements, identities and certifications its members can see. Uses an experimental API. Changes only take effect once the segment is published, see `publish`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Data segment ID",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Segment business name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Segment description",
				Optional:            true,
			},
			"membership": schema.StringAttribute{
				MarkdownDescription: "How members are chosen: `ALL`, `FILTER` (identities matching `member_filter_json`) or `SELECTION` (the identities in `member_selection`). When not set, the value chosen by the API is kept.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"member_filter_json": schema.StringAttribute{
				MarkdownDescription: "Member filter for the `FILTER` membership as a JSON object with an `expression` (`operator` `AND` or `EQUALS`, `attribute`, `value` with `type` and `value`, and one level of `children`). Use `jsonencode()`. Fields the API adds do not cause a diff.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"scopes_json": schema.StringAttribute{
				MarkdownDescription: "Scopes of the segment as a JSON array. Each scope has a `scope` (`ENTITLEMENT`, `CERTIFICATION`, `IDENTITY` or `ENTITLEMENTREQUEST`), a `visibility` (`ALL`, `FILTER`, `SELECTION` or `UNSEGMENTED`) and a `scopeFilter` or `scopeSelection`. Use `jsonencode()`. Fields the API adds do not cause a diff.",
				Optional:            true,
				Validators:          []validator.String{jsonArrayStringValidator{}},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the segment is active, inactive segments have no effect. When not set, the value chosen by the API is kept.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"publish": schema.BoolAttribute{
				MarkdownDescription: "Whether the provider publishes the segment after every create and update, so the changes are applied. Defaults to `false`, which leaves the changes unpublished.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"published": schema.BoolAttribute{
				MarkdownDescription: "Whether the segment as read from the API is published",
				Computed:            true,
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
		Blocks: map[string]schema.Block{
			"member_selection": schema.ListNestedBlock{
				MarkdownDescription: "Identity selected as member for the `SELECTION` membership",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{MarkdownDescription: "Identity ID", Required: true},
						"type": schema.StringAttribute{
							MarkdownDescription: "Object type, `IDENTITY` (default)",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("IDENTITY"),
						},
					},
				},
			},
		},
	}
}

func (r *DataSegmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func dataSegmentSelectionValue(ctx context.Context, list types.List, diags *diag.Diagnostics) []DataSegmentSelection {
	selection := []DataSegmentSelection{}
	if list.IsNull() || list.IsUnknown() {
		return selection
	}
	var models []DataSegmentSelectionModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	for _, model := range models {
		selection = append(selection, DataSegmentSelection{Type: model.Type.ValueString(), ID: model.ID.ValueString()})
	}
	return selection
}

// dataSegmentSelectionState maps the selected members in the prior order.
func dataSegmentSelectionState(ctx context.Context, selection []DataSegmentSelection, prior types.List, diags *diag.Diagnostics) types.List {
	if len(selection) == 0 {
		return types.ListNull(dataSegmentSelectionObjectType)
	}
	ordered := orderByPriorIDs(selection, dataSegmentSelectionValue(ctx, prior, diags), func(s DataSegmentSelection) string { return s.ID })
	models := make([]DataSegmentSelectionModel, 0, len(ordered))
	for _, member := range ordered {
		memberType := member.Type
		if memberType == "" {
			memberType = "IDENTITY"
		}
		models = append(models, DataSegmentSelectionModel{Type: types.StringValue(memberType), ID: types.StringValue(member.ID)})
	}
	list, d := types.ListValueFrom(ctx, dataSegmentSelectionObjectType, models)
	diags.Append(d...)
	return list
}

// dataSegmentFromModel converts the model to the create request body.
func dataSegmentFromModel(ctx context.Context, data DataSegmentModel, diags *diag.Diagnostics) *DataSegment {
	segment := &DataSegment{
		Name:            data.Name.ValueString(),
		Description:     data.Description.ValueString(),
		MemberSelection: dataSegmentSelectionValue(ctx, data.MemberSelection, diags),
		Enabled:         boolPointer(data.Enabled),
	}
	if membership := stringPointer(data.Membership); membership != nil {
		segment.Membership = *membership
	}
	if filter, ok := campaignTemplateJSONValue(data.MemberFilterJSON, "member_filter_json", diags).(map[string]interface{}); ok {
		segment.MemberFilter = filter
	}
	if scopes, ok := campaignTemplateJSONValue(data.ScopesJSON, "scopes_json", diags).([]interface{}); ok {
		segment.Scopes = scopes
	}
	return segment
}

// dataSegmentPatchOps returns JSON Patch operations for the fields changed between state and plan.
func dataSegmentPatchOps(ctx context.Context, plan, state DataSegmentModel, diags *diag.Diagnostics) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	b.replaceIfChanged(plan.Membership, state.Membership, "/membership", plan.Membership.ValueString())
	b.replaceIfChanged(plan.Enabled, state.Enabled, "/enabled", plan.Enabled.ValueBool())
	b.replaceIfChanged(plan.MemberSelection, state.MemberSelection, "/memberSelection", dataSegmentSelectionValue(ctx, plan.MemberSelection, diags))
	if campaignTemplateJSONChanged(plan.MemberFilterJSON, state.MemberFilterJSON) {
		if filter := campaignTemplateJSONValue(plan.MemberFilterJSON, "member_filter_json", diags); filter != nil {
			b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/memberFilter", Value: filter})
		} else {
			b.ops = append(b.ops, jsonPatchOp{Op: "remove", Path: "/memberFilter"})
		}
	}
	if campaignTemplateJSONChanged(plan.ScopesJSON, state.ScopesJSON) {
		scopes := campaignTemplateJSONValue(plan.ScopesJSON, "scopes_json", diags)
		if scopes == nil {
			scopes = []interface{}{}
		}
		b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/scopes", Value: scopes})
	}
	return b.ops
}

// setDataSegmentComputed resolves the computed attributes after apply from the API response. Values
// known from the plan (kept from state) are not changed.
func setDataSegmentComputed(data *DataSegmentModel, segment *DataSegment) {
	data.ID = computedStringFromAPI(data.ID, segment.ID)
	data.Created = computedStringFromAPI(data.Created, segment.Created)
	data.Modified = types.StringValue(segment.Modified)
	data.Membership = computedStringFromAPI(data.Membership, segment.Membership)
	data.Enabled = computedBoolFromAPI(data.Enabled, segment.Enabled)
	data.Published = types.BoolValue(segment.Published != nil && *segment.Published)
}

// setDataSegmentState maps an API segment onto the model when reading.
func setDataSegmentState(ctx context.Context, data *DataSegmentModel, segment *DataSegment, diags *diag.Diagnostics) {
	data.ID = types.StringValue(segment.ID)
	data.Name = types.StringValue(segment.Name)
	data.Description = optionalStringState(data.Description, segment.Description)
	data.Membership = types.StringValue(segment.Membership)
	data.MemberFilterJSON = jsonSubsetState(data.MemberFilterJSON, segment.MemberFilter)
	data.MemberSelection = dataSegmentSelectionState(ctx, segment.MemberSelection, data.MemberSelection, diags)
	data.ScopesJSON = jsonSubsetState(data.ScopesJSON, segment.Scopes)
	data.Enabled = types.BoolValue(segment.Enabled != nil && *segment.Enabled)
	data.Published = types.BoolValue(segment.Published != nil && *segment.Published)
	data.Created = types.StringValue(segment.Created)
	data.Modified = types.StringValue(segment.Modified)
}

// publish publishes the segment when configured and reads it back to resolve the computed values.
func (r *DataSegmentResource) publish(ctx context.Context, client *Client, data *DataSegmentModel, segment *DataSegment, diags *diag.Diagnostics) {
	if !data.Publish.ValueBool() {
		setDataSegmentComputed(data, segment)
		return
	}
	if err := client.PublishDataSegment(ctx, segment.ID); err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to publish data segment %s: %s", segment.ID, err))
		return
	}
	published, err := client.GetDataSegment(ctx, segment.ID)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read published data segment %s: %s", segment.ID, err))
		return
	}
	setDataSegmentComputed(data, published)
}

func (r *DataSegmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DataSegmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	segment := dataSegmentFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateDataSegment(ctx, segment)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create data segment: %s", err))
		return
	}
	r.publish(ctx, client, &data, created, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		// Keep the created segment in state, so it is not orphaned when publishing fails.
		setDataSegmentComputed(&data, created)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DataSegmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DataSegmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	segment, err := client.GetDataSegment(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read data segment: %s", err))
		return
	}
	setDataSegmentState(ctx, &data, segment, &resp.Diagnostics)
	if data.Publish.IsNull() {
		// Imported segments are not published by the provider unless configured.
		data.Publish = types.BoolValue(false)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DataSegmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state DataSegmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := dataSegmentPatchOps(ctx, plan, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var segment *DataSegment
	if len(ops) == 0 {
		segment, err = client.GetDataSegment(ctx, plan.ID.ValueString())
	} else {
		segment, err = client.PatchDataSegment(ctx, plan.ID.ValueString(), ops)
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update data segment: %s", err))
		return
	}
	r.publish(ctx, client, &plan, segment, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the unpublished version of the segment and, when the segment was published, the
// published version as well.
func (r *DataSegmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DataSegmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteDataSegment(ctx, data.ID.ValueString(), false); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete data segment: %s", err))
		return
	}
	if data.Published.ValueBool() || data.Publish.ValueBool() {
		if err := client.DeleteDataSegment(ctx, data.ID.ValueString(), true); err != nil && !isNotFound(err) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete published data segment: %s", err))
		}
	}
}

func (r *DataSegmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &DataSegmentDataSource{}

func NewDataSegmentDataSource() datasource.DataSource {
	return &DataSegmentDataSource{}
}

type DataSegmentDataSource struct {
	client *Config
}

// DataSegmentDataSourceModel is the data source model, which has no publish setting.
type DataSegmentDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Membership       types.String `tfsdk:"membership"`
	MemberFilterJSON types.String `tfsdk:"member_filter_json"`
	MemberSelection  types.List   `tfsdk:"member_selection"`
	ScopesJSON       types.String `tfsdk:"scopes_json"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Published        types.Bool   `tfsdk:"published"`
	Created          types.String `tfsdk:"created"`
	Modified         types.String `tfsdk:"modified"`
}

func (d *DataSegmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_segment"
}

func (d *DataSegmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a data access segment by ID. Uses an experimental API.",
		Attributes: map[string]dsschema.Attribute{
			"id":                 dsschema.StringAttribute{MarkdownDescription: "Data segment ID", Required: true},
			"name":               dsschema.StringAttribute{MarkdownDescription: "Segment business name", Computed: true},
			"description":        dsschema.StringAttribute{MarkdownDescription: "Segment description", Computed: true},
			"membership":         dsschema.StringAttribute{MarkdownDescription: "How members are chosen: `ALL`, `FILTER` or `SELECTION`", Computed: true},
			"member_filter_json": dsschema.StringAttribute{MarkdownDescription: "Member filter as a JSON object", Computed: true},
			"member_selection": dsschema.ListNestedAttribute{
				MarkdownDescription: "Identities selected as members",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"id":   dsschema.StringAttribute{MarkdownDescription: "Identity ID", Computed: true},
						"type": dsschema.StringAttribute{MarkdownDescription: "Object type", Computed: true},
					},
				},
			},
			"scopes_json": dsschema.StringAttribute{MarkdownDescription: "Scopes of the segment as a JSON array", Computed: true},
			"enabled":     dsschema.BoolAttribute{MarkdownDescription: "Whether the segment is active", Computed: true},
			"published":   dsschema.BoolAttribute{MarkdownDescription: "Whether the segment is published", Computed: true},
			"created":     dsschema.StringAttribute{MarkdownDescription: "Creation date", Computed: true},
			"modified":    dsschema.StringAttribute{MarkdownDescription: "Last modification date", Computed: true},
		},
	}
}

func (d *DataSegmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DataSegmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DataSegmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	segment, err := client.GetDataSegment(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read data segment: %s", err))
		return
	}
	data := DataSegmentModel{
		Description:      types.StringNull(),
		MemberFilterJSON: types.StringNull(),
		MemberSelection:  types.ListNull(dataSegmentSelectionObjectType),
		ScopesJSON:       types.StringNull(),
	}
	setDataSegmentState(ctx, &data, segment, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &DataSegmentDataSourceModel{
		ID: data.ID, Name: data.Name, Description: data.Description, Membership: data.Membership,
		MemberFilterJSON: data.MemberFilterJSON, MemberSelection: data.MemberSelection, ScopesJSON: data.ScopesJSON,
		Enabled: data.Enabled, Published: data.Published, Created: data.Created, Modified: data.Modified,
	})...)
}
