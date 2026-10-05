package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	registerResource(NewCampaignTemplateScheduleResource)
}

// CampaignTemplateSchedule is the schedule of a campaign template, /v2026/campaign-templates/{id}/schedule.
type CampaignTemplateSchedule struct {
	Type       string                            `json:"type"`
	Months     *CampaignTemplateScheduleSelector `json:"months,omitempty"`
	Days       *CampaignTemplateScheduleSelector `json:"days,omitempty"`
	Hours      *CampaignTemplateScheduleSelector `json:"hours,omitempty"`
	Expiration *string                           `json:"expiration,omitempty"`
	TimeZoneID *string                           `json:"timeZoneId,omitempty"`
}

// CampaignTemplateScheduleSelector selects the months, days or hours in which a schedule is active.
type CampaignTemplateScheduleSelector struct {
	Type     string   `json:"type"`
	Values   []string `json:"values"`
	Interval *int64   `json:"interval,omitempty"`
}

func (c *Client) GetCampaignTemplateSchedule(ctx context.Context, templateID string) (*CampaignTemplateSchedule, error) {
	var schedule CampaignTemplateSchedule
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/campaign-templates/%s/schedule", templateID), nil, &schedule); err != nil {
		return nil, err
	}
	return &schedule, nil
}

// SetCampaignTemplateSchedule creates or overwrites the schedule. The API responds without a body.
func (c *Client) SetCampaignTemplateSchedule(ctx context.Context, templateID string, schedule *CampaignTemplateSchedule) error {
	return c.doJSON(ctx, http.MethodPut, apiPath("/v2026/campaign-templates/%s/schedule", templateID), schedule, nil)
}

func (c *Client) DeleteCampaignTemplateSchedule(ctx context.Context, templateID string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/campaign-templates/%s/schedule", templateID), nil, nil)
}

var _ resource.Resource = &CampaignTemplateScheduleResource{}
var _ resource.ResourceWithImportState = &CampaignTemplateScheduleResource{}

func NewCampaignTemplateScheduleResource() resource.Resource {
	return &CampaignTemplateScheduleResource{}
}

type CampaignTemplateScheduleResource struct {
	client *Config
}

type CampaignTemplateScheduleModel struct {
	ID                 types.String `tfsdk:"id"`
	CampaignTemplateID types.String `tfsdk:"campaign_template_id"`
	Type               types.String `tfsdk:"type"`
	Months             types.List   `tfsdk:"months"`
	Days               types.List   `tfsdk:"days"`
	Hours              types.List   `tfsdk:"hours"`
	Expiration         types.String `tfsdk:"expiration"`
	TimeZoneID         types.String `tfsdk:"time_zone_id"`
}

type CampaignTemplateScheduleSelectorModel struct {
	Type     types.String `tfsdk:"type"`
	Values   types.List   `tfsdk:"values"`
	Interval types.Int64  `tfsdk:"interval"`
}

var campaignTemplateScheduleSelectorObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"type":     types.StringType,
	"values":   types.ListType{ElemType: types.StringType},
	"interval": types.Int64Type,
}}

func (r *CampaignTemplateScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign_template_schedule"
}

func campaignTemplateScheduleSelectorBlock(description string, min int) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: description,
		Validators:          []validator.List{listSizeBetween(min, 1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					MarkdownDescription: "`LIST` (distinct values) or `RANGE` (two values, the inclusive start and end of the range).",
					Required:            true,
				},
				"values": schema.ListAttribute{
					MarkdownDescription: "Selected values, as strings.",
					Required:            true,
					ElementType:         types.StringType,
				},
				"interval": schema.Int64Attribute{
					MarkdownDescription: "Interval between the selected values, for example `3` with hour `8` runs every three hours from 8 AM.",
					Optional:            true,
				},
			},
		},
	}
}

func (r *CampaignTemplateScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the schedule that generates campaigns from a certification campaign template. A template has at most one schedule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Campaign template ID, the schedule has no ID of its own.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"campaign_template_id": schema.StringAttribute{
				MarkdownDescription: "ID of the scheduled campaign template. Changing this forces a new schedule to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Schedule cadence: `WEEKLY`, `MONTHLY`, `ANNUALLY` or `CALENDAR`. All periods smaller than the cadence can be selected.",
				Required:            true,
			},
			"expiration": schema.StringAttribute{
				MarkdownDescription: "Date and time (ISO-8601) after which the schedule no longer runs.",
				Optional:            true,
			},
			"time_zone_id": schema.StringAttribute{
				MarkdownDescription: "Time zone the schedule runs in, such as `America/New_York`. When not set, the value chosen by IdentityNow is kept.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"months": campaignTemplateScheduleSelectorBlock("Active months (`1`-`12`), only valid for `ANNUALLY` schedules. At most one block.", 0),
			"days":   campaignTemplateScheduleSelectorBlock("Active days: days of the week (`1`-`7`) for `WEEKLY`, days of the month (`1`-`31`, `L` for the last day) for `MONTHLY` and `ANNUALLY`, ISO-8601 dates for `CALENDAR`. At most one block.", 0),
			"hours":  campaignTemplateScheduleSelectorBlock("Active hours (`0`-`23`). Exactly one block is required.", 1),
		},
	}
}

func (r *CampaignTemplateScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func campaignTemplateScheduleSelectorValue(ctx context.Context, list types.List, diags *diag.Diagnostics) *CampaignTemplateScheduleSelector {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil
	}
	var models []CampaignTemplateScheduleSelectorModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if len(models) == 0 {
		return nil
	}
	selector := &CampaignTemplateScheduleSelector{Type: models[0].Type.ValueString(), Values: []string{}, Interval: int64Pointer(models[0].Interval)}
	if !models[0].Values.IsNull() && !models[0].Values.IsUnknown() {
		diags.Append(models[0].Values.ElementsAs(ctx, &selector.Values, false)...)
	}
	return selector
}

