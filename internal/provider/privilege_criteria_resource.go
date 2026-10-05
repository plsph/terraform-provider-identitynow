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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewPrivilegeCriteriaResource)
	registerDataSource(NewPrivilegeCriteriaDataSource)
}

// privilegeCriteriaTypeCustom is the only criteria type that can be created.
const privilegeCriteriaTypeCustom = "CUSTOM"

// PrivilegeCriteria is a privilege criteria as returned by the /v2026/criteria/privilege API.
type PrivilegeCriteria struct {
	ID             string        `json:"id,omitempty"`
	SourceID       string        `json:"sourceId"`
	Type           string        `json:"type"`
	Operator       string        `json:"operator,omitempty"`
	Groups         []interface{} `json:"groups"`
	PrivilegeLevel string        `json:"privilegeLevel"`
}

func (c *Client) GetPrivilegeCriteria(ctx context.Context, id string) (*PrivilegeCriteria, error) {
	var criteria PrivilegeCriteria
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/criteria/privilege/%s", id), nil, &criteria); err != nil {
		return nil, err
	}
	return &criteria, nil
}

func (c *Client) CreatePrivilegeCriteria(ctx context.Context, criteria *PrivilegeCriteria) (*PrivilegeCriteria, error) {
	var created PrivilegeCriteria
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/criteria/privilege", criteria, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdatePrivilegeCriteria(ctx context.Context, id string, criteria *PrivilegeCriteria) (*PrivilegeCriteria, error) {
	var updated PrivilegeCriteria
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/criteria/privilege/%s", id), criteria, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeletePrivilegeCriteria(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/criteria/privilege/%s", id), nil, nil)
}

var _ resource.Resource = &PrivilegeCriteriaResource{}
var _ resource.ResourceWithImportState = &PrivilegeCriteriaResource{}

func NewPrivilegeCriteriaResource() resource.Resource {
	return &PrivilegeCriteriaResource{}
}

type PrivilegeCriteriaResource struct {
	client *Config
}

type PrivilegeCriteriaModel struct {
	ID             types.String `tfsdk:"id"`
	SourceID       types.String `tfsdk:"source_id"`
	Type           types.String `tfsdk:"type"`
	Operator       types.String `tfsdk:"operator"`
	GroupsJSON     types.String `tfsdk:"groups_json"`
	PrivilegeLevel types.String `tfsdk:"privilege_level"`
}

func (r *PrivilegeCriteriaResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privilege_criteria"
}

func (r *PrivilegeCriteriaResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom privilege criteria, which assigns a privilege level to the entitlements of a source that match the criteria.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Privilege criteria ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source_id": schema.StringAttribute{
				MarkdownDescription: "ID of the source the criteria applies to.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Criteria type, always `CUSTOM` for criteria managed with this resource.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"operator": schema.StringAttribute{
				MarkdownDescription: "Logical operator between the groups, `AND` or `OR`.",
				Required:            true,
			},
			"groups_json": schema.StringAttribute{
				MarkdownDescription: "Criteria groups as a JSON array. Each group has an `operator` (`AND` or `OR`) between its `criteriaItems`. Each item has a `targetType` (`group`), a `property` (`displayName`, `description`, `value` or `attributes.<name>`), an `operator` (`IN`, `EQUALS`, `NOT_EQUALS`, `CONTAINS`, `DOES_NOT_CONTAIN`, `STARTS_WITH` or `ENDS_WITH`), 1 to 50 `values` and `ignoreCase`. Use `jsonencode()`. The value is compared on the configured fields only, so formatting, key order and defaults IdentityNow adds do not produce a diff.",
				Required:            true,
				Validators:          []validator.String{jsonArrayStringValidator{}},
			},
			"privilege_level": schema.StringAttribute{
				MarkdownDescription: "Privilege level assigned by the criteria, `HIGH`, `MEDIUM` or `LOW`.",
				Required:            true,
			},
		},
	}
}

