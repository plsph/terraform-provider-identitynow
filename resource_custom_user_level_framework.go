package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewCustomUserLevelResource)
	registerDataSource(NewCustomUserLevelDataSource)
}

// customUserLevelStatusActive is the status of a published user level.
const customUserLevelStatusActive = "ACTIVE"

// CustomUserLevel is a user level as returned by the experimental
// /v2026/authorization/custom-user-levels API.
type CustomUserLevel struct {
	ID                        string                    `json:"id,omitempty"`
	Name                      string                    `json:"name"`
	Description               string                    `json:"description"`
	Owner                     *CustomUserLevelOwner     `json:"owner,omitempty"`
	RightSets                 []CustomUserLevelRightSet `json:"rightSets,omitempty"`
	Status                    string                    `json:"status,omitempty"`
	Created                   string                    `json:"created,omitempty"`
	Modified                  string                    `json:"modified,omitempty"`
	AssociatedIdentitiesCount *int64                    `json:"associatedIdentitiesCount,omitempty"`
}

// CustomUserLevelRequest is the body of the create request. Right sets are referenced by ID.
type CustomUserLevelRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Owner       *CustomUserLevelOwner `json:"owner"`
	RightSets   []string              `json:"rightSets,omitempty"`
}

// CustomUserLevelOwner is the identity that owns a user level.
type CustomUserLevelOwner struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// CustomUserLevelRightSet is a right set assigned to a user level.
type CustomUserLevelRightSet struct {
	ID       string  `json:"id"`
	Name     string  `json:"name,omitempty"`
	ParentID *string `json:"parentId,omitempty"`
}

func (c *Client) GetCustomUserLevel(ctx context.Context, id string) (*CustomUserLevel, error) {
	var level CustomUserLevel
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/authorization/custom-user-levels/%s", id), nil, &level, withExperimental()); err != nil {
		return nil, err
	}
	return &level, nil
}

func (c *Client) CreateCustomUserLevel(ctx context.Context, level *CustomUserLevelRequest) (*CustomUserLevel, error) {
	var created CustomUserLevel
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/authorization/custom-user-levels", level, &created, withExperimental()); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchCustomUserLevel(ctx context.Context, id string, ops []jsonPatchOp) (*CustomUserLevel, error) {
	var updated CustomUserLevel
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/authorization/custom-user-levels/%s", id), ops, &updated, withExperimental(), withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

// PublishCustomUserLevel publishes a draft user level, making it active and assignable.
func (c *Client) PublishCustomUserLevel(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, apiPath("/v2026/authorization/custom-user-levels/%s/publish", id), nil, nil, withExperimental())
}

func (c *Client) DeleteCustomUserLevel(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/authorization/custom-user-levels/%s", id), nil, nil, withExperimental())
}

var _ resource.Resource = &CustomUserLevelResource{}
var _ resource.ResourceWithImportState = &CustomUserLevelResource{}

func NewCustomUserLevelResource() resource.Resource {
	return &CustomUserLevelResource{}
}

type CustomUserLevelResource struct {
	client *Config
}

type CustomUserLevelModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Owner             types.List   `tfsdk:"owner"`
	RightSets         types.Set    `tfsdk:"right_sets"`
	Publish           types.Bool   `tfsdk:"publish"`
	AssignedRightSets types.Set    `tfsdk:"assigned_right_sets"`
	Status            types.String `tfsdk:"status"`
	Created           types.String `tfsdk:"created"`
	Modified          types.String `tfsdk:"modified"`
}

func (r *CustomUserLevelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_user_level"
}

