package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
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
	registerResource(NewSourceProvisioningPolicyResource)
	registerDataSource(NewSourceProvisioningPolicyDataSource)
}

// SourceProvisioningPolicy is a provisioning policy as used by the
// /v2026/sources/{sourceId}/provisioning-policies API.
type SourceProvisioningPolicy struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	UsageType   string          `json:"usageType,omitempty"`
	Fields      json.RawMessage `json:"fields,omitempty"`
}

func (c *Client) GetSourceProvisioningPolicy(ctx context.Context, sourceID, usageType string) (*SourceProvisioningPolicy, error) {
	var policy SourceProvisioningPolicy
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/sources/%s/provisioning-policies/%s", sourceID, usageType), nil, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

func (c *Client) CreateSourceProvisioningPolicy(ctx context.Context, sourceID string, policy *SourceProvisioningPolicy) (*SourceProvisioningPolicy, error) {
	var created SourceProvisioningPolicy
	if err := c.doJSON(ctx, http.MethodPost, apiPath("/v2026/sources/%s/provisioning-policies", sourceID), policy, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateSourceProvisioningPolicy(ctx context.Context, sourceID, usageType string, policy *SourceProvisioningPolicy) (*SourceProvisioningPolicy, error) {
	var updated SourceProvisioningPolicy
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/sources/%s/provisioning-policies/%s", sourceID, usageType), policy, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteSourceProvisioningPolicy(ctx context.Context, sourceID, usageType string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/sources/%s/provisioning-policies/%s", sourceID, usageType), nil, nil)
}

var _ resource.Resource = &SourceProvisioningPolicyResource{}
var _ resource.ResourceWithImportState = &SourceProvisioningPolicyResource{}

func NewSourceProvisioningPolicyResource() resource.Resource {
	return &SourceProvisioningPolicyResource{}
}

type SourceProvisioningPolicyResource struct {
	client *Config
}

type SourceProvisioningPolicyModel struct {
	ID          types.String `tfsdk:"id"`
	SourceID    types.String `tfsdk:"source_id"`
	UsageType   types.String `tfsdk:"usage_type"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	FieldsJSON  types.String `tfsdk:"fields_json"`
}

func (r *SourceProvisioningPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_provisioning_policy"
}

func (r *SourceProvisioningPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the provisioning policy of a source for one usage type, e.g. the account create policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource ID in the format `<source_id>/<usage_type>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source_id": schema.StringAttribute{
				MarkdownDescription: "ID of the source. Changing this forces a new provisioning policy to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"usage_type": schema.StringAttribute{
				MarkdownDescription: "Provisioning operation the policy applies to: `CREATE`, `UPDATE`, `ENABLE`, `DISABLE`, `DELETE`, `ASSIGN`, `UNASSIGN`, `CREATE_GROUP`, `UPDATE_GROUP`, `DELETE_GROUP`, `REGISTER`, `CREATE_IDENTITY`, `UPDATE_IDENTITY`, `EDIT_GROUP`, `UNLOCK` or `CHANGE_PASSWORD`. Changing this forces a new provisioning policy to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Provisioning policy name.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Provisioning policy description.",
				Optional:            true,
			},
			"fields_json": schema.StringAttribute{
				MarkdownDescription: "Policy fields as a JSON array. Each field has a `name`, a `type` (`string`, `int`, `long`, `date`, `boolean` or `secret`) and optionally a `transform`, `attributes` and `isMultiValued`. Use `jsonencode()` for convenience. The value is compared semantically, so formatting and key order do not produce a diff, and keys that hold the API defaults (`transform` and `attributes` as empty objects, `isRequired` and `isMultiValued` set to false, null values) are ignored. When not set, the policy has no fields.",
				Optional:            true,
				Validators:          []validator.String{jsonArrayStringValidator{}},
			},
		},
	}
}

