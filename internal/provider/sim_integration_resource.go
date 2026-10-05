package provider

import (
	"context"
	"encoding/json"
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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewSimIntegrationResource)
	registerDataSource(NewSimIntegrationDataSource)
}

// SimIntegration is a service integration module (SIM) integration of the experimental
// /v2026/sim-integrations API.
type SimIntegration struct {
	ID                     string                     `json:"id,omitempty"`
	Name                   string                     `json:"name"`
	Created                string                     `json:"created,omitempty"`
	Modified               string                     `json:"modified,omitempty"`
	Description            string                     `json:"description,omitempty"`
	Type                   string                     `json:"type,omitempty"`
	Attributes             map[string]interface{}     `json:"attributes,omitempty"`
	Sources                []string                   `json:"sources,omitempty"`
	Cluster                string                     `json:"cluster,omitempty"`
	StatusMap              map[string]interface{}     `json:"statusMap,omitempty"`
	Request                map[string]interface{}     `json:"request,omitempty"`
	BeforeProvisioningRule *ServiceDeskIntegrationRef `json:"beforeProvisioningRule,omitempty"`
}

// simIntegrationResponse decodes a SIM integration response. The spec documents the responses with
// the service desk integration schema (managedSources, clusterRef), so both shapes are accepted.
// The spec example shows the attributes as a JSON encoded string, which is accepted as well.
type simIntegrationResponse struct {
	SimIntegration
	Attributes     json.RawMessage            `json:"attributes,omitempty"`
	ManagedSources []string                   `json:"managedSources,omitempty"`
	ClusterRef     *ServiceDeskIntegrationRef `json:"clusterRef,omitempty"`
}

// simIntegrationNormalize returns the SIM integration of a response, filling sources and cluster
// from the service desk integration fields when only those are returned.
func simIntegrationNormalize(response *simIntegrationResponse) *SimIntegration {
	integration := response.SimIntegration
	integration.Attributes = nil
	var encoded string
	if json.Unmarshal(response.Attributes, &encoded) == nil {
		_ = json.Unmarshal([]byte(encoded), &integration.Attributes)
	} else {
		_ = json.Unmarshal(response.Attributes, &integration.Attributes)
	}
	if len(integration.Sources) == 0 && len(response.ManagedSources) > 0 {
		integration.Sources = response.ManagedSources
	}
	if integration.Cluster == "" && response.ClusterRef != nil {
		integration.Cluster = response.ClusterRef.ID
	}
	return &integration
}

func (c *Client) GetSimIntegration(ctx context.Context, id string) (*SimIntegration, error) {
	var response simIntegrationResponse
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/sim-integrations/%s", id), nil, &response, withExperimental()); err != nil {
		return nil, err
	}
	return simIntegrationNormalize(&response), nil
}

func (c *Client) CreateSimIntegration(ctx context.Context, integration *SimIntegration) (*SimIntegration, error) {
	var response simIntegrationResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/sim-integrations", integration, &response, withExperimental()); err != nil {
		return nil, err
	}
	return simIntegrationNormalize(&response), nil
}

func (c *Client) UpdateSimIntegration(ctx context.Context, id string, integration *SimIntegration) (*SimIntegration, error) {
	var response simIntegrationResponse
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/sim-integrations/%s", id), integration, &response, withExperimental()); err != nil {
		return nil, err
	}
	return simIntegrationNormalize(&response), nil
}

func (c *Client) DeleteSimIntegration(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/sim-integrations/%s", id), nil, nil, withExperimental())
}

var _ resource.Resource = &SimIntegrationResource{}
var _ resource.ResourceWithImportState = &SimIntegrationResource{}

func NewSimIntegrationResource() resource.Resource {
	return &SimIntegrationResource{}
}

type SimIntegrationResource struct {
	client *Config
}

type SimIntegrationModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	Type                   types.String `tfsdk:"type"`
	Sources                types.List   `tfsdk:"sources"`
	Cluster                types.String `tfsdk:"cluster"`
	StatusMapJSON          types.String `tfsdk:"status_map_json"`
	RequestJSON            types.String `tfsdk:"request_json"`
	AttributesJSON         types.String `tfsdk:"attributes_json"`
	BeforeProvisioningRule types.List   `tfsdk:"before_provisioning_rule"`
	Created                types.String `tfsdk:"created"`
	Modified               types.String `tfsdk:"modified"`
}

