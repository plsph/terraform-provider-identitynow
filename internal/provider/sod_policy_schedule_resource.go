package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewSodPolicyScheduleResource)
}

// SodPolicySchedule is the violation report schedule of a SOD policy, /v2026/sod-policies/{id}/schedule.
type SodPolicySchedule struct {
	Name              string                 `json:"name,omitempty"`
	Created           string                 `json:"created,omitempty"`
	Modified          string                 `json:"modified,omitempty"`
	Description       string                 `json:"description,omitempty"`
	Schedule          map[string]interface{} `json:"schedule"`
	Recipients        []SodPolicyRef         `json:"recipients"`
	EmailEmptyResults *bool                  `json:"emailEmptyResults,omitempty"`
	CreatorID         string                 `json:"creatorId,omitempty"`
	ModifierID        string                 `json:"modifierId,omitempty"`
}

func (c *Client) GetSodPolicySchedule(ctx context.Context, policyID string) (*SodPolicySchedule, error) {
	var schedule SodPolicySchedule
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/sod-policies/%s/schedule", policyID), nil, &schedule); err != nil {
		return nil, err
	}
	return &schedule, nil
}

// SetSodPolicySchedule creates or replaces the schedule of a policy.
func (c *Client) SetSodPolicySchedule(ctx context.Context, policyID string, schedule *SodPolicySchedule) (*SodPolicySchedule, error) {
	var updated SodPolicySchedule
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/sod-policies/%s/schedule", policyID), schedule, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteSodPolicySchedule(ctx context.Context, policyID string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/sod-policies/%s/schedule", policyID), nil, nil)
}

var _ resource.Resource = &SodPolicyScheduleResource{}
var _ resource.ResourceWithImportState = &SodPolicyScheduleResource{}

func NewSodPolicyScheduleResource() resource.Resource {
	return &SodPolicyScheduleResource{}
}

type SodPolicyScheduleResource struct {
	client *Config
}

