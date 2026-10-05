package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewDasTaskScheduleResource)
}

// DasTaskScheduleRequest is the body to create or update a Data Access Security task schedule.
type DasTaskScheduleRequest struct {
	TaskTypeName           string   `json:"taskTypeName"`
	ScheduleType           string   `json:"scheduleType"`
	Interval               *int64   `json:"interval,omitempty"`
	ScheduleTaskName       *string  `json:"scheduleTaskName,omitempty"`
	StartTime              *int64   `json:"startTime,omitempty"`
	EndTime                *int64   `json:"endTime,omitempty"`
	DaysOfWeek             []string `json:"daysOfWeek,omitempty"`
	Active                 bool     `json:"active"`
	RunAfterScheduleTaskID *int64   `json:"runAfterScheduleTaskId,omitempty"`
	ApplicationID          *int64   `json:"applicationId,omitempty"`
}

// DasTaskSchedule is a Data Access Security task schedule as returned by the /v2026/das/tasks/schedules API.
type DasTaskSchedule struct {
	ScheduleTaskID           int64    `json:"scheduleTaskId"`
	ScheduleTaskName         string   `json:"scheduleTaskName"`
	TaskTypeName             string   `json:"taskTypeName"`
	Interval                 *int64   `json:"interval"`
	ScheduleType             string   `json:"scheduleType"`
	Active                   bool     `json:"active"`
	StartTime                *int64   `json:"startTime"`
	EndTime                  *int64   `json:"endTime"`
	DaysOfWeek               []string `json:"daysOfWeek"`
	RunAfterScheduleTaskID   *int64   `json:"runAfterScheduleTaskId"`
	RunAfterScheduleTaskName string   `json:"runAfterScheduleTaskName"`
	ApplicationID            *int64   `json:"applicationId"`
	CreatedByDisplayName     string   `json:"createdByDisplayName"`
	NextRun                  *int64   `json:"nextRun"`
	LastRun                  *int64   `json:"lastRun"`
}

func (c *Client) GetDasTaskSchedule(ctx context.Context, id string) (*DasTaskSchedule, error) {
	var schedule DasTaskSchedule
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/das/tasks/schedules/%s", id), nil, &schedule); err != nil {
		return nil, err
	}
	return &schedule, nil
}

// CreateDasTaskSchedule creates a schedule and returns its ID.
func (c *Client) CreateDasTaskSchedule(ctx context.Context, request *DasTaskScheduleRequest) (int64, error) {
	var id int64
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/das/tasks/schedules", request, &id); err != nil {
		return 0, err
	}
	return id, nil
}

func (c *Client) UpdateDasTaskSchedule(ctx context.Context, id string, request *DasTaskScheduleRequest) error {
	return c.doJSON(ctx, http.MethodPut, apiPath("/v2026/das/tasks/schedules/%s", id), request, nil)
}

func (c *Client) DeleteDasTaskSchedule(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/das/tasks/schedules/%s", id), nil, nil)
}

var _ resource.Resource = &DasTaskScheduleResource{}
var _ resource.ResourceWithImportState = &DasTaskScheduleResource{}

func NewDasTaskScheduleResource() resource.Resource {
	return &DasTaskScheduleResource{}
}

type DasTaskScheduleResource struct {
	client *Config
}

type DasTaskScheduleModel struct {
	ID                       types.String `tfsdk:"id"`
	TaskTypeName             types.String `tfsdk:"task_type_name"`
	ScheduleType             types.String `tfsdk:"schedule_type"`
	ScheduleTaskName         types.String `tfsdk:"schedule_task_name"`
	Interval                 types.Int64  `tfsdk:"interval"`
	StartTime                types.Int64  `tfsdk:"start_time"`
	EndTime                  types.Int64  `tfsdk:"end_time"`
	DaysOfWeek               types.List   `tfsdk:"days_of_week"`
	Active                   types.Bool   `tfsdk:"active"`
	RunAfterScheduleTaskID   types.Int64  `tfsdk:"run_after_schedule_task_id"`
	ApplicationID            types.Int64  `tfsdk:"application_id"`
	RunAfterScheduleTaskName types.String `tfsdk:"run_after_schedule_task_name"`
	CreatedByDisplayName     types.String `tfsdk:"created_by_display_name"`
	NextRun                  types.Int64  `tfsdk:"next_run"`
	LastRun                  types.Int64  `tfsdk:"last_run"`
}

