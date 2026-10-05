package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	registerResource(NewServiceDeskIntegrationResource)
	registerDataSource(NewServiceDeskIntegrationDataSource, NewServiceDeskIntegrationTypesDataSource)
}

// ServiceDeskIntegrationRef is a typed reference (owner, cluster, rule) used by service desk and SIM integrations.
type ServiceDeskIntegrationRef struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// ServiceDeskIntegration is a service desk integration as returned by the /v2026/service-desk-integrations API.
type ServiceDeskIntegration struct {
	ID                     string                     `json:"id,omitempty"`
	Name                   string                     `json:"name"`
	Description            string                     `json:"description"`
	Type                   string                     `json:"type"`
	Created                string                     `json:"created,omitempty"`
	Modified               string                     `json:"modified,omitempty"`
	OwnerRef               *ServiceDeskIntegrationRef `json:"ownerRef,omitempty"`
	ClusterRef             *ServiceDeskIntegrationRef `json:"clusterRef,omitempty"`
	ManagedSources         []string                   `json:"managedSources,omitempty"`
	ProvisioningConfig     map[string]interface{}     `json:"provisioningConfig,omitempty"`
	Attributes             map[string]interface{}     `json:"attributes"`
	BeforeProvisioningRule *ServiceDeskIntegrationRef `json:"beforeProvisioningRule,omitempty"`
}

// ServiceDeskIntegrationType is a supported service desk integration type.
type ServiceDeskIntegrationType struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	ScriptName string `json:"scriptName"`
}

func (c *Client) GetServiceDeskIntegration(ctx context.Context, id string) (*ServiceDeskIntegration, error) {
	var integration ServiceDeskIntegration
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/service-desk-integrations/%s", id), nil, &integration); err != nil {
		return nil, err
	}
	return &integration, nil
}

func (c *Client) GetServiceDeskIntegrationByName(ctx context.Context, name string) (*ServiceDeskIntegration, error) {
	return findByName(ctx, c, "/v2026/service-desk-integrations", "service desk integration", name, func(i ServiceDeskIntegration) string { return i.Name })
}

func (c *Client) CreateServiceDeskIntegration(ctx context.Context, integration *ServiceDeskIntegration) (*ServiceDeskIntegration, error) {
	var created ServiceDeskIntegration
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/service-desk-integrations", integration, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateServiceDeskIntegration(ctx context.Context, id string, integration *ServiceDeskIntegration) (*ServiceDeskIntegration, error) {
	var updated ServiceDeskIntegration
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/service-desk-integrations/%s", id), integration, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteServiceDeskIntegration(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/service-desk-integrations/%s", id), nil, nil)
}

func (c *Client) ListServiceDeskIntegrationTypes(ctx context.Context) ([]ServiceDeskIntegrationType, error) {
	var result []ServiceDeskIntegrationType
	if err := c.doJSON(ctx, http.MethodGet, "/v2026/service-desk-integrations/types", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

var _ resource.Resource = &ServiceDeskIntegrationResource{}
var _ resource.ResourceWithImportState = &ServiceDeskIntegrationResource{}

func NewServiceDeskIntegrationResource() resource.Resource {
	return &ServiceDeskIntegrationResource{}
}

type ServiceDeskIntegrationResource struct {
	client *Config
}

type ServiceDeskIntegrationModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	Type                   types.String `tfsdk:"type"`
	OwnerRef               types.List   `tfsdk:"owner_ref"`
	ClusterRef             types.List   `tfsdk:"cluster_ref"`
	ManagedSources         types.List   `tfsdk:"managed_sources"`
	ProvisioningConfigJSON types.String `tfsdk:"provisioning_config_json"`
	AttributesJSON         types.String `tfsdk:"attributes_json"`
	BeforeProvisioningRule types.List   `tfsdk:"before_provisioning_rule"`
	Created                types.String `tfsdk:"created"`
	Modified               types.String `tfsdk:"modified"`
}

func (r *ServiceDeskIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_desk_integration"
}

// serviceDeskIntegrationRefBlock is a single optional reference block with an ID, a type that
// defaults to defaultType and a name resolved by the API.
func serviceDeskIntegrationRefBlock(description, defaultType string) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: description,
		Validators:          []validator.List{listSizeBetween(0, 1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "ID of the referenced object.",
					Required:            true,
				},
				"type": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Type of the referenced object. Defaults to `%s`.", defaultType),
					Optional:            true,
					Computed:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				// No UseStateForUnknown: the name of a changed reference must be resolved again.
				"name": schema.StringAttribute{
					MarkdownDescription: "Name of the referenced object. Resolved by the API when not set.",
					Optional:            true,
					Computed:            true,
				},
			},
		},
	}
}

