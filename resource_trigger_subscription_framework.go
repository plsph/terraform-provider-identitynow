package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewTriggerSubscriptionResource)
}

// TriggerSubscription is an event trigger subscription as returned by the /v2026/trigger-subscriptions API.
type TriggerSubscription struct {
	ID                string                                `json:"id,omitempty"`
	Name              string                                `json:"name"`
	Description       string                                `json:"description,omitempty"`
	TriggerID         string                                `json:"triggerId,omitempty"`
	TriggerName       string                                `json:"triggerName,omitempty"`
	Type              string                                `json:"type"`
	ResponseDeadline  string                                `json:"responseDeadline,omitempty"`
	HTTPConfig        *TriggerSubscriptionHTTPConfig        `json:"httpConfig,omitempty"`
	EventBridgeConfig *TriggerSubscriptionEventBridgeConfig `json:"eventBridgeConfig,omitempty"`
	WorkflowConfig    map[string]interface{}                `json:"workflowConfig,omitempty"`
	Enabled           *bool                                 `json:"enabled,omitempty"`
	Filter            string                                `json:"filter,omitempty"`
}

// TriggerSubscriptionHTTPConfig is the configuration of an HTTP subscription.
type TriggerSubscriptionHTTPConfig struct {
	URL                    string                                    `json:"url"`
	HTTPDispatchMode       string                                    `json:"httpDispatchMode"`
	HTTPAuthenticationType string                                    `json:"httpAuthenticationType,omitempty"`
	BasicAuthConfig        *TriggerSubscriptionBasicAuthConfig       `json:"basicAuthConfig,omitempty"`
	BearerTokenAuthConfig  *TriggerSubscriptionBearerTokenAuthConfig `json:"bearerTokenAuthConfig,omitempty"`
}

// TriggerSubscriptionBasicAuthConfig holds basic authentication credentials. The API never returns the password.
type TriggerSubscriptionBasicAuthConfig struct {
	UserName string  `json:"userName,omitempty"`
	Password *string `json:"password,omitempty"`
}

// TriggerSubscriptionBearerTokenAuthConfig holds a bearer token. The API never returns the token.
type TriggerSubscriptionBearerTokenAuthConfig struct {
	BearerToken *string `json:"bearerToken,omitempty"`
}

// TriggerSubscriptionEventBridgeConfig is the configuration of an AWS EventBridge subscription.
type TriggerSubscriptionEventBridgeConfig struct {
	AWSAccount string `json:"awsAccount"`
	AWSRegion  string `json:"awsRegion"`
}

