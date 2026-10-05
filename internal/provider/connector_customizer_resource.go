package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewConnectorCustomizerResource)
	registerDataSource(NewConnectorCustomizerDataSource)
}

// ConnectorCustomizer is a connector customizer as returned by the /v2026/connector-customizers API.
type ConnectorCustomizer struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name"`
	ImageVersion *int64 `json:"imageVersion,omitempty"`
	ImageID      string `json:"imageID,omitempty"`
	TenantID     string `json:"tenantID,omitempty"`
	Created      string `json:"created,omitempty"`
}

// ConnectorCustomizerRequest is the create and update request body, name is the only writable field.
type ConnectorCustomizerRequest struct {
	Name string `json:"name"`
}

func (c *Client) GetConnectorCustomizer(ctx context.Context, id string) (*ConnectorCustomizer, error) {
	var customizer ConnectorCustomizer
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/connector-customizers/%s", id), nil, &customizer); err != nil {
		return nil, err
	}
	return &customizer, nil
}

// GetConnectorCustomizerByName returns the customizer with the given name. The list endpoint has
// no filters, so customizers are matched client side.
func (c *Client) GetConnectorCustomizerByName(ctx context.Context, name string) (*ConnectorCustomizer, error) {
	customizers, err := listAllPages[ConnectorCustomizer](ctx, c, "/v2026/connector-customizers", url.Values{})
	if err != nil {
		return nil, err
	}
	var match *ConnectorCustomizer
	for i := range customizers {
		if customizers[i].Name != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple connector customizers are named %q", name)
		}
		match = &customizers[i]
	}
	if match == nil {
		return nil, &NotFoundError{fmt.Sprintf("connector customizer with name %q not found", name)}
	}
	return match, nil
}

func (c *Client) CreateConnectorCustomizer(ctx context.Context, request *ConnectorCustomizerRequest) (*ConnectorCustomizer, error) {
	var created ConnectorCustomizer
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/connector-customizers", request, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateConnectorCustomizer(ctx context.Context, id string, request *ConnectorCustomizerRequest) (*ConnectorCustomizer, error) {
	var updated ConnectorCustomizer
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/connector-customizers/%s", id), request, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteConnectorCustomizer(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/connector-customizers/%s", id), nil, nil)
}

var _ resource.Resource = &ConnectorCustomizerResource{}
var _ resource.ResourceWithImportState = &ConnectorCustomizerResource{}

func NewConnectorCustomizerResource() resource.Resource {
	return &ConnectorCustomizerResource{}
}

type ConnectorCustomizerResource struct {
	client *Config
}

type ConnectorCustomizerModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ImageVersion types.Int64  `tfsdk:"image_version"`
	ImageID      types.String `tfsdk:"image_id"`
	TenantID     types.String `tfsdk:"tenant_id"`
	Created      types.String `tfsdk:"created"`
}

func (r *ConnectorCustomizerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector_customizer"
}