func (r *ServiceDeskIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a service desk integration (SDIM), which sends provisioning requests of the managed sources to a service desk such as ServiceNow.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Service desk integration ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Unique name of the service desk integration.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the service desk integration.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Service desk integration type, e.g. `ServiceNowSDIM`. See the `identitynow_service_desk_integration_types` data source for the supported types.",
				Required:            true,
			},
			"managed_sources": schema.ListAttribute{
				MarkdownDescription: "IDs of the sources managed by the integration. Deprecated by the API in favor of `managedResourceRefs` in `provisioning_config_json`; when not set, the value returned by the API is used.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"provisioning_config_json": schema.StringAttribute{
				MarkdownDescription: "Provisioning configuration as a JSON object, with keys such as `managedResourceRefs`, `planInitializerScript`, `noProvisioningRequests` and `provisioningRequestExpiration`. When not set, the value returned by the API is used and kept, so removing the argument does not clear the configuration; set it to `{}` to clear it. Keys added by the API, such as the read-only `universalManager` (never sent to the API), do not cause a diff.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"attributes_json": schema.StringAttribute{
				MarkdownDescription: "Integration attributes as a JSON object, e.g. the service desk URL and credentials. The value is sensitive. Keys that the API does not return, such as passwords, and keys added by the API do not cause a diff.",
				Required:            true,
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
			"owner_ref":                serviceDeskIntegrationRefBlock("Identity that owns the integration.", "IDENTITY"),
			"cluster_ref":              serviceDeskIntegrationRefBlock("Virtual appliance cluster the integration uses.", "CLUSTER"),
			"before_provisioning_rule": serviceDeskIntegrationRefBlock("Before provisioning rule of the integration.", "RULE"),
		},
	}
}

func (r *ServiceDeskIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// serviceDeskIntegrationRefFromList converts a single-element reference block to the API reference.
func serviceDeskIntegrationRefFromList(ctx context.Context, list types.List, defaultType string, diags *diag.Diagnostics) *ServiceDeskIntegrationRef {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil
	}
	var models []OwnerModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if len(models) == 0 {
		return nil
	}
	ref := &ServiceDeskIntegrationRef{ID: models[0].ID.ValueString(), Type: defaultType}
	if !models[0].Type.IsNull() && !models[0].Type.IsUnknown() {
		ref.Type = models[0].Type.ValueString()
	}
	if !models[0].Name.IsNull() && !models[0].Name.IsUnknown() {
		ref.Name = models[0].Name.ValueString()
	}
	return ref
}

// serviceDeskIntegrationRefState returns the state of a reference block. On refresh the value comes
// from the API, but when the referenced ID is unchanged the prior type and name are kept: they are
// Optional and Computed, and a configured name that differs from the display name returned by the
// API must not cause a diff. After apply the planned block is kept and only unknown type and name
// are resolved.
func serviceDeskIntegrationRefState(ctx context.Context, planned types.List, api *ServiceDeskIntegrationRef, defaultType string, refresh bool, diags *diag.Diagnostics) types.List {
	var model OwnerModel
	if refresh {
		if api == nil || api.ID == "" {
			return types.ListNull(objectInfoObjectType)
		}
		model = OwnerModel{ID: types.StringValue(api.ID), Type: types.StringValue(api.Type), Name: types.StringValue(api.Name)}
		if api.Type == "" {
			model.Type = types.StringValue(defaultType)
		}
		if !planned.IsNull() && !planned.IsUnknown() && len(planned.Elements()) > 0 {
			var priors []OwnerModel
			diags.Append(planned.ElementsAs(ctx, &priors, false)...)
			if len(priors) > 0 && priors[0].ID.ValueString() == api.ID {
				if !priors[0].Type.IsNull() && !priors[0].Type.IsUnknown() {
					model.Type = priors[0].Type
				}
				if !priors[0].Name.IsNull() && !priors[0].Name.IsUnknown() {
					model.Name = priors[0].Name
				}
			}
		}
	} else {
		if planned.IsNull() || planned.IsUnknown() || len(planned.Elements()) == 0 {
			return types.ListNull(objectInfoObjectType)
		}
		var models []OwnerModel
		diags.Append(planned.ElementsAs(ctx, &models, false)...)
		if len(models) == 0 {
			return types.ListNull(objectInfoObjectType)
		}
		model = models[0]
		apiType, apiName := defaultType, ""
		if api != nil {
			if api.Type != "" {
				apiType = api.Type
			}
			apiName = api.Name
		}
		if model.Type.IsUnknown() || model.Type.IsNull() {
			model.Type = types.StringValue(apiType)
		}
		if model.Name.IsUnknown() || model.Name.IsNull() {
			model.Name = types.StringValue(apiName)
		}
	}
	list, d := types.ListValueFrom(ctx, objectInfoObjectType, []OwnerModel{model})
	diags.Append(d...)
	return list
}

