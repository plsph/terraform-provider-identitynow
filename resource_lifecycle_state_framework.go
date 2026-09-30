package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewLifecycleStateResource)
	registerDataSource(NewLifecycleStateDataSource)
}

// LifecycleStateEmailNotificationOption is the email configuration of a lifecycle state.
type LifecycleStateEmailNotificationOption struct {
	NotifyManagers      bool     `json:"notifyManagers"`
	NotifyAllAdmins     bool     `json:"notifyAllAdmins"`
	NotifySpecificUsers bool     `json:"notifySpecificUsers"`
	EmailAddressList    []string `json:"emailAddressList"`
}

// LifecycleStateAccessActionConfiguration is the access configuration of a lifecycle state.
type LifecycleStateAccessActionConfiguration struct {
	RemoveAllAccessEnabled bool `json:"removeAllAccessEnabled"`
}

// LifecycleState is a lifecycle state as returned by the
// /v2026/identity-profiles/{identity-profile-id}/lifecycle-states API.
type LifecycleState struct {
	ID                        string                                   `json:"id,omitempty"`
	Name                      string                                   `json:"name"`
	TechnicalName             string                                   `json:"technicalName"`
	Description               *string                                  `json:"description,omitempty"`
	Enabled                   *bool                                    `json:"enabled,omitempty"`
	IdentityCount             *int64                                   `json:"identityCount,omitempty"`
	EmailNotificationOption   *LifecycleStateEmailNotificationOption   `json:"emailNotificationOption,omitempty"`
	AccountActions            json.RawMessage                          `json:"accountActions,omitempty"`
	AccessProfileIDs          []string                                 `json:"accessProfileIds,omitempty"`
	IdentityState             *string                                  `json:"identityState,omitempty"`
	AccessActionConfiguration *LifecycleStateAccessActionConfiguration `json:"accessActionConfiguration,omitempty"`
	Priority                  *int64                                   `json:"priority,omitempty"`
	Created                   string                                   `json:"created,omitempty"`
	Modified                  string                                   `json:"modified,omitempty"`
}

func lifecycleStatePath(profileID, id string) string {
	if id == "" {
		return apiPath("/v2026/identity-profiles/%s/lifecycle-states", profileID)
	}
	return apiPath("/v2026/identity-profiles/%s/lifecycle-states/%s", profileID, id)
}

