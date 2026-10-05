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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewNonEmployeeSourceResource)
	registerDataSource(NewNonEmployeeSourceDataSource)
}

// NonEmployeeSourceIdentityRef references an identity or governance group of a non-employee source.
type NonEmployeeSourceIdentityRef struct {
	ID   string `json:"id"`
	Type string `json:"type,omitempty"`
}

// NonEmployeeSource is a non-employee source as used by the /v2026/non-employee-sources API. Owner
// and managementWorkgroup are only documented in the create request, the API does not return them.
type NonEmployeeSource struct {
	ID                  string                         `json:"id,omitempty"`
	SourceID            string                         `json:"sourceId,omitempty"`
	Name                string                         `json:"name"`
	Description         string                         `json:"description"`
	Owner               *NonEmployeeSourceIdentityRef  `json:"owner,omitempty"`
	ManagementWorkgroup string                         `json:"managementWorkgroup,omitempty"`
	Approvers           []NonEmployeeSourceIdentityRef `json:"approvers,omitempty"`
	AccountManagers     []NonEmployeeSourceIdentityRef `json:"accountManagers,omitempty"`
	CloudExternalID     string                         `json:"cloudExternalId,omitempty"`
	Created             string                         `json:"created,omitempty"`
	Modified            string                         `json:"modified,omitempty"`
}

func (c *Client) GetNonEmployeeSource(ctx context.Context, id string) (*NonEmployeeSource, error) {
	var source NonEmployeeSource
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/non-employee-sources/%s", id), nil, &source); err != nil {
		return nil, err
	}
	return &source, nil
}

// GetNonEmployeeSourceByName lists all non-employee sources and returns the one with the given
// name. The list endpoint cannot filter.
func (c *Client) GetNonEmployeeSourceByName(ctx context.Context, name string) (*NonEmployeeSource, error) {
	items, err := listAllPages[NonEmployeeSource](ctx, c, "/v2026/non-employee-sources", nil)
	if err != nil {
		return nil, err
	}
	var match *NonEmployeeSource
	for i := range items {
		if items[i].Name != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple non-employee sources are named %q", name)
		}
		match = &items[i]
	}
	if match == nil {
		return nil, &NotFoundError{fmt.Sprintf("non-employee source with name %q not found", name)}
	}
	return match, nil
}