func (r *DasTaskScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_das_task_schedule"
}

func (r *DasTaskScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Data Access Security task schedule, which runs a task such as a crawl or permission collection on a recurring schedule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Schedule ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"task_type_name": schema.StringAttribute{
				MarkdownDescription: "Type of the scheduled task.",
				Required:            true,
			},
			"schedule_type": schema.StringAttribute{
				MarkdownDescription: "Schedule cycle, e.g. `Daily`, `Weekly` or `Manual`.",
				Required:            true,
			},
			"schedule_task_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the scheduled task.",
				Optional:            true,
			},
			"interval": schema.Int64Attribute{
				MarkdownDescription: "Interval between runs in units of the schedule cycle, e.g. days for a daily schedule.",
				Optional:            true,
			},
			"start_time": schema.Int64Attribute{
				MarkdownDescription: "Start time of the schedule, in seconds since the epoch. When not set, the value returned by the API is stored and kept: the API always has a start time, so removing the argument keeps the current start time.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"end_time": schema.Int64Attribute{
				MarkdownDescription: "End time of the schedule, in seconds since the epoch.",
				Optional:            true,
			},
			"days_of_week": schema.ListAttribute{
				MarkdownDescription: "Days of the week the task runs on, e.g. `Monday`.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the schedule is active. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"run_after_schedule_task_id": schema.Int64Attribute{
				MarkdownDescription: "ID of another schedule whose completion triggers this task.",
				Optional:            true,
			},
			"application_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the Data Access Security application of the task, e.g. `identitynow_das_application.example.id`.",
				Optional:            true,
			},
			"run_after_schedule_task_name": schema.StringAttribute{
				MarkdownDescription: "Name of the schedule whose completion triggers this task.",
				Computed:            true,
			},
			"created_by_display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the user who created the schedule.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"next_run": schema.Int64Attribute{
				MarkdownDescription: "Next run time, in seconds since the epoch.",
				Computed:            true,
			},
			"last_run": schema.Int64Attribute{
				MarkdownDescription: "Last run time, in seconds since the epoch.",
				Computed:            true,
			},
		},
	}
}