func (c *Client) GetLifecycleState(ctx context.Context, profileID, id string) (*LifecycleState, error) {
	var state LifecycleState
	if err := c.doJSON(ctx, http.MethodGet, lifecycleStatePath(profileID, id), nil, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (c *Client) CreateLifecycleState(ctx context.Context, profileID string, state *LifecycleState) (*LifecycleState, error) {
	var created LifecycleState
	if err := c.doJSON(ctx, http.MethodPost, lifecycleStatePath(profileID, ""), state, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateLifecycleState(ctx context.Context, profileID, id string, ops []jsonPatchOp) (*LifecycleState, error) {
	var updated LifecycleState
	if err := c.doJSON(ctx, http.MethodPatch, lifecycleStatePath(profileID, id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeleteLifecycleState deletes a lifecycle state. The API accepts the request with 202 and returns
// a reference to the deleted state or to a task, which is ignored.
func (c *Client) DeleteLifecycleState(ctx context.Context, profileID, id string) error {
	return c.doJSON(ctx, http.MethodDelete, lifecycleStatePath(profileID, id), nil, nil)
}

var _ resource.Resource = &LifecycleStateResource{}
var _ resource.ResourceWithImportState = &LifecycleStateResource{}

func NewLifecycleStateResource() resource.Resource {
	return &LifecycleStateResource{}
}

type LifecycleStateResource struct {
	client *Config
}

type LifecycleStateModel struct {
	ID                      types.String `tfsdk:"id"`
	IdentityProfileID       types.String `tfsdk:"identity_profile_id"`
	Name                    types.String `tfsdk:"name"`
	TechnicalName           types.String `tfsdk:"technical_name"`
	Description             types.String `tfsdk:"description"`
	Enabled                 types.Bool   `tfsdk:"enabled"`
	IdentityState           types.String `tfsdk:"identity_state"`
	Priority                types.Int64  `tfsdk:"priority"`
	EmailNotificationOption types.List   `tfsdk:"email_notification_option"`
	AccountActionsJSON      types.String `tfsdk:"account_actions_json"`
	AccessProfileIDs        types.Set    `tfsdk:"access_profile_ids"`
	RemoveAllAccessEnabled  types.Bool   `tfsdk:"remove_all_access_enabled"`
	IdentityCount           types.Int64  `tfsdk:"identity_count"`
	Created                 types.String `tfsdk:"created"`
	Modified                types.String `tfsdk:"modified"`
}

type LifecycleStateEmailNotificationModel struct {
	NotifyManagers      types.Bool `tfsdk:"notify_managers"`
	NotifyAllAdmins     types.Bool `tfsdk:"notify_all_admins"`
	NotifySpecificUsers types.Bool `tfsdk:"notify_specific_users"`
	EmailAddressList    types.List `tfsdk:"email_address_list"`
}

var lifecycleStateEmailNotificationObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"notify_managers":       types.BoolType,
	"notify_all_admins":     types.BoolType,
	"notify_specific_users": types.BoolType,
	"email_address_list":    types.ListType{ElemType: types.StringType},
}}

func (r *LifecycleStateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lifecycle_state"
}

func (r *LifecycleStateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a lifecycle state of an identity profile.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Lifecycle state ID",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_profile_id": schema.StringAttribute{
				MarkdownDescription: "ID of the identity profile the lifecycle state belongs to. Changing it forces a new lifecycle state.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Lifecycle state name. The API does not allow renaming, changing it forces a new lifecycle state.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"technical_name": schema.StringAttribute{
				MarkdownDescription: "Technical name of the lifecycle state, used as the value of the `cloudLifecycleState` identity attribute. Changing it forces a new lifecycle state.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Lifecycle state description",
				Optional:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the lifecycle state is enabled. The API defaults it to `false`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"identity_state": schema.StringAttribute{
				MarkdownDescription: "Identity state associated with the lifecycle state: `ACTIVE`, `INACTIVE_SHORT_TERM` or `INACTIVE_LONG_TERM`. The API may assign one when not set. Changing it forces a new lifecycle state.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "Sort order of the lifecycle state, lower numbers first. The API assigns one when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"account_actions_json": schema.StringAttribute{
				MarkdownDescription: "Actions performed on the accounts of identities that enter the lifecycle state, as a JSON array of objects with `action` (`ENABLE`, `DISABLE` or `DELETE`), and `sourceIds`, `excludeSourceIds` or `allSources`.",
				Optional:            true,
				Validators:          []validator.String{jsonArrayStringValidator{}},
			},
			"access_profile_ids": schema.SetAttribute{
				MarkdownDescription: "IDs of the access profiles granted to identities that enter the lifecycle state",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"remove_all_access_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether all access is marked for removal when an identity enters the lifecycle state",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"identity_count": schema.Int64Attribute{
				MarkdownDescription: "Number of identities in the lifecycle state",
				Computed:            true,
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Creation date",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modified": schema.StringAttribute{
				MarkdownDescription: "Last modification date",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"email_notification_option": schema.ListNestedBlock{
				MarkdownDescription: "Email notifications sent when an identity enters the lifecycle state. Without the block no notifications are sent.",
				Validators:          []validator.List{listSizeBetween(0, 1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"notify_managers": schema.BoolAttribute{
							MarkdownDescription: "Notify the manager of the identity",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"notify_all_admins": schema.BoolAttribute{
							MarkdownDescription: "Notify all admins",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"notify_specific_users": schema.BoolAttribute{
							MarkdownDescription: "Notify the users in `email_address_list`",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"email_address_list": schema.ListAttribute{
							MarkdownDescription: "Email addresses notified when `notify_specific_users` is true",
							Optional:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *LifecycleStateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// lifecycleStateEmailValue converts the email block to the API option. Without a block all
// notifications are disabled.
func lifecycleStateEmailValue(ctx context.Context, list types.List, diags *diag.Diagnostics) *LifecycleStateEmailNotificationOption {
	option := &LifecycleStateEmailNotificationOption{EmailAddressList: []string{}}
	if list.IsNull() || list.IsUnknown() {
		return option
	}
	var models []LifecycleStateEmailNotificationModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if len(models) == 0 {
		return option
	}
	option.NotifyManagers = models[0].NotifyManagers.ValueBool()
	option.NotifyAllAdmins = models[0].NotifyAllAdmins.ValueBool()
	option.NotifySpecificUsers = models[0].NotifySpecificUsers.ValueBool()
	if !models[0].EmailAddressList.IsNull() && !models[0].EmailAddressList.IsUnknown() {
		diags.Append(models[0].EmailAddressList.ElementsAs(ctx, &option.EmailAddressList, false)...)
	}
	return option
}

func lifecycleStateEmailIsDefault(option *LifecycleStateEmailNotificationOption) bool {
	return option == nil || (!option.NotifyManagers && !option.NotifyAllAdmins && !option.NotifySpecificUsers && len(option.EmailAddressList) == 0)
}

// lifecycleStateEmailState converts the API option to the block. A default option (no
// notifications) is null, unless always is set (data source). An empty address list stays null
// when it was null before.
func lifecycleStateEmailState(ctx context.Context, prior types.List, option *LifecycleStateEmailNotificationOption, always bool, diags *diag.Diagnostics) types.List {
	if !always && lifecycleStateEmailIsDefault(option) && (prior.IsNull() || prior.IsUnknown()) {
		return types.ListNull(lifecycleStateEmailNotificationObjectType)
	}
	if option == nil {
		option = &LifecycleStateEmailNotificationOption{}
	}
	priorAddressesNull := true
	if !prior.IsNull() && !prior.IsUnknown() {
		var models []LifecycleStateEmailNotificationModel
		diags.Append(prior.ElementsAs(ctx, &models, false)...)
		if len(models) > 0 && !models[0].EmailAddressList.IsNull() {
			priorAddressesNull = false
		}
	}
	addresses := types.ListNull(types.StringType)
	if len(option.EmailAddressList) > 0 || !priorAddressesNull || always {
		list, d := types.ListValueFrom(ctx, types.StringType, append([]string{}, option.EmailAddressList...))
		diags.Append(d...)
		addresses = list
	}
	list, d := types.ListValueFrom(ctx, lifecycleStateEmailNotificationObjectType, []LifecycleStateEmailNotificationModel{{
		NotifyManagers:      types.BoolValue(option.NotifyManagers),
		NotifyAllAdmins:     types.BoolValue(option.NotifyAllAdmins),
		NotifySpecificUsers: types.BoolValue(option.NotifySpecificUsers),
		EmailAddressList:    addresses,
	}})
	diags.Append(d...)
	return list
}

func lifecycleStateAccessProfileIDs(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	ids := []string{}
	if set.IsNull() || set.IsUnknown() {
		return ids
	}
	diags.Append(set.ElementsAs(ctx, &ids, false)...)
	return ids
}

func lifecycleStateAccountActions(value types.String) json.RawMessage {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return json.RawMessage("[]")
	}
	return json.RawMessage(value.ValueString())
}

// lifecycleStateFromModel builds the create request from the plan.
func lifecycleStateFromModel(ctx context.Context, data LifecycleStateModel, diags *diag.Diagnostics) *LifecycleState {
	state := &LifecycleState{
		Name:          data.Name.ValueString(),
		TechnicalName: data.TechnicalName.ValueString(),
		Description:   stringPointer(data.Description),
		Enabled:       boolPointer(data.Enabled),
		IdentityState: stringPointer(data.IdentityState),
		Priority:      int64Pointer(data.Priority),
	}
	if !data.EmailNotificationOption.IsNull() {
		state.EmailNotificationOption = lifecycleStateEmailValue(ctx, data.EmailNotificationOption, diags)
	}
	if !data.AccountActionsJSON.IsNull() {
		state.AccountActions = lifecycleStateAccountActions(data.AccountActionsJSON)
	}
	if !data.AccessProfileIDs.IsNull() {
		state.AccessProfileIDs = lifecycleStateAccessProfileIDs(ctx, data.AccessProfileIDs, diags)
	}
	if remove := boolPointer(data.RemoveAllAccessEnabled); remove != nil {
		state.AccessActionConfiguration = &LifecycleStateAccessActionConfiguration{RemoveAllAccessEnabled: *remove}
	}
	return state
}

func lifecycleStateSetComputed(data *LifecycleStateModel, state *LifecycleState) {
	data.ID = types.StringValue(state.ID)
	count := int64(0)
	if state.IdentityCount != nil {
		count = *state.IdentityCount
	}
	data.IdentityCount = types.Int64Value(count)
	data.Created = types.StringValue(state.Created)
	data.Modified = types.StringValue(state.Modified)
	if data.IdentityState.IsUnknown() {
		identityState := ""
		if state.IdentityState != nil {
			identityState = *state.IdentityState
		}
		data.IdentityState = stringValueOrNull(identityState)
	}
}

func lifecycleStateRemoveAllAccess(state *LifecycleState) *bool {
	value := state.AccessActionConfiguration != nil && state.AccessActionConfiguration.RemoveAllAccessEnabled
	return &value
}

// lifecycleStateResolveApply keeps the planned values after create or update and resolves the
// unknown ones from the API response.
func lifecycleStateResolveApply(data *LifecycleStateModel, state *LifecycleState) {
	lifecycleStateSetComputed(data, state)
	data.Enabled = computedBoolFromAPI(data.Enabled, state.Enabled)
	data.RemoveAllAccessEnabled = computedBoolFromAPI(data.RemoveAllAccessEnabled, lifecycleStateRemoveAllAccess(state))
	if data.Priority.IsUnknown() {
		data.Priority = types.Int64Null()
		if state.Priority != nil {
			data.Priority = types.Int64Value(*state.Priority)
		}
	}
}

// setLifecycleStateState refreshes the model from the API, keeping null for unset optional
// attributes and semantically equal JSON. always returns empty values too (data source).
func setLifecycleStateState(ctx context.Context, data *LifecycleStateModel, state *LifecycleState, always bool, diags *diag.Diagnostics) {
	lifecycleStateSetComputed(data, state)
	data.Name = types.StringValue(state.Name)
	data.TechnicalName = types.StringValue(state.TechnicalName)
	description, identityState := "", ""
	if state.Description != nil {
		description = *state.Description
	}
	if state.IdentityState != nil {
		identityState = *state.IdentityState
	}
	if always {
		data.Description = types.StringValue(description)
		data.IdentityState = stringValueOrNull(identityState)
	} else {
		data.Description = optionalStringState(data.Description, description)
		data.IdentityState = stringValueOrNull(identityState)
	}
	data.Enabled = types.BoolValue(state.Enabled != nil && *state.Enabled)
	data.RemoveAllAccessEnabled = types.BoolValue(*lifecycleStateRemoveAllAccess(state))
	data.Priority = types.Int64Null()
	if state.Priority != nil {
		data.Priority = types.Int64Value(*state.Priority)
	}
	data.EmailNotificationOption = lifecycleStateEmailState(ctx, data.EmailNotificationOption, state.EmailNotificationOption, always, diags)
	data.AccountActionsJSON = jsonSubsetStateRaw(data.AccountActionsJSON, state.AccountActions)
	if len(state.AccessProfileIDs) == 0 && data.AccessProfileIDs.IsNull() && !always {
		data.AccessProfileIDs = types.SetNull(types.StringType)
	} else {
		set, d := types.SetValueFrom(ctx, types.StringType, append([]string{}, state.AccessProfileIDs...))
		diags.Append(d...)
		data.AccessProfileIDs = set
	}
}

// lifecycleStatePatches returns the JSON Patch operations for the changed updatable attributes.
func lifecycleStatePatches(ctx context.Context, plan, state LifecycleStateModel, diags *diag.Diagnostics) []jsonPatchOp {
	b := &patchBuilder{}
	b.replaceOrRemoveIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	b.replaceIfChanged(plan.Enabled, state.Enabled, "/enabled", plan.Enabled.ValueBool())
	if !plan.Priority.IsNull() {
		b.replaceIfChanged(plan.Priority, state.Priority, "/priority", plan.Priority.ValueInt64())
	}
	b.replaceIfChanged(plan.EmailNotificationOption, state.EmailNotificationOption, "/emailNotificationOption", lifecycleStateEmailValue(ctx, plan.EmailNotificationOption, diags))
	if identityProfileJSONChanged(plan.AccountActionsJSON, state.AccountActionsJSON) {
		b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/accountActions", Value: lifecycleStateAccountActions(plan.AccountActionsJSON)})
	}
	b.replaceIfChanged(plan.AccessProfileIDs, state.AccessProfileIDs, "/accessProfileIds", lifecycleStateAccessProfileIDs(ctx, plan.AccessProfileIDs, diags))
	b.replaceIfChanged(plan.RemoveAllAccessEnabled, state.RemoveAllAccessEnabled, "/accessActionConfiguration",
		&LifecycleStateAccessActionConfiguration{RemoveAllAccessEnabled: plan.RemoveAllAccessEnabled.ValueBool()})
	return b.ops
}

func (r *LifecycleStateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LifecycleStateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := lifecycleStateFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateLifecycleState(ctx, data.IdentityProfileID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create lifecycle state: %s", err))
		return
	}
	lifecycleStateResolveApply(&data, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LifecycleStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LifecycleStateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	state, err := client.GetLifecycleState(ctx, data.IdentityProfileID.ValueString(), data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read lifecycle state: %s", err))
		return
	}
	setLifecycleStateState(ctx, &data, state, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LifecycleStateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, prior LifecycleStateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := lifecycleStatePatches(ctx, data, prior, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var updated *LifecycleState
	if len(ops) > 0 {
		updated, err = client.UpdateLifecycleState(ctx, data.IdentityProfileID.ValueString(), data.ID.ValueString(), ops)
	} else {
		updated, err = client.GetLifecycleState(ctx, data.IdentityProfileID.ValueString(), data.ID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update lifecycle state: %s", err))
		return
	}
	lifecycleStateResolveApply(&data, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LifecycleStateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LifecycleStateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteLifecycleState(ctx, data.IdentityProfileID.ValueString(), data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete lifecycle state: %s", err))
	}
}

func (r *LifecycleStateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	profileID, id, ok := strings.Cut(req.ID, "/")
	if !ok || profileID == "" || id == "" || strings.Contains(id, "/") {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected <identity_profile_id>/<lifecycle_state_id>, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("identity_profile_id"), profileID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

var _ datasource.DataSource = &LifecycleStateDataSource{}

func NewLifecycleStateDataSource() datasource.DataSource {
	return &LifecycleStateDataSource{}
}

type LifecycleStateDataSource struct {
	client *Config
}

func (d *LifecycleStateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lifecycle_state"
}

func (d *LifecycleStateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a lifecycle state of an identity profile by ID.",
		Attributes: map[string]dsschema.Attribute{
			"id":                  dsschema.StringAttribute{MarkdownDescription: "Lifecycle state ID", Required: true},
			"identity_profile_id": dsschema.StringAttribute{MarkdownDescription: "ID of the identity profile the lifecycle state belongs to", Required: true},
			"name":                dsschema.StringAttribute{MarkdownDescription: "Lifecycle state name", Computed: true},
			"technical_name":      dsschema.StringAttribute{MarkdownDescription: "Technical name of the lifecycle state", Computed: true},
			"description":         dsschema.StringAttribute{MarkdownDescription: "Lifecycle state description", Computed: true},
			"enabled":             dsschema.BoolAttribute{MarkdownDescription: "Whether the lifecycle state is enabled", Computed: true},
			"identity_state":      dsschema.StringAttribute{MarkdownDescription: "Identity state associated with the lifecycle state", Computed: true},
			"priority":            dsschema.Int64Attribute{MarkdownDescription: "Sort order of the lifecycle state", Computed: true},
			"email_notification_option": dsschema.ListNestedAttribute{
				MarkdownDescription: "Email notifications sent when an identity enters the lifecycle state",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"notify_managers":       dsschema.BoolAttribute{MarkdownDescription: "Notify the manager of the identity", Computed: true},
						"notify_all_admins":     dsschema.BoolAttribute{MarkdownDescription: "Notify all admins", Computed: true},
						"notify_specific_users": dsschema.BoolAttribute{MarkdownDescription: "Notify the users in `email_address_list`", Computed: true},
						"email_address_list":    dsschema.ListAttribute{MarkdownDescription: "Email addresses to notify", Computed: true, ElementType: types.StringType},
					},
				},
			},
			"account_actions_json":      dsschema.StringAttribute{MarkdownDescription: "Account actions as a JSON array", Computed: true},
			"access_profile_ids":        dsschema.SetAttribute{MarkdownDescription: "IDs of the access profiles granted in the lifecycle state", Computed: true, ElementType: types.StringType},
			"remove_all_access_enabled": dsschema.BoolAttribute{MarkdownDescription: "Whether all access is marked for removal", Computed: true},
			"identity_count":            dsschema.Int64Attribute{MarkdownDescription: "Number of identities in the lifecycle state", Computed: true},
			"created":                   dsschema.StringAttribute{MarkdownDescription: "Creation date", Computed: true},
			"modified":                  dsschema.StringAttribute{MarkdownDescription: "Last modification date", Computed: true},
		},
	}
}

func (d *LifecycleStateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LifecycleStateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LifecycleStateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	state, err := client.GetLifecycleState(ctx, data.IdentityProfileID.ValueString(), data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("id"), "Lifecycle state not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read lifecycle state: %s", err))
		return
	}
	setLifecycleStateState(ctx, &data, state, true, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