func (r *PrivilegeCriteriaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// privilegeCriteriaFromModel converts the model to the request body. PUT replaces the whole criteria,
// every field of which is modelled.
func privilegeCriteriaFromModel(data PrivilegeCriteriaModel, diags *diag.Diagnostics) *PrivilegeCriteria {
	criteria := &PrivilegeCriteria{
		ID:             data.ID.ValueString(),
		SourceID:       data.SourceID.ValueString(),
		Type:           privilegeCriteriaTypeCustom,
		Operator:       data.Operator.ValueString(),
		Groups:         []interface{}{},
		PrivilegeLevel: data.PrivilegeLevel.ValueString(),
	}
	if data.ID.IsUnknown() {
		criteria.ID = ""
	}
	if groups, ok := campaignTemplateJSONValue(data.GroupsJSON, "groups_json", diags).([]interface{}); ok {
		criteria.Groups = groups
	}
	return criteria
}

// setPrivilegeCriteriaState maps an API criteria onto the model, keeping the prior groups JSON when
// the API value contains it.
func setPrivilegeCriteriaState(data *PrivilegeCriteriaModel, criteria *PrivilegeCriteria) {
	data.ID = types.StringValue(criteria.ID)
	data.SourceID = types.StringValue(criteria.SourceID)
	data.Type = types.StringValue(criteria.Type)
	data.Operator = types.StringValue(criteria.Operator)
	data.GroupsJSON = jsonSubsetState(data.GroupsJSON, criteria.Groups)
	data.PrivilegeLevel = types.StringValue(criteria.PrivilegeLevel)
}

func (r *PrivilegeCriteriaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivilegeCriteriaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	criteria := privilegeCriteriaFromModel(data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreatePrivilegeCriteria(ctx, criteria)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create privilege criteria: %s", err))
		return
	}
	data.ID = types.StringValue(created.ID)
	data.Type = types.StringValue(privilegeCriteriaTypeCustom)
	if created.Type != "" {
		data.Type = types.StringValue(created.Type)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivilegeCriteriaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivilegeCriteriaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	criteria, err := client.GetPrivilegeCriteria(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read privilege criteria: %s", err))
		return
	}
	setPrivilegeCriteriaState(&data, criteria)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivilegeCriteriaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PrivilegeCriteriaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	criteria := privilegeCriteriaFromModel(data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if _, err := client.UpdatePrivilegeCriteria(ctx, data.ID.ValueString(), criteria); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update privilege criteria: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivilegeCriteriaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrivilegeCriteriaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeletePrivilegeCriteria(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete privilege criteria: %s", err))
	}
}

func (r *PrivilegeCriteriaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &PrivilegeCriteriaDataSource{}

func NewPrivilegeCriteriaDataSource() datasource.DataSource {
	return &PrivilegeCriteriaDataSource{}
}

type PrivilegeCriteriaDataSource struct {
	client *Config
}

func (d *PrivilegeCriteriaDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privilege_criteria"
}

func (d *PrivilegeCriteriaDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a privilege criteria by ID. Connector and single level criteria can be read as well as custom criteria.",
		Attributes: map[string]dsschema.Attribute{
			"id":              dsschema.StringAttribute{MarkdownDescription: "Privilege criteria ID.", Required: true},
			"source_id":       dsschema.StringAttribute{MarkdownDescription: "ID of the source the criteria applies to.", Computed: true},
			"type":            dsschema.StringAttribute{MarkdownDescription: "Criteria type, `CUSTOM`, `CONNECTOR` or `SINGLE_LEVEL`.", Computed: true},
			"operator":        dsschema.StringAttribute{MarkdownDescription: "Logical operator between the groups.", Computed: true},
			"groups_json":     dsschema.StringAttribute{MarkdownDescription: "Criteria groups as a JSON array.", Computed: true},
			"privilege_level": dsschema.StringAttribute{MarkdownDescription: "Privilege level assigned by the criteria.", Computed: true},
		},
	}
}

func (d *PrivilegeCriteriaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PrivilegeCriteriaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivilegeCriteriaModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	criteria, err := client.GetPrivilegeCriteria(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read privilege criteria: %s", err))
		return
	}
	setPrivilegeCriteriaState(&data, criteria)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