func (r *CustomUserLevelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom user level, a set of UI right sets that can be assigned to identities. Uses an experimental API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "User level ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the user level.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the user level.",
				Required:            true,
			},
			"right_sets": schema.SetAttribute{
				MarkdownDescription: "IDs of the right sets assigned to the user level, see `identitynow_authorization_right_sets`. When only a parent right set is listed, IdentityNow assigns all of its children; these are shown in `assigned_right_sets` and do not cause a diff.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"publish": schema.BoolAttribute{
				MarkdownDescription: "Publish the user level after it is created or updated, making it active and assignable. A published user level cannot be unpublished, so setting it to `false` later has no effect. Defaults to `false`, which leaves a new user level in draft status.",
				Optional:            true,
			},
			"assigned_right_sets": schema.SetAttribute{
				MarkdownDescription: "IDs of all right sets assigned to the user level, including children assigned automatically.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the user level, `DRAFT` or `ACTIVE`.",
				Computed:            true,
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
		Blocks: map[string]schema.Block{
			"owner": schema.ListNestedBlock{
				MarkdownDescription: "Identity that owns the user level. Exactly one block is required.",
				Validators:          []validator.List{listSizeBetween(1, 1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Owner identity ID.",
							Required:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Owner type, `IDENTITY` (the default).",
							Optional:            true,
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Owner display name, filled in by IdentityNow when not set. A configured name is kept as long as the owner ID does not change.",
							Optional:            true,
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (r *CustomUserLevelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// customUserLevelOwnerValue converts the owner block to the API owner.
func customUserLevelOwnerValue(ctx context.Context, owner types.List, diags *diag.Diagnostics) *CustomUserLevelOwner {
	if owner.IsNull() || owner.IsUnknown() {
		return nil
	}
	var owners []OwnerModel
	diags.Append(owner.ElementsAs(ctx, &owners, false)...)
	if len(owners) == 0 {
		return nil
	}
	result := &CustomUserLevelOwner{ID: owners[0].ID.ValueString(), Type: "IDENTITY", Name: owners[0].Name.ValueString()}
	if !owners[0].Type.IsNull() && !owners[0].Type.IsUnknown() {
		result.Type = owners[0].Type.ValueString()
	}
	return result
}

// customUserLevelOwnerState converts the API owner to the owner block. Planned values are kept and
// unknown type and name values are resolved from the API.
func customUserLevelOwnerState(ctx context.Context, planned types.List, owner *CustomUserLevelOwner, diags *diag.Diagnostics) types.List {
	if owner == nil {
		return types.ListNull(objectInfoObjectType)
	}
	model := OwnerModel{ID: types.StringValue(owner.ID), Type: types.StringValue(owner.Type), Name: types.StringValue(owner.Name)}
	if owner.Type == "" {
		model.Type = types.StringValue("IDENTITY")
	}
	if !planned.IsNull() && !planned.IsUnknown() {
		var owners []OwnerModel
		diags.Append(planned.ElementsAs(ctx, &owners, false)...)
		if len(owners) == 1 {
			model.ID = owners[0].ID
			if !owners[0].Type.IsUnknown() && !owners[0].Type.IsNull() {
				model.Type = owners[0].Type
			}
			if !owners[0].Name.IsUnknown() && !owners[0].Name.IsNull() {
				model.Name = owners[0].Name
			}
		}
	}
	list, d := types.ListValueFrom(ctx, objectInfoObjectType, []OwnerModel{model})
	diags.Append(d...)
	return list
}

// customUserLevelOwnerRefreshState returns the owner block read from the API. When the owner ID is
// unchanged, the prior type and name are kept: they are Optional and Computed, and a configured name
// that differs from the display name returned by the API must not cause a diff.
func customUserLevelOwnerRefreshState(ctx context.Context, prior types.List, owner *CustomUserLevelOwner, diags *diag.Diagnostics) types.List {
	if owner == nil {
		return types.ListNull(objectInfoObjectType)
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		var owners []OwnerModel
		diags.Append(prior.ElementsAs(ctx, &owners, false)...)
		if len(owners) == 1 && owners[0].ID.ValueString() == owner.ID {
			return customUserLevelOwnerState(ctx, prior, owner, diags)
		}
	}
	return customUserLevelOwnerState(ctx, types.ListNull(objectInfoObjectType), owner, diags)
}

// customUserLevelRightSetIDs returns the sorted IDs of the assigned right sets.
func customUserLevelRightSetIDs(rightSets []CustomUserLevelRightSet) []string {
	ids := make([]string, 0, len(rightSets))
	for _, rightSet := range rightSets {
		ids = append(ids, rightSet.ID)
	}
	sort.Strings(ids)
	return ids
}

// customUserLevelRightSetsState returns the state of right_sets. IdentityNow assigns all children of
// a parent right set that is configured on its own, so the prior value is kept when the assigned
// right sets are the configured ones plus children of configured right sets. A configured parent
// that is not returned as assigned itself counts as assigned when one of its children is assigned.
func customUserLevelRightSetsState(ctx context.Context, prior types.Set, rightSets []CustomUserLevelRightSet, diags *diag.Diagnostics) types.Set {
	if !prior.IsNull() && !prior.IsUnknown() {
		configured := map[string]bool{}
		for _, id := range oauthClientStringSetValue(ctx, prior, diags) {
			configured[id] = true
		}
		assigned := map[string]bool{}
		covered := true
		for _, rightSet := range rightSets {
			assigned[rightSet.ID] = true
			if rightSet.ParentID != nil {
				assigned[*rightSet.ParentID] = true
			}
			if !configured[rightSet.ID] && (rightSet.ParentID == nil || !configured[*rightSet.ParentID]) {
				covered = false
			}
		}
		for id := range configured {
			if !assigned[id] {
				covered = false
			}
		}
		if covered {
			return prior
		}
	}
	return oauthClientStringSetState(ctx, prior, customUserLevelRightSetIDs(rightSets), true, diags)
}

// customUserLevelResolveComputed sets the computed attributes from the API after create or update.
func customUserLevelResolveComputed(ctx context.Context, data *CustomUserLevelModel, level *CustomUserLevel, diags *diag.Diagnostics) {
	if data.ID.IsUnknown() {
		data.ID = types.StringValue(level.ID)
	}
	data.Owner = customUserLevelOwnerState(ctx, data.Owner, level.Owner, diags)
	data.AssignedRightSets = oauthClientStringSetState(ctx, types.SetNull(types.StringType), customUserLevelRightSetIDs(level.RightSets), false, diags)
	data.Status = types.StringValue(level.Status)
	if data.Created.IsUnknown() {
		data.Created = types.StringValue(level.Created)
	}
	data.Modified = types.StringValue(level.Modified)
}

// setCustomUserLevelState refreshes the model from the API. publish is not returned by the API and
// keeps its state value.
func setCustomUserLevelState(ctx context.Context, data *CustomUserLevelModel, level *CustomUserLevel, diags *diag.Diagnostics) {
	data.ID = types.StringValue(level.ID)
	data.Name = types.StringValue(level.Name)
	data.Description = types.StringValue(level.Description)
	data.Owner = customUserLevelOwnerRefreshState(ctx, data.Owner, level.Owner, diags)
	data.RightSets = customUserLevelRightSetsState(ctx, data.RightSets, level.RightSets, diags)
	data.AssignedRightSets = oauthClientStringSetState(ctx, types.SetNull(types.StringType), customUserLevelRightSetIDs(level.RightSets), false, diags)
	data.Status = types.StringValue(level.Status)
	data.Created = types.StringValue(level.Created)
	data.Modified = types.StringValue(level.Modified)
}

// customUserLevelPatchOps returns JSON Patch operations for the changed attributes.
func customUserLevelPatchOps(ctx context.Context, plan, state CustomUserLevelModel, diags *diag.Diagnostics) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	b.replaceIfChanged(plan.Owner, state.Owner, "/owner", customUserLevelOwnerValue(ctx, plan.Owner, diags))
	rightSets := oauthClientStringSetValue(ctx, plan.RightSets, diags)
	if rightSets == nil {
		rightSets = []string{}
	}
	b.replaceIfChanged(plan.RightSets, state.RightSets, "/rightSets", rightSets)
	return b.ops
}

// customUserLevelPublishIfRequested publishes a draft user level when publish is set and returns
// the refreshed user level.
func customUserLevelPublishIfRequested(ctx context.Context, client *Client, data CustomUserLevelModel, level *CustomUserLevel) (*CustomUserLevel, error) {
	if !data.Publish.ValueBool() || level.Status == customUserLevelStatusActive {
		return level, nil
	}
	if err := client.PublishCustomUserLevel(ctx, level.ID); err != nil {
		return nil, err
	}
	return client.GetCustomUserLevel(ctx, level.ID)
}

func (r *CustomUserLevelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CustomUserLevelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request := &CustomUserLevelRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Owner:       customUserLevelOwnerValue(ctx, data.Owner, &resp.Diagnostics),
		RightSets:   oauthClientStringSetValue(ctx, data.RightSets, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateCustomUserLevel(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create custom user level: %s", err))
		return
	}
	// Save the draft first, so a failed publish does not leave an untracked user level.
	customUserLevelResolveComputed(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	published, err := customUserLevelPublishIfRequested(ctx, client, data, created)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to publish custom user level: %s", err))
		return
	}
	customUserLevelResolveComputed(ctx, &data, published, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomUserLevelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomUserLevelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	level, err := client.GetCustomUserLevel(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read custom user level: %s", err))
		return
	}
	setCustomUserLevelState(ctx, &data, level, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomUserLevelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state CustomUserLevelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := customUserLevelPatchOps(ctx, data, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var level *CustomUserLevel
	if len(ops) > 0 {
		level, err = client.PatchCustomUserLevel(ctx, data.ID.ValueString(), ops)
	} else {
		level, err = client.GetCustomUserLevel(ctx, data.ID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update custom user level: %s", err))
		return
	}
	level, err = customUserLevelPublishIfRequested(ctx, client, data, level)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to publish custom user level: %s", err))
		return
	}
	customUserLevelResolveComputed(ctx, &data, level, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomUserLevelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomUserLevelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteCustomUserLevel(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete custom user level: %s", err))
	}
}

func (r *CustomUserLevelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &CustomUserLevelDataSource{}

func NewCustomUserLevelDataSource() datasource.DataSource {
	return &CustomUserLevelDataSource{}
}

type CustomUserLevelDataSource struct {
	client *Config
}

type CustomUserLevelDataSourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Description               types.String `tfsdk:"description"`
	Owner                     types.List   `tfsdk:"owner"`
	RightSets                 types.Set    `tfsdk:"right_sets"`
	Status                    types.String `tfsdk:"status"`
	Created                   types.String `tfsdk:"created"`
	Modified                  types.String `tfsdk:"modified"`
	AssociatedIdentitiesCount types.Int64  `tfsdk:"associated_identities_count"`
}

func (d *CustomUserLevelDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_user_level"
}

func (d *CustomUserLevelDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a custom user level by ID. Uses an experimental API.",
		Attributes: map[string]dsschema.Attribute{
			"id":          dsschema.StringAttribute{MarkdownDescription: "User level ID.", Required: true},
			"name":        dsschema.StringAttribute{MarkdownDescription: "Name of the user level.", Computed: true},
			"description": dsschema.StringAttribute{MarkdownDescription: "Description of the user level.", Computed: true},
			"owner": dsschema.ListNestedAttribute{
				MarkdownDescription: "Identity that owns the user level.",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{Attributes: map[string]dsschema.Attribute{
					"id":   dsschema.StringAttribute{MarkdownDescription: "Owner identity ID.", Computed: true},
					"type": dsschema.StringAttribute{MarkdownDescription: "Owner type.", Computed: true},
					"name": dsschema.StringAttribute{MarkdownDescription: "Owner display name.", Computed: true},
				}},
			},
			"right_sets":                  dsschema.SetAttribute{MarkdownDescription: "IDs of the right sets assigned to the user level.", Computed: true, ElementType: types.StringType},
			"status":                      dsschema.StringAttribute{MarkdownDescription: "Status of the user level, `DRAFT` or `ACTIVE`.", Computed: true},
			"created":                     dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"modified":                    dsschema.StringAttribute{MarkdownDescription: "Last modification date.", Computed: true},
			"associated_identities_count": dsschema.Int64Attribute{MarkdownDescription: "Number of identities the user level is assigned to.", Computed: true},
		},
	}
}

func (d *CustomUserLevelDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CustomUserLevelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CustomUserLevelDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	level, err := client.GetCustomUserLevel(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("id"), "Custom user level not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read custom user level: %s", err))
		return
	}
	data.Name = types.StringValue(level.Name)
	data.Description = types.StringValue(level.Description)
	data.Owner = customUserLevelOwnerState(ctx, types.ListNull(objectInfoObjectType), level.Owner, &resp.Diagnostics)
	data.RightSets = oauthClientStringSetState(ctx, types.SetNull(types.StringType), customUserLevelRightSetIDs(level.RightSets), false, &resp.Diagnostics)
	data.Status = types.StringValue(level.Status)
	data.Created = types.StringValue(level.Created)
	data.Modified = types.StringValue(level.Modified)
	data.AssociatedIdentitiesCount = types.Int64Null()
	if level.AssociatedIdentitiesCount != nil {
		data.AssociatedIdentitiesCount = types.Int64Value(*level.AssociatedIdentitiesCount)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
