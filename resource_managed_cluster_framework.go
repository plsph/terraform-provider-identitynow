package main

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewManagedClusterResource)
	registerDataSource(NewManagedClusterDataSource)
}

// ManagedCluster is a managed (virtual appliance) cluster as returned by the /v2026/managed-clusters API.
type ManagedCluster struct {
	ID                                 string             `json:"id"`
	Name                               string             `json:"name"`
	Pod                                string             `json:"pod,omitempty"`
	Org                                string             `json:"org,omitempty"`
	Type                               string             `json:"type,omitempty"`
	Configuration                      map[string]*string `json:"configuration,omitempty"`
	Description                        string             `json:"description,omitempty"`
	ClientType                         string             `json:"clientType,omitempty"`
	CcgVersion                         string             `json:"ccgVersion,omitempty"`
	PinnedConfig                       bool               `json:"pinnedConfig"`
	Operational                        bool               `json:"operational"`
	Status                             string             `json:"status,omitempty"`
	PublicKey                          string             `json:"publicKey,omitempty"`
	PublicKeyThumbprint                string             `json:"publicKeyThumbprint,omitempty"`
	PublicKeyCertificate               string             `json:"publicKeyCertificate,omitempty"`
	AlertKey                           string             `json:"alertKey,omitempty"`
	ClientIDs                          []string           `json:"clientIds,omitempty"`
	ServiceCount                       int64              `json:"serviceCount"`
	CreatedAt                          string             `json:"createdAt,omitempty"`
	UpdatedAt                          string             `json:"updatedAt,omitempty"`
	CurrentInstalledReleaseVersion     string             `json:"currentInstalledReleaseVersion,omitempty"`
	ConsolidatedHealthIndicatorsStatus string             `json:"consolidatedHealthIndicatorsStatus,omitempty"`
}

// ManagedClusterRequest is the request body to create a managed cluster.
type ManagedClusterRequest struct {
	Name          string            `json:"name"`
	Type          string            `json:"type,omitempty"`
	Configuration map[string]string `json:"configuration,omitempty"`
	Description   *string           `json:"description,omitempty"`
}

func (c *Client) GetManagedCluster(ctx context.Context, id string) (*ManagedCluster, error) {
	var cluster ManagedCluster
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/managed-clusters/%s", id), nil, &cluster); err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (c *Client) GetManagedClusterByName(ctx context.Context, name string) (*ManagedCluster, error) {
	return findByName(ctx, c, "/v2026/managed-clusters", "managed cluster", name, func(m ManagedCluster) string { return m.Name })
}

