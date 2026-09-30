package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	registerResource(NewSodPolicyResource)
	registerDataSource(NewSodPolicyDataSource)
}

const (
	sodPolicyTypeGeneral           = "GENERAL"
	sodPolicyTypeConflictingAccess = "CONFLICTING_ACCESS_BASED"
)

// SodPolicy is a separation of duties policy as returned by the /v2026/sod-policies API.
type SodPolicy struct {
	ID                             string                         `json:"id,omitempty"`
	Name                           string                         `json:"name"`
	Created                        string                         `json:"created,omitempty"`
	Modified                       string                         `json:"modified,omitempty"`
	Description                    *string                        `json:"description,omitempty"`
	OwnerRef                       *SodPolicyRef                  `json:"ownerRef,omitempty"`
	ExternalPolicyReference        *string                        `json:"externalPolicyReference,omitempty"`
	PolicyQuery                    string                         `json:"policyQuery,omitempty"`
	CompensatingControls           *string                        `json:"compensatingControls,omitempty"`
	CorrectionAdvice               *string                        `json:"correctionAdvice,omitempty"`
	State                          string                         `json:"state,omitempty"`
	Tags                           []string                       `json:"tags,omitempty"`
	CreatorID                      string                         `json:"creatorId,omitempty"`
	ModifierID                     string                         `json:"modifierId,omitempty"`
	ViolationOwnerAssignmentConfig *SodPolicyViolationOwnerConfig `json:"violationOwnerAssignmentConfig,omitempty"`
	Scheduled                      *bool                          `json:"scheduled,omitempty"`
	Type                           string                         `json:"type,omitempty"`
	ConflictingAccessCriteria      map[string]interface{}         `json:"conflictingAccessCriteria,omitempty"`
}

// SodPolicyRef is an owner or recipient reference of a SOD policy.
type SodPolicyRef struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// SodPolicyViolationOwnerConfig determines who owns the violations of a SOD policy.
type SodPolicyViolationOwnerConfig struct {
	AssignmentRule *string       `json:"assignmentRule,omitempty"`
	OwnerRef       *SodPolicyRef `json:"ownerRef,omitempty"`
}

