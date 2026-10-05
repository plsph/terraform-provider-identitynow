package provider

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewIdentityProfileResource)
	registerDataSource(NewIdentityProfileDataSource)
}

// IdentityProfileRef is a reference to the owner or the authoritative source of an identity profile.
type IdentityProfileRef struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// IdentityProfileAttributeConfig is the identity attribute mapping configuration of an identity profile.
type IdentityProfileAttributeConfig struct {
	Enabled             *bool           `json:"enabled,omitempty"`
	AttributeTransforms json.RawMessage `json:"attributeTransforms,omitempty"`
}

// IdentityProfileExceptionReportReference references the last identity exception report of a profile.
type IdentityProfileExceptionReportReference struct {
	TaskResultID string `json:"taskResultId,omitempty"`
	ReportName   string `json:"reportName,omitempty"`
}

// IdentityProfile is an identity profile as returned by the /v2026/identity-profiles API.
type IdentityProfile struct {
	ID                               string                                   `json:"id,omitempty"`
	Name                             string                                   `json:"name"`
	Description                      *string                                  `json:"description,omitempty"`
	Created                          string                                   `json:"created,omitempty"`
	Modified                         string                                   `json:"modified,omitempty"`
	Owner                            *IdentityProfileRef                      `json:"owner,omitempty"`
	Priority                         *int64                                   `json:"priority,omitempty"`
	AuthoritativeSource              *IdentityProfileRef                      `json:"authoritativeSource,omitempty"`
	IdentityRefreshRequired          *bool                                    `json:"identityRefreshRequired,omitempty"`
	IdentityCount                    *int64                                   `json:"identityCount,omitempty"`
	IdentityAttributeConfig          *IdentityProfileAttributeConfig          `json:"identityAttributeConfig,omitempty"`
	IdentityExceptionReportReference *IdentityProfileExceptionReportReference `json:"identityExceptionReportReference,omitempty"`
	HasTimeBasedAttr                 *bool                                    `json:"hasTimeBasedAttr,omitempty"`
}

func (c *Client) GetIdentityProfile(ctx context.Context, id string) (*IdentityProfile, error) {
	var profile IdentityProfile
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/identity-profiles/%s", id), nil, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (c *Client) GetIdentityProfileByName(ctx context.Context, name string) (*IdentityProfile, error) {
	return findByName(ctx, c, "/v2026/identity-profiles", "identity profile", name, func(p IdentityProfile) string { return p.Name })
}

func (c *Client) CreateIdentityProfile(ctx context.Context, profile *IdentityProfile) (*IdentityProfile, error) {
	var created IdentityProfile
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/identity-profiles", profile, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateIdentityProfile(ctx context.Context, id string, ops []jsonPatchOp) (*IdentityProfile, error) {
	var updated IdentityProfile
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/identity-profiles/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeleteIdentityProfile deletes an identity profile. The API accepts the request with 202 and
// deletes the profile in a background task; the task reference in the response is ignored.
func (c *Client) DeleteIdentityProfile(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/identity-profiles/%s", id), nil, nil)
}

var _ resource.Resource = &IdentityProfileResource{}
var _ resource.ResourceWithImportState = &IdentityProfileResource{}

func NewIdentityProfileResource() resource.Resource {
	return &IdentityProfileResource{}
}

type IdentityProfileResource struct {
	client *Config
}

type IdentityProfileModel struct {
	ID                             types.String `tfsdk:"id"`
	Name                           types.String `tfsdk:"name"`
	Description                    types.String `tfsdk:"description"`
	Priority                       types.Int64  `tfsdk:"priority"`
	Owner                          types.List   `tfsdk:"owner"`
	AuthoritativeSource            types.List   `tfsdk:"authoritative_source"`
	IdentityAttributeConfigEnabled types.Bool   `tfsdk:"identity_attribute_config_enabled"`
	AttributeConfigJSON            types.String `tfsdk:"attribute_config_json"`
	IdentityRefreshRequired        types.Bool   `tfsdk:"identity_refresh_required"`
	IdentityCount                  types.Int64  `tfsdk:"identity_count"`
	HasTimeBasedAttr               types.Bool   `tfsdk:"has_time_based_attr"`
	ExceptionReportTaskResultID    types.String `tfsdk:"identity_exception_report_task_result_id"`
	ExceptionReportName            types.String `tfsdk:"identity_exception_report_name"`
	Created                        types.String `tfsdk:"created"`
	Modified                       types.String `tfsdk:"modified"`
}

func (r *IdentityProfileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_profile"
}

// identityProfileRefBlock returns the schema of a reference block with a required ID and an
// optional type and name that the API fills in.
func identityProfileRefBlock(description, defaultType string, min int) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: description,
		Validators:          []validator.List{listSizeBetween(min, 1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "Referenced object ID",
					Required:            true,
				},
				"type": schema.StringAttribute{
					MarkdownDescription: fmt.Sprintf("Referenced object type, defaults to `%s`", defaultType),
					Optional:            true,
					Computed:            true,
				},
				"name": schema.StringAttribute{
					MarkdownDescription: "Referenced object name, filled in by the API when not set",
					Optional:            true,
					Computed:            true,
				},
			},
		},
	}
}

