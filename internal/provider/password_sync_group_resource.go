package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewPasswordSyncGroupResource)
	registerDataSource(NewPasswordSyncGroupDataSource)
}

// PasswordSyncGroup is a password sync group as used by the /v2026/password-sync-groups API.
type PasswordSyncGroup struct {
	ID               string   `json:"id,omitempty"`
	Name             string   `json:"name"`
	PasswordPolicyID *string  `json:"passwordPolicyId"`
	SourceIDs        []string `json:"sourceIds"`
	Created          string   `json:"created,omitempty"`
	Modified         string   `json:"modified,omitempty"`
}

func (c *Client) GetPasswordSyncGroup(ctx context.Context, id string) (*PasswordSyncGroup, error) {
	var group PasswordSyncGroup
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/password-sync-groups/%s", id), nil, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// GetPasswordSyncGroupByName returns the sync group with the given name. The list endpoint has no
// filters, so groups are matched client side.
func (c *Client) GetPasswordSyncGroupByName(ctx context.Context, name string) (*PasswordSyncGroup, error) {
	groups, err := listAllPages[PasswordSyncGroup](ctx, c, "/v2026/password-sync-groups", url.Values{})
	if err != nil {
		return nil, err
	}
	var match *PasswordSyncGroup
	for i := range groups {
		if groups[i].Name != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple password sync groups are named %q", name)
		}
		match = &groups[i]
	}
	if match == nil {
		return nil, &NotFoundError{fmt.Sprintf("password sync group with name %q not found", name)}
	}
	return match, nil
}

func (c *Client) CreatePasswordSyncGroup(ctx context.Context, group *PasswordSyncGroup) (*PasswordSyncGroup, error) {
	var created PasswordSyncGroup
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/password-sync-groups", group, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdatePasswordSyncGroup(ctx context.Context, id string, group *PasswordSyncGroup) (*PasswordSyncGroup, error) {
	var updated PasswordSyncGroup
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/password-sync-groups/%s", id), group, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeletePasswordSyncGroup(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/password-sync-groups/%s", id), nil, nil)
}

var _ resource.Resource = &PasswordSyncGroupResource{}
var _ resource.ResourceWithImportState = &PasswordSyncGroupResource{}

func NewPasswordSyncGroupResource() resource.Resource {
	return &PasswordSyncGroupResource{}
}

type PasswordSyncGroupResource struct {
	client *Config
}

type PasswordSyncGroupModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	PasswordPolicyID types.String `tfsdk:"password_policy_id"`
	SourceIDs        types.Set    `tfsdk:"source_ids"`
	Created          types.String `tfsdk:"created"`
	Modified         types.String `tfsdk:"modified"`
}

func (r *PasswordSyncGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password_sync_group"
}