func (c *Client) GetSodPolicy(ctx context.Context, id string) (*SodPolicy, error) {
	var policy SodPolicy
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/sod-policies/%s", id), nil, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

func (c *Client) GetSodPolicyByName(ctx context.Context, name string) (*SodPolicy, error) {
	return findByName(ctx, c, "/v2026/sod-policies", "SOD policy", name, func(p SodPolicy) string { return p.Name })
}

func (c *Client) CreateSodPolicy(ctx context.Context, policy *SodPolicy) (*SodPolicy, error) {
	var created SodPolicy
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/sod-policies", policy, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdateSodPolicy replaces the managed fields with PUT, keeping fields the provider does not manage.
func (c *Client) UpdateSodPolicy(ctx context.Context, id string, managed map[string]interface{}) (*SodPolicy, error) {
	var updated SodPolicy
	if err := c.putMerged(ctx, apiPath("/v2026/sod-policies/%s", id), managed, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// PatchSodPolicy applies JSON Patch operations. The API only accepts it for conflicting access based policies.
func (c *Client) PatchSodPolicy(ctx context.Context, id string, ops []jsonPatchOp) (*SodPolicy, error) {
	var updated SodPolicy
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/sod-policies/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeleteSodPolicy permanently deletes a policy. The API soft deletes by default (logical=true).
func (c *Client) DeleteSodPolicy(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/sod-policies/%s", id)+"?logical=false", nil, nil)
}

var _ resource.Resource = &SodPolicyResource{}
var _ resource.ResourceWithImportState = &SodPolicyResource{}
var _ resource.ResourceWithValidateConfig = &SodPolicyResource{}

func NewSodPolicyResource() resource.Resource {
	return &SodPolicyResource{}
}

type SodPolicyResource struct {
	client *Config
}

type SodPolicyModel struct {
	ID                             types.String `tfsdk:"id"`
	Name                           types.String `tfsdk:"name"`
	Description                    types.String `tfsdk:"description"`
	OwnerRef                       types.List   `tfsdk:"owner_ref"`
	ExternalPolicyReference        types.String `tfsdk:"external_policy_reference"`
	PolicyQuery                    types.String `tfsdk:"policy_query"`
	CompensatingControls           types.String `tfsdk:"compensating_controls"`
	CorrectionAdvice               types.String `tfsdk:"correction_advice"`
	State                          types.String `tfsdk:"state"`
	Tags                           types.List   `tfsdk:"tags"`
	ViolationOwnerAssignmentConfig types.List   `tfsdk:"violation_owner_assignment_config"`
	Scheduled                      types.Bool   `tfsdk:"scheduled"`
	Type                           types.String `tfsdk:"type"`
	ConflictingAccessCriteriaJSON  types.String `tfsdk:"conflicting_access_criteria_json"`
	CreatorID                      types.String `tfsdk:"creator_id"`
	ModifierID                     types.String `tfsdk:"modifier_id"`
	Created                        types.String `tfsdk:"created"`
	Modified                       types.String `tfsdk:"modified"`
}

type SodPolicyViolationOwnerConfigModel struct {
	AssignmentRule types.String `tfsdk:"assignment_rule"`
	OwnerRef       types.List   `tfsdk:"owner_ref"`
}

var sodPolicyViolationOwnerConfigObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"assignment_rule": types.StringType,
	"owner_ref":       types.ListType{ElemType: objectInfoObjectType},
}}

func (r *SodPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sod_policy"
}

func sodPolicyRefBlock(description, typeDescription string, min int) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: description,
		Validators:          []validator.List{listSizeBetween(min, 1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"id":   schema.StringAttribute{MarkdownDescription: "Owner ID", Required: true},
				"type": schema.StringAttribute{MarkdownDescription: typeDescription, Required: true},
				"name": schema.StringAttribute{MarkdownDescription: "Owner name. The API resolves it, so it can be left out.", Optional: true},
			},
		},
	}
}

func (r *SodPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a separation of duties (SOD) policy, either a general policy defined by a search query or a conflicting access based policy defined by two sets of entitlements.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "SOD policy ID",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Policy business name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Policy description",
				Optional:            true,
			},
			"external_policy_reference": schema.StringAttribute{
				MarkdownDescription: "Reference to an external policy",
				Optional:            true,
			},
			"policy_query": schema.StringAttribute{
				MarkdownDescription: "Search query of the policy. Required for `GENERAL` policies. For `CONFLICTING_ACCESS_BASED` policies the API generates it from `conflicting_access_criteria_json`, so it must not be set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{sodPolicyQueryPlanModifier{}},
			},
			"compensating_controls": schema.StringAttribute{
				MarkdownDescription: "Compensating (mitigating) controls",
				Optional:            true,
			},
			"correction_advice": schema.StringAttribute{
				MarkdownDescription: "Advice on how to correct a violation",
				Optional:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Whether the policy is enforced: `ENFORCED` or `NOT_ENFORCED`. When not set, the value chosen by the API is kept.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tags": schema.ListAttribute{
				MarkdownDescription: "Tags of the policy",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"scheduled": schema.BoolAttribute{
				MarkdownDescription: "Whether the policy is scheduled. When not set, the value chosen by the API is kept.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Policy type: `GENERAL` (query based, the default) or `CONFLICTING_ACCESS_BASED`. Changing it forces a new policy.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(sodPolicyTypeGeneral),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"conflicting_access_criteria_json": schema.StringAttribute{
				MarkdownDescription: "Conflicting access criteria of a `CONFLICTING_ACCESS_BASED` policy as a JSON object with `leftCriteria` and `rightCriteria`, each with a `name` and a `criteriaList` of 1 to 50 entitlement references (`type = \"ENTITLEMENT\"`, `id`). Use `jsonencode()`. Fields the API adds, such as entitlement names, do not cause a diff.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"creator_id": schema.StringAttribute{
				MarkdownDescription: "ID of the identity that created the policy",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modifier_id": schema.StringAttribute{
				MarkdownDescription: "ID of the identity that last modified the policy",
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
			"owner_ref": sodPolicyRefBlock("Owner of the policy. Exactly one block is required.", "Owner type: `IDENTITY` or `GOVERNANCE_GROUP`", 1),
			"violation_owner_assignment_config": schema.ListNestedBlock{
				MarkdownDescription: "Who owns the violations of the policy. At most one block. When the block is not configured, the setting is not managed and the value in IdentityNow is kept.",
				Validators:          []validator.List{listSizeBetween(0, 1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"assignment_rule": schema.StringAttribute{
							MarkdownDescription: "`MANAGER` (the manager of the identity in violation) or `STATIC` (the identity or governance group in `owner_ref`)",
							Optional:            true,
						},
					},
					Blocks: map[string]schema.Block{
						"owner_ref": sodPolicyRefBlock("Owner of the violations for the `STATIC` rule. At most one block. Leave it out for the `MANAGER` rule; a reference without ID returned for that rule is not shown.", "Owner type: `IDENTITY`, `GOVERNANCE_GROUP` or `MANAGER`", 0),
					},
				},
			},
		},
	}
}

