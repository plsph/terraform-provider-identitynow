package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewCampaignTemplateResource)
	registerDataSource(NewCampaignTemplateDataSource)
}

// CampaignTemplate is a certification campaign template as returned by the /v2026/campaign-templates API.
type CampaignTemplate struct {
	ID               string                 `json:"id,omitempty"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Created          string                 `json:"created,omitempty"`
	Modified         string                 `json:"modified,omitempty"`
	Scheduled        *bool                  `json:"scheduled,omitempty"`
	OwnerRef         *ObjectInfo            `json:"ownerRef,omitempty"`
	DeadlineDuration *string                `json:"deadlineDuration,omitempty"`
	Campaign         map[string]interface{} `json:"campaign"`
}

// campaignTemplateCampaignReadOnlyKeys are campaign fields set by the API. They are removed from the
// API value before it is stored in campaign_json, since they cannot be configured.
var campaignTemplateCampaignReadOnlyKeys = []string{
	"id", "status", "created", "modified", "totalCertifications", "completedCertifications", "alerts",
	"sourcesWithOrphanEntitlements",
}

func (c *Client) GetCampaignTemplate(ctx context.Context, id string) (*CampaignTemplate, error) {
	var template CampaignTemplate
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/campaign-templates/%s", id), nil, &template); err != nil {
		return nil, err
	}
	return &template, nil
}

func (c *Client) GetCampaignTemplateByName(ctx context.Context, name string) (*CampaignTemplate, error) {
	return findByName(ctx, c, "/v2026/campaign-templates", "campaign template", name, func(t CampaignTemplate) string { return t.Name })
}

func (c *Client) CreateCampaignTemplate(ctx context.Context, template *CampaignTemplate) (*CampaignTemplate, error) {
	var created CampaignTemplate
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/campaign-templates", template, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchCampaignTemplate(ctx context.Context, id string, ops []jsonPatchOp) (*CampaignTemplate, error) {
	var updated CampaignTemplate
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/campaign-templates/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteCampaignTemplate(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/campaign-templates/%s", id), nil, nil)
}

// campaignTemplateJSONValue decodes a JSON attribute for a request body. Null and unknown values return nil.
func campaignTemplateJSONValue(value types.String, attribute string, diags *diag.Diagnostics) interface{} {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(value.ValueString()), &decoded); err != nil {
		diags.AddAttributeError(path.Root(attribute), "Invalid JSON", err.Error())
		return nil
	}
	return decoded
}

// campaignTemplateJSONChanged reports whether a planned JSON attribute differs semantically from state.
func campaignTemplateJSONChanged(planned, prior types.String) bool {
	if planned.IsUnknown() {
		return false
	}
	if planned.IsNull() || prior.IsNull() {
		return planned.IsNull() != prior.IsNull()
	}
	return !jsonSemanticallyEqual([]byte(planned.ValueString()), []byte(prior.ValueString()))
}

var _ resource.Resource = &CampaignTemplateResource{}
var _ resource.ResourceWithImportState = &CampaignTemplateResource{}

func NewCampaignTemplateResource() resource.Resource {
	return &CampaignTemplateResource{}
}

type CampaignTemplateResource struct {
	client *Config
}

type CampaignTemplateModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	DeadlineDuration types.String `tfsdk:"deadline_duration"`
	CampaignJSON     types.String `tfsdk:"campaign_json"`
	Scheduled        types.Bool   `tfsdk:"scheduled"`
	OwnerRef         types.List   `tfsdk:"owner_ref"`
	Created          types.String `tfsdk:"created"`
	Modified         types.String `tfsdk:"modified"`
}

func (r *CampaignTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign_template"
}

func (r *CampaignTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a certification campaign template. Campaigns can be generated from the template manually or on a schedule with `identitynow_campaign_template_schedule`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Campaign template ID",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Template name. It has no bearing on the names of generated campaigns.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Template description. It has no bearing on the descriptions of generated campaigns.",
				Required:            true,
			},
			"deadline_duration": schema.StringAttribute{
				MarkdownDescription: "Time period in which generated campaigns must be completed, as an ISO-8601 duration, e.g. `P2W`. The campaign deadline is the generation date plus this duration.",
				Optional:            true,
			},
			"campaign_json": schema.StringAttribute{
				MarkdownDescription: "Definition of the campaigns generated from this template as a JSON object, e.g. `name`, `description`, `type`, `emailNotificationEnabled` and the type specific `searchCampaignInfo`, `sourceOwnerCampaignInfo`, `roleCompositionCampaignInfo` or `machineAccountCampaignInfo`. Use `jsonencode()`. Fields the API adds, such as defaults and read-only fields, do not cause a diff.",
				Required:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"scheduled": schema.BoolAttribute{
				MarkdownDescription: "Whether the template has a schedule",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"owner_ref": schema.ListNestedAttribute{
				MarkdownDescription: "Owner of the template and of the campaigns generated by its schedule, set by the API to the identity that created the template",
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{MarkdownDescription: "Owner ID", Computed: true},
						"type": schema.StringAttribute{MarkdownDescription: "Owner type", Computed: true},
						"name": schema.StringAttribute{MarkdownDescription: "Owner name", Computed: true},
					},
				},
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

func (r *CampaignTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// campaignTemplateFromModel converts the model to the API request body.
func campaignTemplateFromModel(data CampaignTemplateModel, diags *diag.Diagnostics) *CampaignTemplate {
	template := &CampaignTemplate{
		Name:             data.Name.ValueString(),
		Description:      data.Description.ValueString(),
		DeadlineDuration: stringPointer(data.DeadlineDuration),
		Campaign:         map[string]interface{}{},
	}
	if campaign, ok := campaignTemplateJSONValue(data.CampaignJSON, "campaign_json", diags).(map[string]interface{}); ok {
		template.Campaign = campaign
	}
	return template
}

// campaignTemplateCampaignState maps the API campaign to campaign_json without read-only fields.
func campaignTemplateCampaignState(prior types.String, campaign map[string]interface{}) types.String {
	stripped := make(map[string]interface{}, len(campaign))
	for key, value := range campaign {
		stripped[key] = value
	}
	for _, key := range campaignTemplateCampaignReadOnlyKeys {
		delete(stripped, key)
	}
	return jsonSubsetState(prior, stripped)
}

// setCampaignTemplateComputed resolves the computed attributes from an API response.
func setCampaignTemplateComputed(ctx context.Context, data *CampaignTemplateModel, template *CampaignTemplate, diags *diag.Diagnostics) {
	data.ID = types.StringValue(template.ID)
	data.Created = types.StringValue(template.Created)
	data.Modified = types.StringValue(template.Modified)
	data.Scheduled = types.BoolValue(template.Scheduled != nil && *template.Scheduled)
	data.OwnerRef = objectInfoListState(ctx, template.OwnerRef, diags)
}

// setCampaignTemplateState maps an API template onto the model when reading, keeping null for unset
// optional attributes and the prior campaign JSON when the API value contains it.
func setCampaignTemplateState(ctx context.Context, data *CampaignTemplateModel, template *CampaignTemplate, diags *diag.Diagnostics) {
	setCampaignTemplateComputed(ctx, data, template, diags)
	data.Name = types.StringValue(template.Name)
	data.Description = types.StringValue(template.Description)
	deadline := ""
	if template.DeadlineDuration != nil {
		deadline = *template.DeadlineDuration
	}
	data.DeadlineDuration = optionalStringState(data.DeadlineDuration, deadline)
	data.CampaignJSON = campaignTemplateCampaignState(data.CampaignJSON, template.Campaign)
}

// campaignTemplatePatchOps returns JSON Patch operations for the attributes changed between state and plan.
func campaignTemplatePatchOps(plan, state CampaignTemplateModel, diags *diag.Diagnostics) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	b.replaceOrRemoveIfChanged(plan.DeadlineDuration, state.DeadlineDuration, "/deadlineDuration", plan.DeadlineDuration.ValueString())
	if campaignTemplateJSONChanged(plan.CampaignJSON, state.CampaignJSON) {
		b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/campaign", Value: campaignTemplateJSONValue(plan.CampaignJSON, "campaign_json", diags)})
	}
	return b.ops
}

func (r *CampaignTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CampaignTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	template := campaignTemplateFromModel(data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateCampaignTemplate(ctx, template)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create campaign template: %s", err))
		return
	}
	setCampaignTemplateComputed(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CampaignTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CampaignTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	template, err := client.GetCampaignTemplate(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read campaign template: %s", err))
		return
	}
	setCampaignTemplateState(ctx, &data, template, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CampaignTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state CampaignTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := campaignTemplatePatchOps(plan, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(ops) == 0 {
		plan.Modified = state.Modified
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.PatchCampaignTemplate(ctx, plan.ID.ValueString(), ops)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update campaign template: %s", err))
		return
	}
	plan.Modified = types.StringValue(updated.Modified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CampaignTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CampaignTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteCampaignTemplate(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete campaign template: %s", err))
	}
}

func (r *CampaignTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &CampaignTemplateDataSource{}
var _ datasource.DataSourceWithValidateConfig = &CampaignTemplateDataSource{}

func NewCampaignTemplateDataSource() datasource.DataSource {
	return &CampaignTemplateDataSource{}
}

type CampaignTemplateDataSource struct {
	client *Config
}

func (d *CampaignTemplateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_campaign_template"
}

func (d *CampaignTemplateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a certification campaign template by ID or name.",
		Attributes: map[string]dsschema.Attribute{
			"id": dsschema.StringAttribute{
				MarkdownDescription: "Campaign template ID. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"name": dsschema.StringAttribute{
				MarkdownDescription: "Campaign template name. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"description":       dsschema.StringAttribute{MarkdownDescription: "Template description", Computed: true},
			"deadline_duration": dsschema.StringAttribute{MarkdownDescription: "Completion period of generated campaigns as an ISO-8601 duration", Computed: true},
			"campaign_json":     dsschema.StringAttribute{MarkdownDescription: "Definition of the generated campaigns as a JSON object, without read-only fields", Computed: true},
			"scheduled":         dsschema.BoolAttribute{MarkdownDescription: "Whether the template has a schedule", Computed: true},
			"owner_ref": dsschema.ListNestedAttribute{
				MarkdownDescription: "Owner of the template",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{
					Attributes: map[string]dsschema.Attribute{
						"id":   dsschema.StringAttribute{MarkdownDescription: "Owner ID", Computed: true},
						"type": dsschema.StringAttribute{MarkdownDescription: "Owner type", Computed: true},
						"name": dsschema.StringAttribute{MarkdownDescription: "Owner name", Computed: true},
					},
				},
			},
			"created":  dsschema.StringAttribute{MarkdownDescription: "Creation date", Computed: true},
			"modified": dsschema.StringAttribute{MarkdownDescription: "Last modification date", Computed: true},
		},
	}
}

func (d *CampaignTemplateDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *CampaignTemplateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CampaignTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CampaignTemplateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var template *CampaignTemplate
	if !data.ID.IsNull() {
		template, err = client.GetCampaignTemplate(ctx, data.ID.ValueString())
	} else {
		template, err = client.GetCampaignTemplateByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Campaign template not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read campaign template: %s", err))
		return
	}
	setCampaignTemplateState(ctx, &data, template, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
