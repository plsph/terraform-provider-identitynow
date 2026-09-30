package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewReassignmentConfigurationResource)
}

// ReassignmentConfigurationRequest creates or updates the reassignment of one work type of an identity.
type ReassignmentConfigurationRequest struct {
	ReassignedFromID string  `json:"reassignedFromId"`
	ReassignedToID   string  `json:"reassignedToId"`
	ConfigType       string  `json:"configType"`
	StartDate        string  `json:"startDate,omitempty"`
	EndDate          *string `json:"endDate"`
}

// ReassignmentConfigurationIdentity is an identity reference of the reassignment configuration API.
type ReassignmentConfigurationIdentity struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// ReassignmentConfigurationAuditDetails holds the audit dates of a reassignment.
type ReassignmentConfigurationAuditDetails struct {
	Created  string `json:"created,omitempty"`
	Modified string `json:"modified,omitempty"`
}

// ReassignmentConfigurationDetail is the reassignment of one work type.
type ReassignmentConfigurationDetail struct {
	ConfigType     string                                 `json:"configType"`
	TargetIdentity *ReassignmentConfigurationIdentity     `json:"targetIdentity,omitempty"`
	StartDate      string                                 `json:"startDate,omitempty"`
	EndDate        string                                 `json:"endDate,omitempty"`
	AuditDetails   *ReassignmentConfigurationAuditDetails `json:"auditDetails,omitempty"`
}

// ReassignmentConfiguration is the reassignment configuration of an identity as returned by the
// experimental /v2026/reassignment-configurations API.
type ReassignmentConfiguration struct {
	Identity      *ReassignmentConfigurationIdentity `json:"identity,omitempty"`
	ConfigDetails []ReassignmentConfigurationDetail  `json:"configDetails,omitempty"`
}

// reassignmentConfigurationDetail returns the detail of the given work type, or nil.
func reassignmentConfigurationDetail(config *ReassignmentConfiguration, configType string) *ReassignmentConfigurationDetail {
	if config == nil {
		return nil
	}
	for i := range config.ConfigDetails {
		if config.ConfigDetails[i].ConfigType == configType {
			return &config.ConfigDetails[i]
		}
	}
	return nil
}

func (c *Client) CreateReassignmentConfiguration(ctx context.Context, request *ReassignmentConfigurationRequest) (*ReassignmentConfiguration, error) {
	var created ReassignmentConfiguration
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/reassignment-configurations", request, &created, withExperimental()); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) GetReassignmentConfiguration(ctx context.Context, identityID string) (*ReassignmentConfiguration, error) {
	var config ReassignmentConfiguration
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/reassignment-configurations/%s", identityID), nil, &config, withExperimental()); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Client) UpdateReassignmentConfiguration(ctx context.Context, request *ReassignmentConfigurationRequest) (*ReassignmentConfiguration, error) {
	var updated ReassignmentConfiguration
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/reassignment-configurations/%s", request.ReassignedFromID), request, &updated, withExperimental()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteReassignmentConfiguration(ctx context.Context, identityID, configType string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/reassignment-configurations/%s/%s", identityID, configType), nil, nil, withExperimental())
}

var _ resource.Resource = &ReassignmentConfigurationResource{}
var _ resource.ResourceWithImportState = &ReassignmentConfigurationResource{}

func NewReassignmentConfigurationResource() resource.Resource {
	return &ReassignmentConfigurationResource{}
}

type ReassignmentConfigurationResource struct {
	client *Config
}

type ReassignmentConfigurationModel struct {
	ID             types.String `tfsdk:"id"`
	IdentityID     types.String `tfsdk:"identity_id"`
	ConfigType     types.String `tfsdk:"config_type"`
	ReassignedToID types.String `tfsdk:"reassigned_to_id"`
	StartDate      types.String `tfsdk:"start_date"`
	EndDate        types.String `tfsdk:"end_date"`
	Created        types.String `tfsdk:"created"`
	Modified       types.String `tfsdk:"modified"`
}

// reassignmentConfigurationDateValidator checks that a string is an RFC 3339 date-time.
type reassignmentConfigurationDateValidator struct{}

func (v reassignmentConfigurationDateValidator) Description(ctx context.Context) string {
	return "value must be an RFC 3339 date-time, e.g. 2026-01-31T00:00:00Z"
}

func (v reassignmentConfigurationDateValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v reassignmentConfigurationDateValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := time.Parse(time.RFC3339, req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid date-time", fmt.Sprintf("%s: %s", v.Description(ctx), err))
	}
}

func (r *ReassignmentConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reassignment_configuration"
}

func (r *ReassignmentConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the reassignment of one work type of an identity to another identity (work reassignment, e.g. during an absence). Uses an experimental API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource ID in the form `<identity_id>/<config_type>`",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				MarkdownDescription: "ID of the identity whose work is reassigned. Changing it forces a new reassignment.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"config_type": schema.StringAttribute{
				MarkdownDescription: "Work type to reassign: `ACCESS_REQUESTS`, `CERTIFICATIONS`, `MANUAL_TASKS` or `GENERIC_APPROVALS`. Changing it forces a new reassignment.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{triggerSubscriptionOneOfValidator{values: []string{"ACCESS_REQUESTS", "CERTIFICATIONS", "MANUAL_TASKS", "GENERIC_APPROVALS"}}},
			},
			"reassigned_to_id": schema.StringAttribute{
				MarkdownDescription: "ID of the identity that receives the work items",
				Required:            true,
			},
			"start_date": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 date-time from which work items are reassigned. Defaults to the API default when not set. Equal times in a different format do not show a difference.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{reassignmentConfigurationDateValidator{}},
			},
			"end_date": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 date-time at which the reassignment ends. When not set, the reassignment is permanent.",
				Optional:            true,
				Validators:          []validator.String{reassignmentConfigurationDateValidator{}},
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
	}
}