func (c *Client) CreateManagedCluster(ctx context.Context, request *ManagedClusterRequest) (*ManagedCluster, error) {
	var created ManagedCluster
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/managed-clusters", request, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchManagedCluster(ctx context.Context, id string, ops []jsonPatchOp) (*ManagedCluster, error) {
	var updated ManagedCluster
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/managed-clusters/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeleteManagedCluster deletes a cluster. With removeClients, clusters that still have clients are
// deleted together with their clients.
func (c *Client) DeleteManagedCluster(ctx context.Context, id string, removeClients bool) error {
	p := apiPath("/v2026/managed-clusters/%s", id)
	if removeClients {
		p += "?removeClients=true"
	}
	return c.doJSON(ctx, http.MethodDelete, p, nil, nil)
}

var _ resource.Resource = &ManagedClusterResource{}
var _ resource.ResourceWithImportState = &ManagedClusterResource{}

func NewManagedClusterResource() resource.Resource {
	return &ManagedClusterResource{}
}

type ManagedClusterResource struct {
	client *Config
}

// ManagedClusterComputedModel holds the attributes shared by the resource and the data source.
type ManagedClusterComputedModel struct {
	ID                                 types.String `tfsdk:"id"`
	Name                               types.String `tfsdk:"name"`
	Type                               types.String `tfsdk:"type"`
	Description                        types.String `tfsdk:"description"`
	Configuration                      types.Map    `tfsdk:"configuration"`
	Pod                                types.String `tfsdk:"pod"`
	Org                                types.String `tfsdk:"org"`
	ClientType                         types.String `tfsdk:"client_type"`
	CcgVersion                         types.String `tfsdk:"ccg_version"`
	PinnedConfig                       types.Bool   `tfsdk:"pinned_config"`
	Operational                        types.Bool   `tfsdk:"operational"`
	Status                             types.String `tfsdk:"status"`
	PublicKey                          types.String `tfsdk:"public_key"`
	PublicKeyThumbprint                types.String `tfsdk:"public_key_thumbprint"`
	PublicKeyCertificate               types.String `tfsdk:"public_key_certificate"`
	AlertKey                           types.String `tfsdk:"alert_key"`
	ClientIDs                          types.List   `tfsdk:"client_ids"`
	ServiceCount                       types.Int64  `tfsdk:"service_count"`
	CreatedAt                          types.String `tfsdk:"created_at"`
	UpdatedAt                          types.String `tfsdk:"updated_at"`
	CurrentInstalledReleaseVersion     types.String `tfsdk:"current_installed_release_version"`
	ConsolidatedHealthIndicatorsStatus types.String `tfsdk:"consolidated_health_indicators_status"`
}

type ManagedClusterModel struct {
	ManagedClusterComputedModel
	RemoveClientsOnDestroy types.Bool `tfsdk:"remove_clients_on_destroy"`
}

func (r *ManagedClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_cluster"
}

// managedClusterComputedDescriptions documents the read-only attributes of a managed cluster.
var managedClusterComputedDescriptions = map[string]string{
	"pod":                                   "Pod of the cluster.",
	"org":                                   "Org (tenant) of the cluster.",
	"client_type":                           "Type of the clients of the cluster, e.g. `VA` or `CCG`.",
	"ccg_version":                           "CCG version used by the cluster.",
	"status":                                "Cluster status, e.g. `NORMAL`, `NO_CLIENTS` or `FAILED`.",
	"public_key":                            "Public key of the cluster.",
	"public_key_thumbprint":                 "Public key thumbprint of the cluster.",
	"public_key_certificate":                "Public key certificate of the cluster.",
	"alert_key":                             "Key describing any immediate cluster alerts.",
	"updated_at":                            "Last update date.",
	"current_installed_release_version":     "Release installed on the cluster.",
	"consolidated_health_indicators_status": "Consolidated health status of the cluster.",
}

func (r *ManagedClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "Managed cluster ID.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Name of the cluster.",
			Required:            true,
		},
		"type": schema.StringAttribute{
			MarkdownDescription: "Cluster type, one of `idn`, `iai`, `spConnectCluster`, `sqsCluster`, `das-rc`, `das-pc`, `das-dc`, `pag`, `das-am` or `standard`. When not set, the API default is used. Changing it forces a new cluster.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Description of the cluster. When not set, the value returned by the API is used and kept, so removing the argument does not clear the description; set it to an empty string to clear it.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"configuration": schema.MapAttribute{
			MarkdownDescription: "Cluster configuration entries, e.g. `gmtOffset`. Only the configured keys are managed: keys added by the API or outside Terraform are neither shown nor removed.",
			ElementType:         types.StringType,
			Optional:            true,
		},
		"remove_clients_on_destroy": schema.BoolAttribute{
			MarkdownDescription: "Whether destroying the resource also deletes the clients of the cluster. Without it, deleting a cluster that still has clients fails. Defaults to `false`.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"pinned_config": schema.BoolAttribute{
			MarkdownDescription: "Whether the cluster configuration is pinned.",
			Computed:            true,
		},
		"operational": schema.BoolAttribute{
			MarkdownDescription: "Whether the cluster is operational.",
			Computed:            true,
		},
		"client_ids": schema.ListAttribute{
			MarkdownDescription: "IDs of the clients of the cluster.",
			ElementType:         types.StringType,
			Computed:            true,
		},
		"service_count": schema.Int64Attribute{
			MarkdownDescription: "Number of services bound to the cluster.",
			Computed:            true,
		},
		"created_at": schema.StringAttribute{
			MarkdownDescription: "Creation date.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}
	for name, description := range managedClusterComputedDescriptions {
		attribute := schema.StringAttribute{MarkdownDescription: description, Computed: true}
		if name == "pod" || name == "org" {
			attribute.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
		}
		attributes[name] = attribute
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a managed cluster, a group of virtual appliances or other clients that connect IdentityNow to on-premises sources.",
		Attributes:          attributes,
	}
}

func (r *ManagedClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// managedClusterConfigurationValue returns the elements of a known configuration map.
func managedClusterConfigurationValue(ctx context.Context, value types.Map, diags *diag.Diagnostics) map[string]string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := map[string]string{}
	diags.Append(value.ElementsAs(ctx, &result, false)...)
	return result
}

// managedClusterConfigurationState returns the configuration state read from the API. With
// onlyKeys, only the keys of prior are kept, so entries the user does not manage are ignored.
func managedClusterConfigurationState(ctx context.Context, prior types.Map, api map[string]*string, onlyKeys bool, diags *diag.Diagnostics) types.Map {
	if onlyKeys && (prior.IsNull() || prior.IsUnknown()) {
		return types.MapNull(types.StringType)
	}
	values := map[string]string{}
	if onlyKeys {
		for key := range managedClusterConfigurationValue(ctx, prior, diags) {
			if value, ok := api[key]; ok {
				values[key] = ""
				if value != nil {
					values[key] = *value
				}
			}
		}
	} else {
		if len(api) == 0 {
			return types.MapNull(types.StringType)
		}
		for key, value := range api {
			values[key] = ""
			if value != nil {
				values[key] = *value
			}
		}
	}
	result, d := types.MapValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return result
}

// managedClusterMergedConfiguration applies the planned configuration to the current one: keys
// removed from the configuration are deleted, planned keys are set, other keys are kept.
func managedClusterMergedConfiguration(current map[string]*string, prior, planned map[string]string) map[string]*string {
	merged := map[string]*string{}
	for key, value := range current {
		merged[key] = value
	}
	for key := range prior {
		if _, ok := planned[key]; !ok {
			delete(merged, key)
		}
	}
	for key, value := range planned {
		v := value
		merged[key] = &v
	}
	return merged
}

// setManagedClusterComputedState sets the read-only attributes from the API.
func setManagedClusterComputedState(ctx context.Context, data *ManagedClusterComputedModel, cluster *ManagedCluster, diags *diag.Diagnostics) {
	data.ID = types.StringValue(cluster.ID)
	data.Pod = types.StringValue(cluster.Pod)
	data.Org = types.StringValue(cluster.Org)
	data.ClientType = stringValueOrNull(cluster.ClientType)
	data.CcgVersion = types.StringValue(cluster.CcgVersion)
	data.PinnedConfig = types.BoolValue(cluster.PinnedConfig)
	data.Operational = types.BoolValue(cluster.Operational)
	data.Status = types.StringValue(cluster.Status)
	data.PublicKey = stringValueOrNull(cluster.PublicKey)
	data.PublicKeyThumbprint = stringValueOrNull(cluster.PublicKeyThumbprint)
	data.PublicKeyCertificate = stringValueOrNull(cluster.PublicKeyCertificate)
	data.AlertKey = types.StringValue(cluster.AlertKey)
	clientIDs, d := types.ListValueFrom(ctx, types.StringType, append([]string{}, cluster.ClientIDs...))
	diags.Append(d...)
	data.ClientIDs = clientIDs
	data.ServiceCount = types.Int64Value(cluster.ServiceCount)
	data.CreatedAt = stringValueOrNull(cluster.CreatedAt)
	data.UpdatedAt = stringValueOrNull(cluster.UpdatedAt)
	data.CurrentInstalledReleaseVersion = stringValueOrNull(cluster.CurrentInstalledReleaseVersion)
	data.ConsolidatedHealthIndicatorsStatus = stringValueOrNull(cluster.ConsolidatedHealthIndicatorsStatus)
}

// setManagedClusterState maps the API cluster onto the model. With refresh the configurable
// attributes are refreshed from the API; otherwise the planned values are kept and unknown
// values are resolved.
func setManagedClusterState(ctx context.Context, data *ManagedClusterModel, cluster *ManagedCluster, refresh bool, diags *diag.Diagnostics) {
	planned := data.ManagedClusterComputedModel
	setManagedClusterComputedState(ctx, &data.ManagedClusterComputedModel, cluster, diags)
	if !refresh {
		// id, pod, org and created_at never change (UseStateForUnknown): an update keeps the planned values.
		for _, attribute := range []struct{ planned, target *types.String }{
			{&planned.ID, &data.ID}, {&planned.Pod, &data.Pod}, {&planned.Org, &data.Org}, {&planned.CreatedAt, &data.CreatedAt},
		} {
			if !attribute.planned.IsUnknown() {
				*attribute.target = *attribute.planned
			}
		}
	}
	if refresh {
		data.Name = types.StringValue(cluster.Name)
		data.Type = types.StringValue(cluster.Type)
		data.Description = types.StringValue(cluster.Description)
		data.Configuration = managedClusterConfigurationState(ctx, data.Configuration, cluster.Configuration, true, diags)
		if data.RemoveClientsOnDestroy.IsNull() || data.RemoveClientsOnDestroy.IsUnknown() {
			data.RemoveClientsOnDestroy = types.BoolValue(false)
		}
		return
	}
	data.Type = computedStringFromAPI(data.Type, cluster.Type)
	data.Description = computedStringFromAPI(data.Description, cluster.Description)
}

// managedClusterPatchOps returns the operations for the changed attributes. The configuration is
// replaced with the merge of the current configuration and the planned entries.
func managedClusterPatchOps(ctx context.Context, plan, state ManagedClusterModel, current *ManagedCluster, diags *diag.Diagnostics) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	if !plan.Configuration.Equal(state.Configuration) && current != nil {
		merged := managedClusterMergedConfiguration(
			current.Configuration,
			managedClusterConfigurationValue(ctx, state.Configuration, diags),
			managedClusterConfigurationValue(ctx, plan.Configuration, diags),
		)
		b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/configuration", Value: merged})
	}
	return b.ops
}

