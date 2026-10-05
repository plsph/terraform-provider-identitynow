package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// serviceDeskIntegrationRefDataSourceAttribute is a computed single-element reference list.
func serviceDeskIntegrationRefDataSourceAttribute(description string) schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: description + ", a list with at most one item with `id`, `type` and `name`.",
		Computed:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id":   schema.StringAttribute{MarkdownDescription: "ID of the referenced object.", Computed: true},
				"type": schema.StringAttribute{MarkdownDescription: "Reference type.", Computed: true},
				"name": schema.StringAttribute{MarkdownDescription: "Name of the referenced object.", Computed: true},
			},
		},
	}
}

var _ datasource.DataSource = &ServiceDeskIntegrationDataSource{}
var _ datasource.DataSourceWithValidateConfig = &ServiceDeskIntegrationDataSource{}

func NewServiceDeskIntegrationDataSource() datasource.DataSource {
	return &ServiceDeskIntegrationDataSource{}
}

type ServiceDeskIntegrationDataSource struct {
	client *Config
}

func (d *ServiceDeskIntegrationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_desk_integration"
}

func (d *ServiceDeskIntegrationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a service desk integration by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Service desk integration ID. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Service desk integration name. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{MarkdownDescription: "Description of the integration.", Computed: true},
			"type":        schema.StringAttribute{MarkdownDescription: "Service desk integration type.", Computed: true},
			"managed_sources": schema.ListAttribute{
				MarkdownDescription: "IDs of the sources managed by the integration.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"provisioning_config_json": schema.StringAttribute{MarkdownDescription: "Provisioning configuration as a JSON object.", Computed: true},
			"attributes_json": schema.StringAttribute{
				MarkdownDescription: "Integration attributes as a JSON object, as returned by the API. The value is sensitive.",
				Computed:            true,
				Sensitive:           true,
			},
			"owner_ref":                serviceDeskIntegrationRefDataSourceAttribute("Identity that owns the integration"),
			"cluster_ref":              serviceDeskIntegrationRefDataSourceAttribute("Virtual appliance cluster of the integration"),
			"before_provisioning_rule": serviceDeskIntegrationRefDataSourceAttribute("Before provisioning rule"),
			"created":                  schema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"modified":                 schema.StringAttribute{MarkdownDescription: "Last modification date.", Computed: true},
		},
	}
}

func (d *ServiceDeskIntegrationDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *ServiceDeskIntegrationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ServiceDeskIntegrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceDeskIntegrationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var integration *ServiceDeskIntegration
	if !data.ID.IsNull() {
		integration, err = client.GetServiceDeskIntegration(ctx, data.ID.ValueString())
	} else {
		integration, err = client.GetServiceDeskIntegrationByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Service desk integration not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service desk integration: %s", err))
		return
	}
	setServiceDeskIntegrationState(ctx, &data, integration, true, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

var _ datasource.DataSource = &ServiceDeskIntegrationTypesDataSource{}

func NewServiceDeskIntegrationTypesDataSource() datasource.DataSource {
	return &ServiceDeskIntegrationTypesDataSource{}
}

type ServiceDeskIntegrationTypesDataSource struct {
	client *Config
}

type ServiceDeskIntegrationTypesModel struct {
	ID    types.String                      `tfsdk:"id"`
	Types []ServiceDeskIntegrationTypeModel `tfsdk:"types"`
}

type ServiceDeskIntegrationTypeModel struct {
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	ScriptName types.String `tfsdk:"script_name"`
}

func (d *ServiceDeskIntegrationTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_desk_integration_types"
}

func (d *ServiceDeskIntegrationTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the supported service desk integration types.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Fixed identifier of the data source, `service-desk-integration-types`.",
				Computed:            true,
			},
			"types": schema.ListNestedAttribute{
				MarkdownDescription: "Supported service desk integration types.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":        schema.StringAttribute{MarkdownDescription: "Display name of the type.", Computed: true},
						"type":        schema.StringAttribute{MarkdownDescription: "Type value, used as `type` of the integration.", Computed: true},
						"script_name": schema.StringAttribute{MarkdownDescription: "Script name of the integration template of the type.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *ServiceDeskIntegrationTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// serviceDeskIntegrationTypesState converts the API types to the data source model.
func serviceDeskIntegrationTypesState(list []ServiceDeskIntegrationType) ServiceDeskIntegrationTypesModel {
	data := ServiceDeskIntegrationTypesModel{
		ID:    types.StringValue("service-desk-integration-types"),
		Types: make([]ServiceDeskIntegrationTypeModel, 0, len(list)),
	}
	for _, t := range list {
		data.Types = append(data.Types, ServiceDeskIntegrationTypeModel{
			Name:       types.StringValue(t.Name),
			Type:       types.StringValue(t.Type),
			ScriptName: types.StringValue(t.ScriptName),
		})
	}
	return data
}

func (d *ServiceDeskIntegrationTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	list, err := client.ListServiceDeskIntegrationTypes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list service desk integration types: %s", err))
		return
	}
	data := serviceDeskIntegrationTypesState(list)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
