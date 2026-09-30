package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewDasApplicationResource)
	registerDataSource(NewDasApplicationDataSource)
}

// DasApplicationTag is a key/value tag in a Data Access Security application request.
type DasApplicationTag struct {
	Key   int64   `json:"key"`
	Value *string `json:"value,omitempty"`
}

// DasApplicationRequest is the body to create or update a Data Access Security application.
type DasApplicationRequest struct {
	ApplicationType               int64                  `json:"applicationType"`
	Name                          string                 `json:"name"`
	Description                   *string                `json:"description,omitempty"`
	Tags                          []DasApplicationTag    `json:"tags,omitempty"`
	IdentityCollectorID           *int64                 `json:"identityCollectorId,omitempty"`
	ADIdentityCollectorID         *int64                 `json:"adIdentityCollectorId,omitempty"`
	NISIdentityCollectorID        *int64                 `json:"nisIdentityCollectorId,omitempty"`
	ApplicationCrawlerSettings    map[string]interface{} `json:"applicationCrawlerSettings,omitempty"`
	PermissionCollectorSettings   map[string]interface{} `json:"permissionCollectorSettings,omitempty"`
	DataClassificationSettings    map[string]interface{} `json:"dataClassificationSettings,omitempty"`
	ActivityConfigurationSettings map[string]interface{} `json:"activityConfigurationSettings,omitempty"`
	ExecuteNow                    bool                   `json:"executeNow"`
}

// DasApplicationTagRef is a tag of a Data Access Security application as returned by the API.
type DasApplicationTagRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// DasApplication is a Data Access Security application as returned by the /v2026/das/applications API.
type DasApplication struct {
	ID                   int64                  `json:"id"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	Type                 string                 `json:"type"`
	Tags                 []DasApplicationTagRef `json:"tags"`
	TestConnectionStatus string                 `json:"testConnectionStatus"`
	TestConnectionDate   *int64                 `json:"testConnectionDate"`
	RcClusterID          string                 `json:"rcClusterId"`
	DcClusterID          string                 `json:"dcClusterId"`
	PcClusterID          string                 `json:"pcClusterId"`
}

func (c *Client) GetDasApplication(ctx context.Context, id string) (*DasApplication, error) {
	var application DasApplication
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/das/applications/%s", id), nil, &application); err != nil {
		return nil, err
	}
	return &application, nil
}

// ListDasApplicationsByName lists the applications with the given name.
func (c *Client) ListDasApplicationsByName(ctx context.Context, name string) ([]DasApplication, error) {
	applications, err := listAllPages[DasApplication](ctx, c, "/v2026/das/applications", url.Values{"filters": {eqFilter("appName", name)}})
	if err != nil {
		return nil, err
	}
	var matching []DasApplication
	for _, application := range applications {
		if application.Name == name {
			matching = append(matching, application)
		}
	}
	return matching, nil
}

// CreateDasApplication creates an application. The API responds without content, so the new
// application is identified as the application with the requested name that did not exist before.
func (c *Client) CreateDasApplication(ctx context.Context, request *DasApplicationRequest) (*DasApplication, error) {
	existing, err := c.ListDasApplicationsByName(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	known := map[int64]bool{}
	for _, application := range existing {
		known[application.ID] = true
	}
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/das/applications", request, nil); err != nil {
		return nil, err
	}
	after, err := c.ListDasApplicationsByName(ctx, request.Name)
	if err != nil {
		return nil, fmt.Errorf("application %q was created, but looking it up failed: %w", request.Name, err)
	}
	var created []DasApplication
	for _, application := range after {
		if !known[application.ID] {
			created = append(created, application)
		}
	}
	if len(created) != 1 {
		return nil, fmt.Errorf("application %q was created, but %d new applications with that name were found; import it once it can be identified", request.Name, len(created))
	}
	return &created[0], nil
}

func (c *Client) UpdateDasApplication(ctx context.Context, id string, request *DasApplicationRequest) error {
	return c.doJSON(ctx, http.MethodPut, apiPath("/v2026/das/applications/%s", id), request, nil)
}

func (c *Client) DeleteDasApplication(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/das/applications/%s", id), nil, nil)
}

var _ resource.Resource = &DasApplicationResource{}
var _ resource.ResourceWithImportState = &DasApplicationResource{}

func NewDasApplicationResource() resource.Resource {
	return &DasApplicationResource{}
}

type DasApplicationResource struct {
	client *Config
}

type DasApplicationModel struct {
	ID                                types.String `tfsdk:"id"`
	Name                              types.String `tfsdk:"name"`
	ApplicationType                   types.Int64  `tfsdk:"application_type"`
	Description                       types.String `tfsdk:"description"`
	Tag                               types.List   `tfsdk:"tag"`
	IdentityCollectorID               types.Int64  `tfsdk:"identity_collector_id"`
	ADIdentityCollectorID             types.Int64  `tfsdk:"ad_identity_collector_id"`
	NISIdentityCollectorID            types.Int64  `tfsdk:"nis_identity_collector_id"`
	ApplicationCrawlerSettingsJSON    types.String `tfsdk:"application_crawler_settings_json"`
	PermissionCollectorSettingsJSON   types.String `tfsdk:"permission_collector_settings_json"`
	DataClassificationSettingsJSON    types.String `tfsdk:"data_classification_settings_json"`
	ActivityConfigurationSettingsJSON types.String `tfsdk:"activity_configuration_settings_json"`
	ExecuteNow                        types.Bool   `tfsdk:"execute_now"`
	Type                              types.String `tfsdk:"type"`
	TestConnectionStatus              types.String `tfsdk:"test_connection_status"`
	TestConnectionDate                types.Int64  `tfsdk:"test_connection_date"`
	RcClusterID                       types.String `tfsdk:"rc_cluster_id"`
	DcClusterID                       types.String `tfsdk:"dc_cluster_id"`
	PcClusterID                       types.String `tfsdk:"pc_cluster_id"`
}

type DasApplicationTagModel struct {
	Key   types.Int64  `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

func (r *DasApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_das_application"
}

// dasApplicationSettingsAttribute is an optional JSON object attribute for one settings object.
func dasApplicationSettingsAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: description + " The API does not return the settings, so changes made outside Terraform are not detected.",
		Optional:            true,
		Validators:          []validator.String{jsonObjectStringValidator{}},
	}
}