// campaignTemplateScheduleFromModel converts the model to the API request body.
func campaignTemplateScheduleFromModel(ctx context.Context, data CampaignTemplateScheduleModel, diags *diag.Diagnostics) *CampaignTemplateSchedule {
	return &CampaignTemplateSchedule{
		Type:       data.Type.ValueString(),
		Months:     campaignTemplateScheduleSelectorValue(ctx, data.Months, diags),
		Days:       campaignTemplateScheduleSelectorValue(ctx, data.Days, diags),
		Hours:      campaignTemplateScheduleSelectorValue(ctx, data.Hours, diags),
		Expiration: stringPointer(data.Expiration),
		TimeZoneID: stringPointer(data.TimeZoneID),
	}
}

// campaignTemplateScheduleSelectorState maps an API selector to a single-element list. The interval
// stays null when it is not configured and the API returns none or zero.
func campaignTemplateScheduleSelectorState(ctx context.Context, selector *CampaignTemplateScheduleSelector, prior types.List, diags *diag.Diagnostics) types.List {
	if selector == nil {
		return types.ListNull(campaignTemplateScheduleSelectorObjectType)
	}
	priorInterval := types.Int64Null()
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorModels []CampaignTemplateScheduleSelectorModel
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
		if len(priorModels) > 0 {
			priorInterval = priorModels[0].Interval
		}
	}
	values, d := types.ListValueFrom(ctx, types.StringType, selector.Values)
	diags.Append(d...)
	if selector.Values == nil {
		values, d = types.ListValue(types.StringType, []attr.Value{})
		diags.Append(d...)
	}
	list, d := types.ListValueFrom(ctx, campaignTemplateScheduleSelectorObjectType, []CampaignTemplateScheduleSelectorModel{{
		Type:     types.StringValue(selector.Type),
		Values:   values,
		Interval: optionalInt64State(priorInterval, selector.Interval),
	}})
	diags.Append(d...)
	return list
}

// campaignTemplateScheduleTimeState keeps the prior date-time when the API returns the same instant
// in a different format.
func campaignTemplateScheduleTimeState(prior types.String, value *string) types.String {
	if value == nil || *value == "" {
		return optionalStringState(prior, "")
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		priorTime, err1 := time.Parse(time.RFC3339, prior.ValueString())
		apiTime, err2 := time.Parse(time.RFC3339, *value)
		if err1 == nil && err2 == nil && priorTime.Equal(apiTime) {
			return prior
		}
	}
	return types.StringValue(*value)
}

// setCampaignTemplateScheduleState maps the API schedule onto the model when reading.
func setCampaignTemplateScheduleState(ctx context.Context, data *CampaignTemplateScheduleModel, schedule *CampaignTemplateSchedule, diags *diag.Diagnostics) {
	data.Type = types.StringValue(schedule.Type)
	data.Months = campaignTemplateScheduleSelectorState(ctx, schedule.Months, data.Months, diags)
	data.Days = campaignTemplateScheduleSelectorState(ctx, schedule.Days, data.Days, diags)
	data.Hours = campaignTemplateScheduleSelectorState(ctx, schedule.Hours, data.Hours, diags)
	data.Expiration = campaignTemplateScheduleTimeState(data.Expiration, schedule.Expiration)
	timeZone := ""
	if schedule.TimeZoneID != nil {
		timeZone = *schedule.TimeZoneID
	}
	data.TimeZoneID = types.StringValue(timeZone)
}

// apply sets the planned schedule and resolves the time zone when it is not configured. The PUT
// response has no body, so the time zone chosen by the API is read back.
func (r *CampaignTemplateScheduleResource) apply(ctx context.Context, data *CampaignTemplateScheduleModel, diags *diag.Diagnostics) {
	schedule := campaignTemplateScheduleFromModel(ctx, *data, diags)
	if diags.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		diags.AddError("Client Error", err.Error())
		return
	}
	templateID := data.CampaignTemplateID.ValueString()
	if err := client.SetCampaignTemplateSchedule(ctx, templateID, schedule); err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to set campaign template schedule: %s", err))
		return
	}
	data.ID = types.StringValue(templateID)
	if data.TimeZoneID.IsUnknown() {
		timeZone := ""
		if current, err := client.GetCampaignTemplateSchedule(ctx, templateID); err == nil && current.TimeZoneID != nil {
			timeZone = *current.TimeZoneID
		}
		data.TimeZoneID = types.StringValue(timeZone)
	}
}

func (r *CampaignTemplateScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CampaignTemplateScheduleModel
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

func (r *CampaignTemplateScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CampaignTemplateScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	schedule, err := client.GetCampaignTemplateSchedule(ctx, data.CampaignTemplateID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read campaign template schedule: %s", err))
		return
	}
	data.ID = data.CampaignTemplateID
	setCampaignTemplateScheduleState(ctx, &data, schedule, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CampaignTemplateScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CampaignTemplateScheduleModel
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

func (r *CampaignTemplateScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CampaignTemplateScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteCampaignTemplateSchedule(ctx, data.CampaignTemplateID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete campaign template schedule: %s", err))
	}
}

// ImportState imports a schedule by the ID of its campaign template.
func (r *CampaignTemplateScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected the campaign template ID.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("campaign_template_id"), req.ID)...)
}