func (r *IdentityProfileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an identity profile, which defines how identities are built from an authoritative source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identity profile ID",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Identity profile name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Identity profile description",
				Optional:            true,
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "Identity profile priority. The API assigns one when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"identity_attribute_config_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the identity attribute mappings are enabled. Values are only promoted to identities when enabled.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"attribute_config_json": schema.StringAttribute{
				MarkdownDescription: "Identity attribute mappings (`identityAttributeConfig.attributeTransforms`) as a JSON array of objects with `identityAttributeName` and `transformDefinition`. When not set, the mappings generated by IdentityNow are kept and not managed.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.String{jsonArrayStringValidator{}},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_refresh_required": schema.BoolAttribute{
				MarkdownDescription: "Whether an identity refresh is required for the profile",
				Computed:            true,
			},
			"identity_count": schema.Int64Attribute{
				MarkdownDescription: "Number of identities belonging to the profile",
				Computed:            true,
			},
			"has_time_based_attr": schema.BoolAttribute{
				MarkdownDescription: "Whether the profile has time based attributes that require a periodic refresh",
				Computed:            true,
			},
			"identity_exception_report_task_result_id": schema.StringAttribute{
				MarkdownDescription: "Task result ID of the last identity exception report",
				Computed:            true,
			},
			"identity_exception_report_name": schema.StringAttribute{
				MarkdownDescription: "Name of the last identity exception report",
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
			"owner":                identityProfileRefBlock("Identity profile owner, an identity.", "IDENTITY", 0),
			"authoritative_source": identityProfileRefBlock("Authoritative source of the identity profile. Exactly one block is required.", "SOURCE", 1),
		},
	}
}

