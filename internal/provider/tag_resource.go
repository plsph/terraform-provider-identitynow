package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewTagResource)
	registerDataSource(NewTagDataSource)
}

// Tag is a tag as returned by the /v2026/tags API.
type Tag struct {
	ID              string        `json:"id,omitempty"`
	Name            string        `json:"name"`
	Created         string        `json:"created,omitempty"`
	Modified        string        `json:"modified,omitempty"`
	TagCategoryRefs []*ObjectInfo `json:"tagCategoryRefs,omitempty"`
}

func (c *Client) GetTag(ctx context.Context, id string) (*Tag, error) {
	var tag Tag
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/tags/%s", id), nil, &tag); err != nil {
		return nil, err
	}
	return &tag, nil
}

func (c *Client) GetTagByName(ctx context.Context, name string) (*Tag, error) {
	return findByName(ctx, c, "/v2026/tags", "tag", name, func(t Tag) string { return t.Name })
}

func (c *Client) CreateTag(ctx context.Context, tag *Tag) (*Tag, error) {
	var created Tag
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/tags", tag, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) DeleteTag(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/tags/%s", id), nil, nil)
}

var _ resource.Resource = &TagResource{}
var _ resource.ResourceWithImportState = &TagResource{}

func NewTagResource() resource.Resource {
	return &TagResource{}
}

type TagResource struct {
	client *Config
}

type TagModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Created         types.String `tfsdk:"created"`
	Modified        types.String `tfsdk:"modified"`
	TagCategoryRefs types.List   `tfsdk:"tag_category_refs"`
}

func (r *TagResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *TagResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a tag that can be assigned to objects with `identitynow_tagged_object`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Tag ID",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Tag name. Tags cannot be renamed, changing it forces a new tag.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Creation date",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modified": schema.StringAttribute{
				MarkdownDescription: "Last modification date",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tag_category_refs": schema.ListNestedAttribute{
				MarkdownDescription: "Objects the tag is assigned to",
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (r *TagResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func setTagState(ctx context.Context, data *TagModel, tag *Tag, diags *diag.Diagnostics) {
	data.ID = types.StringValue(tag.ID)
	data.Name = types.StringValue(tag.Name)
	data.Created = types.StringValue(tag.Created)
	data.Modified = types.StringValue(tag.Modified)
	data.TagCategoryRefs = objectInfoSliceState(ctx, tag.TagCategoryRefs, diags)
}

func (r *TagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateTag(ctx, &Tag{Name: data.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tag: %s", err))
		return
	}
	setTagState(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	tag, err := client.GetTag(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tag: %s", err))
		return
	}
	setTagState(ctx, &data, tag, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is never called with changes, since every configurable attribute forces replacement.
func (r *TagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteTag(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete tag: %s", err))
	}
}

func (r *TagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &TagDataSource{}
var _ datasource.DataSourceWithValidateConfig = &TagDataSource{}

func NewTagDataSource() datasource.DataSource {
	return &TagDataSource{}
}

type TagDataSource struct {
	client *Config
}

func (d *TagDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (d *TagDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a tag by ID or name.",
		Attributes: map[string]dsschema.Attribute{
			"id": dsschema.StringAttribute{
				MarkdownDescription: "Tag ID. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"name": dsschema.StringAttribute{
				MarkdownDescription: "Tag name. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"created":  dsschema.StringAttribute{MarkdownDescription: "Creation date", Computed: true},
			"modified": dsschema.StringAttribute{MarkdownDescription: "Last modification date", Computed: true},
			"tag_category_refs": dsschema.ListNestedAttribute{
				MarkdownDescription: "Objects the tag is assigned to",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"id":   dsschema.StringAttribute{Computed: true},
						"type": dsschema.StringAttribute{Computed: true},
						"name": dsschema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *TagDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *TagDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TagModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var tag *Tag
	if !data.ID.IsNull() {
		tag, err = client.GetTag(ctx, data.ID.ValueString())
	} else {
		tag, err = client.GetTagByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Tag not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tag: %s", err))
		return
	}
	setTagState(ctx, &data, tag, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
