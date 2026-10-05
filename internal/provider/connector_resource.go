package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewConnectorResource)
	registerDataSource(NewConnectorDataSource)
}

// connectorCreateStatuses are the statuses a custom connector can be created with.
var connectorCreateStatuses = []string{"DEVELOPMENT", "DEMO", "RELEASED"}

// Connector is a connector as returned by the /v2026/connectors API.
type Connector struct {
	Name                 string                 `json:"name,omitempty"`
	Type                 string                 `json:"type,omitempty"`
	ClassName            string                 `json:"className,omitempty"`
	ScriptName           string                 `json:"scriptName,omitempty"`
	ApplicationXML       string                 `json:"applicationXml,omitempty"`
	CorrelationConfigXML string                 `json:"correlationConfigXml,omitempty"`
	SourceConfigXML      string                 `json:"sourceConfigXml,omitempty"`
	S3Location           string                 `json:"s3Location,omitempty"`
	FileUpload           *bool                  `json:"fileUpload,omitempty"`
	DirectConnect        *bool                  `json:"directConnect,omitempty"`
	ConnectorMetadata    map[string]interface{} `json:"connectorMetadata,omitempty"`
	Status               string                 `json:"status,omitempty"`
}

// ConnectorCreateRequest is the request body to create a custom connector.
type ConnectorCreateRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type,omitempty"`
	ClassName     string `json:"className"`
	DirectConnect *bool  `json:"directConnect,omitempty"`
	Status        string `json:"status,omitempty"`
}

func (c *Client) GetConnector(ctx context.Context, scriptName string) (*Connector, error) {
	var connector Connector
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/connectors/%s", scriptName), nil, &connector); err != nil {
		return nil, err
	}
	return &connector, nil
}

func (c *Client) CreateConnector(ctx context.Context, request *ConnectorCreateRequest) (*Connector, error) {
	var created Connector
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/connectors", request, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// FindConnectorByName lists the connectors whose name starts with name (the list endpoint does not
// support eq on name) and returns the one whose name matches exactly. Connector names are unique
// per tenant; it returns a NotFoundError when nothing matches. The API documents that the list
// only contains connectors with the RELEASED status.
func (c *Client) FindConnectorByName(ctx context.Context, name string) (*Connector, error) {
	filter := strings.Replace(eqFilter("name", name), "name eq ", "name sw ", 1)
	connectors, err := listAllPages[Connector](ctx, c, "/v2026/connectors", url.Values{"filters": {filter}})
	if err != nil {
		return nil, err
	}
	var match *Connector
	for i := range connectors {
		if connectors[i].Name != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple connectors are named %q", name)
		}
		match = &connectors[i]
	}
	if match == nil {
		return nil, &NotFoundError{fmt.Sprintf("connector with name %q not found", name)}
	}
	return match, nil
}

func (c *Client) PatchConnector(ctx context.Context, scriptName string, ops []jsonPatchOp) (*Connector, error) {
	var updated Connector
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/connectors/%s", scriptName), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteConnector(ctx context.Context, scriptName string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/connectors/%s", scriptName), nil, nil)
}

// connectorStatusValidator checks that the status is one a connector can be created with.
type connectorStatusValidator struct{}

func (v connectorStatusValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be one of %s", strings.Join(connectorCreateStatuses, ", "))
}

