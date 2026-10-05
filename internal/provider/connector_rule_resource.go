package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	registerResource(NewConnectorRuleResource)
	registerDataSource(NewConnectorRuleDataSource)
}

// connectorRulePageSize is the maximum page size of the connector rules list endpoint.
const connectorRulePageSize = 50

// ConnectorRule is a connector rule as used by the /v2026/connector-rules API.
type ConnectorRule struct {
	ID          string                  `json:"id,omitempty"`
	Name        string                  `json:"name"`
	Description *string                 `json:"description"`
	Type        string                  `json:"type"`
	Signature   json.RawMessage         `json:"signature,omitempty"`
	SourceCode  ConnectorRuleSourceCode `json:"sourceCode"`
	Attributes  map[string]interface{}  `json:"attributes"`
	Created     string                  `json:"created,omitempty"`
	Modified    string                  `json:"modified,omitempty"`
}

// ConnectorRuleSourceCode is the code of a connector rule.
type ConnectorRuleSourceCode struct {
	Version string `json:"version"`
	Script  string `json:"script"`
}

func (c *Client) GetConnectorRule(ctx context.Context, id string) (*ConnectorRule, error) {
	var rule ConnectorRule
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/connector-rules/%s", id), nil, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListConnectorRules lists all connector rules. The endpoint accepts at most 50 items per page.
func (c *Client) ListConnectorRules(ctx context.Context) ([]ConnectorRule, error) {
	return listAllPagesSized[ConnectorRule](ctx, c, "/v2026/connector-rules", nil, connectorRulePageSize)
}

// GetConnectorRuleByName returns the connector rule with the given name. The list endpoint has no
// name filter, so rules are matched client side.
func (c *Client) GetConnectorRuleByName(ctx context.Context, name string) (*ConnectorRule, error) {
	rules, err := c.ListConnectorRules(ctx)
	if err != nil {
		return nil, err
	}
	var match *ConnectorRule
	for i := range rules {
		if rules[i].Name != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple connector rules are named %q", name)
		}
		match = &rules[i]
	}
	if match == nil {
		return nil, &NotFoundError{fmt.Sprintf("connector rule with name %q not found", name)}
	}
	return match, nil
}

func (c *Client) CreateConnectorRule(ctx context.Context, rule *ConnectorRule) (*ConnectorRule, error) {
	var created ConnectorRule
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/connector-rules", rule, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateConnectorRule(ctx context.Context, id string, rule *ConnectorRule) (*ConnectorRule, error) {
	var updated ConnectorRule
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/connector-rules/%s", id), rule, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteConnectorRule(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/connector-rules/%s", id), nil, nil)
}

var _ resource.Resource = &ConnectorRuleResource{}
var _ resource.ResourceWithImportState = &ConnectorRuleResource{}

func NewConnectorRuleResource() resource.Resource {
	return &ConnectorRuleResource{}
}

type ConnectorRuleResource struct {
	client *Config
}

type ConnectorRuleModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Type           types.String `tfsdk:"type"`
	SignatureJSON  types.String `tfsdk:"signature_json"`
	SourceCode     types.List   `tfsdk:"source_code"`
	AttributesJSON types.String `tfsdk:"attributes_json"`
	Created        types.String `tfsdk:"created"`
	Modified       types.String `tfsdk:"modified"`
}

type ConnectorRuleSourceCodeModel struct {
	Version types.String `tfsdk:"version"`
	Script  types.String `tfsdk:"script"`
}

var connectorRuleSourceCodeObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"version": types.StringType,
	"script":  types.StringType,
}}

func (r *ConnectorRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector_rule"
}