func (r *DasApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Data Access Security application, a monitored data store such as a file share or a cloud storage service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Application ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the application. Names should be unique: the API does not return the ID of a new application, so it is looked up by name after creation.",
				Required:            true,
			},
			"application_type": schema.Int64Attribute{
				MarkdownDescription: "Numeric application type, one of `1`, `8`, `9`, `11`, `15`, `20`, `21`, `24`, `25`, `27`, `28`, `29`, `33`, `35` or `37` (e.g. Active Directory or AWS S3). Changing it forces a new application. The API does not return it, so after an import the configured value is taken over without a replacement.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplaceIf(
					dasApplicationTypeRequiresReplace,
					"Changing the application type forces a new application.",
					"Changing the application type forces a new application.",
				)},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the application.",
				Optional:            true,
			},
			"identity_collector_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the identity collector of the application.",
				Optional:            true,
			},
			"ad_identity_collector_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the Active Directory identity collector.",
				Optional:            true,
			},
			"nis_identity_collector_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the NIS identity collector.",
				Optional:            true,
			},
			"application_crawler_settings_json": dasApplicationSettingsAttribute(
				"Resource crawler settings as a JSON object, with keys such as `isEnabled`, `clusterId`, `calculateResourceSize`, `excludedPathsByRegex` and `includeResources`."),
			"permission_collector_settings_json": dasApplicationSettingsAttribute(
				"Permission collector settings as a JSON object, with keys such as `isEnabled`, `clusterId`, `analyzeUniquePermissions` and `calculateEffectivePermissions`."),
			"data_classification_settings_json": dasApplicationSettingsAttribute(
				"Data classification settings as a JSON object, with the keys `isEnabled` and `clusterId`."),
			"activity_configuration_settings_json": dasApplicationSettingsAttribute(
				"Activity monitoring settings as a JSON object, with keys such as `isEnabled`, `clusterId`, `retentionTimePeriod`, `retentionTimeType` and `excludeUsers`."),
			"execute_now": schema.BoolAttribute{
				MarkdownDescription: "Whether the application setup is executed immediately when the application is created or updated. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Name of the application type.",
				Computed:            true,
			},
			"test_connection_status": schema.StringAttribute{
				MarkdownDescription: "Status of the last connection test.",
				Computed:            true,
			},
			"test_connection_date": schema.Int64Attribute{
				MarkdownDescription: "Time of the last connection test, in milliseconds since the epoch.",
				Computed:            true,
			},
			"rc_cluster_id": schema.StringAttribute{
				MarkdownDescription: "ID of the cluster that crawls resources.",
				Computed:            true,
			},
			"dc_cluster_id": schema.StringAttribute{
				MarkdownDescription: "ID of the cluster that classifies data.",
				Computed:            true,
			},
			"pc_cluster_id": schema.StringAttribute{
				MarkdownDescription: "ID of the cluster that collects permissions.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"tag": schema.ListNestedBlock{
				MarkdownDescription: "Tags that categorize the application. The API returns tags in a different form, so changes made outside Terraform are not detected.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.Int64Attribute{
							MarkdownDescription: "Key of the tag.",
							Required:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Value of the tag.",
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

func (r *DasApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// dasApplicationTypeRequiresReplace forces a replacement when the application type changes. The
// API does not return the type as a number, so the state value is null after an import; setting it
// then takes the configured value over instead of replacing the application.
func dasApplicationTypeRequiresReplace(ctx context.Context, req planmodifier.Int64Request, resp *int64planmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = !req.StateValue.IsNull() && !req.StateValue.IsUnknown()
}

// dasApplicationUnmanagedWarning warns when an update sends no value for attributes that the API
// does not return (settings and tags): PUT replaces the whole application, so the values of those
// attributes in IdentityNow, e.g. after an import, may be reset. Attributes whose configuration is
// removed (set in the prior state) are cleared on purpose and are not reported.
func dasApplicationUnmanagedWarning(plan, prior DasApplicationModel) string {
	var unset []string
	for _, attribute := range []struct {
		name        string
		plan, prior attr.Value
	}{
		{"application_crawler_settings_json", plan.ApplicationCrawlerSettingsJSON, prior.ApplicationCrawlerSettingsJSON},
		{"permission_collector_settings_json", plan.PermissionCollectorSettingsJSON, prior.PermissionCollectorSettingsJSON},
		{"data_classification_settings_json", plan.DataClassificationSettingsJSON, prior.DataClassificationSettingsJSON},
		{"activity_configuration_settings_json", plan.ActivityConfigurationSettingsJSON, prior.ActivityConfigurationSettingsJSON},
		{"tag", plan.Tag, prior.Tag},
	} {
		if attribute.plan.IsNull() && attribute.prior.IsNull() {
			unset = append(unset, "`"+attribute.name+"`")
		}
	}
	if len(unset) == 0 {
		return ""
	}
	return fmt.Sprintf("The update replaces the whole application, but these attributes are not configured: %s. The API does not "+
		"return their values, so the provider cannot send the current values back, and IdentityNow may reset them. Configure them "+
		"to keep them, in particular after an import.", strings.Join(unset, ", "))
}

// dasApplicationFromModel converts the model to the create and update request.
func dasApplicationFromModel(ctx context.Context, data DasApplicationModel, diags *diag.Diagnostics) *DasApplicationRequest {
	request := &DasApplicationRequest{
		ApplicationType:               data.ApplicationType.ValueInt64(),
		Name:                          data.Name.ValueString(),
		Description:                   stringPointer(data.Description),
		IdentityCollectorID:           int64Pointer(data.IdentityCollectorID),
		ADIdentityCollectorID:         int64Pointer(data.ADIdentityCollectorID),
		NISIdentityCollectorID:        int64Pointer(data.NISIdentityCollectorID),
		ApplicationCrawlerSettings:    serviceDeskIntegrationJSONMap(data.ApplicationCrawlerSettingsJSON, "application_crawler_settings_json", diags),
		PermissionCollectorSettings:   serviceDeskIntegrationJSONMap(data.PermissionCollectorSettingsJSON, "permission_collector_settings_json", diags),
		DataClassificationSettings:    serviceDeskIntegrationJSONMap(data.DataClassificationSettingsJSON, "data_classification_settings_json", diags),
		ActivityConfigurationSettings: serviceDeskIntegrationJSONMap(data.ActivityConfigurationSettingsJSON, "activity_configuration_settings_json", diags),
		ExecuteNow:                    data.ExecuteNow.ValueBool(),
	}
	if !data.Tag.IsNull() && !data.Tag.IsUnknown() {
		var tags []DasApplicationTagModel
		diags.Append(data.Tag.ElementsAs(ctx, &tags, false)...)
		for _, tag := range tags {
			request.Tags = append(request.Tags, DasApplicationTag{Key: tag.Key.ValueInt64(), Value: stringPointer(tag.Value)})
		}
	}
	return request
}

// setDasApplicationState maps the API application onto the model. The API only returns the name,
// description and read-only attributes; the other attributes keep their planned or prior values.
func setDasApplicationState(data *DasApplicationModel, application *DasApplication, refresh bool) {
	if application.ID != 0 {
		data.ID = types.StringValue(strconv.FormatInt(application.ID, 10))
	}
	data.Type = stringValueOrNull(application.Type)
	data.TestConnectionStatus = stringValueOrNull(application.TestConnectionStatus)
	data.TestConnectionDate = types.Int64PointerValue(application.TestConnectionDate)
	data.RcClusterID = stringValueOrNull(application.RcClusterID)
	data.DcClusterID = stringValueOrNull(application.DcClusterID)
	data.PcClusterID = stringValueOrNull(application.PcClusterID)
	if refresh {
		data.Name = types.StringValue(application.Name)
		data.Description = optionalStringState(data.Description, application.Description)
		if data.ExecuteNow.IsNull() || data.ExecuteNow.IsUnknown() {
			data.ExecuteNow = types.BoolValue(false)
		}
	}
}

func (r *DasApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DasApplicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request := dasApplicationFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateDasApplication(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Data Access Security application: %s", err))
		return
	}
	setDasApplicationState(&data, created, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DasApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DasApplicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	application, err := client.GetDasApplication(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Data Access Security application: %s", err))
		return
	}
	setDasApplicationState(&data, application, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DasApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, prior DasApplicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request := dasApplicationFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// The model covers the whole request body, so PUT sends the complete planned application. The API
	// does not return settings and tags, so unconfigured ones cannot be preserved.
	if warning := dasApplicationUnmanagedWarning(data, prior); warning != "" {
		resp.Diagnostics.AddWarning("Unconfigured Data Access Security application settings", warning)
	}
	if err := client.UpdateDasApplication(ctx, data.ID.ValueString(), request); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Data Access Security application: %s", err))
		return
	}
	// PUT responds without content, the read-only attributes are read back.
	application, err := client.GetDasApplication(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Data Access Security application: %s", err))
		return
	}
	setDasApplicationState(&data, application, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DasApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DasApplicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteDasApplication(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Data Access Security application: %s", err))
	}
}

func (r *DasApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.ParseInt(req.ID, 10, 64); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected a numeric application ID, got %q.", req.ID))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &DasApplicationDataSource{}

func NewDasApplicationDataSource() datasource.DataSource {
	return &DasApplicationDataSource{}
}

type DasApplicationDataSource struct {
	client *Config
}

type DasApplicationDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Type                 types.String `tfsdk:"type"`
	Tags                 types.List   `tfsdk:"tags"`
	TestConnectionStatus types.String `tfsdk:"test_connection_status"`
	TestConnectionDate   types.Int64  `tfsdk:"test_connection_date"`
	RcClusterID          types.String `tfsdk:"rc_cluster_id"`
	DcClusterID          types.String `tfsdk:"dc_cluster_id"`
	PcClusterID          types.String `tfsdk:"pc_cluster_id"`
}

type DasApplicationTagRefModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

var dasApplicationTagRefObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":   types.Int64Type,
	"name": types.StringType,
}}

func (d *DasApplicationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_das_application"
}

func (d *DasApplicationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a Data Access Security application by ID.",
		Attributes: map[string]dsschema.Attribute{
			"id":          dsschema.StringAttribute{MarkdownDescription: "Application ID.", Required: true},
			"name":        dsschema.StringAttribute{MarkdownDescription: "Display name of the application.", Computed: true},
			"description": dsschema.StringAttribute{MarkdownDescription: "Description of the application.", Computed: true},
			"type":        dsschema.StringAttribute{MarkdownDescription: "Name of the application type.", Computed: true},
			"tags": dsschema.ListNestedAttribute{
				MarkdownDescription: "Tags of the application.",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"id":   dsschema.Int64Attribute{MarkdownDescription: "Tag ID.", Computed: true},
						"name": dsschema.StringAttribute{MarkdownDescription: "Tag name.", Computed: true},
					},
				},
			},
			"test_connection_status": dsschema.StringAttribute{MarkdownDescription: "Status of the last connection test.", Computed: true},
			"test_connection_date":   dsschema.Int64Attribute{MarkdownDescription: "Time of the last connection test, in milliseconds since the epoch.", Computed: true},
			"rc_cluster_id":          dsschema.StringAttribute{MarkdownDescription: "ID of the cluster that crawls resources.", Computed: true},
			"dc_cluster_id":          dsschema.StringAttribute{MarkdownDescription: "ID of the cluster that classifies data.", Computed: true},
			"pc_cluster_id":          dsschema.StringAttribute{MarkdownDescription: "ID of the cluster that collects permissions.", Computed: true},
		},
	}
}