func (r *SourceProvisioningPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// sourceProvisioningPolicyFromModel builds the full request body. PUT replaces the policy, so
// unset fields are sent empty and removed in IdentityNow.
func sourceProvisioningPolicyFromModel(data SourceProvisioningPolicyModel) *SourceProvisioningPolicy {
	policy := &SourceProvisioningPolicy{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		UsageType:   data.UsageType.ValueString(),
		Fields:      json.RawMessage("[]"),
	}
	if !data.FieldsJSON.IsNull() && !data.FieldsJSON.IsUnknown() && data.FieldsJSON.ValueString() != "" {
		policy.Fields = json.RawMessage(data.FieldsJSON.ValueString())
	}
	return policy
}

// sourceProvisioningPolicyID returns the resource ID of a policy.
func sourceProvisioningPolicyID(sourceID, usageType string) string {
	return sourceID + "/" + usageType
}

// setSourceProvisioningPolicyState maps an API policy onto the model during refresh.
func setSourceProvisioningPolicyState(data *SourceProvisioningPolicyModel, policy *SourceProvisioningPolicy) {
	if policy.UsageType != "" {
		data.UsageType = types.StringValue(policy.UsageType)
	}
	data.ID = types.StringValue(sourceProvisioningPolicyID(data.SourceID.ValueString(), data.UsageType.ValueString()))
	data.Name = types.StringValue(policy.Name)
	data.Description = optionalStringState(data.Description, policy.Description)
	data.FieldsJSON = sourceProvisioningPolicyFieldsState(data.FieldsJSON, policy.Fields)
}

// sourceProvisioningPolicyNormalizeFields decodes a fields array and drops keys that hold the
// API defaults, so a configuration that omits them compares equal to the API response.
func sourceProvisioningPolicyNormalizeFields(raw []byte) ([]interface{}, bool) {
	var fields []interface{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, false
	}
	for _, field := range fields {
		object, ok := field.(map[string]interface{})
		if !ok {
			continue
		}
		for key, value := range object {
			switch v := value.(type) {
			case nil:
				delete(object, key)
			case bool:
				if !v && (key == "isRequired" || key == "isMultiValued") {
					delete(object, key)
				}
			case map[string]interface{}:
				if len(v) == 0 && (key == "transform" || key == "attributes") {
					delete(object, key)
				}
			}
		}
	}
	return fields, true
}

// sourceProvisioningPolicyFieldsState returns the state value of fields_json. The prior value is
// kept when it is equal to the API value after dropping API defaults. An empty API value maps to
// null unless the prior value is also an empty array.
func sourceProvisioningPolicyFieldsState(prior types.String, raw json.RawMessage) types.String {
	priorSet := !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() != ""
	if isEmptyJSONArray(raw) {
		if priorSet && isEmptyJSONArray(json.RawMessage(prior.ValueString())) {
			return prior
		}
		return types.StringNull()
	}
	fromAPI, ok := sourceProvisioningPolicyNormalizeFields(raw)
	if !ok {
		return types.StringValue(string(raw))
	}
	if priorSet {
		if fromPrior, ok := sourceProvisioningPolicyNormalizeFields([]byte(prior.ValueString())); ok && reflect.DeepEqual(fromPrior, fromAPI) {
			return prior
		}
	}
	encoded, err := json.Marshal(fromAPI)
	if err != nil {
		return types.StringValue(string(raw))
	}
	return types.StringValue(string(encoded))
}

func (r *SourceProvisioningPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SourceProvisioningPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if _, err := client.CreateSourceProvisioningPolicy(ctx, data.SourceID.ValueString(), sourceProvisioningPolicyFromModel(data)); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create provisioning policy: %s", err))
		return
	}
	data.ID = types.StringValue(sourceProvisioningPolicyID(data.SourceID.ValueString(), data.UsageType.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceProvisioningPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SourceProvisioningPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	policy, err := client.GetSourceProvisioningPolicy(ctx, data.SourceID.ValueString(), data.UsageType.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read provisioning policy: %s", err))
		return
	}
	setSourceProvisioningPolicyState(&data, policy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceProvisioningPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SourceProvisioningPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// The policy has no fields besides the modelled ones, so the full object is sent.
	if _, err := client.UpdateSourceProvisioningPolicy(ctx, data.SourceID.ValueString(), data.UsageType.ValueString(), sourceProvisioningPolicyFromModel(data)); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update provisioning policy: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceProvisioningPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SourceProvisioningPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteSourceProvisioningPolicy(ctx, data.SourceID.ValueString(), data.UsageType.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete provisioning policy: %s", err))
	}
}

func (r *SourceProvisioningPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	sourceID, usageType, ok := strings.Cut(req.ID, "/")
	if !ok || sourceID == "" || usageType == "" || strings.Contains(usageType, "/") {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected <source_id>/<usage_type>, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source_id"), sourceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("usage_type"), usageType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), sourceProvisioningPolicyID(sourceID, usageType))...)
}

var _ datasource.DataSource = &SourceProvisioningPolicyDataSource{}

func NewSourceProvisioningPolicyDataSource() datasource.DataSource {
	return &SourceProvisioningPolicyDataSource{}
}

type SourceProvisioningPolicyDataSource struct {
	client *Config
}

func (d *SourceProvisioningPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_provisioning_policy"
}

func (d *SourceProvisioningPolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up the provisioning policy of a source by usage type.",
		Attributes: map[string]dsschema.Attribute{
			"id":          dsschema.StringAttribute{MarkdownDescription: "ID in the format `<source_id>/<usage_type>`.", Computed: true},
			"source_id":   dsschema.StringAttribute{MarkdownDescription: "ID of the source.", Required: true},
			"usage_type":  dsschema.StringAttribute{MarkdownDescription: "Provisioning operation the policy applies to, e.g. `CREATE`.", Required: true},
			"name":        dsschema.StringAttribute{MarkdownDescription: "Provisioning policy name.", Computed: true},
			"description": dsschema.StringAttribute{MarkdownDescription: "Provisioning policy description.", Computed: true},
			"fields_json": dsschema.StringAttribute{MarkdownDescription: "Policy fields as a JSON array.", Computed: true},
		},
	}
}

func (d *SourceProvisioningPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourceProvisioningPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SourceProvisioningPolicyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	policy, err := client.GetSourceProvisioningPolicy(ctx, data.SourceID.ValueString(), data.UsageType.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("usage_type"), "Provisioning policy not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read provisioning policy: %s", err))
		return
	}
	data.Description = types.StringValue(policy.Description)
	setSourceProvisioningPolicyState(&data, policy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
