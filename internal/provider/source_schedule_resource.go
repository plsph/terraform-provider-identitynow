package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewSourceScheduleResource)
	registerDataSource(NewSourceScheduleDataSource)
}

// sourceScheduleTypes are the schedule types documented by the API.
var sourceScheduleTypes = []string{"ACCOUNT_AGGREGATION", "GROUP_AGGREGATION"}

// SourceSchedule is a source schedule as used by the /v2026/sources/{sourceId}/schedules API.
type SourceSchedule struct {
	Type           string `json:"type"`
	CronExpression string `json:"cronExpression"`
}

func (c *Client) GetSourceSchedule(ctx context.Context, sourceID, scheduleType string) (*SourceSchedule, error) {
	var schedule SourceSchedule
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/sources/%s/schedules/%s", sourceID, scheduleType), nil, &schedule); err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (c *Client) CreateSourceSchedule(ctx context.Context, sourceID string, schedule *SourceSchedule) (*SourceSchedule, error) {
	var created SourceSchedule
	if err := c.doJSON(ctx, http.MethodPost, apiPath("/v2026/sources/%s/schedules", sourceID), schedule, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchSourceSchedule(ctx context.Context, sourceID, scheduleType string, ops []jsonPatchOp) (*SourceSchedule, error) {
	var updated SourceSchedule
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/sources/%s/schedules/%s", sourceID, scheduleType), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteSourceSchedule(ctx context.Context, sourceID, scheduleType string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/sources/%s/schedules/%s", sourceID, scheduleType), nil, nil)
}

// sourceScheduleTypeValidator checks that the schedule type is one of sourceScheduleTypes.
type sourceScheduleTypeValidator struct{}

func (v sourceScheduleTypeValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be one of %s", strings.Join(sourceScheduleTypes, ", "))
}

func (v sourceScheduleTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v sourceScheduleTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, allowed := range sourceScheduleTypes {
		if req.ConfigValue.ValueString() == allowed {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid schedule type", fmt.Sprintf("%s, got %q.", v.Description(ctx), req.ConfigValue.ValueString()))
}

var _ resource.Resource = &SourceScheduleResource{}
var _ resource.ResourceWithImportState = &SourceScheduleResource{}

func NewSourceScheduleResource() resource.Resource {
	return &SourceScheduleResource{}
}

type SourceScheduleResource struct {
	client *Config
}

type SourceScheduleModel struct {
	ID             types.String `tfsdk:"id"`
	SourceID       types.String `tfsdk:"source_id"`
	Type           types.String `tfsdk:"type"`
	CronExpression types.String `tfsdk:"cron_expression"`
}

func (r *SourceScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_schedule"
}

func (r *SourceScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an aggregation schedule of a source with the v2026 source schedules API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource ID in the format `<source_id>/<type>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source_id": schema.StringAttribute{
				MarkdownDescription: "ID of the source. Changing this forces a new schedule to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Schedule type, `ACCOUNT_AGGREGATION` or `GROUP_AGGREGATION`. The type cannot be changed, changing this forces a new schedule to be created.",
				Required:            true,
				Validators:          []validator.String{sourceScheduleTypeValidator{}},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cron_expression": schema.StringAttribute{
				MarkdownDescription: "Cron expression of the schedule, e.g. `0 0 12 1/1 * ? *` for every day at 12:00. Days of the week are 1-7 (Sunday-Saturday).",
				Required:            true,
			},
		},
	}
}

func (r *SourceScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func sourceScheduleID(sourceID, scheduleType string) string {
	return sourceID + "/" + scheduleType
}

// setSourceScheduleState maps an API schedule onto the model during refresh.
func setSourceScheduleState(data *SourceScheduleModel, schedule *SourceSchedule) {
	if schedule.Type != "" {
		data.Type = types.StringValue(schedule.Type)
	}
	data.ID = types.StringValue(sourceScheduleID(data.SourceID.ValueString(), data.Type.ValueString()))
	data.CronExpression = types.StringValue(schedule.CronExpression)
}

// sourceSchedulePatch returns the JSON Patch operations for the changed attributes. The type is
// immutable, so only the cron expression is patched.
func sourceSchedulePatch(plan, state SourceScheduleModel) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.CronExpression, state.CronExpression, "/cronExpression", plan.CronExpression.ValueString())
	return b.ops
}

func (r *SourceScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SourceScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	schedule := &SourceSchedule{Type: data.Type.ValueString(), CronExpression: data.CronExpression.ValueString()}
	if _, err := client.CreateSourceSchedule(ctx, data.SourceID.ValueString(), schedule); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create source schedule: %s", err))
		return
	}
	data.ID = types.StringValue(sourceScheduleID(data.SourceID.ValueString(), data.Type.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SourceScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	schedule, err := client.GetSourceSchedule(ctx, data.SourceID.ValueString(), data.Type.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source schedule: %s", err))
		return
	}
	setSourceScheduleState(&data, schedule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state SourceScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if ops := sourceSchedulePatch(data, state); len(ops) > 0 {
		client, err := r.client.IdentityNowClient(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		if _, err := client.PatchSourceSchedule(ctx, data.SourceID.ValueString(), data.Type.ValueString(), ops); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update source schedule: %s", err))
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SourceScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteSourceSchedule(ctx, data.SourceID.ValueString(), data.Type.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete source schedule: %s", err))
	}
}

func (r *SourceScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	sourceID, scheduleType, ok := strings.Cut(req.ID, "/")
	if !ok || sourceID == "" || scheduleType == "" || strings.Contains(scheduleType, "/") {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected <source_id>/<type>, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source_id"), sourceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), scheduleType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), sourceScheduleID(sourceID, scheduleType))...)
}

var _ datasource.DataSource = &SourceScheduleDataSource{}

func NewSourceScheduleDataSource() datasource.DataSource {
	return &SourceScheduleDataSource{}
}

type SourceScheduleDataSource struct {
	client *Config
}

func (d *SourceScheduleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_schedule"
}

func (d *SourceScheduleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up an aggregation schedule of a source by type.",
		Attributes: map[string]dsschema.Attribute{
			"id":              dsschema.StringAttribute{MarkdownDescription: "ID in the format `<source_id>/<type>`.", Computed: true},
			"source_id":       dsschema.StringAttribute{MarkdownDescription: "ID of the source.", Required: true},
			"type":            dsschema.StringAttribute{MarkdownDescription: "Schedule type, `ACCOUNT_AGGREGATION` or `GROUP_AGGREGATION`.", Required: true, Validators: []validator.String{sourceScheduleTypeValidator{}}},
			"cron_expression": dsschema.StringAttribute{MarkdownDescription: "Cron expression of the schedule.", Computed: true},
		},
	}
}

func (d *SourceScheduleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourceScheduleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SourceScheduleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	schedule, err := client.GetSourceSchedule(ctx, data.SourceID.ValueString(), data.Type.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("type"), "Source schedule not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source schedule: %s", err))
		return
	}
	setSourceScheduleState(&data, schedule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