// ValidateConfig checks the attributes required or forbidden by the policy type.
func (r *SodPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var policyType, query, criteria types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("type"), &policyType)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("policy_query"), &query)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("conflicting_access_criteria_json"), &criteria)...)
	if resp.Diagnostics.HasError() || policyType.IsUnknown() {
		return
	}
	switch policyType.ValueString() {
	case "", sodPolicyTypeGeneral:
		if query.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("policy_query"), "Missing policy query", "policy_query is required for GENERAL policies.")
		}
		if !criteria.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("conflicting_access_criteria_json"), "Invalid attribute", "conflicting_access_criteria_json can only be set for CONFLICTING_ACCESS_BASED policies.")
		}
	case sodPolicyTypeConflictingAccess:
		if criteria.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("conflicting_access_criteria_json"), "Missing criteria", "conflicting_access_criteria_json is required for CONFLICTING_ACCESS_BASED policies.")
		}
		if !query.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("policy_query"), "Invalid attribute", "policy_query is generated by the API for CONFLICTING_ACCESS_BASED policies and cannot be set.")
		}
	}
}

// sodPolicyQueryPlanModifier keeps the prior policy query when it is not configured and the
// conflicting access criteria it is generated from are unchanged. Otherwise the generated query is
// unknown until apply.
type sodPolicyQueryPlanModifier struct{}

func (m sodPolicyQueryPlanModifier) Description(ctx context.Context) string {
	return "Keeps the generated policy query unless the conflicting access criteria change."
}

func (m sodPolicyQueryPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m sodPolicyQueryPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() || req.StateValue.IsNull() || !req.PlanValue.IsUnknown() {
		return
	}
	var planned, prior types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("conflicting_access_criteria_json"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("conflicting_access_criteria_json"), &prior)...)
	if resp.Diagnostics.HasError() || planned.IsUnknown() || campaignTemplateJSONChanged(planned, prior) {
		return
	}
	resp.PlanValue = req.StateValue
}

func (r *SodPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// sodPolicyRefValue converts a single-element reference block to the API reference.
func sodPolicyRefValue(ctx context.Context, list types.List, diags *diag.Diagnostics) *SodPolicyRef {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil
	}
	var models []OwnerModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if len(models) == 0 {
		return nil
	}
	return &SodPolicyRef{ID: models[0].ID.ValueString(), Type: models[0].Type.ValueString(), Name: models[0].Name.ValueString()}
}

// sodPolicyRefState maps an API reference to a single-element list. The name stays null when it is
// not configured, to avoid a diff against the name the API resolves.
func sodPolicyRefState(ctx context.Context, ref *SodPolicyRef, prior types.List, diags *diag.Diagnostics) types.List {
	if ref == nil || (ref.ID == "" && ref.Type == "") {
		return types.ListNull(objectInfoObjectType)
	}
	name := types.StringValue(ref.Name)
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorRefs []OwnerModel
		diags.Append(prior.ElementsAs(ctx, &priorRefs, false)...)
		if len(priorRefs) > 0 && priorRefs[0].Name.IsNull() {
			name = types.StringNull()
		}
	}
	list, d := types.ListValueFrom(ctx, objectInfoObjectType, []OwnerModel{{
		ID: types.StringValue(ref.ID), Type: types.StringValue(ref.Type), Name: name,
	}})
	diags.Append(d...)
	return list
}