func (d *DasApplicationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// dasApplicationDataSourceState converts the API application to the data source model.
func dasApplicationDataSourceState(ctx context.Context, application *DasApplication, diags *diag.Diagnostics) DasApplicationDataSourceModel {
	tags := make([]DasApplicationTagRefModel, 0, len(application.Tags))
	for _, tag := range application.Tags {
		tags = append(tags, DasApplicationTagRefModel{ID: types.Int64Value(tag.ID), Name: types.StringValue(tag.Name)})
	}
	tagList, d := types.ListValueFrom(ctx, dasApplicationTagRefObjectType, tags)
	diags.Append(d...)
	return DasApplicationDataSourceModel{
		ID:                   types.StringValue(strconv.FormatInt(application.ID, 10)),
		Name:                 types.StringValue(application.Name),
		Description:          stringValueOrNull(application.Description),
		Type:                 stringValueOrNull(application.Type),
		Tags:                 tagList,
		TestConnectionStatus: stringValueOrNull(application.TestConnectionStatus),
		TestConnectionDate:   types.Int64PointerValue(application.TestConnectionDate),
		RcClusterID:          stringValueOrNull(application.RcClusterID),
		DcClusterID:          stringValueOrNull(application.DcClusterID),
		PcClusterID:          stringValueOrNull(application.PcClusterID),
	}
}

func (d *DasApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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
	application, err := client.GetDasApplication(ctx, id.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("id"), "Data Access Security application not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Data Access Security application: %s", err))
		return
	}
	data := dasApplicationDataSourceState(ctx, application, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
