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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewManagedClusterTypeResource)
	registerDataSource(NewManagedClusterTypeDataSource)
}

// ManagedClusterType is a managed cluster type as returned by the /v2026/managed-cluster-types API.
type ManagedClusterType struct {
	ID                string   `json:"id,omitempty"`
	Type              string   `json:"type"`
	Pod               string   `json:"pod"`
	Org               string   `json:"org"`
	ManagedProcessIDs []string `json:"managedProcessIds,omitempty"`
}

func (c *Client) GetManagedClusterType(ctx context.Context, id string) (*ManagedClusterType, error) {
	var clusterType ManagedClusterType
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/managed-cluster-types/%s", id), nil, &clusterType); err != nil {
		return nil, err
	}
	return &clusterType, nil
}

func (c *Client) CreateManagedClusterType(ctx context.Context, clusterType *ManagedClusterType) (*ManagedClusterType, error) {
	var created ManagedClusterType
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/managed-cluster-types", clusterType, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// PatchManagedClusterType sends an RFC 6902 JSON Patch array. The spec models the body as an object
// with an operations array, but describes it as an RFC 6902 document like the other PATCH endpoints.
func (c *Client) PatchManagedClusterType(ctx context.Context, id string, ops []jsonPatchOp) (*ManagedClusterType, error) {
	var updated ManagedClusterType
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/managed-cluster-types/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteManagedClusterType(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/managed-cluster-types/%s", id), nil, nil)
}

var _ resource.Resource = &ManagedClusterTypeResource{}
var _ resource.ResourceWithImportState = &ManagedClusterTypeResource{}

func NewManagedClusterTypeResource() resource.Resource {
	return &ManagedClusterTypeResource{}
}

type ManagedClusterTypeResource struct {
	client *Config
}

type ManagedClusterTypeModel struct {
	ID                types.String `tfsdk:"id"`
	Type              types.String `tfsdk:"type"`
	Pod               types.String `tfsdk:"pod"`
	Org               types.String `tfsdk:"org"`
	ManagedProcessIDs types.List   `tfsdk:"managed_process_ids"`
}

func (r *ManagedClusterTypeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_cluster_type"
}

func (r *ManagedClusterTypeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a managed cluster type, which defines the processes that run on clusters of that type for a pod or org.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Managed cluster type ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Name of the cluster type.",
				Required:            true,
			},
			"pod": schema.StringAttribute{
				MarkdownDescription: "Pod the cluster type applies to.",
				Required:            true,
			},
			"org": schema.StringAttribute{
				MarkdownDescription: "Org the cluster type applies to.",
				Required:            true,
			},
			"managed_process_ids": schema.ListAttribute{
				MarkdownDescription: "IDs of the processes that run on clusters of this type. Removing the attribute clears the list.",
				ElementType:         types.StringType,
				Optional:            true,
			},
		},
	}
}

func (r *ManagedClusterTypeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// setManagedClusterTypeState refreshes the model from the API.
func setManagedClusterTypeState(ctx context.Context, data *ManagedClusterTypeModel, clusterType *ManagedClusterType, diags *diag.Diagnostics) {
	data.ID = types.StringValue(clusterType.ID)
	data.Type = types.StringValue(clusterType.Type)
	data.Pod = types.StringValue(clusterType.Pod)
	data.Org = types.StringValue(clusterType.Org)
	data.ManagedProcessIDs = serviceDeskIntegrationStringListState(ctx, data.ManagedProcessIDs, clusterType.ManagedProcessIDs, diags)
}

// managedClusterTypePatchOps returns replace operations for the changed attributes. Removed
// process IDs are replaced with an empty list.
func managedClusterTypePatchOps(ctx context.Context, plan, state ManagedClusterTypeModel, diags *diag.Diagnostics) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.Type, state.Type, "/type", plan.Type.ValueString())
	b.replaceIfChanged(plan.Pod, state.Pod, "/pod", plan.Pod.ValueString())
	b.replaceIfChanged(plan.Org, state.Org, "/org", plan.Org.ValueString())
	processIDs := serviceDeskIntegrationStringList(ctx, plan.ManagedProcessIDs, diags)
	if processIDs == nil {
		processIDs = []string{}
	}
	b.replaceIfChanged(plan.ManagedProcessIDs, state.ManagedProcessIDs, "/managedProcessIds", processIDs)
	return b.ops
}

func (r *ManagedClusterTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ManagedClusterTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	clusterType := &ManagedClusterType{
		Type:              data.Type.ValueString(),
		Pod:               data.Pod.ValueString(),
		Org:               data.Org.ValueString(),
		ManagedProcessIDs: serviceDeskIntegrationStringList(ctx, data.ManagedProcessIDs, &resp.Diagnostics),
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateManagedClusterType(ctx, clusterType)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create managed cluster type: %s", err))
		return
	}
	data.ID = types.StringValue(created.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedClusterTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ManagedClusterTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	clusterType, err := client.GetManagedClusterType(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed cluster type: %s", err))
		return
	}
	setManagedClusterTypeState(ctx, &data, clusterType, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedClusterTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ManagedClusterTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := managedClusterTypePatchOps(ctx, data, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(ops) > 0 {
		client, err := r.client.IdentityNowClient(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		if _, err := client.PatchManagedClusterType(ctx, data.ID.ValueString(), ops); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update managed cluster type: %s", err))
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedClusterTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ManagedClusterTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteManagedClusterType(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete managed cluster type: %s", err))
	}
}

func (r *ManagedClusterTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &ManagedClusterTypeDataSource{}

func NewManagedClusterTypeDataSource() datasource.DataSource {
	return &ManagedClusterTypeDataSource{}
}

type ManagedClusterTypeDataSource struct {
	client *Config
}

func (d *ManagedClusterTypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_cluster_type"
}

func (d *ManagedClusterTypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a managed cluster type by ID.",
		Attributes: map[string]dsschema.Attribute{
			"id":   dsschema.StringAttribute{MarkdownDescription: "Managed cluster type ID.", Required: true},
			"type": dsschema.StringAttribute{MarkdownDescription: "Name of the cluster type.", Computed: true},
			"pod":  dsschema.StringAttribute{MarkdownDescription: "Pod the cluster type applies to.", Computed: true},
			"org":  dsschema.StringAttribute{MarkdownDescription: "Org the cluster type applies to.", Computed: true},
			"managed_process_ids": dsschema.ListAttribute{
				MarkdownDescription: "IDs of the processes that run on clusters of this type.",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (d *ManagedClusterTypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ManagedClusterTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ManagedClusterTypeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	clusterType, err := client.GetManagedClusterType(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("id"), "Managed cluster type not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed cluster type: %s", err))
		return
	}
	setManagedClusterTypeState(ctx, &data, clusterType, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