func sodPolicyViolationOwnerConfigValue(ctx context.Context, list types.List, diags *diag.Diagnostics) *SodPolicyViolationOwnerConfig {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil
	}
	var models []SodPolicyViolationOwnerConfigModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if len(models) == 0 {
		return nil
	}
	return &SodPolicyViolationOwnerConfig{
		AssignmentRule: stringPointer(models[0].AssignmentRule),
		OwnerRef:       sodPolicyRefValue(ctx, models[0].OwnerRef, diags),
	}
}

// sodPolicyViolationOwnerConfigState maps the violation owner configuration. When all is false and
// the block is not in the prior state, it stays null: the setting is only managed when configured.
func sodPolicyViolationOwnerConfigState(ctx context.Context, config *SodPolicyViolationOwnerConfig, prior types.List, all bool, diags *diag.Diagnostics) types.List {
	priorSet := !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) > 0
	if (!all && !priorSet) || config == nil {
		return types.ListNull(sodPolicyViolationOwnerConfigObjectType)
	}
	priorModel := SodPolicyViolationOwnerConfigModel{AssignmentRule: types.StringNull(), OwnerRef: types.ListNull(objectInfoObjectType)}
	if priorSet {
		var priorModels []SodPolicyViolationOwnerConfigModel
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
		if len(priorModels) > 0 {
			priorModel = priorModels[0]
		}
	}
	rule := ""
	if config.AssignmentRule != nil {
		rule = *config.AssignmentRule
	}
	if all && rule == "" && config.OwnerRef == nil {
		return types.ListNull(sodPolicyViolationOwnerConfigObjectType)
	}
	ownerRef := sodPolicyRefState(ctx, config.OwnerRef, priorModel.OwnerRef, diags)
	priorRefSet := !priorModel.OwnerRef.IsNull() && !priorModel.OwnerRef.IsUnknown() && len(priorModel.OwnerRef.Elements()) > 0
	if !priorRefSet && config.OwnerRef != nil && (rule == "MANAGER" || config.OwnerRef.ID == "") {
		// For the MANAGER rule the API may return a reference without ID ({"type": "MANAGER"}); it
		// only repeats the rule, so it is not shown when no owner_ref block is configured.
		ownerRef = types.ListNull(objectInfoObjectType)
	}
	list, d := types.ListValueFrom(ctx, sodPolicyViolationOwnerConfigObjectType, []SodPolicyViolationOwnerConfigModel{{
		AssignmentRule: optionalStringState(priorModel.AssignmentRule, rule),
		OwnerRef:       ownerRef,
	}})
	diags.Append(d...)
	return list
}

func sodPolicyTags(ctx context.Context, list types.List, diags *diag.Diagnostics) []string {
	tags := []string{}
	if !list.IsNull() && !list.IsUnknown() {
		diags.Append(list.ElementsAs(ctx, &tags, false)...)
	}
	return tags
}

func sodPolicyTagsState(ctx context.Context, tags []string, prior types.List, diags *diag.Diagnostics) types.List {
	if len(tags) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.ListNull(types.StringType)
	}
	if tags == nil {
		tags = []string{}
	}
	list, d := types.ListValueFrom(ctx, types.StringType, tags)
	diags.Append(d...)
	return list
}

// sodPolicyFromModel converts the model to the create request body.
func sodPolicyFromModel(ctx context.Context, data SodPolicyModel, diags *diag.Diagnostics) *SodPolicy {
	policy := &SodPolicy{
		Name:                           data.Name.ValueString(),
		Description:                    stringPointer(data.Description),
		OwnerRef:                       sodPolicyRefValue(ctx, data.OwnerRef, diags),
		ExternalPolicyReference:        stringPointer(data.ExternalPolicyReference),
		CompensatingControls:           stringPointer(data.CompensatingControls),
		CorrectionAdvice:               stringPointer(data.CorrectionAdvice),
		Tags:                           sodPolicyTags(ctx, data.Tags, diags),
		ViolationOwnerAssignmentConfig: sodPolicyViolationOwnerConfigValue(ctx, data.ViolationOwnerAssignmentConfig, diags),
		Scheduled:                      boolPointer(data.Scheduled),
		Type:                           data.Type.ValueString(),
	}
	if query := stringPointer(data.PolicyQuery); query != nil {
		policy.PolicyQuery = *query
	}
	if state := stringPointer(data.State); state != nil {
		policy.State = *state
	}
	if criteria, ok := campaignTemplateJSONValue(data.ConflictingAccessCriteriaJSON, "conflicting_access_criteria_json", diags).(map[string]interface{}); ok {
		policy.ConflictingAccessCriteria = criteria
	}
	return policy
}

