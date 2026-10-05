package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewSearchAttributeConfigResource)
	registerDataSource(NewSearchAttributeConfigDataSource)
}

// SearchAttributeConfig is an extended search attribute as returned by the experimental
// /v2026/accounts/search-attribute-config API.
type SearchAttributeConfig struct {
	Name                  string            `json:"name"`
	DisplayName           string            `json:"displayName"`
	ApplicationAttributes map[string]string `json:"applicationAttributes"`
}

func (c *Client) GetSearchAttributeConfig(ctx context.Context, name string) (*SearchAttributeConfig, error) {
	var config SearchAttributeConfig
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/accounts/search-attribute-config/%s", name), nil, &config, withExperimental()); err != nil {
		return nil, err
	}
	// The API answers 204 without a body when the configuration does not exist.
	if config.Name == "" {
		return nil, &NotFoundError{fmt.Sprintf("search attribute config %q not found", name)}
	}
	return &config, nil
}

// CreateSearchAttributeConfig creates the configuration. The API accepts it with 202 and does
// not return the configuration.
func (c *Client) CreateSearchAttributeConfig(ctx context.Context, config *SearchAttributeConfig) error {
	return c.doJSON(ctx, http.MethodPost, "/v2026/accounts/search-attribute-config", config, nil, withExperimental())
}

// searchAttributeConfigCreateTimeout, searchAttributeConfigPollInterval and
// searchAttributeConfigMaxPollInterval control how long Create waits for an accepted configuration
// to become readable, and the backoff between the reads. They are variables for the tests.
var (
	searchAttributeConfigCreateTimeout   = 60 * time.Second
	searchAttributeConfigPollInterval    = time.Second
	searchAttributeConfigMaxPollInterval = 10 * time.Second
)

// WaitForSearchAttributeConfig reads the configuration until it exists. The API creates
// configurations asynchronously (POST answers 202) and GET answers 204 until the configuration
// exists. It returns an error when the configuration still does not exist after timeout, when
// the context is cancelled, or when a read fails with another error.
func (c *Client) WaitForSearchAttributeConfig(ctx context.Context, name string, timeout time.Duration) (*SearchAttributeConfig, error) {
	deadline := time.Now().Add(timeout)
	interval := searchAttributeConfigPollInterval
	for {
		config, err := c.GetSearchAttributeConfig(ctx, name)
		if err == nil {
			return config, nil
		}
		if !isNotFound(err) {
			return nil, err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("search attribute config %q was accepted but did not become available within %s", name, timeout)
		}
		if interval > remaining {
			interval = remaining
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
		interval *= 2
		if interval > searchAttributeConfigMaxPollInterval {
			interval = searchAttributeConfigMaxPollInterval
		}
	}
}

func (c *Client) UpdateSearchAttributeConfig(ctx context.Context, name string, ops []jsonPatchOp) (*SearchAttributeConfig, error) {
	var updated SearchAttributeConfig
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/accounts/search-attribute-config/%s", name), ops, &updated, withExperimental(), withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteSearchAttributeConfig(ctx context.Context, name string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/accounts/search-attribute-config/%s", name), nil, nil, withExperimental())
}

var _ resource.Resource = &SearchAttributeConfigResource{}
var _ resource.ResourceWithImportState = &SearchAttributeConfigResource{}

func NewSearchAttributeConfigResource() resource.Resource {
	return &SearchAttributeConfigResource{}
}

type SearchAttributeConfigResource struct {
	client *Config
}

type SearchAttributeConfigModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	DisplayName           types.String `tfsdk:"display_name"`
	ApplicationAttributes types.Map    `tfsdk:"application_attributes"`
}

// searchAttributeConfigIDFromName plans the ID as the planned name, since the configuration is
// identified by its name and a rename changes the ID.
type searchAttributeConfigIDFromName struct{}

func (m searchAttributeConfigIDFromName) Description(ctx context.Context) string {
	return "The ID is the same as the name."
}

func (m searchAttributeConfigIDFromName) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m searchAttributeConfigIDFromName) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var name types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("name"), &name)...)
	if !name.IsNull() && !name.IsUnknown() {
		resp.PlanValue = name
	}
}

func (r *SearchAttributeConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_search_attribute_config"
}