func (r *ConnectorRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a connector rule, a BeanShell rule that runs on the virtual appliance, e.g. a `ConnectorBeforeCreate` rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Connector rule ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Rule name. The name is immutable. Changing this forces a new rule to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the rule's purpose.",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Rule type, e.g. `BuildMap`, `ConnectorAfterCreate`, `ConnectorAfterDelete`, `ConnectorAfterModify`, `ConnectorBeforeCreate`, `ConnectorBeforeDelete`, `ConnectorBeforeModify`, `JDBCBuildMap`, `JDBCOperationProvisioning`, `JDBCProvision`, `PeopleSoftHRMSBuildMap`, `RACFPermissionCustomization`, `SAPBuildMap`, `SapHrManagerRule`, `SapHrOperationProvisioning`, `SapHrProvision`, `SuccessFactorsOperationProvisioning`, `WebServiceAfterOperationRule` or `WebServiceBeforeOperationRule`. The type is immutable. Changing this forces a new rule to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"signature_json": schema.StringAttribute{
				MarkdownDescription: "Function signature of the rule as a JSON object with an `input` array and an optional `output` object; each argument has a `name`, `description` and `type`. Use `jsonencode()` for convenience. The value is compared semantically. When not set, the signature returned by IdentityNow is kept and not managed.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"attributes_json": schema.StringAttribute{
				MarkdownDescription: "Rule attributes as a JSON object. Use `jsonencode()` for convenience. The value is compared semantically.",
				Optional:            true,
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
			"source_code": schema.ListNestedBlock{
				MarkdownDescription: "Code of the rule. Exactly one block is required.",
				Validators:          []validator.List{listSizeBetween(1, 1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"version": schema.StringAttribute{
							MarkdownDescription: "Version of the code, e.g. `1.0`.",
							Required:            true,
						},
						"script": schema.StringAttribute{
							MarkdownDescription: "The BeanShell code. Use `file()` to load it from a file. Differences only in line endings or trailing whitespace are ignored.",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (r *ConnectorRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// connectorRuleFromModel builds the full rule from the plan. PUT replaces the rule, and the
// request covers every writable field, so nothing set outside Terraform is kept except an
// unmanaged signature, which is taken from the prior state.
func connectorRuleFromModel(ctx context.Context, data ConnectorRuleModel, diags *diag.Diagnostics) *ConnectorRule {
	rule := &ConnectorRule{
		Name:        data.Name.ValueString(),
		Description: stringPointer(data.Description),
		Type:        data.Type.ValueString(),
	}
	if !data.SignatureJSON.IsNull() && !data.SignatureJSON.IsUnknown() && data.SignatureJSON.ValueString() != "" {
		rule.Signature = json.RawMessage(data.SignatureJSON.ValueString())
	}
	if !data.AttributesJSON.IsNull() && !data.AttributesJSON.IsUnknown() && data.AttributesJSON.ValueString() != "" {
		if err := json.Unmarshal([]byte(data.AttributesJSON.ValueString()), &rule.Attributes); err != nil {
			diags.AddAttributeError(path.Root("attributes_json"), "Invalid JSON", err.Error())
		}
	}
	var code []ConnectorRuleSourceCodeModel
	if !data.SourceCode.IsNull() && !data.SourceCode.IsUnknown() {
		diags.Append(data.SourceCode.ElementsAs(ctx, &code, false)...)
	}
	if len(code) > 0 {
		rule.SourceCode = ConnectorRuleSourceCode{Version: code[0].Version.ValueString(), Script: code[0].Script.ValueString()}
	}
	return rule
}

// connectorRuleSourceCodeState returns the source_code state from the API, keeping the prior
// script when it differs only in line endings or trailing whitespace.
func connectorRuleSourceCodeState(ctx context.Context, prior types.List, code ConnectorRuleSourceCode, diags *diag.Diagnostics) types.List {
	priorScript := types.StringNull()
	if !prior.IsNull() && !prior.IsUnknown() {
		var models []ConnectorRuleSourceCodeModel
		diags.Append(prior.ElementsAs(ctx, &models, false)...)
		if len(models) > 0 {
			priorScript = models[0].Script
		}
	}
	list, d := types.ListValueFrom(ctx, connectorRuleSourceCodeObjectType, []ConnectorRuleSourceCodeModel{{
		Version: types.StringValue(code.Version),
		Script:  connectorTextState(priorScript, code.Script),
	}})
	diags.Append(d...)
	return list
}

// setConnectorRuleState maps an API rule onto the model during refresh.
func setConnectorRuleState(ctx context.Context, data *ConnectorRuleModel, rule *ConnectorRule, diags *diag.Diagnostics) {
	data.ID = types.StringValue(rule.ID)
	data.Name = types.StringValue(rule.Name)
	description := ""
	if rule.Description != nil {
		description = *rule.Description
	}
	data.Description = optionalStringState(data.Description, description)
	data.Type = types.StringValue(rule.Type)
	data.SignatureJSON = connectorRuleSignatureState(data.SignatureJSON, rule.Signature)
	data.SourceCode = connectorRuleSourceCodeState(ctx, data.SourceCode, rule.SourceCode, diags)
	data.AttributesJSON = jsonStringState(data.AttributesJSON, rule.Attributes)
	data.Created = types.StringValue(rule.Created)
	data.Modified = stringValueOrNull(rule.Modified)
}

// connectorRuleSignatureState returns the state of signature_json, keeping the prior value when the
// API value contains it, since the API adds null descriptions and types.
func connectorRuleSignatureState(prior types.String, raw json.RawMessage) types.String {
	var signature interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &signature); err != nil {
			return types.StringValue(string(raw))
		}
	}
	if prior.IsUnknown() {
		prior = types.StringNull()
	}
	return jsonSubsetState(prior, signature)
}

// applyConnectorRuleResponse resolves the computed values after create or update. Configured
// values are kept as planned.
func applyConnectorRuleResponse(data *ConnectorRuleModel, rule *ConnectorRule) {
	if data.ID.IsUnknown() {
		data.ID = types.StringValue(rule.ID)
	}
	if data.SignatureJSON.IsUnknown() {
		data.SignatureJSON = connectorRuleSignatureState(types.StringNull(), rule.Signature)
	}
	if data.Created.IsUnknown() {
		data.Created = types.StringValue(rule.Created)
	}
	data.Modified = stringValueOrNull(rule.Modified)
}

func (r *ConnectorRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectorRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule := connectorRuleFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateConnectorRule(ctx, rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create connector rule: %s", err))
		return
	}
	applyConnectorRuleResponse(&data, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectorRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectorRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	rule, err := client.GetConnectorRule(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read connector rule: %s", err))
		return
	}
	setConnectorRuleState(ctx, &data, rule, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectorRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectorRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule := connectorRuleFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	// The update request must contain the (immutable) ID.
	rule.ID = data.ID.ValueString()
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.UpdateConnectorRule(ctx, data.ID.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update connector rule: %s", err))
		return
	}
	applyConnectorRuleResponse(&data, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConnectorRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectorRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteConnectorRule(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete connector rule: %s", err))
	}
}

func (r *ConnectorRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &ConnectorRuleDataSource{}
var _ datasource.DataSourceWithValidateConfig = &ConnectorRuleDataSource{}

func NewConnectorRuleDataSource() datasource.DataSource {
	return &ConnectorRuleDataSource{}
}

type ConnectorRuleDataSource struct {
	client *Config
}

func (d *ConnectorRuleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector_rule"
}

func (d *ConnectorRuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a connector rule by ID or name.",
		Attributes: map[string]dsschema.Attribute{
			"id":              dsschema.StringAttribute{MarkdownDescription: "Connector rule ID. Exactly one of `id` or `name` must be set.", Optional: true, Computed: true},
			"name":            dsschema.StringAttribute{MarkdownDescription: "Rule name. Exactly one of `id` or `name` must be set.", Optional: true, Computed: true},
			"description":     dsschema.StringAttribute{MarkdownDescription: "Description of the rule's purpose.", Computed: true},
			"type":            dsschema.StringAttribute{MarkdownDescription: "Rule type.", Computed: true},
			"signature_json":  dsschema.StringAttribute{MarkdownDescription: "Function signature of the rule as a JSON object.", Computed: true},
			"attributes_json": dsschema.StringAttribute{MarkdownDescription: "Rule attributes as a JSON object.", Computed: true},
			"created":         dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"modified":        dsschema.StringAttribute{MarkdownDescription: "Last modification date.", Computed: true},
			"source_code": dsschema.ListNestedAttribute{
				MarkdownDescription: "Code of the rule, a list with one element.",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"version": dsschema.StringAttribute{MarkdownDescription: "Version of the code.", Computed: true},
						"script":  dsschema.StringAttribute{MarkdownDescription: "The BeanShell code.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *ConnectorRuleDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *ConnectorRuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ConnectorRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConnectorRuleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var rule *ConnectorRule
	if !data.ID.IsNull() {
		rule, err = client.GetConnectorRule(ctx, data.ID.ValueString())
	} else {
		rule, err = client.GetConnectorRuleByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Connector rule not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read connector rule: %s", err))
		return
	}
	data.Description = types.StringValue("")
	setConnectorRuleState(ctx, &data, rule, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
