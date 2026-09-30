package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewMultihostResource)
	registerDataSource(NewMultihostDataSource)
}

const (
	// multihostMaxSourcesPerAggGroupKey and multihostMaxAllowedSourcesKey are the connector
	// attributes modelled as typed attributes.
	multihostMaxSourcesPerAggGroupKey = "maxSourcesPerAggGroup"
	multihostMaxAllowedSourcesKey     = "maxAllowedSources"
)

// Multihost is a Multi-Host Integration as used by the /v2026/multihosts API.
type Multihost struct {
	ID                  string                 `json:"id,omitempty"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	Owner               *MultihostRef          `json:"owner,omitempty"`
	Cluster             *MultihostRef          `json:"cluster,omitempty"`
	Connector           string                 `json:"connector"`
	ConnectorAttributes map[string]interface{} `json:"connectorAttributes,omitempty"`
	ManagementWorkgroup *MultihostRef          `json:"managementWorkgroup,omitempty"`
	Type                string                 `json:"type,omitempty"`
	Created             string                 `json:"created,omitempty"`
	Modified            string                 `json:"modified,omitempty"`
}

// MultihostRef is a reference to an identity, cluster or governance group.
type MultihostRef struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func (c *Client) GetMultihost(ctx context.Context, id string) (*Multihost, error) {
	var multihost Multihost
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/multihosts/%s", id), nil, &multihost); err != nil {
		return nil, err
	}
	return &multihost, nil
}

func (c *Client) CreateMultihost(ctx context.Context, multihost *Multihost) (*Multihost, error) {
	var created Multihost
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/multihosts", multihost, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// PatchMultihost applies JSON Patch operations. The API only supports add and replace, and does
// not document a response body, so the updated object is read separately.
func (c *Client) PatchMultihost(ctx context.Context, id string, ops []jsonPatchOp) error {
	return c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/multihosts/%s", id), ops, nil, withJSONPatch())
}

func (c *Client) DeleteMultihost(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/multihosts/%s", id), nil, nil)
}

var _ resource.Resource = &MultihostResource{}
var _ resource.ResourceWithImportState = &MultihostResource{}
var _ resource.ResourceWithValidateConfig = &MultihostResource{}

func NewMultihostResource() resource.Resource {
	return &MultihostResource{}
}

type MultihostResource struct {
	client *Config
}

type MultihostModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	Connector               types.String `tfsdk:"connector"`
	ConnectorAttributesJSON types.String `tfsdk:"connector_attributes_json"`
	MaxSourcesPerAggGroup   types.Int64  `tfsdk:"max_sources_per_agg_group"`
	MaxAllowedSources       types.Int64  `tfsdk:"max_allowed_sources"`
	Type                    types.String `tfsdk:"type"`
	Created                 types.String `tfsdk:"created"`
	Modified                types.String `tfsdk:"modified"`
	Owner                   types.List   `tfsdk:"owner"`
	Cluster                 types.List   `tfsdk:"cluster"`
	ManagementWorkgroup     types.List   `tfsdk:"management_workgroup"`
}

// MultihostRefModel is a reference block with a configured ID and the name resolved by the API.
type MultihostRefModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

var multihostRefObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":   types.StringType,
	"name": types.StringType,
}}

func (r *MultihostResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_multihost"
}

func multihostRefBlock(description string, min int) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: description,
		Validators:          []validator.List{listSizeBetween(min, 1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "ID of the referenced object.",
					Required:            true,
				},
				"name": schema.StringAttribute{
					MarkdownDescription: "Name of the referenced object, resolved by IdentityNow.",
					Computed:            true,
				},
			},
		},
	}
}

func (r *MultihostResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Multi-Host Integration, which holds sources of the same type. Uploading connector files and creating the sources of the integration are not supported.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Multi-Host Integration ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the Multi-Host Integration.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the Multi-Host Integration.",
				Required:            true,
			},
			"connector": schema.StringAttribute{
				MarkdownDescription: "Connector script name, e.g. `multihost-microsoft-sql-server`. Changing it forces a new Multi-Host Integration.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"connector_attributes_json": schema.StringAttribute{
				MarkdownDescription: "Connector attributes of the Multi-Host Integration as a JSON object, e.g. `multiHostAttributes` with `authType`, `user` and `password`. Use `jsonencode()` for convenience. The value is sensitive, since it may contain credentials. Only the configured keys are managed and compared, also inside nested objects such as `multiHostAttributes`: attributes added by IdentityNow are ignored, and values the API does not return, or returns masked or encrypted, keep the configured value. Updates only send the changed values, so attributes of a nested object that are not configured are kept. Keys removed from the configuration are left unchanged in IdentityNow, since the API cannot remove attributes. Use `max_sources_per_agg_group` and `max_allowed_sources` instead of the `maxSourcesPerAggGroup` and `maxAllowedSources` keys.",
				Optional:            true,
				Sensitive:           true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"max_sources_per_agg_group": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of sources per aggregation group (connector attribute `maxSourcesPerAggGroup`). When not set, the value is not managed.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.Int64{int64AtLeastValidator{min: 1}},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"max_allowed_sources": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of sources in the Multi-Host Integration (connector attribute `maxAllowedSources`). When not set, the value is not managed.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.Int64{int64AtLeastValidator{min: 1}},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of system managed, e.g. `Multi-Host - Microsoft SQL Server`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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
			"owner":                multihostRefBlock("Identity that owns the Multi-Host Integration. Exactly one block is required.", 1),
			"cluster":              multihostRefBlock("Virtual appliance cluster of the Multi-Host Integration. At most one block.", 0),
			"management_workgroup": multihostRefBlock("Governance group that manages the Multi-Host Integration. At most one block.", 0),
		},
	}
}

func (r *MultihostResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var raw types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("connector_attributes_json"), &raw)...)
	attributes, err := multihostConnectorAttributesValue(raw)
	if err != nil {
		return // reported by the attribute validator
	}
	for _, key := range []string{multihostMaxSourcesPerAggGroupKey, multihostMaxAllowedSourcesKey} {
		if _, ok := attributes[key]; ok {
			resp.Diagnostics.AddAttributeError(path.Root("connector_attributes_json"), "Invalid connector attributes",
				fmt.Sprintf("Set %q with the max_sources_per_agg_group or max_allowed_sources attribute instead.", key))
		}
	}
}

func (r *MultihostResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// multihostConnectorAttributesValue decodes connector_attributes_json; null and unknown are empty.
func multihostConnectorAttributesValue(value types.String) (map[string]interface{}, error) {
	attributes := map[string]interface{}{}
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return attributes, nil
	}
	err := json.Unmarshal([]byte(value.ValueString()), &attributes)
	return attributes, err
}

// multihostRefValue converts a reference block to the API reference with the given type.
func multihostRefValue(ctx context.Context, list types.List, refType string, diags *diag.Diagnostics) *MultihostRef {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var models []MultihostRefModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if len(models) == 0 {
		return nil
	}
	ref := &MultihostRef{Type: refType, ID: models[0].ID.ValueString()}
	if !models[0].Name.IsNull() && !models[0].Name.IsUnknown() {
		ref.Name = models[0].Name.ValueString()
	}
	return ref
}

// multihostRefState returns the state of a reference block. With refresh, the API reference is
// used. Otherwise the planned ID is kept and an unknown name is resolved from the API reference.
func multihostRefState(ctx context.Context, planned types.List, ref *MultihostRef, refresh bool, diags *diag.Diagnostics) types.List {
	if refresh {
		if ref == nil || ref.ID == "" {
			return types.ListNull(multihostRefObjectType)
		}
		list, d := types.ListValueFrom(ctx, multihostRefObjectType, []MultihostRefModel{{ID: types.StringValue(ref.ID), Name: types.StringValue(ref.Name)}})
		diags.Append(d...)
		return list
	}
	if planned.IsNull() || planned.IsUnknown() {
		return types.ListNull(multihostRefObjectType)
	}
	var models []MultihostRefModel
	diags.Append(planned.ElementsAs(ctx, &models, false)...)
	for i := range models {
		if models[i].Name.IsUnknown() {
			name := ""
			if ref != nil && ref.ID == models[i].ID.ValueString() {
				name = ref.Name
			}
			models[i].Name = types.StringValue(name)
		}
	}
	list, d := types.ListValueFrom(ctx, multihostRefObjectType, models)
	diags.Append(d...)
	return list
}

// multihostRefID returns the configured ID of a reference block, or "" when the block is not set.
func multihostRefID(ctx context.Context, list types.List, diags *diag.Diagnostics) string {
	if ref := multihostRefValue(ctx, list, "", diags); ref != nil {
		return ref.ID
	}
	return ""
}

// multihostFromModel builds the create request from the plan.
func multihostFromModel(ctx context.Context, data MultihostModel, diags *diag.Diagnostics) *Multihost {
	multihost := &Multihost{
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueString(),
		Connector:           data.Connector.ValueString(),
		Owner:               multihostRefValue(ctx, data.Owner, "IDENTITY", diags),
		Cluster:             multihostRefValue(ctx, data.Cluster, "CLUSTER", diags),
		ManagementWorkgroup: multihostRefValue(ctx, data.ManagementWorkgroup, "GOVERNANCE_GROUP", diags),
	}
	attributes, err := multihostConnectorAttributesValue(data.ConnectorAttributesJSON)
	if err != nil {
		diags.AddAttributeError(path.Root("connector_attributes_json"), "Invalid JSON", err.Error())
		return nil
	}
	if v := int64Pointer(data.MaxSourcesPerAggGroup); v != nil {
		attributes[multihostMaxSourcesPerAggGroupKey] = *v
	}
	if v := int64Pointer(data.MaxAllowedSources); v != nil {
		attributes[multihostMaxAllowedSourcesKey] = *v
	}
	if len(attributes) > 0 {
		multihost.ConnectorAttributes = attributes
	}
	return multihost
}

// multihostJSONPointerEscape escapes a key for use in a JSON Pointer path.
func multihostJSONPointerEscape(key string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(key)
}

// multihostSortedKeys returns the keys of a map in sorted order.
func multihostSortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// multihostAttributesPatch returns add operations for the configured connector attributes that
// changed. Changes are detected against the prior state (which holds the configured values, also
// for secrets the API masks). Nested objects are patched per changed leaf when the object already
// exists in IdentityNow (current, the attributes returned by the API), so attributes that are not
// configured, e.g. connector_files in multiHostAttributes, are not overwritten. When the object does
// not exist yet, or is not an object in IdentityNow, the whole configured value is added.
func multihostAttributesPatch(prefix string, planned, prior, current map[string]interface{}) []jsonPatchOp {
	var ops []jsonPatchOp
	for _, key := range multihostSortedKeys(planned) {
		plannedValue := planned[key]
		keyPath := prefix + "/" + multihostJSONPointerEscape(key)
		priorValue, inPrior := prior[key]
		if plannedMap, ok := plannedValue.(map[string]interface{}); ok {
			if currentMap, ok := current[key].(map[string]interface{}); ok {
				priorMap, _ := priorValue.(map[string]interface{})
				ops = append(ops, multihostAttributesPatch(keyPath, plannedMap, priorMap, currentMap)...)
				continue
			}
		}
		if inPrior && reflect.DeepEqual(priorValue, plannedValue) {
			continue
		}
		ops = append(ops, jsonPatchOp{Op: "add", Path: keyPath, Value: plannedValue})
	}
	return ops
}

// multihostRemovedAttributes returns the connector attributes, also nested ones as JSON Pointer
// style paths relative to connectorAttributes, that are in the prior state but not in the plan.
func multihostRemovedAttributes(prefix string, planned, prior map[string]interface{}) []string {
	var removed []string
	for _, key := range multihostSortedKeys(prior) {
		plannedValue, ok := planned[key]
		if !ok {
			removed = append(removed, prefix+key)
			continue
		}
		plannedMap, plannedIsMap := plannedValue.(map[string]interface{})
		priorMap, priorIsMap := prior[key].(map[string]interface{})
		if plannedIsMap && priorIsMap {
			removed = append(removed, multihostRemovedAttributes(prefix+key+"/", plannedMap, priorMap)...)
		}
	}
	return removed
}

// multihostPatch returns the JSON Patch operations for the changed attributes and the connector
// attribute keys that were removed from the configuration. The API only supports add and replace,
// so removed references are replaced with null and removed connector attributes are left as is.
// current holds the connector attributes currently returned by the API; it decides whether nested
// objects are patched per leaf (see multihostAttributesPatch).
func multihostPatch(ctx context.Context, plan, state MultihostModel, current map[string]interface{}, diags *diag.Diagnostics) ([]jsonPatchOp, []string) {
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	refs := []struct {
		planned, prior types.List
		path, refType  string
	}{
		{plan.Owner, state.Owner, "/owner", "IDENTITY"},
		{plan.Cluster, state.Cluster, "/cluster", "CLUSTER"},
		{plan.ManagementWorkgroup, state.ManagementWorkgroup, "/managementWorkgroup", "GOVERNANCE_GROUP"},
	}
	for _, ref := range refs {
		if multihostRefID(ctx, ref.planned, diags) == multihostRefID(ctx, ref.prior, diags) {
			continue
		}
		if value := multihostRefValue(ctx, ref.planned, ref.refType, diags); value != nil {
			value.Name = ""
			b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: ref.path, Value: value})
		} else {
			b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: ref.path, Value: json.RawMessage("null")})
		}
	}

	planned, err := multihostConnectorAttributesValue(plan.ConnectorAttributesJSON)
	if err != nil {
		diags.AddAttributeError(path.Root("connector_attributes_json"), "Invalid JSON", err.Error())
		return nil, nil
	}
	prior, _ := multihostConnectorAttributesValue(state.ConnectorAttributesJSON)
	b.ops = append(b.ops, multihostAttributesPatch("/connectorAttributes", planned, prior, current)...)
	removed := multihostRemovedAttributes("", planned, prior)

	addInt := func(planned, prior types.Int64, key string) {
		if planned.IsNull() || planned.IsUnknown() || planned.Equal(prior) {
			return
		}
		b.ops = append(b.ops, jsonPatchOp{Op: "add", Path: "/connectorAttributes/" + key, Value: planned.ValueInt64()})
	}
	addInt(plan.MaxSourcesPerAggGroup, state.MaxSourcesPerAggGroup, multihostMaxSourcesPerAggGroupKey)
	addInt(plan.MaxAllowedSources, state.MaxAllowedSources, multihostMaxAllowedSourcesKey)
	return b.ops, removed
}

// multihostMaskedValue reports whether an API value looks like a masked or encrypted secret.
func multihostMaskedValue(value interface{}) bool {
	s, ok := value.(string)
	if !ok || s == "" {
		return false
	}
	return strings.Trim(s, "*") == "" || strings.Contains(s, ":ENC:")
}

// multihostMergeAttribute merges an API connector attribute value into the configured value.
// Objects are merged recursively and keep only the configured keys; configured keys the API does
// not return keep the configured value. Arrays of the same length are merged per element. Other
// values are taken from the API, except masked or encrypted secrets, which keep the configured
// value, at every nesting level.
func multihostMergeAttribute(configured, api interface{}) interface{} {
	switch c := configured.(type) {
	case map[string]interface{}:
		if a, ok := api.(map[string]interface{}); ok {
			result := make(map[string]interface{}, len(c))
			for key, value := range c {
				result[key] = value
				if apiValue, ok := a[key]; ok {
					result[key] = multihostMergeAttribute(value, apiValue)
				}
			}
			return result
		}
	case []interface{}:
		if a, ok := api.([]interface{}); ok && len(a) == len(c) {
			result := make([]interface{}, len(c))
			for i := range c {
				result[i] = multihostMergeAttribute(c[i], a[i])
			}
			return result
		}
	}
	if multihostMaskedValue(api) {
		return configured
	}
	return api
}

// multihostConnectorAttributesState returns the state of connector_attributes_json during refresh.
// Only the configured keys are tracked, also in nested objects such as multiHostAttributes: other
// attributes returned by the API are ignored, and configured values that the API omits or masks are
// kept (see multihostMergeAttribute). A null prior value stays null.
func multihostConnectorAttributesState(prior types.String, attributes map[string]interface{}) types.String {
	if prior.IsNull() || prior.IsUnknown() || prior.ValueString() == "" {
		return types.StringNull()
	}
	configured, err := multihostConnectorAttributesValue(prior)
	if err != nil {
		return prior
	}
	if attributes == nil {
		return prior
	}
	result := multihostMergeAttribute(configured, attributes)
	if reflect.DeepEqual(result, configured) {
		return prior
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return prior
	}
	return types.StringValue(string(encoded))
}

// multihostInt64Attribute returns a numeric connector attribute, or nil when it is missing.
func multihostInt64Attribute(attributes map[string]interface{}, key string) *int64 {
	switch v := attributes[key].(type) {
	case float64:
		n := int64(v)
		return &n
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return &n
		}
	}
	return nil
}

// setMultihostState maps an API Multi-Host Integration onto the model. With refresh, values are
// taken from the API; otherwise known planned values are kept and only unknown values are resolved.
func setMultihostState(ctx context.Context, data *MultihostModel, multihost *Multihost, refresh bool, diags *diag.Diagnostics) {
	if refresh || data.ID.IsUnknown() {
		data.ID = types.StringValue(multihost.ID)
	}
	if refresh {
		data.Name = types.StringValue(multihost.Name)
		data.Description = types.StringValue(multihost.Description)
		data.Connector = types.StringValue(multihost.Connector)
		data.ConnectorAttributesJSON = multihostConnectorAttributesState(data.ConnectorAttributesJSON, multihost.ConnectorAttributes)
	}
	if refresh || data.MaxSourcesPerAggGroup.IsUnknown() {
		data.MaxSourcesPerAggGroup = types.Int64PointerValue(multihostInt64Attribute(multihost.ConnectorAttributes, multihostMaxSourcesPerAggGroupKey))
	}
	if refresh || data.MaxAllowedSources.IsUnknown() {
		data.MaxAllowedSources = types.Int64PointerValue(multihostInt64Attribute(multihost.ConnectorAttributes, multihostMaxAllowedSourcesKey))
	}
	if refresh || data.Type.IsUnknown() {
		data.Type = stringValueOrNull(multihost.Type)
	}
	if refresh || data.Created.IsUnknown() {
		data.Created = stringValueOrNull(multihost.Created)
	}
	data.Modified = stringValueOrNull(multihost.Modified)
	data.Owner = multihostRefState(ctx, data.Owner, multihost.Owner, refresh, diags)
	data.Cluster = multihostRefState(ctx, data.Cluster, multihost.Cluster, refresh, diags)
	data.ManagementWorkgroup = multihostRefState(ctx, data.ManagementWorkgroup, multihost.ManagementWorkgroup, refresh, diags)
}

// multihostResolveUnknown sets values that are still unknown after a failed read to null.
func multihostResolveUnknown(ctx context.Context, data *MultihostModel, diags *diag.Diagnostics) {
	for _, value := range []*types.String{&data.Type, &data.Created, &data.Modified} {
		if value.IsUnknown() {
			*value = types.StringNull()
		}
	}
	for _, value := range []*types.Int64{&data.MaxSourcesPerAggGroup, &data.MaxAllowedSources} {
		if value.IsUnknown() {
			*value = types.Int64Null()
		}
	}
	for _, list := range []*types.List{&data.Owner, &data.Cluster, &data.ManagementWorkgroup} {
		*list = multihostRefState(ctx, *list, nil, false, diags)
	}
}

func (r *MultihostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MultihostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	multihost := multihostFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateMultihost(ctx, multihost)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Multi-Host Integration: %s", err))
		return
	}
	setMultihostState(ctx, &data, created, false, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MultihostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MultihostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	multihost, err := client.GetMultihost(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Multi-Host Integration: %s", err))
		return
	}
	setMultihostState(ctx, &data, multihost, true, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MultihostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state MultihostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// The current connector attributes decide whether nested objects are patched per changed
	// value, so attributes that are not configured are kept.
	current, err := client.GetMultihost(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Multi-Host Integration before update: %s", err))
		return
	}
	ops, removed := multihostPatch(ctx, data, state, current.ConnectorAttributes, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(removed) > 0 {
		resp.Diagnostics.AddWarning("Connector attributes left unchanged",
			fmt.Sprintf("The Multi-Host Integration API cannot remove connector attributes. The attributes %s were removed from the configuration but are left unchanged in IdentityNow.", strings.Join(removed, ", ")))
	}
	if len(ops) > 0 {
		if err := client.PatchMultihost(ctx, data.ID.ValueString(), ops); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Multi-Host Integration: %s", err))
			return
		}
	}
	updated, err := client.GetMultihost(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Multi-Host Integration after update: %s", err))
		multihostResolveUnknown(ctx, &data, &resp.Diagnostics)
	} else {
		setMultihostState(ctx, &data, updated, false, &resp.Diagnostics)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MultihostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MultihostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteMultihost(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Multi-Host Integration: %s", err))
	}
}

func (r *MultihostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &MultihostDataSource{}

func NewMultihostDataSource() datasource.DataSource {
	return &MultihostDataSource{}
}

type MultihostDataSource struct {
	client *Config
}

func (d *MultihostDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_multihost"
}

func (d *MultihostDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	ref := func(description string) dsschema.ListNestedAttribute {
		return dsschema.ListNestedAttribute{
			MarkdownDescription: description,
			Computed:            true,
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"id":   dsschema.StringAttribute{MarkdownDescription: "ID of the referenced object.", Computed: true},
					"name": dsschema.StringAttribute{MarkdownDescription: "Name of the referenced object.", Computed: true},
				},
			},
		}
	}
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a Multi-Host Integration by ID.",
		Attributes: map[string]dsschema.Attribute{
			"id":                        dsschema.StringAttribute{MarkdownDescription: "Multi-Host Integration ID.", Required: true},
			"name":                      dsschema.StringAttribute{MarkdownDescription: "Name of the Multi-Host Integration.", Computed: true},
			"description":               dsschema.StringAttribute{MarkdownDescription: "Description of the Multi-Host Integration.", Computed: true},
			"connector":                 dsschema.StringAttribute{MarkdownDescription: "Connector script name.", Computed: true},
			"connector_attributes_json": dsschema.StringAttribute{MarkdownDescription: "All connector attributes as a JSON object. Sensitive, since it may contain credentials.", Computed: true, Sensitive: true},
			"max_sources_per_agg_group": dsschema.Int64Attribute{MarkdownDescription: "Maximum number of sources per aggregation group.", Computed: true},
			"max_allowed_sources":       dsschema.Int64Attribute{MarkdownDescription: "Maximum number of sources in the Multi-Host Integration.", Computed: true},
			"type":                      dsschema.StringAttribute{MarkdownDescription: "Type of system managed.", Computed: true},
			"created":                   dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"modified":                  dsschema.StringAttribute{MarkdownDescription: "Last modification date.", Computed: true},
			"owner":                     ref("Owner identity, a list with one element."),
			"cluster":                   ref("Virtual appliance cluster, a list with at most one element."),
			"management_workgroup":      ref("Management governance group, a list with at most one element."),
		},
	}
}

func (d *MultihostDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MultihostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MultihostModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	multihost, err := client.GetMultihost(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("id"), "Multi-Host Integration not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Multi-Host Integration: %s", err))
		return
	}
	setMultihostState(ctx, &data, multihost, true, &resp.Diagnostics)
	data.ConnectorAttributesJSON = jsonStringState(types.StringNull(), multihost.ConnectorAttributes)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