// sodPolicyManagedFields returns the fields sent with PUT. Unset optional values are sent as null
// to clear them. The violation owner configuration, state and scheduled flag are only sent when
// known, so the values in IdentityNow are kept otherwise.
func sodPolicyManagedFields(ctx context.Context, data SodPolicyModel, diags *diag.Diagnostics) map[string]interface{} {
	policy := sodPolicyFromModel(ctx, data, diags)
	managed := map[string]interface{}{
		"name":                    policy.Name,
		"description":             policy.Description,
		"ownerRef":                policy.OwnerRef,
		"externalPolicyReference": policy.ExternalPolicyReference,
		"compensatingControls":    policy.CompensatingControls,
		"correctionAdvice":        policy.CorrectionAdvice,
		"tags":                    policy.Tags,
		"type":                    policy.Type,
	}
	if policy.PolicyQuery != "" {
		managed["policyQuery"] = policy.PolicyQuery
	}
	if policy.State != "" {
		managed["state"] = policy.State
	}
	if policy.Scheduled != nil {
		managed["scheduled"] = *policy.Scheduled
	}
	if policy.ViolationOwnerAssignmentConfig != nil {
		managed["violationOwnerAssignmentConfig"] = policy.ViolationOwnerAssignmentConfig
	}
	if policy.ConflictingAccessCriteria != nil {
		managed["conflictingAccessCriteria"] = policy.ConflictingAccessCriteria
	}
	return managed
}

// sodPolicyPatchOps returns JSON Patch operations for the fields changed between state and plan.
// The API only accepts PATCH for conflicting access based policies.
func sodPolicyPatchOps(ctx context.Context, plan, state SodPolicyModel, diags *diag.Diagnostics) []jsonPatchOp {
	policy := sodPolicyFromModel(ctx, plan, diags)
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", policy.Name)
	b.replaceOrRemoveIfChanged(plan.Description, state.Description, "/description", policy.Description)
	b.replaceIfChanged(plan.OwnerRef, state.OwnerRef, "/ownerRef", policy.OwnerRef)
	b.replaceOrRemoveIfChanged(plan.ExternalPolicyReference, state.ExternalPolicyReference, "/externalPolicyReference", policy.ExternalPolicyReference)
	b.replaceOrRemoveIfChanged(plan.CompensatingControls, state.CompensatingControls, "/compensatingControls", policy.CompensatingControls)
	b.replaceOrRemoveIfChanged(plan.CorrectionAdvice, state.CorrectionAdvice, "/correctionAdvice", policy.CorrectionAdvice)
	b.replaceIfChanged(plan.State, state.State, "/state", policy.State)
	b.replaceIfChanged(plan.Tags, state.Tags, "/tags", policy.Tags)
	b.replaceIfChanged(plan.Scheduled, state.Scheduled, "/scheduled", policy.Scheduled)
	if policy.ViolationOwnerAssignmentConfig != nil {
		b.replaceIfChanged(plan.ViolationOwnerAssignmentConfig, state.ViolationOwnerAssignmentConfig, "/violationOwnerAssignmentConfig", policy.ViolationOwnerAssignmentConfig)
	}
	if campaignTemplateJSONChanged(plan.ConflictingAccessCriteriaJSON, state.ConflictingAccessCriteriaJSON) && policy.ConflictingAccessCriteria != nil {
		b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/conflictingAccessCriteria", Value: policy.ConflictingAccessCriteria})
	}
	return b.ops
}