func (c *Client) CreateNonEmployeeSource(ctx context.Context, source *NonEmployeeSource) (*NonEmployeeSource, error) {
	var created NonEmployeeSource
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/non-employee-sources", source, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchNonEmployeeSource(ctx context.Context, id string, ops []jsonPatchOp) (*NonEmployeeSource, error) {
	var updated NonEmployeeSource
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/non-employee-sources/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteNonEmployeeSource(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/non-employee-sources/%s", id), nil, nil)
}

var _ resource.Resource = &NonEmployeeSourceResource{}
var _ resource.ResourceWithImportState = &NonEmployeeSourceResource{}

func NewNonEmployeeSourceResource() resource.Resource {
	return &NonEmployeeSourceResource{}
}

type NonEmployeeSourceResource struct {
	client *Config
}

type NonEmployeeSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	SourceID            types.String `tfsdk:"source_id"`
	CloudExternalID     types.String `tfsdk:"cloud_external_id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Owner               types.List   `tfsdk:"owner"`
	ManagementWorkgroup types.String `tfsdk:"management_workgroup"`
	Approvers           types.List   `tfsdk:"approvers"`
	AccountManagers     types.Set    `tfsdk:"account_managers"`
	Created             types.String `tfsdk:"created"`
	Modified            types.String `tfsdk:"modified"`
}

type NonEmployeeSourceOwnerModel struct {
	ID types.String `tfsdk:"id"`
}

var nonEmployeeSourceOwnerObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.StringType}}

// nonEmployeeSourceWriteOnlyKey is the private state key recording that owner and
// management_workgroup were sent to the API by this resource. They are not returned by the API,
// so after an import they are unknown to the provider and are adopted from the configuration.
const nonEmployeeSourceWriteOnlyKey = "write_only_fields_applied"

// nonEmployeeSourcePrivateGetter is the part of the private state used by the plan modifiers.
type nonEmployeeSourcePrivateGetter interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

// nonEmployeeSourceWriteOnlyApplied reports whether owner and management_workgroup in state are
// known to match the API.
func nonEmployeeSourceWriteOnlyApplied(ctx context.Context, private nonEmployeeSourcePrivateGetter) bool {
	if private == nil {
		return false
	}
	value, _ := private.GetKey(ctx, nonEmployeeSourceWriteOnlyKey)
	return string(value) == "true"
}

// nonEmployeeSourceRequiresReplaceString forces replacement when a create-only attribute changes,
// except right after an import, when the prior value is unknown and the configuration is adopted.
var nonEmployeeSourceRequiresReplaceString = stringplanmodifier.RequiresReplaceIf(
	func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = req.Private != nil && nonEmployeeSourceWriteOnlyApplied(ctx, req.Private)
	},
	"Changing the value forces a new non-employee source, except after an import.",
	"Changing the value forces a new non-employee source, except after an import.",
)

var nonEmployeeSourceRequiresReplaceList = listplanmodifier.RequiresReplaceIf(
	func(ctx context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = req.Private != nil && nonEmployeeSourceWriteOnlyApplied(ctx, req.Private)
	},
	"Changing the value forces a new non-employee source, except after an import.",
	"Changing the value forces a new non-employee source, except after an import.",
)

func (r *NonEmployeeSourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_non_employee_source"
}

func (r *NonEmployeeSourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a non-employee source used by Non-Employee Lifecycle Management. Creating it also creates the backing source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Non-employee source ID",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source_id": schema.StringAttribute{
				MarkdownDescription: "ID of the source that backs the non-employee source",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"cloud_external_id": schema.StringAttribute{
				MarkdownDescription: "Legacy V1 ID of the source. Only returned when the source is created, so it is empty after an import.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the non-employee source",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the non-employee source",
				Required:            true,
			},
			"management_workgroup": schema.StringAttribute{
				MarkdownDescription: "ID of the governance group that contains the source sub-admins. It cannot be changed and is not returned by the API: changing it forces a new source, and changes made outside Terraform are not detected.",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{nonEmployeeSourceRequiresReplaceString},
			},
			"approvers": schema.ListAttribute{
				MarkdownDescription: "IDs of up to 3 identities or governance groups that approve non-employee requests, in approval order",
				Optional:            true,
				ElementType:         types.StringType,
				Validators:          []validator.List{listSizeBetween(0, 3)},
			},
			"account_managers": schema.SetAttribute{
				MarkdownDescription: "IDs of up to 10 identities or governance groups that manage the non-employee accounts",
				Optional:            true,
				ElementType:         types.StringType,
				Validators:          []validator.Set{nonEmployeeSourceSetSizeValidator{max: 10}},
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
			"owner": schema.ListNestedBlock{
				MarkdownDescription: "Owner identity of the source. Exactly one block. It cannot be changed and is not returned by the API: changing it forces a new source, and changes made outside Terraform are not detected.",
				Validators:          []validator.List{listSizeBetween(1, 1)},
				PlanModifiers:       []planmodifier.List{nonEmployeeSourceRequiresReplaceList},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Owner identity ID",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

// nonEmployeeSourceSetSizeValidator checks that a set has at most max elements.
type nonEmployeeSourceSetSizeValidator struct {
	max int
}

func (v nonEmployeeSourceSetSizeValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("set must contain at most %d elements", v.max)
}

func (v nonEmployeeSourceSetSizeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v nonEmployeeSourceSetSizeValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if size := len(req.ConfigValue.Elements()); size > v.max {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid set size", fmt.Sprintf("%s, got %d.", v.Description(ctx), size))
	}
}

func (r *NonEmployeeSourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// nonEmployeeSourceRefs converts a list or set of IDs to identity references. A null value gives
// an empty, non-nil slice, so a patch clears the field.
func nonEmployeeSourceRefs(ctx context.Context, value interface {
	IsNull() bool
	IsUnknown() bool
	ElementsAs(context.Context, interface{}, bool) diag.Diagnostics
}, diags *diag.Diagnostics) []NonEmployeeSourceIdentityRef {
	refs := []NonEmployeeSourceIdentityRef{}
	if value.IsNull() || value.IsUnknown() {
		return refs
	}
	var ids []string
	diags.Append(value.ElementsAs(ctx, &ids, false)...)
	for _, id := range ids {
		refs = append(refs, NonEmployeeSourceIdentityRef{ID: id})
	}
	return refs
}

func nonEmployeeSourceOwnerFromModel(ctx context.Context, owner types.List, diags *diag.Diagnostics) *NonEmployeeSourceIdentityRef {
	if owner.IsNull() || owner.IsUnknown() || len(owner.Elements()) == 0 {
		return nil
	}
	var models []NonEmployeeSourceOwnerModel
	diags.Append(owner.ElementsAs(ctx, &models, false)...)
	if len(models) == 0 {
		return nil
	}
	return &NonEmployeeSourceIdentityRef{ID: models[0].ID.ValueString()}
}

func nonEmployeeSourceFromModel(ctx context.Context, data NonEmployeeSourceModel, diags *diag.Diagnostics) *NonEmployeeSource {
	source := &NonEmployeeSource{
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		Owner:               nonEmployeeSourceOwnerFromModel(ctx, data.Owner, diags),
		ManagementWorkgroup: data.ManagementWorkgroup.ValueString(),
	}
	if approvers := nonEmployeeSourceRefs(ctx, data.Approvers, diags); len(approvers) > 0 {
		source.Approvers = approvers
	}
	if managers := nonEmployeeSourceRefs(ctx, data.AccountManagers, diags); len(managers) > 0 {
		source.AccountManagers = managers
	}
	return source
}

func nonEmployeeSourceRefIDs(refs []NonEmployeeSourceIdentityRef) []attr.Value {
	values := make([]attr.Value, 0, len(refs))
	for _, ref := range refs {
		values = append(values, types.StringValue(ref.ID))
	}
	return values
}

// nonEmployeeSourceListState returns the approvers list, null when the API has none and the
// attribute is not configured.
func nonEmployeeSourceListState(prior types.List, refs []NonEmployeeSourceIdentityRef, diags *diag.Diagnostics) types.List {
	if len(refs) == 0 && prior.IsNull() {
		return types.ListNull(types.StringType)
	}
	list, d := types.ListValue(types.StringType, nonEmployeeSourceRefIDs(refs))
	diags.Append(d...)
	return list
}

// nonEmployeeSourceSetState returns the account managers set, null when the API has none and the
// attribute is not configured.
func nonEmployeeSourceSetState(prior types.Set, refs []NonEmployeeSourceIdentityRef, diags *diag.Diagnostics) types.Set {
	if len(refs) == 0 && prior.IsNull() {
		return types.SetNull(types.StringType)
	}
	set, d := types.SetValue(types.StringType, nonEmployeeSourceRefIDs(refs))
	diags.Append(d...)
	return set
}

// setNonEmployeeSourceState refreshes the model from the API. Owner and management workgroup are
// only refreshed when the API returns them.
func setNonEmployeeSourceState(ctx context.Context, data *NonEmployeeSourceModel, source *NonEmployeeSource, diags *diag.Diagnostics) {
	data.ID = types.StringValue(source.ID)
	data.SourceID = types.StringValue(source.SourceID)
	if source.CloudExternalID != "" || data.CloudExternalID.IsNull() || data.CloudExternalID.IsUnknown() {
		data.CloudExternalID = types.StringValue(source.CloudExternalID)
	}
	data.Name = types.StringValue(source.Name)
	data.Description = types.StringValue(source.Description)
	if source.Owner != nil && source.Owner.ID != "" {
		owners, d := types.ListValueFrom(ctx, nonEmployeeSourceOwnerObjectType, []NonEmployeeSourceOwnerModel{{ID: types.StringValue(source.Owner.ID)}})
		diags.Append(d...)
		data.Owner = owners
	} else if data.Owner.IsUnknown() {
		data.Owner = types.ListNull(nonEmployeeSourceOwnerObjectType)
	}
	if source.ManagementWorkgroup != "" {
		data.ManagementWorkgroup = types.StringValue(source.ManagementWorkgroup)
	} else if data.ManagementWorkgroup.IsUnknown() {
		data.ManagementWorkgroup = types.StringNull()
	}
	data.Approvers = nonEmployeeSourceListState(data.Approvers, source.Approvers, diags)
	data.AccountManagers = nonEmployeeSourceSetState(data.AccountManagers, source.AccountManagers, diags)
	data.Created = types.StringValue(source.Created)
	data.Modified = types.StringValue(source.Modified)
}

// nonEmployeeSourcePatchOps returns JSON Patch operations for the patchable attributes that changed.
func nonEmployeeSourcePatchOps(ctx context.Context, plan, state NonEmployeeSourceModel, diags *diag.Diagnostics) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	b.replaceIfChanged(plan.Approvers, state.Approvers, "/approvers", nonEmployeeSourceRefs(ctx, plan.Approvers, diags))
	b.replaceIfChanged(plan.AccountManagers, state.AccountManagers, "/accountManagers", nonEmployeeSourceRefs(ctx, plan.AccountManagers, diags))
	return b.ops
}

func (r *NonEmployeeSourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NonEmployeeSourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	source := nonEmployeeSourceFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateNonEmployeeSource(ctx, source)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create non-employee source: %s", err))
		return
	}
	data.ID = types.StringValue(created.ID)
	data.SourceID = types.StringValue(created.SourceID)
	data.CloudExternalID = types.StringValue(created.CloudExternalID)
	data.Created = types.StringValue(created.Created)
	data.Modified = types.StringValue(created.Modified)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, nonEmployeeSourceWriteOnlyKey, []byte("true"))...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NonEmployeeSourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NonEmployeeSourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	source, err := client.GetNonEmployeeSource(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read non-employee source: %s", err))
		return
	}
	setNonEmployeeSourceState(ctx, &data, source, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NonEmployeeSourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NonEmployeeSourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := nonEmployeeSourcePatchOps(ctx, plan, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Modified = state.Modified
	if len(ops) > 0 {
		client, err := r.client.IdentityNowClient(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		updated, err := client.PatchNonEmployeeSource(ctx, plan.ID.ValueString(), ops)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update non-employee source: %s", err))
			return
		}
		plan.Modified = types.StringValue(updated.Modified)
	}
	// Owner and management workgroup changes only reach Update right after an import, when the
	// configured values are adopted. From now on they are known to be managed by this resource.
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, nonEmployeeSourceWriteOnlyKey, []byte("true"))...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NonEmployeeSourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NonEmployeeSourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteNonEmployeeSource(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete non-employee source: %s", err))
	}
}

func (r *NonEmployeeSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &NonEmployeeSourceDataSource{}
var _ datasource.DataSourceWithValidateConfig = &NonEmployeeSourceDataSource{}

func NewNonEmployeeSourceDataSource() datasource.DataSource {
	return &NonEmployeeSourceDataSource{}
}

type NonEmployeeSourceDataSource struct {
	client *Config
}

type NonEmployeeSourceDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	SourceID        types.String `tfsdk:"source_id"`
	Description     types.String `tfsdk:"description"`
	Approvers       types.List   `tfsdk:"approvers"`
	AccountManagers types.Set    `tfsdk:"account_managers"`
	Created         types.String `tfsdk:"created"`
	Modified        types.String `tfsdk:"modified"`
}

func (d *NonEmployeeSourceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_non_employee_source"
}

func (d *NonEmployeeSourceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a non-employee source by ID or name.",
		Attributes: map[string]dsschema.Attribute{
			"id": dsschema.StringAttribute{
				MarkdownDescription: "Non-employee source ID. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"name": dsschema.StringAttribute{
				MarkdownDescription: "Non-employee source name. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"source_id":   dsschema.StringAttribute{MarkdownDescription: "ID of the source that backs the non-employee source", Computed: true},
			"description": dsschema.StringAttribute{MarkdownDescription: "Description of the non-employee source", Computed: true},
			"approvers": dsschema.ListAttribute{
				MarkdownDescription: "IDs of the approvers, in approval order",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"account_managers": dsschema.SetAttribute{
				MarkdownDescription: "IDs of the account managers",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"created":  dsschema.StringAttribute{MarkdownDescription: "Creation date", Computed: true},
			"modified": dsschema.StringAttribute{MarkdownDescription: "Last modification date", Computed: true},
		},
	}
}

func (d *NonEmployeeSourceDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *NonEmployeeSourceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NonEmployeeSourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NonEmployeeSourceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var source *NonEmployeeSource
	attribute := path.Root("id")
	if !data.ID.IsNull() {
		source, err = client.GetNonEmployeeSource(ctx, data.ID.ValueString())
	} else {
		attribute = path.Root("name")
		source, err = client.GetNonEmployeeSourceByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(attribute, "Non-employee source not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read non-employee source: %s", err))
		return
	}
	data.ID = types.StringValue(source.ID)
	data.Name = types.StringValue(source.Name)
	data.SourceID = types.StringValue(source.SourceID)
	data.Description = types.StringValue(source.Description)
	data.Approvers = nonEmployeeSourceListState(types.ListValueMust(types.StringType, nil), source.Approvers, &resp.Diagnostics)
	data.AccountManagers = nonEmployeeSourceSetState(types.SetValueMust(types.StringType, nil), source.AccountManagers, &resp.Diagnostics)
	data.Created = types.StringValue(source.Created)
	data.Modified = types.StringValue(source.Modified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