func (r *SearchAttributeConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an extended account search attribute, which promotes source account attributes to a searchable account attribute. Uses an experimental API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Configuration ID, the same as `name`",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{searchAttributeConfigIDFromName{}},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the search attribute. It must not be the same as an account or source attribute name.",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the search attribute",
				Required:            true,
			},
			"application_attributes": schema.MapAttribute{
				MarkdownDescription: "Map of source ID to the name of the account attribute promoted to the search attribute",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *SearchAttributeConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func searchAttributeConfigApplicationAttributes(ctx context.Context, value types.Map, diags *diag.Diagnostics) map[string]string {
	attributes := map[string]string{}
	if value.IsNull() || value.IsUnknown() {
		return attributes
	}
	diags.Append(value.ElementsAs(ctx, &attributes, false)...)
	return attributes
}

func searchAttributeConfigFromModel(ctx context.Context, data SearchAttributeConfigModel, diags *diag.Diagnostics) *SearchAttributeConfig {
	return &SearchAttributeConfig{
		Name:                  data.Name.ValueString(),
		DisplayName:           data.DisplayName.ValueString(),
		ApplicationAttributes: searchAttributeConfigApplicationAttributes(ctx, data.ApplicationAttributes, diags),
	}
}

func setSearchAttributeConfigState(ctx context.Context, data *SearchAttributeConfigModel, config *SearchAttributeConfig, diags *diag.Diagnostics) {
	data.ID = types.StringValue(config.Name)
	data.Name = types.StringValue(config.Name)
	data.DisplayName = types.StringValue(config.DisplayName)
	attributes := config.ApplicationAttributes
	if attributes == nil {
		attributes = map[string]string{}
	}
	value, d := types.MapValueFrom(ctx, types.StringType, attributes)
	diags.Append(d...)
	data.ApplicationAttributes = value
}

// searchAttributeConfigPatches returns the JSON Patch operations for the changed fields.
func searchAttributeConfigPatches(ctx context.Context, plan, state SearchAttributeConfigModel, diags *diag.Diagnostics) []jsonPatchOp {
	b := &patchBuilder{}
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceIfChanged(plan.DisplayName, state.DisplayName, "/displayName", plan.DisplayName.ValueString())
	b.replaceIfChanged(plan.ApplicationAttributes, state.ApplicationAttributes, "/applicationAttributes", searchAttributeConfigApplicationAttributes(ctx, plan.ApplicationAttributes, diags))
	return b.ops
}

func (r *SearchAttributeConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SearchAttributeConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := searchAttributeConfigFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.CreateSearchAttributeConfig(ctx, body); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create search attribute config: %s", err))
		return
	}
	data.ID = types.StringValue(data.Name.ValueString())
	// The API creates the configuration asynchronously; wait until it can be read, so a refresh
	// right after the create does not remove it from state. When waiting fails, the state is saved
	// with the error, so Terraform marks the resource as tainted and replaces it on the next apply.
	if _, err := client.WaitForSearchAttributeConfig(ctx, data.Name.ValueString(), searchAttributeConfigCreateTimeout); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Search attribute config was accepted, but could not be read after the create: %s", err))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SearchAttributeConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SearchAttributeConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	config, err := client.GetSearchAttributeConfig(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read search attribute config: %s", err))
		return
	}
	setSearchAttributeConfigState(ctx, &data, config, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SearchAttributeConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state SearchAttributeConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := searchAttributeConfigPatches(ctx, data, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(ops) > 0 {
		client, err := r.client.IdentityNowClient(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		// The configuration is addressed by its current name, a rename is one of the operations.
		if _, err := client.UpdateSearchAttributeConfig(ctx, state.ID.ValueString(), ops); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update search attribute config: %s", err))
			return
		}
	}
	data.ID = types.StringValue(data.Name.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SearchAttributeConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SearchAttributeConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteSearchAttributeConfig(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete search attribute config: %s", err))
	}
}

func (r *SearchAttributeConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

var _ datasource.DataSource = &SearchAttributeConfigDataSource{}

func NewSearchAttributeConfigDataSource() datasource.DataSource {
	return &SearchAttributeConfigDataSource{}
}

type SearchAttributeConfigDataSource struct {
	client *Config
}

func (d *SearchAttributeConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_search_attribute_config"
}

func (d *SearchAttributeConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up an extended account search attribute by name. Uses an experimental API.",
		Attributes: map[string]dsschema.Attribute{
			"id":           dsschema.StringAttribute{MarkdownDescription: "Configuration ID, the same as `name`", Computed: true},
			"name":         dsschema.StringAttribute{MarkdownDescription: "Name of the search attribute", Required: true},
			"display_name": dsschema.StringAttribute{MarkdownDescription: "Display name of the search attribute", Computed: true},
			"application_attributes": dsschema.MapAttribute{
				MarkdownDescription: "Map of source ID to the name of the promoted account attribute",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *SearchAttributeConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SearchAttributeConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SearchAttributeConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	config, err := client.GetSearchAttributeConfig(ctx, data.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Search attribute config not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read search attribute config: %s", err))
		return
	}
	setSearchAttributeConfigState(ctx, &data, config, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
