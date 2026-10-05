package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewManagedClientResource)
	registerDataSource(NewManagedClientDataSource)
}

// ManagedClient is a managed client (virtual appliance or CCG) as returned by the /v2026/managed-clients API.
type ManagedClient struct {
	ID                string `json:"id,omitempty"`
	AlertKey          string `json:"alertKey,omitempty"`
	APIGatewayBaseURL string `json:"apiGatewayBaseUrl,omitempty"`
	ClientID          string `json:"clientId,omitempty"`
	ClusterID         string `json:"clusterId"`
	Description       string `json:"description,omitempty"`
	IPAddress         string `json:"ipAddress,omitempty"`
	LastSeen          string `json:"lastSeen,omitempty"`
	Name              string `json:"name,omitempty"`
	SinceLastSeen     string `json:"sinceLastSeen,omitempty"`
	Status            string `json:"status,omitempty"`
	Type              string `json:"type,omitempty"`
	ClusterType       string `json:"clusterType,omitempty"`
	VaDownloadURL     string `json:"vaDownloadUrl,omitempty"`
	VaVersion         string `json:"vaVersion,omitempty"`
	Secret            string `json:"secret,omitempty"`
	CreatedAt         string `json:"createdAt,omitempty"`
	UpdatedAt         string `json:"updatedAt,omitempty"`
	ProvisionStatus   string `json:"provisionStatus,omitempty"`
}

// ManagedClientRequest is the request body to create a managed client.
type ManagedClientRequest struct {
	ClusterID   string  `json:"clusterId"`
	Description *string `json:"description,omitempty"`
	Name        *string `json:"name,omitempty"`
	Type        *string `json:"type,omitempty"`
}

func (c *Client) GetManagedClient(ctx context.Context, id string) (*ManagedClient, error) {
	var managedClient ManagedClient
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/managed-clients/%s", id), nil, &managedClient); err != nil {
		return nil, err
	}
	return &managedClient, nil
}