func (v connectorStatusValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v connectorStatusValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, allowed := range connectorCreateStatuses {
		if req.ConfigValue.ValueString() == allowed {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid connector status", fmt.Sprintf("%s, got %q.", v.Description(ctx), req.ConfigValue.ValueString()))
}

// connectorTextState returns the state value of a large text attribute (XML or script) read from
// the API. The prior value is kept when it differs only in line endings or trailing whitespace.
func connectorTextState(prior types.String, value string) types.String {
	normalize := func(s string) string {
		return strings.TrimRight(strings.ReplaceAll(s, "\r\n", "\n"), " \t\r\n")
	}
	if !prior.IsNull() && !prior.IsUnknown() && normalize(prior.ValueString()) == normalize(value) {
		return prior
	}
	return types.StringValue(value)
}

var _ resource.Resource = &ConnectorResource{}
var _ resource.ResourceWithImportState = &ConnectorResource{}

func NewConnectorResource() resource.Resource {
	return &ConnectorResource{}
}

type ConnectorResource struct {
	client *Config
}

type ConnectorModel struct {
	ID                    types.String `tfsdk:"id"`
	ScriptName            types.String `tfsdk:"script_name"`
	Name                  types.String `tfsdk:"name"`
	Type                  types.String `tfsdk:"type"`
	ClassName             types.String `tfsdk:"class_name"`
	DirectConnect         types.Bool   `tfsdk:"direct_connect"`
	Status                types.String `tfsdk:"status"`
	ConnectorMetadataJSON types.String `tfsdk:"connector_metadata_json"`
	ApplicationXML        types.String `tfsdk:"application_xml"`
	CorrelationConfigXML  types.String `tfsdk:"correlation_config_xml"`
	SourceConfigXML       types.String `tfsdk:"source_config_xml"`
	FileUpload            types.Bool   `tfsdk:"file_upload"`
}

func (r *ConnectorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector"
}

func (r *ConnectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	computedString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: description,
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	metadata := computedString("UI metadata of the connector as a JSON object. Use `jsonencode()` for convenience. The value is compared semantically. When not set, the value is not managed.")
	metadata.Validators = []validator.String{jsonObjectStringValidator{}}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom connector. Uploading connector files and translations is not supported.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Connector ID, the same as `script_name`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"script_name": schema.StringAttribute{
				MarkdownDescription: "Unique script name of the connector, generated by IdentityNow from the name.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Connector name, unique in the tenant. Changing this forces a new connector to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Connector type. Defaults to `custom <name>`. Changing this forces a new connector to be created.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"class_name": schema.StringAttribute{
				MarkdownDescription: "Connector class name. Connectors that implement the open connector standard use `sailpoint.connector.OpenConnectorAdapter`. Changing this forces a new connector to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"direct_connect": schema.BoolAttribute{
				MarkdownDescription: "Whether sources of the connector are direct connect sources. Defaults to `true`. Changing this forces a new connector to be created.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown(), boolplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Connector status, `DEVELOPMENT`, `DEMO` or `RELEASED`. Changing this forces a new connector to be created.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{connectorStatusValidator{}},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"connector_metadata_json": metadata,
			"application_xml":         computedString("Application XML of the connector. When not set, the value is not managed."),
			"correlation_config_xml":  computedString("Correlation config XML of the connector. When not set, the value is not managed."),
			"source_config_xml":       computedString("Source config XML of the connector. When not set, the value is not managed."),
			"file_upload": schema.BoolAttribute{
				MarkdownDescription: "Whether sources of the connector are file upload sources.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *ConnectorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// connectorCreateRequestFromModel builds the create request from the plan.
func connectorCreateRequestFromModel(data ConnectorModel) *ConnectorCreateRequest {
	return &ConnectorCreateRequest{
		Name:          data.Name.ValueString(),
		Type:          data.Type.ValueString(),
		ClassName:     data.ClassName.ValueString(),
		DirectConnect: boolPointer(data.DirectConnect),
		Status:        data.Status.ValueString(),
	}
}

// connectorMetadataValue decodes connector_metadata_json. The value is validated as a JSON object.
func connectorMetadataValue(value types.String) (map[string]interface{}, error) {
	metadata := map[string]interface{}{}
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return metadata, nil
	}
	err := json.Unmarshal([]byte(value.ValueString()), &metadata)
	return metadata, err
}

// connectorPatch returns JSON Patch operations for the patchable attributes that changed. The
// attributes are optional and computed, so null or unknown planned values are not managed.
func connectorPatch(plan, state ConnectorModel) ([]jsonPatchOp, error) {
	var b patchBuilder
	add := func(planned, prior types.String, path string, value interface{}) {
		if planned.IsNull() || planned.IsUnknown() {
			return
		}
		b.replaceIfChanged(planned, prior, path, value)
	}
	if !plan.ConnectorMetadataJSON.IsNull() && !plan.ConnectorMetadataJSON.IsUnknown() &&
		(state.ConnectorMetadataJSON.IsNull() || !jsonSemanticallyEqual([]byte(plan.ConnectorMetadataJSON.ValueString()), []byte(state.ConnectorMetadataJSON.ValueString()))) {
		metadata, err := connectorMetadataValue(plan.ConnectorMetadataJSON)
		if err != nil {
			return nil, err
		}
		b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/connectorMetadata", Value: metadata})
	}
	add(plan.ApplicationXML, state.ApplicationXML, "/applicationXml", plan.ApplicationXML.ValueString())
	add(plan.CorrelationConfigXML, state.CorrelationConfigXML, "/correlationConfigXml", plan.CorrelationConfigXML.ValueString())
	add(plan.SourceConfigXML, state.SourceConfigXML, "/sourceConfigXml", plan.SourceConfigXML.ValueString())
	return b.ops, nil
}

// setConnectorState maps an API connector onto the model. With refresh, all values are taken from
// the API; otherwise known planned values are kept and only unknown values are resolved.
func setConnectorState(data *ConnectorModel, connector *Connector, refresh bool) {
	resolve := func(current types.String, value string) types.String {
		if refresh || current.IsUnknown() {
			return types.StringValue(value)
		}
		return current
	}
	resolveText := func(current types.String, value string) types.String {
		if refresh {
			return connectorTextState(current, value)
		}
		return resolve(current, value)
	}
	if connector.ScriptName != "" {
		data.ScriptName = types.StringValue(connector.ScriptName)
		data.ID = data.ScriptName
	}
	if refresh {
		data.Name = types.StringValue(connector.Name)
		data.ClassName = types.StringValue(connector.ClassName)
	}
	data.Type = resolve(data.Type, connector.Type)
	data.Status = resolve(data.Status, connector.Status)
	data.DirectConnect = boolFromAPI(data.DirectConnect, connector.DirectConnect, refresh)
	if data.DirectConnect.IsNull() && !refresh {
		data.DirectConnect = types.BoolValue(false)
	}
	data.FileUpload = boolFromAPI(data.FileUpload, connector.FileUpload, refresh)
	if data.FileUpload.IsNull() {
		data.FileUpload = types.BoolValue(false)
	}
	if refresh || data.ConnectorMetadataJSON.IsUnknown() {
		prior := data.ConnectorMetadataJSON
		if prior.IsUnknown() {
			prior = types.StringNull()
		}
		data.ConnectorMetadataJSON = jsonStringState(prior, connector.ConnectorMetadata)
	}
	data.ApplicationXML = resolveText(data.ApplicationXML, connector.ApplicationXML)
	data.CorrelationConfigXML = resolveText(data.CorrelationConfigXML, connector.CorrelationConfigXML)
	data.SourceConfigXML = resolveText(data.SourceConfigXML, connector.SourceConfigXML)
}

// connectorResolveUnknown sets values that are still unknown after a failed apply step to null,
// so the partially created connector can be saved in state.
func connectorResolveUnknown(data *ConnectorModel) {
	for _, value := range []*types.String{&data.Type, &data.Status, &data.ConnectorMetadataJSON, &data.ApplicationXML, &data.CorrelationConfigXML, &data.SourceConfigXML} {
		if value.IsUnknown() {
			*value = types.StringNull()
		}
	}
	for _, value := range []*types.Bool{&data.DirectConnect, &data.FileUpload} {
		if value.IsUnknown() {
			*value = types.BoolNull()
		}
	}
}

func (r *ConnectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateConnector(ctx, connectorCreateRequestFromModel(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create connector: %s", err))
		return
	}
	if created.ScriptName == "" {
		// The connector was created, but the response has no script name: look it up by name.
		found, err := client.FindConnectorByName(ctx, data.Name.ValueString())
		if err != nil || found.ScriptName == "" {
			if err == nil {
				err = fmt.Errorf("the connector found by name has no script name")
			}
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf(
				"The connector %q was created, but the API did not return its script name and it could not be found by name: %s. "+
					"Import it with its script name (terraform import) to manage it.", data.Name.ValueString(), err))
			return
		}
		created = found
	}
	data.ScriptName = types.StringValue(created.ScriptName)
	data.ID = data.ScriptName

	// The patchable attributes are not part of the create request, they are applied afterwards.
	// On errors the connector is saved in state anyway, so Terraform marks it tainted.
	unset := ConnectorModel{ConnectorMetadataJSON: types.StringNull(), ApplicationXML: types.StringNull(), CorrelationConfigXML: types.StringNull(), SourceConfigXML: types.StringNull()}
	r.finishApply(ctx, client, &data, unset, "create", &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// finishApply patches the changed patchable attributes and resolves unknown values from the API.
// It reports whether the patch was applied. Unknown values are always set to null afterwards, so a
// partially created connector can still be saved in state.
func (r *ConnectorResource) finishApply(ctx context.Context, client *Client, data *ConnectorModel, prior ConnectorModel, action string, diags *diag.Diagnostics) bool {
	defer connectorResolveUnknown(data)
	ops, err := connectorPatch(*data, prior)
	if err != nil {
		diags.AddAttributeError(path.Root("connector_metadata_json"), "Invalid JSON", err.Error())
		return false
	}
	if len(ops) > 0 {
		if _, err := client.PatchConnector(ctx, data.ScriptName.ValueString(), ops); err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Unable to %s connector: %s", action, err))
			return false
		}
	}
	connector, err := client.GetConnector(ctx, data.ScriptName.ValueString())
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to read connector after %s: %s", action, err))
		return true
	}
	setConnectorState(data, connector, false)
	return true
}

func (r *ConnectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	connector, err := client.GetConnector(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read connector: %s", err))
		return
	}
	data.ScriptName = data.ID
	setConnectorState(&data, connector, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ConnectorModel
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
	if !r.finishApply(ctx, client, &data, state, "update", &resp.Diagnostics) {
		// Nothing was changed, the prior state is kept.
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteConnector(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete connector: %s", err))
	}
}

func (r *ConnectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("script_name"), req.ID)...)
}

var _ datasource.DataSource = &ConnectorDataSource{}

func NewConnectorDataSource() datasource.DataSource {
	return &ConnectorDataSource{}
}

type ConnectorDataSource struct {
	client *Config
}

func (d *ConnectorDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector"
}

func (d *ConnectorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a connector by script name.",
		Attributes: map[string]dsschema.Attribute{
			"id":                      dsschema.StringAttribute{MarkdownDescription: "Connector ID, the same as `script_name`.", Computed: true},
			"script_name":             dsschema.StringAttribute{MarkdownDescription: "Script name of the connector.", Required: true},
			"name":                    dsschema.StringAttribute{MarkdownDescription: "Connector name.", Computed: true},
			"type":                    dsschema.StringAttribute{MarkdownDescription: "Connector type.", Computed: true},
			"class_name":              dsschema.StringAttribute{MarkdownDescription: "Connector class name.", Computed: true},
			"direct_connect":          dsschema.BoolAttribute{MarkdownDescription: "Whether sources of the connector are direct connect sources.", Computed: true},
			"status":                  dsschema.StringAttribute{MarkdownDescription: "Connector status.", Computed: true},
			"connector_metadata_json": dsschema.StringAttribute{MarkdownDescription: "UI metadata of the connector as a JSON object.", Computed: true},
			"application_xml":         dsschema.StringAttribute{MarkdownDescription: "Application XML of the connector.", Computed: true},
			"correlation_config_xml":  dsschema.StringAttribute{MarkdownDescription: "Correlation config XML of the connector.", Computed: true},
			"source_config_xml":       dsschema.StringAttribute{MarkdownDescription: "Source config XML of the connector.", Computed: true},
			"file_upload":             dsschema.BoolAttribute{MarkdownDescription: "Whether sources of the connector are file upload sources.", Computed: true},
		},
	}
}

func (d *ConnectorDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ConnectorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConnectorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	connector, err := client.GetConnector(ctx, data.ScriptName.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("script_name"), "Connector not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read connector: %s", err))
		return
	}
	data.ID = data.ScriptName
	setConnectorState(&data, connector, true)
	if data.DirectConnect.IsNull() {
		data.DirectConnect = types.BoolValue(false)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
