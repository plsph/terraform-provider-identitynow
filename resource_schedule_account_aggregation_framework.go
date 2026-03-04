package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ScheduleAccountAggregationResource{}

func NewScheduleAccountAggregationResource() resource.Resource {
	return &ScheduleAccountAggregationResource{}
}

type ScheduleAccountAggregationResource struct {
	client *Config
}

type ScheduleAccountAggregationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	SourceID        types.String `tfsdk:"source_id"`
	CronExpressions types.List   `tfsdk:"cron_expressions"`
}

func (r *ScheduleAccountAggregationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule_account_aggregation"
}

func (r *ScheduleAccountAggregationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Schedule Account Aggregation resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Schedule ID (same as source_id)",
			},
			"source_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source ID",
			},
			"cron_expressions": schema.ListAttribute{
				Required:            true,
				MarkdownDescription: "Account aggregation scheduling in cron expression format",
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *ScheduleAccountAggregationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScheduleAccountAggregationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ScheduleAccountAggregationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var cronExpressions []string
	resp.Diagnostics.Append(data.CronExpressions.ElementsAs(ctx, &cronExpressions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule := &AccountAggregationSchedule{
		SourceID:        data.SourceID.ValueString(),
		CronExpressions: cronExpressions,
	}

	tflog.Info(ctx, "Creating Account Aggregation Schedule", map[string]interface{}{"source_id": schedule.SourceID})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	newSchedule, err := client.ManageAccountAggregationSchedule(ctx, schedule, true)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create account aggregation schedule: %s", err))
		return
	}

	newSchedule.SourceID = schedule.SourceID
	data.ID = types.StringValue(newSchedule.SourceID)

	cronList, diags := types.ListValueFrom(ctx, types.StringType, newSchedule.CronExpressions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.CronExpressions = cronList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduleAccountAggregationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ScheduleAccountAggregationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Account Aggregation Schedule", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	schedule, err := client.GetAccountAggregationSchedule(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read account aggregation schedule: %s", err))
		return
	}

	if schedule.CronExpressions != nil {
		schedule.SourceID = data.ID.ValueString()
		data.SourceID = types.StringValue(schedule.SourceID)

		cronList, diags := types.ListValueFrom(ctx, types.StringType, schedule.CronExpressions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.CronExpressions = cronList
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduleAccountAggregationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ScheduleAccountAggregationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var cronExpressions []string
	resp.Diagnostics.Append(data.CronExpressions.ElementsAs(ctx, &cronExpressions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule := &AccountAggregationSchedule{
		SourceID:        data.SourceID.ValueString(),
		CronExpressions: cronExpressions,
	}

	tflog.Info(ctx, "Updating Account Aggregation Schedule", map[string]interface{}{"source_id": schedule.SourceID})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	newSchedule, err := client.ManageAccountAggregationSchedule(ctx, schedule, true)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update account aggregation schedule: %s", err))
		return
	}

	newSchedule.SourceID = schedule.SourceID
	data.ID = types.StringValue(newSchedule.SourceID)

	cronList, diags := types.ListValueFrom(ctx, types.StringType, newSchedule.CronExpressions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.CronExpressions = cronList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduleAccountAggregationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ScheduleAccountAggregationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Account Aggregation Schedule", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	schedule, err := client.GetAccountAggregationSchedule(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get account aggregation schedule: %s", err))
		return
	}

	if schedule.CronExpressions != nil {
		schedule.SourceID = data.ID.ValueString()
		_, err = client.ManageAccountAggregationSchedule(ctx, schedule, false)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete account aggregation schedule: %s", err))
			return
		}
	}
}