type SodPolicyScheduleModel struct {
	ID                types.String `tfsdk:"id"`
	PolicyID          types.String `tfsdk:"policy_id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ScheduleJSON      types.String `tfsdk:"schedule_json"`
	Recipient         types.List   `tfsdk:"recipient"`
	EmailEmptyResults types.Bool   `tfsdk:"email_empty_results"`
	CreatorID         types.String `tfsdk:"creator_id"`
	ModifierID        types.String `tfsdk:"modifier_id"`
	Created           types.String `tfsdk:"created"`
	Modified          types.String `tfsdk:"modified"`
}

func (r *SodPolicyScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sod_policy_schedule"
}

func (r *SodPolicyScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the schedule on which the violation report of a SOD policy is run and emailed. A policy has at most one schedule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "SOD policy ID, the schedule has no ID of its own.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"policy_id": schema.StringAttribute{
				MarkdownDescription: "ID of the scheduled SOD policy. Changing this forces a new schedule to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Schedule name.",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Schedule description.",
				Optional:            true,
			},
			"schedule_json": schema.StringAttribute{
				MarkdownDescription: "Schedule as a JSON object with `type` (`DAILY`, `WEEKLY`, `MONTHLY`, `CALENDAR` or `ANNUALLY`), the `months`, `days` and `hours` selectors (each with `type` `LIST` or `RANGE`, `values` and an optional `interval`), `expiration` and `timeZoneId`. `hours` is required. Use `jsonencode()`. The value is compared on the configured fields only, so formatting, key order and fields IdentityNow adds do not produce a diff.",
				Required:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"email_empty_results": schema.BoolAttribute{
				MarkdownDescription: "Whether the report is emailed when it has no results. When not set, the value chosen by IdentityNow is kept.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"creator_id": schema.StringAttribute{
				MarkdownDescription: "ID of the identity that created the schedule.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modifier_id": schema.StringAttribute{
				MarkdownDescription: "ID of the identity that last modified the schedule.",
				Computed:            true,
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
			"recipient": schema.ListNestedBlock{
				MarkdownDescription: "Identity that receives the violation report.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{MarkdownDescription: "Identity ID.", Required: true},
						"type": schema.StringAttribute{
							MarkdownDescription: "Recipient type, `IDENTITY` (default).",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("IDENTITY"),
						},
						"name": schema.StringAttribute{MarkdownDescription: "Identity display name. When omitted, the name resolved by the API is not tracked.", Optional: true},
					},
				},
			},
		},
	}
}

func (r *SodPolicyScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// sodPolicyScheduleFromModel converts the model to the PUT request body, which replaces the schedule.
func sodPolicyScheduleFromModel(ctx context.Context, data SodPolicyScheduleModel, diags *diag.Diagnostics) *SodPolicySchedule {
	schedule := &SodPolicySchedule{
		Name:              data.Name.ValueString(),
		Description:       data.Description.ValueString(),
		Schedule:          map[string]interface{}{},
		Recipients:        []SodPolicyRef{},
		EmailEmptyResults: boolPointer(data.EmailEmptyResults),
	}
	if value, ok := campaignTemplateJSONValue(data.ScheduleJSON, "schedule_json", diags).(map[string]interface{}); ok {
		schedule.Schedule = value
	}
	if !data.Recipient.IsNull() && !data.Recipient.IsUnknown() {
		var recipients []OwnerModel
		diags.Append(data.Recipient.ElementsAs(ctx, &recipients, false)...)
		for _, recipient := range recipients {
			schedule.Recipients = append(schedule.Recipients, SodPolicyRef{
				ID: recipient.ID.ValueString(), Type: recipient.Type.ValueString(), Name: recipient.Name.ValueString(),
			})
		}
	}
	return schedule
}

// sodPolicyScheduleRecipientsState maps the API recipients in the prior order. Names stay null for
// recipients configured without a name.
func sodPolicyScheduleRecipientsState(ctx context.Context, recipients []SodPolicyRef, prior types.List, diags *diag.Diagnostics) types.List {
	var priorModels []OwnerModel
	if !prior.IsNull() && !prior.IsUnknown() {
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
	}
	if len(recipients) == 0 {
		return types.ListNull(objectInfoObjectType)
	}
	priorByID := map[string]OwnerModel{}
	priorRefs := make([]SodPolicyRef, 0, len(priorModels))
	for _, model := range priorModels {
		priorByID[model.ID.ValueString()] = model
		priorRefs = append(priorRefs, SodPolicyRef{ID: model.ID.ValueString()})
	}
	ordered := orderByPriorIDs(recipients, priorRefs, func(r SodPolicyRef) string { return r.ID })
	models := make([]OwnerModel, 0, len(ordered))
	for _, recipient := range ordered {
		name := types.StringValue(recipient.Name)
		if priorModel, ok := priorByID[recipient.ID]; ok && priorModel.Name.IsNull() {
			name = types.StringNull()
		}
		recipientType := recipient.Type
		if recipientType == "" {
			recipientType = "IDENTITY"
		}
		models = append(models, OwnerModel{ID: types.StringValue(recipient.ID), Type: types.StringValue(recipientType), Name: name})
	}
	list, d := types.ListValueFrom(ctx, objectInfoObjectType, models)
	diags.Append(d...)
	return list
}

// setSodPolicyScheduleComputed resolves the computed attributes after apply from the API response.
// Values known from the plan (kept from state) are not changed.
func setSodPolicyScheduleComputed(data *SodPolicyScheduleModel, schedule *SodPolicySchedule) {
	data.ID = data.PolicyID
	data.CreatorID = computedStringFromAPI(data.CreatorID, schedule.CreatorID)
	data.ModifierID = types.StringValue(schedule.ModifierID)
	data.Created = computedStringFromAPI(data.Created, schedule.Created)
	data.Modified = types.StringValue(schedule.Modified)
	data.EmailEmptyResults = computedBoolFromAPI(data.EmailEmptyResults, schedule.EmailEmptyResults)
}

// setSodPolicyScheduleState maps the API schedule onto the model when reading.
func setSodPolicyScheduleState(ctx context.Context, data *SodPolicyScheduleModel, schedule *SodPolicySchedule, diags *diag.Diagnostics) {
	data.ID = data.PolicyID
	data.CreatorID = types.StringValue(schedule.CreatorID)
	data.ModifierID = types.StringValue(schedule.ModifierID)
	data.Created = types.StringValue(schedule.Created)
	data.Modified = types.StringValue(schedule.Modified)
	data.Name = optionalStringState(data.Name, schedule.Name)
	data.Description = optionalStringState(data.Description, schedule.Description)
	data.ScheduleJSON = jsonSubsetState(data.ScheduleJSON, schedule.Schedule)
	data.Recipient = sodPolicyScheduleRecipientsState(ctx, schedule.Recipients, data.Recipient, diags)
	data.EmailEmptyResults = types.BoolValue(schedule.EmailEmptyResults != nil && *schedule.EmailEmptyResults)
}

func (r *SodPolicyScheduleResource) apply(ctx context.Context, data *SodPolicyScheduleModel, diags *diag.Diagnostics) {
	schedule := sodPolicyScheduleFromModel(ctx, *data, diags)
	if diags.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		diags.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.SetSodPolicySchedule(ctx, data.PolicyID.ValueString(), schedule)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to set SOD policy schedule: %s", err))
		return
	}
	setSodPolicyScheduleComputed(data, updated)
}

func (r *SodPolicyScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SodPolicyScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SodPolicyScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SodPolicyScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	schedule, err := client.GetSodPolicySchedule(ctx, data.PolicyID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SOD policy schedule: %s", err))
		return
	}
	setSodPolicyScheduleState(ctx, &data, schedule, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SodPolicyScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SodPolicyScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SodPolicyScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SodPolicyScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteSodPolicySchedule(ctx, data.PolicyID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete SOD policy schedule: %s", err))
	}
}

// ImportState imports a schedule by the ID of its SOD policy.
func (r *SodPolicyScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected the SOD policy ID.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_id"), req.ID)...)
}