func (r *ReassignmentConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func reassignmentConfigurationFromModel(data ReassignmentConfigurationModel) *ReassignmentConfigurationRequest {
	return &ReassignmentConfigurationRequest{
		ReassignedFromID: data.IdentityID.ValueString(),
		ReassignedToID:   data.ReassignedToID.ValueString(),
		ConfigType:       data.ConfigType.ValueString(),
		StartDate:        data.StartDate.ValueString(),
		EndDate:          stringPointer(data.EndDate),
	}
}

// reassignmentConfigurationSameTime reports whether two date-time strings denote the same instant.
func reassignmentConfigurationSameTime(a, b string) bool {
	ta, errA := time.Parse(time.RFC3339Nano, a)
	tb, errB := time.Parse(time.RFC3339Nano, b)
	return errA == nil && errB == nil && ta.Equal(tb)
}

// reassignmentConfigurationTimeState returns the state value of a date-time read from the API. An
// empty value maps to null, and a prior value denoting the same instant is kept.
func reassignmentConfigurationTimeState(prior types.String, value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	if !prior.IsNull() && !prior.IsUnknown() && reassignmentConfigurationSameTime(prior.ValueString(), value) {
		return prior
	}
	return types.StringValue(value)
}

func reassignmentConfigurationAudit(detail *ReassignmentConfigurationDetail) (string, string) {
	if detail.AuditDetails == nil {
		return "", ""
	}
	return detail.AuditDetails.Created, detail.AuditDetails.Modified
}

// setReassignmentConfigurationState refreshes the model from the API detail.
func setReassignmentConfigurationState(data *ReassignmentConfigurationModel, identityID string, detail *ReassignmentConfigurationDetail) {
	data.ID = types.StringValue(identityID + "/" + detail.ConfigType)
	data.IdentityID = types.StringValue(identityID)
	data.ConfigType = types.StringValue(detail.ConfigType)
	if detail.TargetIdentity != nil {
		data.ReassignedToID = types.StringValue(detail.TargetIdentity.ID)
	}
	data.StartDate = reassignmentConfigurationTimeState(data.StartDate, detail.StartDate)
	data.EndDate = reassignmentConfigurationTimeState(data.EndDate, detail.EndDate)
	created, modified := reassignmentConfigurationAudit(detail)
	data.Created = types.StringValue(created)
	data.Modified = types.StringValue(modified)
}

// resolveReassignmentConfigurationComputed keeps the planned values and resolves the computed ones
// after Create or Update. A missing detail in the response leaves the prior values.
func resolveReassignmentConfigurationComputed(data *ReassignmentConfigurationModel, prior ReassignmentConfigurationModel, detail *ReassignmentConfigurationDetail) {
	data.ID = types.StringValue(data.IdentityID.ValueString() + "/" + data.ConfigType.ValueString())
	startDate, created, modified := "", prior.Created.ValueString(), prior.Modified.ValueString()
	if detail != nil {
		startDate = detail.StartDate
		if c, m := reassignmentConfigurationAudit(detail); c != "" || m != "" {
			created, modified = c, m
		}
	}
	if data.StartDate.IsUnknown() {
		data.StartDate = reassignmentConfigurationTimeState(types.StringNull(), startDate)
	}
	if data.Created.IsUnknown() {
		data.Created = types.StringValue(created)
	}
	data.Modified = types.StringValue(modified)
}

func (r *ReassignmentConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ReassignmentConfigurationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateReassignmentConfiguration(ctx, reassignmentConfigurationFromModel(data))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create reassignment configuration: %s", err))
		return
	}
	detail := reassignmentConfigurationDetail(created, data.ConfigType.ValueString())
	if detail == nil {
		if current, err := client.GetReassignmentConfiguration(ctx, data.IdentityID.ValueString()); err == nil {
			detail = reassignmentConfigurationDetail(current, data.ConfigType.ValueString())
		}
	}
	resolveReassignmentConfigurationComputed(&data, ReassignmentConfigurationModel{}, detail)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ReassignmentConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ReassignmentConfigurationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	config, err := client.GetReassignmentConfiguration(ctx, data.IdentityID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read reassignment configuration: %s", err))
		return
	}
	detail := reassignmentConfigurationDetail(config, data.ConfigType.ValueString())
	if detail == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	setReassignmentConfigurationState(&data, data.IdentityID.ValueString(), detail)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update replaces the reassignment of the configured work type with PUT.
func (r *ReassignmentConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ReassignmentConfigurationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.UpdateReassignmentConfiguration(ctx, reassignmentConfigurationFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update reassignment configuration: %s", err))
		return
	}
	resolveReassignmentConfigurationComputed(&plan, state, reassignmentConfigurationDetail(updated, plan.ConfigType.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ReassignmentConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ReassignmentConfigurationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteReassignmentConfiguration(ctx, data.IdentityID.ValueString(), data.ConfigType.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete reassignment configuration: %s", err))
	}
}

func (r *ReassignmentConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	identityID, configType, ok := strings.Cut(req.ID, "/")
	if !ok || identityID == "" || configType == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected <identity_id>/<config_type>, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("identity_id"), identityID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("config_type"), configType)...)
}