func (r *SimIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sim_integration"
}

func (r *SimIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a service integration module (SIM) integration, which creates service desk tickets for provisioning requests of the managed sources. Uses an experimental API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "SIM integration ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the integration.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the integration.",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Integration type, e.g. `ServiceNow Service Desk`. When not set, the value returned by the API is used.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sources": schema.ListAttribute{
				MarkdownDescription: "IDs of the sources (managed resources) of the integration.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"cluster": schema.StringAttribute{
				MarkdownDescription: "ID of the virtual appliance cluster the integration uses.",
				Optional:            true,
			},
			"status_map_json": schema.StringAttribute{
				MarkdownDescription: "Mapping between ticket statuses and provisioning results as a JSON object, e.g. `{\"closed_complete\" = \"Committed\"}`. When not set, the value returned by the API is ignored. Keys added by the API do not cause a diff.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"request_json": schema.StringAttribute{
				MarkdownDescription: "Request data that customizes the description and body of the created tickets, as a JSON object. When not set, the value returned by the API is ignored. Keys added by the API do not cause a diff.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"attributes_json": schema.StringAttribute{
				MarkdownDescription: "Integration attributes as a JSON object, including the credentials used to connect to the service desk. The value is sensitive. Keys that the API does not return, such as passwords, and keys added by the API do not cause a diff. When not set, the value returned by the API is ignored.",
				Optional:            true,
				Sensitive:           true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
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
			"before_provisioning_rule": serviceDeskIntegrationRefBlock("Before provisioning rule of the integration.", "RULE"),
		},
	}
}