// serviceDeskIntegrationStringListState returns the state of a string list read from the API. The
// order of the prior value is kept, and an empty API value keeps a null prior value.
func serviceDeskIntegrationStringListState(ctx context.Context, prior types.List, values []string, diags *diag.Diagnostics) types.List {
	if len(values) == 0 {
		if prior.IsNull() || prior.IsUnknown() {
			return types.ListNull(types.StringType)
		}
		return types.ListValueMust(types.StringType, []attr.Value{})
	}
	var priorValues []string
	if !prior.IsNull() && !prior.IsUnknown() {
		diags.Append(prior.ElementsAs(ctx, &priorValues, false)...)
	}
	ordered := orderByPriorIDs(values, priorValues, func(s string) string { return s })
	list, d := types.ListValueFrom(ctx, types.StringType, ordered)
	diags.Append(d...)
	return list
}

// serviceDeskIntegrationStringList returns the elements of a known string list.
func serviceDeskIntegrationStringList(ctx context.Context, list types.List, diags *diag.Diagnostics) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var values []string
	diags.Append(list.ElementsAs(ctx, &values, false)...)
	return values
}

// serviceDeskIntegrationJSONMap decodes a JSON object attribute. Null and unknown values return nil.
func serviceDeskIntegrationJSONMap(value types.String, attribute string, diags *diag.Diagnostics) map[string]interface{} {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(value.ValueString()), &decoded); err != nil {
		diags.AddAttributeError(path.Root(attribute), "Invalid JSON", err.Error())
		return nil
	}
	return decoded
}

// serviceDeskIntegrationSubsetJSONState returns the state of a JSON object attribute read from the
// API. The prior value is kept when every key it has is either equal in the API value or missing
// from it: the API omits secrets and adds defaults and read-only keys, which must not cause a diff.
// Otherwise the API value is used, so changed values are detected as drift.
func serviceDeskIntegrationSubsetJSONState(prior types.String, value map[string]interface{}) types.String {
	if !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() != "" {
		var priorMap, apiMap map[string]interface{}
		encoded, err := json.Marshal(value)
		if err == nil && json.Unmarshal([]byte(prior.ValueString()), &priorMap) == nil && json.Unmarshal(encoded, &apiMap) == nil {
			equal := true
			for key, priorValue := range priorMap {
				if apiValue, ok := apiMap[key]; ok && !reflect.DeepEqual(priorValue, apiValue) {
					equal = false
					break
				}
			}
			if equal {
				return prior
			}
		}
	}
	return jsonStringState(prior, value)
}