func (r *IdentityProfileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// identityProfileRefFromList converts a reference block to the API reference, using defaultType
// when the type is not known.
func identityProfileRefFromList(ctx context.Context, list types.List, defaultType string, diags *diag.Diagnostics) *IdentityProfileRef {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var refs []OwnerModel
	diags.Append(list.ElementsAs(ctx, &refs, false)...)
	if len(refs) == 0 {
		return nil
	}
	ref := &IdentityProfileRef{ID: refs[0].ID.ValueString(), Type: defaultType}
	if !refs[0].Type.IsNull() && !refs[0].Type.IsUnknown() {
		ref.Type = refs[0].Type.ValueString()
	}
	if !refs[0].Name.IsNull() && !refs[0].Name.IsUnknown() {
		ref.Name = refs[0].Name.ValueString()
	}
	return ref
}

// identityProfileRefState converts an API reference to a single element list, or null.
func identityProfileRefState(ctx context.Context, ref *IdentityProfileRef, diags *diag.Diagnostics) types.List {
	if ref == nil || ref.ID == "" {
		return types.ListNull(objectInfoObjectType)
	}
	list, d := types.ListValueFrom(ctx, objectInfoObjectType, []OwnerModel{{
		ID: types.StringValue(ref.ID), Type: types.StringValue(ref.Type), Name: types.StringValue(ref.Name),
	}})
	diags.Append(d...)
	return list
}

// identityProfileRefResolve keeps the planned reference and resolves an unknown type and name
// from the API response.
func identityProfileRefResolve(ctx context.Context, planned types.List, api *IdentityProfileRef, defaultType string, diags *diag.Diagnostics) types.List {
	if planned.IsNull() || planned.IsUnknown() {
		return planned
	}
	var refs []OwnerModel
	diags.Append(planned.ElementsAs(ctx, &refs, false)...)
	if len(refs) == 0 {
		return planned
	}
	if api == nil {
		api = &IdentityProfileRef{}
	}
	if refs[0].Type.IsUnknown() {
		value := api.Type
		if value == "" {
			value = defaultType
		}
		refs[0].Type = types.StringValue(value)
	}
	if refs[0].Name.IsUnknown() {
		refs[0].Name = types.StringValue(api.Name)
	}
	list, d := types.ListValueFrom(ctx, objectInfoObjectType, refs)
	diags.Append(d...)
	return list
}

// identityProfileTransformsJSON returns the attribute transforms of the model as raw JSON.
func identityProfileTransformsJSON(value types.String) json.RawMessage {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	return json.RawMessage(value.ValueString())
}

// identityProfileAttributeConfigValue returns the identity attribute configuration of the model
// for a replace operation: the full object with the planned enabled flag and transforms.
func identityProfileAttributeConfigValue(data IdentityProfileModel) map[string]interface{} {
	transforms := identityProfileTransformsJSON(data.AttributeConfigJSON)
	if transforms == nil {
		transforms = json.RawMessage("[]")
	}
	return map[string]interface{}{
		"enabled":             data.IdentityAttributeConfigEnabled.ValueBool(),
		"attributeTransforms": transforms,
	}
}

// identityProfileFromModel builds the create request from the plan.
func identityProfileFromModel(ctx context.Context, data IdentityProfileModel, diags *diag.Diagnostics) *IdentityProfile {
	profile := &IdentityProfile{
		Name:                data.Name.ValueString(),
		Description:         stringPointer(data.Description),
		Owner:               identityProfileRefFromList(ctx, data.Owner, "IDENTITY", diags),
		Priority:            int64Pointer(data.Priority),
		AuthoritativeSource: identityProfileRefFromList(ctx, data.AuthoritativeSource, "SOURCE", diags),
	}
	enabled := boolPointer(data.IdentityAttributeConfigEnabled)
	transforms := identityProfileTransformsJSON(data.AttributeConfigJSON)
	if enabled != nil || transforms != nil {
		profile.IdentityAttributeConfig = &IdentityProfileAttributeConfig{Enabled: enabled, AttributeTransforms: transforms}
	}
	return profile
}

// identityProfileTransformsRaw returns the attribute transforms of an API profile.
func identityProfileTransformsRaw(profile *IdentityProfile) json.RawMessage {
	if profile.IdentityAttributeConfig == nil {
		return nil
	}
	return profile.IdentityAttributeConfig.AttributeTransforms
}

// identityProfileSetComputed sets the computed-only attributes from the API.
func identityProfileSetComputed(data *IdentityProfileModel, profile *IdentityProfile) {
	data.ID = types.StringValue(profile.ID)
	data.IdentityRefreshRequired = types.BoolValue(profile.IdentityRefreshRequired != nil && *profile.IdentityRefreshRequired)
	data.HasTimeBasedAttr = types.BoolValue(profile.HasTimeBasedAttr != nil && *profile.HasTimeBasedAttr)
	count := int64(0)
	if profile.IdentityCount != nil {
		count = *profile.IdentityCount
	}
	data.IdentityCount = types.Int64Value(count)
	report := IdentityProfileExceptionReportReference{}
	if profile.IdentityExceptionReportReference != nil {
		report = *profile.IdentityExceptionReportReference
	}
	data.ExceptionReportTaskResultID = types.StringValue(report.TaskResultID)
	data.ExceptionReportName = types.StringValue(report.ReportName)
	data.Created = types.StringValue(profile.Created)
	data.Modified = types.StringValue(profile.Modified)
}

// identityProfileResolveApply keeps the planned values after create or update and resolves the
// unknown ones from the API response.
func identityProfileResolveApply(ctx context.Context, data *IdentityProfileModel, profile *IdentityProfile, diags *diag.Diagnostics) {
	identityProfileSetComputed(data, profile)
	if data.Priority.IsUnknown() {
		data.Priority = types.Int64Null()
		if profile.Priority != nil {
			data.Priority = types.Int64Value(*profile.Priority)
		}
	}
	var enabled *bool
	if profile.IdentityAttributeConfig != nil {
		enabled = profile.IdentityAttributeConfig.Enabled
	}
	data.IdentityAttributeConfigEnabled = computedBoolFromAPI(data.IdentityAttributeConfigEnabled, enabled)
	if data.AttributeConfigJSON.IsUnknown() {
		data.AttributeConfigJSON = formDefinitionJSONState(types.StringNull(), identityProfileTransformsRaw(profile))
	}
	data.Owner = identityProfileRefResolve(ctx, data.Owner, profile.Owner, "IDENTITY", diags)
	data.AuthoritativeSource = identityProfileRefResolve(ctx, data.AuthoritativeSource, profile.AuthoritativeSource, "SOURCE", diags)
}

// setIdentityProfileState refreshes the model from the API, keeping null for unset optional
// attributes and semantically equal JSON.
func setIdentityProfileState(ctx context.Context, data *IdentityProfileModel, profile *IdentityProfile, diags *diag.Diagnostics) {
	identityProfileSetComputed(data, profile)
	data.Name = types.StringValue(profile.Name)
	description := ""
	if profile.Description != nil {
		description = *profile.Description
	}
	data.Description = optionalStringState(data.Description, description)
	data.Priority = types.Int64Null()
	if profile.Priority != nil {
		data.Priority = types.Int64Value(*profile.Priority)
	}
	enabled := false
	if profile.IdentityAttributeConfig != nil && profile.IdentityAttributeConfig.Enabled != nil {
		enabled = *profile.IdentityAttributeConfig.Enabled
	}
	data.IdentityAttributeConfigEnabled = types.BoolValue(enabled)
	data.AttributeConfigJSON = jsonSubsetStateRaw(data.AttributeConfigJSON, identityProfileTransformsRaw(profile))
	data.Owner = identityProfileRefState(ctx, profile.Owner, diags)
	data.AuthoritativeSource = identityProfileRefState(ctx, profile.AuthoritativeSource, diags)
}

// identityProfileJSONChanged reports whether the planned JSON differs semantically from the state.
func identityProfileJSONChanged(planned, prior types.String) bool {
	if planned.IsUnknown() {
		return false
	}
	if planned.IsNull() || prior.IsNull() || prior.IsUnknown() {
		return !planned.Equal(prior)
	}
	return !jsonSemanticallyEqual([]byte(planned.ValueString()), []byte(prior.ValueString()))
}

// identityProfilePatches returns the JSON Patch requests for the changed attributes. The API does
// not accept changes of the authoritative source and the identity attribute configuration in the
// same request, so the attribute configuration is sent in a second request when both changed.
func identityProfilePatches(ctx context.Context, plan, state IdentityProfileModel, diags *diag.Diagnostics) [][]jsonPatchOp {
	b := &patchBuilder{}
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceOrRemoveIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	if !plan.Priority.IsNull() {
		b.replaceIfChanged(plan.Priority, state.Priority, "/priority", plan.Priority.ValueInt64())
	}
	b.replaceOrRemoveIfChanged(plan.Owner, state.Owner, "/owner", identityProfileRefFromList(ctx, plan.Owner, "IDENTITY", diags))
	b.replaceIfChanged(plan.AuthoritativeSource, state.AuthoritativeSource, "/authoritativeSource", identityProfileRefFromList(ctx, plan.AuthoritativeSource, "SOURCE", diags))
	sourceChanged := !equalIgnoringUnknown(plan.AuthoritativeSource, state.AuthoritativeSource)

	var batches [][]jsonPatchOp
	configChanged := !equalIgnoringUnknown(plan.IdentityAttributeConfigEnabled, state.IdentityAttributeConfigEnabled) ||
		identityProfileJSONChanged(plan.AttributeConfigJSON, state.AttributeConfigJSON)
	configOp := jsonPatchOp{Op: "replace", Path: "/identityAttributeConfig", Value: identityProfileAttributeConfigValue(plan)}
	if configChanged && !sourceChanged {
		b.ops = append(b.ops, configOp)
	}
	if len(b.ops) > 0 {
		batches = append(batches, b.ops)
	}
	if configChanged && sourceChanged {
		batches = append(batches, []jsonPatchOp{configOp})
	}
	return batches
}

func (r *IdentityProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IdentityProfileModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	profile := identityProfileFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateIdentityProfile(ctx, profile)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create identity profile: %s", err))
		return
	}
	identityProfileResolveApply(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IdentityProfileModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	profile, err := client.GetIdentityProfile(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read identity profile: %s", err))
		return
	}
	setIdentityProfileState(ctx, &data, profile, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state IdentityProfileModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	batches := identityProfilePatches(ctx, data, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var profile *IdentityProfile
	for _, ops := range batches {
		profile, err = client.UpdateIdentityProfile(ctx, data.ID.ValueString(), ops)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update identity profile: %s", err))
			return
		}
	}
	if profile == nil {
		// Nothing configurable changed, only computed values are refreshed.
		profile, err = client.GetIdentityProfile(ctx, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read identity profile: %s", err))
			return
		}
	}
	identityProfileResolveApply(ctx, &data, profile, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IdentityProfileModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteIdentityProfile(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete identity profile: %s", err))
	}
}

func (r *IdentityProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &IdentityProfileDataSource{}
var _ datasource.DataSourceWithValidateConfig = &IdentityProfileDataSource{}

func NewIdentityProfileDataSource() datasource.DataSource {
	return &IdentityProfileDataSource{}
}

type IdentityProfileDataSource struct {
	client *Config
}

func (d *IdentityProfileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_profile"
}

func identityProfileRefDataSourceAttribute(description string) dsschema.ListNestedAttribute {
	return dsschema.ListNestedAttribute{
		MarkdownDescription: description,
		Computed:            true,
		NestedObject: dsschema.NestedAttributeObject{
			Attributes: map[string]dsschema.Attribute{
				"id":   dsschema.StringAttribute{MarkdownDescription: "Referenced object ID", Computed: true},
				"type": dsschema.StringAttribute{MarkdownDescription: "Referenced object type", Computed: true},
				"name": dsschema.StringAttribute{MarkdownDescription: "Referenced object name", Computed: true},
			},
		},
	}
}

func (d *IdentityProfileDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up an identity profile by ID or name.",
		Attributes: map[string]dsschema.Attribute{
			"id": dsschema.StringAttribute{
				MarkdownDescription: "Identity profile ID. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"name": dsschema.StringAttribute{
				MarkdownDescription: "Identity profile name. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"description":                              dsschema.StringAttribute{MarkdownDescription: "Identity profile description", Computed: true},
			"priority":                                 dsschema.Int64Attribute{MarkdownDescription: "Identity profile priority", Computed: true},
			"owner":                                    identityProfileRefDataSourceAttribute("Identity profile owner"),
			"authoritative_source":                     identityProfileRefDataSourceAttribute("Authoritative source of the identity profile"),
			"identity_attribute_config_enabled":        dsschema.BoolAttribute{MarkdownDescription: "Whether the identity attribute mappings are enabled", Computed: true},
			"attribute_config_json":                    dsschema.StringAttribute{MarkdownDescription: "Identity attribute mappings as a JSON array", Computed: true},
			"identity_refresh_required":                dsschema.BoolAttribute{MarkdownDescription: "Whether an identity refresh is required for the profile", Computed: true},
			"identity_count":                           dsschema.Int64Attribute{MarkdownDescription: "Number of identities belonging to the profile", Computed: true},
			"has_time_based_attr":                      dsschema.BoolAttribute{MarkdownDescription: "Whether the profile has time based attributes", Computed: true},
			"identity_exception_report_task_result_id": dsschema.StringAttribute{MarkdownDescription: "Task result ID of the last identity exception report", Computed: true},
			"identity_exception_report_name":           dsschema.StringAttribute{MarkdownDescription: "Name of the last identity exception report", Computed: true},
			"created":                                  dsschema.StringAttribute{MarkdownDescription: "Creation date", Computed: true},
			"modified":                                 dsschema.StringAttribute{MarkdownDescription: "Last modification date", Computed: true},
		},
	}
}

func (d *IdentityProfileDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *IdentityProfileDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IdentityProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IdentityProfileModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var profile *IdentityProfile
	if !data.ID.IsNull() {
		profile, err = client.GetIdentityProfile(ctx, data.ID.ValueString())
	} else {
		profile, err = client.GetIdentityProfileByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Identity profile not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read identity profile: %s", err))
		return
	}
	// Data sources have no prior state: return every value, including empty ones.
	data.Description = types.StringValue("")
	setIdentityProfileState(ctx, &data, profile, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