func (r *DasTaskScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// dasTaskScheduleFromModel converts the model to the create and update request.
func dasTaskScheduleFromModel(ctx context.Context, data DasTaskScheduleModel, diags *diag.Diagnostics) *DasTaskScheduleRequest {
	return &DasTaskScheduleRequest{
		TaskTypeName:           data.TaskTypeName.ValueString(),
		ScheduleType:           data.ScheduleType.ValueString(),
		Interval:               int64Pointer(data.Interval),
		ScheduleTaskName:       stringPointer(data.ScheduleTaskName),
		StartTime:              int64Pointer(data.StartTime),
		EndTime:                int64Pointer(data.EndTime),
		DaysOfWeek:             serviceDeskIntegrationStringList(ctx, data.DaysOfWeek, diags),
		Active:                 data.Active.ValueBool(),
		RunAfterScheduleTaskID: int64Pointer(data.RunAfterScheduleTaskID),
		ApplicationID:          int64Pointer(data.ApplicationID),
	}
}

// dasTaskScheduleInt64FromAPI resolves an optional and computed integer after apply: a known
// planned value is kept, an unknown one is taken from the API.
func dasTaskScheduleInt64FromAPI(planned types.Int64, api *int64) types.Int64 {
	if planned.IsUnknown() {
		return types.Int64PointerValue(api)
	}
	return planned
}

// setDasTaskScheduleState maps the API schedule onto the model. With refresh the configurable
// attributes are refreshed from the API; otherwise the planned values are kept and unknown values
// are resolved.
func setDasTaskScheduleState(ctx context.Context, data *DasTaskScheduleModel, schedule *DasTaskSchedule, refresh bool, diags *diag.Diagnostics) {
	if schedule.ScheduleTaskID != 0 {
		data.ID = types.StringValue(strconv.FormatInt(schedule.ScheduleTaskID, 10))
	}
	data.RunAfterScheduleTaskName = stringValueOrNull(schedule.RunAfterScheduleTaskName)
	data.CreatedByDisplayName = stringValueOrNull(schedule.CreatedByDisplayName)
	data.NextRun = types.Int64PointerValue(schedule.NextRun)
	data.LastRun = types.Int64PointerValue(schedule.LastRun)
	if !refresh {
		data.StartTime = dasTaskScheduleInt64FromAPI(data.StartTime, schedule.StartTime)
		return
	}
	data.TaskTypeName = types.StringValue(schedule.TaskTypeName)
	data.ScheduleType = types.StringValue(schedule.ScheduleType)
	data.ScheduleTaskName = optionalStringState(data.ScheduleTaskName, schedule.ScheduleTaskName)
	data.Interval = optionalInt64State(data.Interval, schedule.Interval)
	data.StartTime = types.Int64PointerValue(schedule.StartTime)
	data.EndTime = optionalInt64State(data.EndTime, schedule.EndTime)
	data.DaysOfWeek = serviceDeskIntegrationStringListState(ctx, data.DaysOfWeek, schedule.DaysOfWeek, diags)
	data.Active = types.BoolValue(schedule.Active)
	data.RunAfterScheduleTaskID = optionalInt64State(data.RunAfterScheduleTaskID, schedule.RunAfterScheduleTaskID)
	data.ApplicationID = optionalInt64State(data.ApplicationID, schedule.ApplicationID)
}

func (r *DasTaskScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DasTaskScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request := dasTaskScheduleFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	id, err := client.CreateDasTaskSchedule(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Data Access Security task schedule: %s", err))
		return
	}
	// The create response only contains the ID, the computed attributes are read back.
	schedule, err := client.GetDasTaskSchedule(ctx, strconv.FormatInt(id, 10))
	if err != nil {
		// Keep the created schedule in state, so it is not orphaned; the next refresh completes it.
		setDasTaskScheduleState(ctx, &data, &DasTaskSchedule{ScheduleTaskID: id}, false, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Created Data Access Security task schedule %d, but reading it failed: %s", id, err))
		return
	}
	data.ID = types.StringValue(strconv.FormatInt(id, 10))
	setDasTaskScheduleState(ctx, &data, schedule, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DasTaskScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DasTaskScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	schedule, err := client.GetDasTaskSchedule(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Data Access Security task schedule: %s", err))
		return
	}
	setDasTaskScheduleState(ctx, &data, schedule, true, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DasTaskScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DasTaskScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request := dasTaskScheduleFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// The model covers the whole request body, so PUT sends the complete planned schedule.
	if err := client.UpdateDasTaskSchedule(ctx, data.ID.ValueString(), request); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Data Access Security task schedule: %s", err))
		return
	}
	// PUT responds without content, the computed attributes are read back.
	schedule, err := client.GetDasTaskSchedule(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Data Access Security task schedule: %s", err))
		return
	}
	setDasTaskScheduleState(ctx, &data, schedule, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DasTaskScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DasTaskScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteDasTaskSchedule(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Data Access Security task schedule: %s", err))
	}
}

func (r *DasTaskScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.ParseInt(req.ID, 10, 64); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected a numeric schedule ID, got %q.", req.ID))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