func (r *ConnectorCustomizerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a connector customizer. Customizer versions (code uploads) are not managed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Connector customizer ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Connector customizer name. A rename is verified after the update, see above. Changes are sent with an update request and verified by reading the customizer back; the apply fails when IdentityNow does not apply the new name, since the API documents the name as immutable.",
				Required:            true,
			},
			"image_version": schema.Int64Attribute{
				MarkdownDescription: "Current image version of the customizer. Null until a version is created.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"image_id": schema.StringAttribute{
				MarkdownDescription: "Current image ID of the customizer. Null until a version is created.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "Tenant ID of the customizer.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Creation date.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *ConnectorCustomizerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// setConnectorCustomizerState maps an API customizer onto the model. With refresh, all values are
// taken from the API; otherwise only unknown values are resolved.
func setConnectorCustomizerState(data *ConnectorCustomizerModel, customizer *ConnectorCustomizer, refresh bool) {
	resolve := func(current types.String, value string) types.String {
		if refresh || current.IsUnknown() {
			return stringValueOrNull(value)
		}
		return current
	}
	if refresh || data.ID.IsUnknown() {
		data.ID = types.StringValue(customizer.ID)
	}
	if refresh {
		data.Name = types.StringValue(customizer.Name)
	}
	if refresh || data.ImageVersion.IsUnknown() {
		data.ImageVersion = types.Int64PointerValue(customizer.ImageVersion)
	}
	data.ImageID = resolve(data.ImageID, customizer.ImageID)
	data.TenantID = resolve(data.TenantID, customizer.TenantID)
	data.Created = resolve(data.Created, customizer.Created)
}

// connectorCustomizerCheckRename returns an error when the customizer read back after an update
// does not have the planned name, i.e. the API accepted the update but ignored the rename.
func connectorCustomizerCheckRename(planned string, customizer *ConnectorCustomizer) error {
	if customizer.Name == planned {
		return nil
	}
	return fmt.Errorf("the API accepted the update, but the connector customizer is still named %q instead of %q. "+
		"The API documents the name as immutable; to rename the customizer, create a new one (e.g. with terraform apply -replace) or revert the name in the configuration", customizer.Name, planned)
}

func (r *ConnectorCustomizerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorCustomizerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateConnectorCustomizer(ctx, &ConnectorCustomizerRequest{Name: data.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create connector customizer: %s", err))
		return
	}
	setConnectorCustomizerState(&data, created, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectorCustomizerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorCustomizerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	customizer, err := client.GetConnectorCustomizer(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read connector customizer: %s", err))
		return
	}
	setConnectorCustomizerState(&data, customizer, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectorCustomizerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorCustomizerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// Name is the only writable field, so the request contains the full object. The spec lists name
	// as immutable in the PUT description although it is the only field of the request body, so the
	// rename is verified by reading the customizer back. When it failed, the prior state is kept.
	if _, err := client.UpdateConnectorCustomizer(ctx, data.ID.ValueString(), &ConnectorCustomizerRequest{Name: data.Name.ValueString()}); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update connector customizer: %s", err))
		return
	}
	updated, err := client.GetConnectorCustomizer(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read connector customizer after update: %s", err))
		return
	}
	if err := connectorCustomizerCheckRename(data.Name.ValueString(), updated); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Connector customizer not renamed", err.Error())
		return
	}
	setConnectorCustomizerState(&data, updated, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectorCustomizerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorCustomizerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteConnectorCustomizer(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete connector customizer: %s", err))
	}
}

func (r *ConnectorCustomizerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &ConnectorCustomizerDataSource{}
var _ datasource.DataSourceWithValidateConfig = &ConnectorCustomizerDataSource{}

func NewConnectorCustomizerDataSource() datasource.DataSource {
	return &ConnectorCustomizerDataSource{}
}

type ConnectorCustomizerDataSource struct {
	client *Config
}

func (d *ConnectorCustomizerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector_customizer"
}

func (d *ConnectorCustomizerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a connector customizer by ID or name.",
		Attributes: map[string]dsschema.Attribute{
			"id":            dsschema.StringAttribute{MarkdownDescription: "Connector customizer ID. Exactly one of `id` or `name` must be set.", Optional: true, Computed: true},
			"name":          dsschema.StringAttribute{MarkdownDescription: "Connector customizer name. Exactly one of `id` or `name` must be set.", Optional: true, Computed: true},
			"image_version": dsschema.Int64Attribute{MarkdownDescription: "Current image version of the customizer.", Computed: true},
			"image_id":      dsschema.StringAttribute{MarkdownDescription: "Current image ID of the customizer.", Computed: true},
			"tenant_id":     dsschema.StringAttribute{MarkdownDescription: "Tenant ID of the customizer.", Computed: true},
			"created":       dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
		},
	}
}

func (d *ConnectorCustomizerDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *ConnectorCustomizerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ConnectorCustomizerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConnectorCustomizerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var customizer *ConnectorCustomizer
	if !data.ID.IsNull() {
		customizer, err = client.GetConnectorCustomizer(ctx, data.ID.ValueString())
	} else {
		customizer, err = client.GetConnectorCustomizerByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Connector customizer not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read connector customizer: %s", err))
		return
	}
	setConnectorCustomizerState(&data, customizer, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