func (r *PasswordSyncGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a password sync group, a set of sources that share the same password.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Password sync group ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the sync group.",
				Required:            true,
			},
			"password_policy_id": schema.StringAttribute{
				MarkdownDescription: "ID of the password policy of the sync group.",
				Optional:            true,
			},
			"source_ids": schema.SetAttribute{
				MarkdownDescription: "IDs of the password managed sources in the sync group.",
				ElementType:         types.StringType,
				Optional:            true,
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

func (r *PasswordSyncGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// passwordSyncGroupFromModel builds the full sync group from the plan. PUT replaces the group and
// every writable field is modelled, so unset fields are sent empty.
func passwordSyncGroupFromModel(ctx context.Context, data PasswordSyncGroupModel, diags *diag.Diagnostics) *PasswordSyncGroup {
	group := &PasswordSyncGroup{
		Name:             data.Name.ValueString(),
		PasswordPolicyID: stringPointer(data.PasswordPolicyID),
		SourceIDs:        []string{},
	}
	if !data.SourceIDs.IsNull() && !data.SourceIDs.IsUnknown() {
		diags.Append(data.SourceIDs.ElementsAs(ctx, &group.SourceIDs, false)...)
	}
	sort.Strings(group.SourceIDs)
	return group
}

// setPasswordSyncGroupState maps an API sync group onto the model during refresh.
func setPasswordSyncGroupState(ctx context.Context, data *PasswordSyncGroupModel, group *PasswordSyncGroup, diags *diag.Diagnostics) {
	data.ID = types.StringValue(group.ID)
	data.Name = types.StringValue(group.Name)
	policyID := ""
	if group.PasswordPolicyID != nil {
		policyID = *group.PasswordPolicyID
	}
	data.PasswordPolicyID = optionalStringState(data.PasswordPolicyID, policyID)
	if len(group.SourceIDs) == 0 && data.SourceIDs.IsNull() {
		data.SourceIDs = types.SetNull(types.StringType)
	} else {
		sourceIDs := group.SourceIDs
		if sourceIDs == nil {
			sourceIDs = []string{}
		}
		set, d := types.SetValueFrom(ctx, types.StringType, sourceIDs)
		diags.Append(d...)
		data.SourceIDs = set
	}
	data.Created = stringValueOrNull(group.Created)
	data.Modified = stringValueOrNull(group.Modified)
}

// applyPasswordSyncGroupResponse resolves the computed values after create or update.
func applyPasswordSyncGroupResponse(data *PasswordSyncGroupModel, group *PasswordSyncGroup) {
	if data.ID.IsUnknown() {
		data.ID = types.StringValue(group.ID)
	}
	if data.Created.IsUnknown() {
		data.Created = stringValueOrNull(group.Created)
	}
	data.Modified = stringValueOrNull(group.Modified)
}

func (r *PasswordSyncGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PasswordSyncGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	group := passwordSyncGroupFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreatePasswordSyncGroup(ctx, group)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create password sync group: %s", err))
		return
	}
	applyPasswordSyncGroupResponse(&data, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PasswordSyncGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PasswordSyncGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	group, err := client.GetPasswordSyncGroup(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read password sync group: %s", err))
		return
	}
	setPasswordSyncGroupState(ctx, &data, group, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PasswordSyncGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PasswordSyncGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	group := passwordSyncGroupFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	group.ID = data.ID.ValueString()
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.UpdatePasswordSyncGroup(ctx, data.ID.ValueString(), group)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update password sync group: %s", err))
		return
	}
	applyPasswordSyncGroupResponse(&data, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PasswordSyncGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PasswordSyncGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeletePasswordSyncGroup(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete password sync group: %s", err))
	}
}

func (r *PasswordSyncGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &PasswordSyncGroupDataSource{}
var _ datasource.DataSourceWithValidateConfig = &PasswordSyncGroupDataSource{}

func NewPasswordSyncGroupDataSource() datasource.DataSource {
	return &PasswordSyncGroupDataSource{}
}

type PasswordSyncGroupDataSource struct {
	client *Config
}

func (d *PasswordSyncGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password_sync_group"
}

func (d *PasswordSyncGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a password sync group by ID or name.",
		Attributes: map[string]dsschema.Attribute{
			"id":                 dsschema.StringAttribute{MarkdownDescription: "Password sync group ID. Exactly one of `id` or `name` must be set.", Optional: true, Computed: true},
			"name":               dsschema.StringAttribute{MarkdownDescription: "Name of the sync group. Exactly one of `id` or `name` must be set.", Optional: true, Computed: true},
			"password_policy_id": dsschema.StringAttribute{MarkdownDescription: "ID of the password policy of the sync group.", Computed: true},
			"source_ids":         dsschema.SetAttribute{MarkdownDescription: "IDs of the sources in the sync group.", ElementType: types.StringType, Computed: true},
			"created":            dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"modified":           dsschema.StringAttribute{MarkdownDescription: "Last modification date.", Computed: true},
		},
	}
}

func (d *PasswordSyncGroupDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *PasswordSyncGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PasswordSyncGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PasswordSyncGroupModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var group *PasswordSyncGroup
	if !data.ID.IsNull() {
		group, err = client.GetPasswordSyncGroup(ctx, data.ID.ValueString())
	} else {
		group, err = client.GetPasswordSyncGroupByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Password sync group not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read password sync group: %s", err))
		return
	}
	data.PasswordPolicyID = types.StringValue("")
	data.SourceIDs = types.SetValueMust(types.StringType, nil)
	setPasswordSyncGroupState(ctx, &data, group, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