func (c *Client) CreateManagedClient(ctx context.Context, request *ManagedClientRequest) (*ManagedClient, error) {
	var created ManagedClient
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/managed-clients", request, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchManagedClient(ctx context.Context, id string, ops []jsonPatchOp) (*ManagedClient, error) {
	var updated ManagedClient
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/managed-clients/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteManagedClient(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/managed-clients/%s", id), nil, nil)
}

var _ resource.Resource = &ManagedClientResource{}
var _ resource.ResourceWithImportState = &ManagedClientResource{}

func NewManagedClientResource() resource.Resource {
	return &ManagedClientResource{}
}

type ManagedClientResource struct {
	client *Config
}

type ManagedClientModel struct {
	ID                types.String `tfsdk:"id"`
	ClusterID         types.String `tfsdk:"cluster_id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Type              types.String `tfsdk:"type"`
	ClientID          types.String `tfsdk:"client_id"`
	Secret            types.String `tfsdk:"secret"`
	Status            types.String `tfsdk:"status"`
	ClusterType       types.String `tfsdk:"cluster_type"`
	AlertKey          types.String `tfsdk:"alert_key"`
	APIGatewayBaseURL types.String `tfsdk:"api_gateway_base_url"`
	IPAddress         types.String `tfsdk:"ip_address"`
	LastSeen          types.String `tfsdk:"last_seen"`
	SinceLastSeen     types.String `tfsdk:"since_last_seen"`
	VaDownloadURL     types.String `tfsdk:"va_download_url"`
	VaVersion         types.String `tfsdk:"va_version"`
	ProvisionStatus   types.String `tfsdk:"provision_status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (r *ManagedClientResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_client"
}

// managedClientComputedDescriptions documents the read-only attributes that change over time.
var managedClientComputedDescriptions = map[string]string{
	"status":               "Status of the client, e.g. `NORMAL`, `CONFIGURING` or `ERROR`.",
	"cluster_type":         "Type of the cluster the client belongs to.",
	"alert_key":            "Key describing any immediate client alerts.",
	"api_gateway_base_url": "API gateway base URL of the client.",
	"ip_address":           "Public IP address of the client.",
	"last_seen":            "When the client was last seen by the server.",
	"since_last_seen":      "Milliseconds since the client last polled the server.",
	"va_download_url":      "Virtual appliance download URL.",
	"va_version":           "Version of the virtual appliance software the client runs.",
	"provision_status":     "Provisioning status of the client, `PROVISIONED` or `DRAFT`.",
	"updated_at":           "Last update date.",
}

func (r *ManagedClientResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "Managed client ID.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"cluster_id": schema.StringAttribute{
			MarkdownDescription: "ID of the managed cluster the client belongs to. Changing this forces a new client to be created.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Name of the client. When not set, the API generates one (`VA-<client_id>`). Since the API always has a name, removing the argument keeps the current name instead of clearing it.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Description of the client.",
			Optional:            true,
		},
		"type": schema.StringAttribute{
			MarkdownDescription: "Client type, `VA` or `CCG`. When not set, the API default is stored. Changing this forces a new client to be created.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
		},
		"client_id": schema.StringAttribute{
			MarkdownDescription: "Client ID used in API management.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"secret": schema.StringAttribute{
			MarkdownDescription: "API key of the client. The value is sensitive. It is taken from the create response and kept in state, since the API does not return it again.",
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"created_at": schema.StringAttribute{
			MarkdownDescription: "Creation date.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}
	for name, description := range managedClientComputedDescriptions {
		attributes[name] = schema.StringAttribute{MarkdownDescription: description, Computed: true}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a managed client, a virtual appliance or connector gateway registered in a managed cluster.",
		Attributes:          attributes,
	}
}

func (r *ManagedClientResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// setManagedClientState maps the API client onto the model. With refresh the configurable
// attributes are refreshed from the API; otherwise the planned values are kept and unknown values
// are resolved. The secret is only taken from the API while state has none.
func setManagedClientState(data *ManagedClientModel, managedClient *ManagedClient, refresh bool) {
	if refresh || data.ID.IsUnknown() {
		data.ID = types.StringValue(managedClient.ID)
	}
	if refresh || data.ClientID.IsUnknown() {
		data.ClientID = types.StringValue(managedClient.ClientID)
	}
	// id, client_id and created_at never change (UseStateForUnknown): an update keeps the planned values.
	if refresh || data.CreatedAt.IsUnknown() {
		data.CreatedAt = stringValueOrNull(managedClient.CreatedAt)
	}
	if data.Secret.IsNull() || data.Secret.IsUnknown() || data.Secret.ValueString() == "" {
		data.Secret = stringValueOrNull(managedClient.Secret)
	}
	data.Status = stringValueOrNull(managedClient.Status)
	data.ClusterType = stringValueOrNull(managedClient.ClusterType)
	data.AlertKey = stringValueOrNull(managedClient.AlertKey)
	data.APIGatewayBaseURL = stringValueOrNull(managedClient.APIGatewayBaseURL)
	data.IPAddress = stringValueOrNull(managedClient.IPAddress)
	data.LastSeen = stringValueOrNull(managedClient.LastSeen)
	data.SinceLastSeen = stringValueOrNull(managedClient.SinceLastSeen)
	data.VaDownloadURL = stringValueOrNull(managedClient.VaDownloadURL)
	data.VaVersion = stringValueOrNull(managedClient.VaVersion)
	data.ProvisionStatus = stringValueOrNull(managedClient.ProvisionStatus)
	data.UpdatedAt = stringValueOrNull(managedClient.UpdatedAt)
	if refresh {
		data.ClusterID = types.StringValue(managedClient.ClusterID)
		data.Name = types.StringValue(managedClient.Name)
		data.Description = optionalStringState(data.Description, managedClient.Description)
		data.Type = types.StringValue(managedClient.Type)
		return
	}
	data.Name = computedStringFromAPI(data.Name, managedClient.Name)
	data.Type = computedStringFromAPI(data.Type, managedClient.Type)
}

// managedClientPatchOps returns the operations for the changed attributes. A removed description
// is replaced with an empty string, the API default.
func managedClientPatchOps(plan, state ManagedClientModel) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	return b.ops
}

func (r *ManagedClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ManagedClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateManagedClient(ctx, &ManagedClientRequest{
		ClusterID:   data.ClusterID.ValueString(),
		Description: stringPointer(data.Description),
		Name:        stringPointer(data.Name),
		Type:        stringPointer(data.Type),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create managed client: %s", err))
		return
	}
	setManagedClientState(&data, created, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ManagedClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	managedClient, err := client.GetManagedClient(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed client: %s", err))
		return
	}
	setManagedClientState(&data, managedClient, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ManagedClientModel
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
	var updated *ManagedClient
	if ops := managedClientPatchOps(data, state); len(ops) > 0 {
		updated, err = client.PatchManagedClient(ctx, data.ID.ValueString(), ops)
	} else {
		updated, err = client.GetManagedClient(ctx, data.ID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update managed client: %s", err))
		return
	}
	setManagedClientState(&data, updated, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ManagedClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteManagedClient(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete managed client: %s", err))
	}
}

func (r *ManagedClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &ManagedClientDataSource{}

func NewManagedClientDataSource() datasource.DataSource {
	return &ManagedClientDataSource{}
}

type ManagedClientDataSource struct {
	client *Config
}

// ManagedClientDataSourceModel is the data source model, which does not expose the secret.
type ManagedClientDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	ClusterID         types.String `tfsdk:"cluster_id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Type              types.String `tfsdk:"type"`
	ClientID          types.String `tfsdk:"client_id"`
	Status            types.String `tfsdk:"status"`
	ClusterType       types.String `tfsdk:"cluster_type"`
	AlertKey          types.String `tfsdk:"alert_key"`
	APIGatewayBaseURL types.String `tfsdk:"api_gateway_base_url"`
	IPAddress         types.String `tfsdk:"ip_address"`
	LastSeen          types.String `tfsdk:"last_seen"`
	SinceLastSeen     types.String `tfsdk:"since_last_seen"`
	VaDownloadURL     types.String `tfsdk:"va_download_url"`
	VaVersion         types.String `tfsdk:"va_version"`
	ProvisionStatus   types.String `tfsdk:"provision_status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (d *ManagedClientDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_client"
}

func (d *ManagedClientDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]dsschema.Attribute{
		"id":          dsschema.StringAttribute{MarkdownDescription: "Managed client ID.", Required: true},
		"cluster_id":  dsschema.StringAttribute{MarkdownDescription: "ID of the managed cluster the client belongs to.", Computed: true},
		"name":        dsschema.StringAttribute{MarkdownDescription: "Name of the client.", Computed: true},
		"description": dsschema.StringAttribute{MarkdownDescription: "Description of the client.", Computed: true},
		"type":        dsschema.StringAttribute{MarkdownDescription: "Client type, `VA` or `CCG`.", Computed: true},
		"client_id":   dsschema.StringAttribute{MarkdownDescription: "Client ID used in API management.", Computed: true},
		"created_at":  dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
	}
	for name, description := range managedClientComputedDescriptions {
		attributes[name] = dsschema.StringAttribute{MarkdownDescription: description, Computed: true}
	}
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a managed client by ID.",
		Attributes:          attributes,
	}
}

func (d *ManagedClientDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// managedClientDataSourceState converts the API client to the data source model.
func managedClientDataSourceState(managedClient *ManagedClient) ManagedClientDataSourceModel {
	var full ManagedClientModel
	setManagedClientState(&full, managedClient, true)
	return ManagedClientDataSourceModel{
		ID:                full.ID,
		ClusterID:         full.ClusterID,
		Name:              full.Name,
		Description:       stringValueOrNull(managedClient.Description),
		Type:              full.Type,
		ClientID:          full.ClientID,
		Status:            full.Status,
		ClusterType:       full.ClusterType,
		AlertKey:          full.AlertKey,
		APIGatewayBaseURL: full.APIGatewayBaseURL,
		IPAddress:         full.IPAddress,
		LastSeen:          full.LastSeen,
		SinceLastSeen:     full.SinceLastSeen,
		VaDownloadURL:     full.VaDownloadURL,
		VaVersion:         full.VaVersion,
		ProvisionStatus:   full.ProvisionStatus,
		CreatedAt:         full.CreatedAt,
		UpdatedAt:         full.UpdatedAt,
	}
}

func (d *ManagedClientDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var id types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	managedClient, err := client.GetManagedClient(ctx, id.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("id"), "Managed client not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed client: %s", err))
		return
	}
	data := managedClientDataSourceState(managedClient)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