func (r *ManagedClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ManagedClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request := &ManagedClusterRequest{
		Name:          data.Name.ValueString(),
		Type:          data.Type.ValueString(),
		Configuration: managedClusterConfigurationValue(ctx, data.Configuration, &resp.Diagnostics),
		Description:   stringPointer(data.Description),
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateManagedCluster(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create managed cluster: %s", err))
		return
	}
	setManagedClusterState(ctx, &data, created, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ManagedClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	cluster, err := client.GetManagedCluster(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed cluster: %s", err))
		return
	}
	setManagedClusterState(ctx, &data, cluster, true, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ManagedClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// The current cluster is needed to merge the configuration and to resolve the computed
	// attributes when only remove_clients_on_destroy changed.
	current, err := client.GetManagedCluster(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed cluster: %s", err))
		return
	}
	ops := managedClusterPatchOps(ctx, data, state, current, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(ops) > 0 {
		current, err = client.PatchManagedCluster(ctx, data.ID.ValueString(), ops)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update managed cluster: %s", err))
			return
		}
	}
	setManagedClusterState(ctx, &data, current, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ManagedClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteManagedCluster(ctx, data.ID.ValueString(), data.RemoveClientsOnDestroy.ValueBool()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete managed cluster: %s", err))
	}
}

func (r *ManagedClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &ManagedClusterDataSource{}
var _ datasource.DataSourceWithValidateConfig = &ManagedClusterDataSource{}

func NewManagedClusterDataSource() datasource.DataSource {
	return &ManagedClusterDataSource{}
}

type ManagedClusterDataSource struct {
	client *Config
}

func (d *ManagedClusterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_cluster"
}

func (d *ManagedClusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			MarkdownDescription: "Managed cluster ID. Exactly one of `id` or `name` must be set.",
			Optional:            true,
			Computed:            true,
		},
		"name": dsschema.StringAttribute{
			MarkdownDescription: "Managed cluster name. Exactly one of `id` or `name` must be set.",
			Optional:            true,
			Computed:            true,
		},
		"type":        dsschema.StringAttribute{MarkdownDescription: "Cluster type.", Computed: true},
		"description": dsschema.StringAttribute{MarkdownDescription: "Description of the cluster.", Computed: true},
		"configuration": dsschema.MapAttribute{
			MarkdownDescription: "Cluster configuration entries.",
			ElementType:         types.StringType,
			Computed:            true,
		},
		"pinned_config": dsschema.BoolAttribute{MarkdownDescription: "Whether the cluster configuration is pinned.", Computed: true},
		"operational":   dsschema.BoolAttribute{MarkdownDescription: "Whether the cluster is operational.", Computed: true},
		"client_ids": dsschema.ListAttribute{
			MarkdownDescription: "IDs of the clients of the cluster.",
			ElementType:         types.StringType,
			Computed:            true,
		},
		"service_count": dsschema.Int64Attribute{MarkdownDescription: "Number of services bound to the cluster.", Computed: true},
		"created_at":    dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
	}
	for name, description := range managedClusterComputedDescriptions {
		attributes[name] = dsschema.StringAttribute{MarkdownDescription: description, Computed: true}
	}
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a managed cluster by ID or name.",
		Attributes:          attributes,
	}
}

func (d *ManagedClusterDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *ManagedClusterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// setManagedClusterDataSourceState maps the API cluster onto the data source model.
func setManagedClusterDataSourceState(ctx context.Context, data *ManagedClusterComputedModel, cluster *ManagedCluster, diags *diag.Diagnostics) {
	setManagedClusterComputedState(ctx, data, cluster, diags)
	data.Name = types.StringValue(cluster.Name)
	data.Type = types.StringValue(cluster.Type)
	data.Description = types.StringValue(cluster.Description)
	data.Configuration = managedClusterConfigurationState(ctx, types.MapNull(types.StringType), cluster.Configuration, false, diags)
}

func (d *ManagedClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ManagedClusterComputedModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var cluster *ManagedCluster
	if !data.ID.IsNull() {
		cluster, err = client.GetManagedCluster(ctx, data.ID.ValueString())
	} else {
		cluster, err = client.GetManagedClusterByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Managed cluster not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed cluster: %s", err))
		return
	}
	setManagedClusterDataSourceState(ctx, &data, cluster, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