func (c *Client) CreateTriggerSubscription(ctx context.Context, subscription *TriggerSubscription) (*TriggerSubscription, error) {
	var created TriggerSubscription
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/trigger-subscriptions", subscription, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// GetTriggerSubscription finds a subscription by ID. The API has no GET by ID, so the list is
// filtered by ID.
func (c *Client) GetTriggerSubscription(ctx context.Context, id string) (*TriggerSubscription, error) {
	items, err := listAllPages[TriggerSubscription](ctx, c, "/v2026/trigger-subscriptions", url.Values{"filters": {eqFilter("id", id)}})
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, &NotFoundError{fmt.Sprintf("trigger subscription %q not found", id)}
}

func (c *Client) PatchTriggerSubscription(ctx context.Context, id string, ops []jsonPatchOp) (*TriggerSubscription, error) {
	var updated TriggerSubscription
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/trigger-subscriptions/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteTriggerSubscription(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/trigger-subscriptions/%s", id), nil, nil)
}

var _ resource.Resource = &TriggerSubscriptionResource{}
var _ resource.ResourceWithImportState = &TriggerSubscriptionResource{}
var _ resource.ResourceWithValidateConfig = &TriggerSubscriptionResource{}

func NewTriggerSubscriptionResource() resource.Resource {
	return &TriggerSubscriptionResource{}
}

type TriggerSubscriptionResource struct {
	client *Config
}

type TriggerSubscriptionResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	TriggerID          types.String `tfsdk:"trigger_id"`
	TriggerName        types.String `tfsdk:"trigger_name"`
	Type               types.String `tfsdk:"type"`
	ResponseDeadline   types.String `tfsdk:"response_deadline"`
	HTTPConfig         types.List   `tfsdk:"http_config"`
	EventBridgeConfig  types.List   `tfsdk:"event_bridge_config"`
	WorkflowConfigJSON types.String `tfsdk:"workflow_config_json"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	Filter             types.String `tfsdk:"filter"`
}

type TriggerSubscriptionHTTPConfigModel struct {
	URL                    types.String `tfsdk:"url"`
	HTTPDispatchMode       types.String `tfsdk:"http_dispatch_mode"`
	HTTPAuthenticationType types.String `tfsdk:"http_authentication_type"`
	BasicAuthUserName      types.String `tfsdk:"basic_auth_user_name"`
	BasicAuthPassword      types.String `tfsdk:"basic_auth_password"`
	BearerToken            types.String `tfsdk:"bearer_token"`
}

type TriggerSubscriptionEventBridgeConfigModel struct {
	AWSAccount types.String `tfsdk:"aws_account"`
	AWSRegion  types.String `tfsdk:"aws_region"`
}

var triggerSubscriptionHTTPConfigObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"url":                      types.StringType,
	"http_dispatch_mode":       types.StringType,
	"http_authentication_type": types.StringType,
	"basic_auth_user_name":     types.StringType,
	"basic_auth_password":      types.StringType,
	"bearer_token":             types.StringType,
}}

var triggerSubscriptionEventBridgeConfigObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"aws_account": types.StringType,
	"aws_region":  types.StringType,
}}

// triggerSubscriptionOneOfValidator checks that a string attribute is one of the allowed values.
type triggerSubscriptionOneOfValidator struct {
	values []string
}

func (v triggerSubscriptionOneOfValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be one of %s", strings.Join(v.values, ", "))
}

func (v triggerSubscriptionOneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v triggerSubscriptionOneOfValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, value := range v.values {
		if req.ConfigValue.ValueString() == value {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid value", fmt.Sprintf("%s, got %q.", v.Description(ctx), req.ConfigValue.ValueString()))
}

func (r *TriggerSubscriptionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger_subscription"
}

func (r *TriggerSubscriptionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a subscription to an event trigger. The subscription defines where and how trigger invocations are delivered.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Subscription ID",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Subscription name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Subscription description",
				Optional:            true,
			},
			"trigger_id": schema.StringAttribute{
				MarkdownDescription: "ID of the trigger to subscribe to, e.g. `idn:identity-created`. Changing it forces a new subscription.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"trigger_name": schema.StringAttribute{
				MarkdownDescription: "Name of the trigger",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Subscription type: `HTTP`, `EVENTBRIDGE`, `INLINE`, `SCRIPT` or `WORKFLOW`. `HTTP` requires `http_config`, `EVENTBRIDGE` requires `event_bridge_config`.",
				Required:            true,
				Validators:          []validator.String{triggerSubscriptionOneOfValidator{values: []string{"HTTP", "EVENTBRIDGE", "INLINE", "SCRIPT", "WORKFLOW"}}},
			},
			"response_deadline": schema.StringAttribute{
				MarkdownDescription: "Deadline for completing a `REQUEST_RESPONSE` trigger invocation as an ISO-8601 duration, e.g. `PT1H`. Defaults to the API default (`PT1H`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"workflow_config_json": schema.StringAttribute{
				MarkdownDescription: "Configuration of a `WORKFLOW` subscription as a JSON object. The v2026 API documents the field as patchable but does not describe its content, so it is passed through as is. Compared semantically.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the subscription receives real-time trigger invocations. Test invocations are always enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"filter": schema.StringAttribute{
				MarkdownDescription: "JSONPath filter; the trigger is only invoked when the expression evaluates to true.",
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"http_config": schema.ListNestedBlock{
				MarkdownDescription: "Configuration of an `HTTP` subscription. At most one block.",
				Validators:          []validator.List{listSizeBetween(0, 1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							MarkdownDescription: "URL of the external integration",
							Required:            true,
						},
						"http_dispatch_mode": schema.StringAttribute{
							MarkdownDescription: "HTTP response mode: `SYNC`, `ASYNC` or `DYNAMIC`",
							Required:            true,
							Validators:          []validator.String{triggerSubscriptionOneOfValidator{values: []string{"SYNC", "ASYNC", "DYNAMIC"}}},
						},
						"http_authentication_type": schema.StringAttribute{
							MarkdownDescription: "Authentication type: `NO_AUTH`, `BASIC_AUTH` or `BEARER_TOKEN`. Defaults to `NO_AUTH`.",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("NO_AUTH"),
							Validators:          []validator.String{triggerSubscriptionOneOfValidator{values: []string{"NO_AUTH", "BASIC_AUTH", "BEARER_TOKEN"}}},
						},
						"basic_auth_user_name": schema.StringAttribute{
							MarkdownDescription: "User name for `BASIC_AUTH`",
							Optional:            true,
						},
						"basic_auth_password": schema.StringAttribute{
							MarkdownDescription: "Password for `BASIC_AUTH`. The API never returns it, so changes made outside Terraform are not detected.",
							Optional:            true,
							Sensitive:           true,
						},
						"bearer_token": schema.StringAttribute{
							MarkdownDescription: "Token for `BEARER_TOKEN`. The API never returns it, so changes made outside Terraform are not detected.",
							Optional:            true,
							Sensitive:           true,
						},
					},
				},
			},
			"event_bridge_config": schema.ListNestedBlock{
				MarkdownDescription: "Configuration of an `EVENTBRIDGE` subscription. At most one block.",
				Validators:          []validator.List{listSizeBetween(0, 1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"aws_account": schema.StringAttribute{
							MarkdownDescription: "12-digit AWS account number that has the EventBridge partner event source",
							Required:            true,
						},
						"aws_region": schema.StringAttribute{
							MarkdownDescription: "AWS region that has the EventBridge partner event source, e.g. `us-east-1`",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (r *TriggerSubscriptionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data TriggerSubscriptionResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.Type.IsNull() || data.Type.IsUnknown() {
		return
	}
	switch data.Type.ValueString() {
	case "HTTP":
		if !data.HTTPConfig.IsUnknown() && len(data.HTTPConfig.Elements()) == 0 {
			resp.Diagnostics.AddAttributeError(path.Root("http_config"), "Missing http_config", "Subscriptions of type HTTP require an http_config block.")
		}
	case "EVENTBRIDGE":
		if !data.EventBridgeConfig.IsUnknown() && len(data.EventBridgeConfig.Elements()) == 0 {
			resp.Diagnostics.AddAttributeError(path.Root("event_bridge_config"), "Missing event_bridge_config", "Subscriptions of type EVENTBRIDGE require an event_bridge_config block.")
		}
	}
}

func (r *TriggerSubscriptionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// triggerSubscriptionHTTPConfigFromModel converts the http_config block to the API object.
func triggerSubscriptionHTTPConfigFromModel(ctx context.Context, list types.List, diags *diag.Diagnostics) *TriggerSubscriptionHTTPConfig {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil
	}
	var models []TriggerSubscriptionHTTPConfigModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if len(models) == 0 {
		return nil
	}
	m := models[0]
	config := &TriggerSubscriptionHTTPConfig{
		URL:                    m.URL.ValueString(),
		HTTPDispatchMode:       m.HTTPDispatchMode.ValueString(),
		HTTPAuthenticationType: m.HTTPAuthenticationType.ValueString(),
	}
	if !m.BasicAuthUserName.IsNull() || !m.BasicAuthPassword.IsNull() {
		config.BasicAuthConfig = &TriggerSubscriptionBasicAuthConfig{UserName: m.BasicAuthUserName.ValueString(), Password: stringPointer(m.BasicAuthPassword)}
	}
	if !m.BearerToken.IsNull() {
		config.BearerTokenAuthConfig = &TriggerSubscriptionBearerTokenAuthConfig{BearerToken: stringPointer(m.BearerToken)}
	}
	return config
}

// triggerSubscriptionEventBridgeConfigFromModel converts the event_bridge_config block to the API object.
func triggerSubscriptionEventBridgeConfigFromModel(ctx context.Context, list types.List, diags *diag.Diagnostics) *TriggerSubscriptionEventBridgeConfig {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil
	}
	var models []TriggerSubscriptionEventBridgeConfigModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if len(models) == 0 {
		return nil
	}
	return &TriggerSubscriptionEventBridgeConfig{AWSAccount: models[0].AWSAccount.ValueString(), AWSRegion: models[0].AWSRegion.ValueString()}
}

// triggerSubscriptionWorkflowConfigFromModel decodes workflow_config_json, nil when unset.
func triggerSubscriptionWorkflowConfigFromModel(value types.String, diags *diag.Diagnostics) map[string]interface{} {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(value.ValueString()), &config); err != nil {
		diags.AddAttributeError(path.Root("workflow_config_json"), "Invalid JSON", err.Error())
		return nil
	}
	return config
}

// triggerSubscriptionFromModel builds the create request from the plan.
func triggerSubscriptionFromModel(ctx context.Context, data TriggerSubscriptionResourceModel, diags *diag.Diagnostics) *TriggerSubscription {
	return &TriggerSubscription{
		Name:              data.Name.ValueString(),
		Description:       data.Description.ValueString(),
		TriggerID:         data.TriggerID.ValueString(),
		Type:              data.Type.ValueString(),
		ResponseDeadline:  data.ResponseDeadline.ValueString(),
		HTTPConfig:        triggerSubscriptionHTTPConfigFromModel(ctx, data.HTTPConfig, diags),
		EventBridgeConfig: triggerSubscriptionEventBridgeConfigFromModel(ctx, data.EventBridgeConfig, diags),
		WorkflowConfig:    triggerSubscriptionWorkflowConfigFromModel(data.WorkflowConfigJSON, diags),
		Enabled:           boolPointer(data.Enabled),
		Filter:            data.Filter.ValueString(),
	}
}

// triggerSubscriptionHTTPConfigState maps the API HTTP config to the block. The API does not return
// secrets, so the password and bearer token are kept from the prior state.
func triggerSubscriptionHTTPConfigState(ctx context.Context, prior types.List, config *TriggerSubscriptionHTTPConfig, diags *diag.Diagnostics) types.List {
	if config == nil {
		return types.ListNull(triggerSubscriptionHTTPConfigObjectType)
	}
	var priorModel TriggerSubscriptionHTTPConfigModel
	if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) > 0 {
		var models []TriggerSubscriptionHTTPConfigModel
		diags.Append(prior.ElementsAs(ctx, &models, false)...)
		if len(models) > 0 {
			priorModel = models[0]
		}
	}
	authType := config.HTTPAuthenticationType
	if authType == "" {
		authType = "NO_AUTH"
	}
	model := TriggerSubscriptionHTTPConfigModel{
		URL:                    types.StringValue(config.URL),
		HTTPDispatchMode:       types.StringValue(config.HTTPDispatchMode),
		HTTPAuthenticationType: types.StringValue(authType),
		BasicAuthUserName:      types.StringNull(),
		BasicAuthPassword:      priorModel.BasicAuthPassword,
		BearerToken:            priorModel.BearerToken,
	}
	if priorModel.BasicAuthPassword.IsUnknown() {
		model.BasicAuthPassword = types.StringNull()
	}
	if priorModel.BearerToken.IsUnknown() {
		model.BearerToken = types.StringNull()
	}
	userName := ""
	if config.BasicAuthConfig != nil {
		userName = config.BasicAuthConfig.UserName
		if config.BasicAuthConfig.Password != nil && *config.BasicAuthConfig.Password != "" {
			model.BasicAuthPassword = types.StringValue(*config.BasicAuthConfig.Password)
		}
	}
	if config.BearerTokenAuthConfig != nil && config.BearerTokenAuthConfig.BearerToken != nil && *config.BearerTokenAuthConfig.BearerToken != "" {
		model.BearerToken = types.StringValue(*config.BearerTokenAuthConfig.BearerToken)
	}
	priorUserName := priorModel.BasicAuthUserName
	if priorUserName.IsUnknown() {
		priorUserName = types.StringNull()
	}
	model.BasicAuthUserName = optionalStringState(priorUserName, userName)
	list, d := types.ListValueFrom(ctx, triggerSubscriptionHTTPConfigObjectType, []TriggerSubscriptionHTTPConfigModel{model})
	diags.Append(d...)
	return list
}

func triggerSubscriptionEventBridgeConfigState(ctx context.Context, config *TriggerSubscriptionEventBridgeConfig, diags *diag.Diagnostics) types.List {
	if config == nil {
		return types.ListNull(triggerSubscriptionEventBridgeConfigObjectType)
	}
	list, d := types.ListValueFrom(ctx, triggerSubscriptionEventBridgeConfigObjectType, []TriggerSubscriptionEventBridgeConfigModel{{
		AWSAccount: types.StringValue(config.AWSAccount),
		AWSRegion:  types.StringValue(config.AWSRegion),
	}})
	diags.Append(d...)
	return list
}

// setTriggerSubscriptionState refreshes the model from the API, keeping null for unset optional
// attributes, semantically equal JSON and the secrets the API does not return.
func setTriggerSubscriptionState(ctx context.Context, data *TriggerSubscriptionResourceModel, subscription *TriggerSubscription, diags *diag.Diagnostics) {
	data.ID = types.StringValue(subscription.ID)
	data.Name = types.StringValue(subscription.Name)
	data.Description = optionalStringState(data.Description, subscription.Description)
	data.TriggerID = types.StringValue(subscription.TriggerID)
	data.TriggerName = types.StringValue(subscription.TriggerName)
	data.Type = types.StringValue(subscription.Type)
	data.ResponseDeadline = types.StringValue(subscription.ResponseDeadline)
	data.HTTPConfig = triggerSubscriptionHTTPConfigState(ctx, data.HTTPConfig, subscription.HTTPConfig, diags)
	data.EventBridgeConfig = triggerSubscriptionEventBridgeConfigState(ctx, subscription.EventBridgeConfig, diags)
	data.WorkflowConfigJSON = jsonStringState(data.WorkflowConfigJSON, subscription.WorkflowConfig)
	data.Enabled = types.BoolValue(subscription.Enabled == nil || *subscription.Enabled)
	data.Filter = optionalStringState(data.Filter, subscription.Filter)
}

// triggerSubscriptionPatchOps returns JSON Patch operations for the attributes that changed.
// Optional members that are null in the state may be missing from the API object (e.g. the
// eventBridgeConfig of an HTTP subscription whose type changes), and replace fails for a missing
// member (RFC 6902), so they are set with add, which creates or replaces an object member.
func triggerSubscriptionPatchOps(ctx context.Context, plan, state TriggerSubscriptionResourceModel, diags *diag.Diagnostics) []jsonPatchOp {
	var b patchBuilder
	addOrReplace := func(set func(), prior attr.Value) {
		before := len(b.ops)
		set()
		if len(b.ops) > before && b.ops[before].Op == "replace" && prior.IsNull() {
			b.ops[before].Op = "add"
		}
	}
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	addOrReplace(func() {
		b.replaceIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	}, state.Description)
	b.replaceIfChanged(plan.Type, state.Type, "/type", plan.Type.ValueString())
	b.replaceIfChanged(plan.Enabled, state.Enabled, "/enabled", plan.Enabled.ValueBool())
	addOrReplace(func() { b.replaceOrRemoveIfChanged(plan.Filter, state.Filter, "/filter", plan.Filter.ValueString()) }, state.Filter)
	b.replaceIfChanged(plan.ResponseDeadline, state.ResponseDeadline, "/responseDeadline", plan.ResponseDeadline.ValueString())
	addOrReplace(func() {
		b.replaceOrRemoveIfChanged(plan.HTTPConfig, state.HTTPConfig, "/httpConfig", triggerSubscriptionHTTPConfigFromModel(ctx, plan.HTTPConfig, diags))
	}, state.HTTPConfig)
	addOrReplace(func() {
		b.replaceOrRemoveIfChanged(plan.EventBridgeConfig, state.EventBridgeConfig, "/eventBridgeConfig", triggerSubscriptionEventBridgeConfigFromModel(ctx, plan.EventBridgeConfig, diags))
	}, state.EventBridgeConfig)
	addOrReplace(func() {
		b.replaceOrRemoveIfChanged(plan.WorkflowConfigJSON, state.WorkflowConfigJSON, "/workflowConfig", triggerSubscriptionWorkflowConfigFromModel(plan.WorkflowConfigJSON, diags))
	}, state.WorkflowConfigJSON)
	return b.ops
}

func (r *TriggerSubscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TriggerSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	subscription := triggerSubscriptionFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateTriggerSubscription(ctx, subscription)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create trigger subscription: %s", err))
		return
	}
	data.ID = types.StringValue(created.ID)
	data.TriggerName = types.StringValue(created.TriggerName)
	data.ResponseDeadline = computedStringFromAPI(data.ResponseDeadline, created.ResponseDeadline)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TriggerSubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TriggerSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	subscription, err := client.GetTriggerSubscription(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read trigger subscription: %s", err))
		return
	}
	setTriggerSubscriptionState(ctx, &data, subscription, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TriggerSubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TriggerSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := triggerSubscriptionPatchOps(ctx, plan, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(ops) > 0 {
		client, err := r.client.IdentityNowClient(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		updated, err := client.PatchTriggerSubscription(ctx, plan.ID.ValueString(), ops)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update trigger subscription: %s", err))
			return
		}
		plan.ResponseDeadline = computedStringFromAPI(plan.ResponseDeadline, updated.ResponseDeadline)
	}
	if plan.ResponseDeadline.IsUnknown() {
		plan.ResponseDeadline = state.ResponseDeadline
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TriggerSubscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TriggerSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteTriggerSubscription(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete trigger subscription: %s", err))
	}
}

func (r *TriggerSubscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