// setSodPolicyComputed resolves the computed attributes after apply from the API response. Values
// known from the plan (kept from state) are not changed.
func setSodPolicyComputed(data *SodPolicyModel, policy *SodPolicy) {
	data.ID = computedStringFromAPI(data.ID, policy.ID)
	data.Created = computedStringFromAPI(data.Created, policy.Created)
	data.Modified = types.StringValue(policy.Modified)
	data.CreatorID = computedStringFromAPI(data.CreatorID, policy.CreatorID)
	data.ModifierID = types.StringValue(policy.ModifierID)
	data.PolicyQuery = computedStringFromAPI(data.PolicyQuery, policy.PolicyQuery)
	data.State = computedStringFromAPI(data.State, policy.State)
	data.Scheduled = computedBoolFromAPI(data.Scheduled, policy.Scheduled)
	data.Type = computedStringFromAPI(data.Type, policy.Type)
}

// setSodPolicyState maps an API policy onto the model when reading. all reports every field, as
// needed by the data source; otherwise unset optional attributes stay null.
func setSodPolicyState(ctx context.Context, data *SodPolicyModel, policy *SodPolicy, all bool, diags *diag.Diagnostics) {
	deref := func(v *string) string {
		if v == nil {
			return ""
		}
		return *v
	}
	data.ID = types.StringValue(policy.ID)
	data.Name = types.StringValue(policy.Name)
	data.Description = optionalStringState(data.Description, deref(policy.Description))
	data.OwnerRef = sodPolicyRefState(ctx, policy.OwnerRef, data.OwnerRef, diags)
	data.ExternalPolicyReference = optionalStringState(data.ExternalPolicyReference, deref(policy.ExternalPolicyReference))
	data.PolicyQuery = types.StringValue(policy.PolicyQuery)
	data.CompensatingControls = optionalStringState(data.CompensatingControls, deref(policy.CompensatingControls))
	data.CorrectionAdvice = optionalStringState(data.CorrectionAdvice, deref(policy.CorrectionAdvice))
	data.State = types.StringValue(policy.State)
	data.Tags = sodPolicyTagsState(ctx, policy.Tags, data.Tags, diags)
	data.ViolationOwnerAssignmentConfig = sodPolicyViolationOwnerConfigState(ctx, policy.ViolationOwnerAssignmentConfig, data.ViolationOwnerAssignmentConfig, all, diags)
	data.Scheduled = types.BoolValue(policy.Scheduled != nil && *policy.Scheduled)
	policyType := policy.Type
	if policyType == "" {
		policyType = sodPolicyTypeGeneral
	}
	data.Type = types.StringValue(policyType)
	data.ConflictingAccessCriteriaJSON = jsonSubsetState(data.ConflictingAccessCriteriaJSON, policy.ConflictingAccessCriteria)
	data.CreatorID = types.StringValue(policy.CreatorID)
	data.ModifierID = types.StringValue(policy.ModifierID)
	data.Created = types.StringValue(policy.Created)
	data.Modified = types.StringValue(policy.Modified)
}

func (r *SodPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SodPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	policy := sodPolicyFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateSodPolicy(ctx, policy)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create SOD policy: %s", err))
		return
	}
	setSodPolicyComputed(&data, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SodPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SodPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	policy, err := client.GetSodPolicy(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read SOD policy: %s", err))
		return
	}
	setSodPolicyState(ctx, &data, policy, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SodPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SodPolicyModel
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
	var updated *SodPolicy
	if plan.Type.ValueString() == sodPolicyTypeConflictingAccess {
		// The policy query of these policies is generated by the API and cannot be patched.
		ops := sodPolicyPatchOps(ctx, plan, state, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(ops) == 0 {
			updated, err = client.GetSodPolicy(ctx, plan.ID.ValueString())
		} else {
			updated, err = client.PatchSodPolicy(ctx, plan.ID.ValueString(), ops)
		}
	} else {
		managed := sodPolicyManagedFields(ctx, plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		updated, err = client.UpdateSodPolicy(ctx, plan.ID.ValueString(), managed)
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update SOD policy: %s", err))
		return
	}
	setSodPolicyComputed(&plan, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SodPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SodPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteSodPolicy(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete SOD policy: %s", err))
	}
}

func (r *SodPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