func (r *SimIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// simIntegrationFromModel converts the model to the API object.
func simIntegrationFromModel(ctx context.Context, data SimIntegrationModel, diags *diag.Diagnostics) *SimIntegration {
	return &SimIntegration{
		Name:                   data.Name.ValueString(),
		Description:            data.Description.ValueString(),
		Type:                   data.Type.ValueString(),
		Attributes:             serviceDeskIntegrationJSONMap(data.AttributesJSON, "attributes_json", diags),
		Sources:                serviceDeskIntegrationStringList(ctx, data.Sources, diags),
		Cluster:                data.Cluster.ValueString(),
		StatusMap:              serviceDeskIntegrationJSONMap(data.StatusMapJSON, "status_map_json", diags),
		Request:                serviceDeskIntegrationJSONMap(data.RequestJSON, "request_json", diags),
		BeforeProvisioningRule: serviceDeskIntegrationRefFromList(ctx, data.BeforeProvisioningRule, "RULE", diags),
	}
}

// simIntegrationOptionalJSONState returns the state of an optional (not computed) JSON attribute
// read from the API. An unset attribute stays null, since the API can return defaults for it; with
// populate (import, data source) the API value is used. A set attribute keeps its prior value when
// the API value contains it (keys added by the API are ignored), see jsonSubsetState.
func simIntegrationOptionalJSONState(prior types.String, value map[string]interface{}, populate bool) types.String {
	if (prior.IsNull() || prior.IsUnknown()) && !populate {
		return types.StringNull()
	}
	if prior.IsUnknown() {
		prior = types.StringNull()
	}
	return jsonSubsetState(prior, value)
}

// setSimIntegrationState maps the API object onto the model. With refresh every attribute comes
// from the API; otherwise the planned values are kept and only computed values are resolved.
func setSimIntegrationState(ctx context.Context, data *SimIntegrationModel, integration *SimIntegration, refresh bool, diags *diag.Diagnostics) {
	if refresh {
		data.ID = types.StringValue(integration.ID)
		data.Created = types.StringValue(integration.Created)
	} else {
		// id and created are known in the plan of an update (UseStateForUnknown) and keep that value.
		data.ID = computedStringFromAPI(data.ID, integration.ID)
		data.Created = computedStringFromAPI(data.Created, integration.Created)
	}
	data.Modified = types.StringValue(integration.Modified)
	data.BeforeProvisioningRule = serviceDeskIntegrationRefState(ctx, data.BeforeProvisioningRule, integration.BeforeProvisioningRule, "RULE", refresh, diags)
	if !refresh {
		data.Type = computedStringFromAPI(data.Type, integration.Type)
		return
	}
	// The required name is only null after an import and in the data source: then the optional JSON
	// attributes are populated from the API, otherwise unset attributes stay null.
	populate := data.Name.IsNull()
	data.Name = types.StringValue(integration.Name)
	data.Description = optionalStringState(data.Description, integration.Description)
	data.Type = types.StringValue(integration.Type)
	data.Sources = serviceDeskIntegrationStringListState(ctx, data.Sources, integration.Sources, diags)
	data.Cluster = optionalStringState(data.Cluster, integration.Cluster)
	data.StatusMapJSON = simIntegrationOptionalJSONState(data.StatusMapJSON, integration.StatusMap, populate)
	data.RequestJSON = simIntegrationOptionalJSONState(data.RequestJSON, integration.Request, populate)
	if (data.AttributesJSON.IsNull() || data.AttributesJSON.IsUnknown()) && !populate {
		data.AttributesJSON = types.StringNull()
	} else {
		data.AttributesJSON = serviceDeskIntegrationSubsetJSONState(data.AttributesJSON, integration.Attributes)
	}
}

func (r *SimIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SimIntegrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	integration := simIntegrationFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateSimIntegration(ctx, integration)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create SIM integration: %s", err))
		return
	}
	setSimIntegrationState(ctx, &data, created, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SimIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SimIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	integration, err := client.GetSimIntegration(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SIM integration: %s", err))
		return
	}
	setSimIntegrationState(ctx, &data, integration, true, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SimIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SimIntegrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	integration := simIntegrationFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// The model covers the whole request body, so PUT sends the complete planned integration.
	updated, err := client.UpdateSimIntegration(ctx, data.ID.ValueString(), integration)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update SIM integration: %s", err))
		return
	}
	setSimIntegrationState(ctx, &data, updated, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SimIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SimIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteSimIntegration(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete SIM integration: %s", err))
	}
}

func (r *SimIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &SimIntegrationDataSource{}

func NewSimIntegrationDataSource() datasource.DataSource {
	return &SimIntegrationDataSource{}
}

type SimIntegrationDataSource struct {
	client *Config
}

func (d *SimIntegrationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sim_integration"
}

func (d *SimIntegrationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a SIM integration by ID. Uses an experimental API.",
		Attributes: map[string]dsschema.Attribute{
			"id":          dsschema.StringAttribute{MarkdownDescription: "SIM integration ID.", Required: true},
			"name":        dsschema.StringAttribute{MarkdownDescription: "Name of the integration.", Computed: true},
			"description": dsschema.StringAttribute{MarkdownDescription: "Description of the integration.", Computed: true},
			"type":        dsschema.StringAttribute{MarkdownDescription: "Integration type.", Computed: true},
			"sources": dsschema.ListAttribute{
				MarkdownDescription: "IDs of the sources of the integration.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"cluster":         dsschema.StringAttribute{MarkdownDescription: "ID of the virtual appliance cluster.", Computed: true},
			"status_map_json": dsschema.StringAttribute{MarkdownDescription: "Status mapping as a JSON object.", Computed: true},
			"request_json":    dsschema.StringAttribute{MarkdownDescription: "Ticket request data as a JSON object.", Computed: true},
			"attributes_json": dsschema.StringAttribute{
				MarkdownDescription: "Integration attributes as a JSON object, as returned by the API.",
				Computed:            true,
				Sensitive:           true,
			},
			"before_provisioning_rule": serviceDeskIntegrationRefDataSourceAttribute("Before provisioning rule of the integration."),
			"created":                  dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"modified":                 dsschema.StringAttribute{MarkdownDescription: "Last modification date.", Computed: true},
		},
	}
}

func (d *SimIntegrationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SimIntegrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SimIntegrationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	integration, err := client.GetSimIntegration(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("id"), "SIM integration not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SIM integration: %s", err))
		return
	}
	setSimIntegrationState(ctx, &data, integration, true, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