// serviceDeskIntegrationFromModel converts the model to the API object.
func serviceDeskIntegrationFromModel(ctx context.Context, data ServiceDeskIntegrationModel, diags *diag.Diagnostics) *ServiceDeskIntegration {
	integration := &ServiceDeskIntegration{
		Name:                   data.Name.ValueString(),
		Description:            data.Description.ValueString(),
		Type:                   data.Type.ValueString(),
		OwnerRef:               serviceDeskIntegrationRefFromList(ctx, data.OwnerRef, "IDENTITY", diags),
		ClusterRef:             serviceDeskIntegrationRefFromList(ctx, data.ClusterRef, "CLUSTER", diags),
		ManagedSources:         serviceDeskIntegrationStringList(ctx, data.ManagedSources, diags),
		ProvisioningConfig:     serviceDeskIntegrationJSONMap(data.ProvisioningConfigJSON, "provisioning_config_json", diags),
		Attributes:             serviceDeskIntegrationJSONMap(data.AttributesJSON, "attributes_json", diags),
		BeforeProvisioningRule: serviceDeskIntegrationRefFromList(ctx, data.BeforeProvisioningRule, "RULE", diags),
	}
	if integration.Attributes == nil {
		integration.Attributes = map[string]interface{}{}
	}
	// universalManager is read-only. It is part of the API value kept in state when
	// provisioning_config_json is not configured, and must not be sent back.
	delete(integration.ProvisioningConfig, "universalManager")
	return integration
}

// setServiceDeskIntegrationState maps the API object onto the model. With refresh (Read, import,
// data source) every attribute comes from the API; otherwise (Create, Update) the planned values
// are kept and only computed and unknown values are resolved from the API response.
func setServiceDeskIntegrationState(ctx context.Context, data *ServiceDeskIntegrationModel, integration *ServiceDeskIntegration, refresh bool, diags *diag.Diagnostics) {
	if refresh {
		data.ID = types.StringValue(integration.ID)
		data.Created = types.StringValue(integration.Created)
	} else {
		// id and created are known in the plan of an update (UseStateForUnknown) and keep that value.
		data.ID = computedStringFromAPI(data.ID, integration.ID)
		data.Created = computedStringFromAPI(data.Created, integration.Created)
	}
	data.Modified = types.StringValue(integration.Modified)
	data.OwnerRef = serviceDeskIntegrationRefState(ctx, data.OwnerRef, integration.OwnerRef, "IDENTITY", refresh, diags)
	data.ClusterRef = serviceDeskIntegrationRefState(ctx, data.ClusterRef, integration.ClusterRef, "CLUSTER", refresh, diags)
	data.BeforeProvisioningRule = serviceDeskIntegrationRefState(ctx, data.BeforeProvisioningRule, integration.BeforeProvisioningRule, "RULE", refresh, diags)
	if refresh {
		data.Name = types.StringValue(integration.Name)
		data.Description = types.StringValue(integration.Description)
		data.Type = types.StringValue(integration.Type)
		data.ManagedSources = serviceDeskIntegrationStringListState(ctx, data.ManagedSources, integration.ManagedSources, diags)
		data.ProvisioningConfigJSON = jsonSubsetState(data.ProvisioningConfigJSON, integration.ProvisioningConfig)
		data.AttributesJSON = serviceDeskIntegrationSubsetJSONState(data.AttributesJSON, integration.Attributes)
		return
	}
	if data.ManagedSources.IsUnknown() {
		data.ManagedSources = serviceDeskIntegrationStringListState(ctx, types.ListNull(types.StringType), integration.ManagedSources, diags)
	}
	if data.ProvisioningConfigJSON.IsUnknown() {
		data.ProvisioningConfigJSON = jsonStringState(types.StringNull(), integration.ProvisioningConfig)
	}
}

func (r *ServiceDeskIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceDeskIntegrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	integration := serviceDeskIntegrationFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateServiceDeskIntegration(ctx, integration)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service desk integration: %s", err))
		return
	}
	setServiceDeskIntegrationState(ctx, &data, created, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceDeskIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceDeskIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	integration, err := client.GetServiceDeskIntegration(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service desk integration: %s", err))
		return
	}
	setServiceDeskIntegrationState(ctx, &data, integration, true, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceDeskIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ServiceDeskIntegrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	integration := serviceDeskIntegrationFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// PUT replaces the integration with the planned configuration. Only the deprecated cluster
	// field is not modeled, the API derives it from clusterRef.
	updated, err := client.UpdateServiceDeskIntegration(ctx, data.ID.ValueString(), integration)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service desk integration: %s", err))
		return
	}
	setServiceDeskIntegrationState(ctx, &data, updated, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceDeskIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceDeskIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteServiceDeskIntegration(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete service desk integration: %s", err))
	}
}

func (r *ServiceDeskIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
