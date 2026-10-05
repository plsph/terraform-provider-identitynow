package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// segmentVisibilityDepth is the number of nested expression levels supported in visibility criteria.
const segmentVisibilityDepth = 3

var _ resource.Resource = &SegmentResource{}
var _ resource.ResourceWithImportState = &SegmentResource{}
var _ resource.ResourceWithValidateConfig = &SegmentResource{}

func NewSegmentResource() resource.Resource {
	return &SegmentResource{}
}

type SegmentResource struct {
	client *Config
}

type SegmentResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	Owner                  types.List   `tfsdk:"owner"`
	VisibilityCriteria     types.List   `tfsdk:"visibility_criteria"`
	VisibilityCriteriaJSON types.String `tfsdk:"visibility_criteria_json"`
	Active                 types.Bool   `tfsdk:"active"`
	Created                types.String `tfsdk:"created"`
	Modified               types.String `tfsdk:"modified"`
}

type SegmentVisibilityCriteriaModel struct {
	Expression types.List `tfsdk:"expression"`
}

type SegmentVisibilityExpressionModel struct {
	Operator  types.String `tfsdk:"operator"`
	Attribute types.String `tfsdk:"attribute"`
	Value     types.List   `tfsdk:"value"`
	Children  types.List   `tfsdk:"children"`
}

// SegmentVisibilityLeafExpressionModel is the expression at the deepest level, which has no children.
type SegmentVisibilityLeafExpressionModel struct {
	Operator  types.String `tfsdk:"operator"`
	Attribute types.String `tfsdk:"attribute"`
	Value     types.List   `tfsdk:"value"`
}

type SegmentVisibilityValueModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

func (r *SegmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segment"
}

func (r *SegmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SailPoint identity segment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The segment ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The segment business name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The segment description.",
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the segment is active. Inactive segments have no effect. Defaults to `false`.",
			},
			"created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The segment creation timestamp.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"modified": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The segment modification timestamp.",
			},
			"visibility_criteria_json": schema.StringAttribute{
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
				MarkdownDescription: "Visibility criteria as a JSON object following the SailPoint Visibility Criteria schema. Must be a JSON object. Conflicts with `visibility_criteria`. The value is compared semantically, so differences in whitespace, key order or unset fields don't produce a diff, and it is refreshed from the API, so changes made outside Terraform are detected.",
			},
		},
		Blocks: map[string]schema.Block{
			"visibility_criteria": visibilityCriteriaBlock(segmentVisibilityDepth, false),
			"owner": schema.ListNestedBlock{
				MarkdownDescription: "The segment owner.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Required: true, MarkdownDescription: "Owner identity ID."},
						"type": schema.StringAttribute{Required: true, MarkdownDescription: "Owner type, `IDENTITY`."},
						"name": schema.StringAttribute{Required: true, MarkdownDescription: "Owner name."},
					},
				},
			},
		},
	}
}

func (r *SegmentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var criteria types.List
	var criteriaJSON types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("visibility_criteria"), &criteria)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("visibility_criteria_json"), &criteriaJSON)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if visibilityCriteriaConfigured(criteria) && visibilityCriteriaJSONConfigured(criteriaJSON) {
		resp.Diagnostics.AddAttributeError(
			path.Root("visibility_criteria_json"),
			"Conflicting visibility criteria configuration",
			"Only one of visibility_criteria or visibility_criteria_json may be configured.",
		)
	}
}

func visibilityCriteriaBlock(depth int, computed bool) schema.Block {
	object := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"expression": visibilityExpressionBlock(depth, computed),
	}}
	description := "Visibility criteria following the SailPoint Visibility Criteria schema. Conflicts with `visibility_criteria_json`."
	if computed {
		return schema.ListNestedBlock{MarkdownDescription: description, NestedObject: object}
	}
	return schema.ListNestedBlock{MarkdownDescription: description, NestedObject: object}
}

func visibilityExpressionBlock(depth int, computed bool) schema.Block {
	attribute := func(description string) schema.StringAttribute {
		if computed {
			return schema.StringAttribute{Computed: true, MarkdownDescription: description}
		}
		return schema.StringAttribute{Optional: true, MarkdownDescription: description}
	}
	object := schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"operator":  attribute("Operator, e.g. `EQUALS`, `AND` or `OR`."),
			"attribute": attribute("Identity attribute to compare, for comparison operators."),
		},
		Blocks: map[string]schema.Block{
			"value": schema.ListNestedBlock{
				MarkdownDescription: "Value to compare with.",
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"type":  attribute("Value type, e.g. `STRING`."),
					"value": attribute("The value."),
				}},
			},
		},
	}
	if depth > 1 {
		object.Blocks["children"] = schema.ListNestedBlock{
			MarkdownDescription: "Child criteria for `AND` and `OR` operators. Each `children` block contains an `expression` block with the same arguments. Supports up to 3 levels of nesting.",
			NestedObject: schema.NestedBlockObject{Blocks: map[string]schema.Block{
				"expression": visibilityExpressionBlock(depth-1, computed),
			}},
		}
	}
	return schema.ListNestedBlock{MarkdownDescription: "Expression block.", NestedObject: object}
}

func (r *SegmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func segmentVisibilityValue(ctx context.Context, value types.List, diagnostics *diag.Diagnostics) *SegmentVisibilityCriteria {
	if value.IsNull() || value.IsUnknown() || len(value.Elements()) == 0 {
		return nil
	}
	var models []SegmentVisibilityCriteriaModel
	diagnostics.Append(value.ElementsAs(ctx, &models, false)...)
	if diagnostics.HasError() || len(models) == 0 {
		return nil
	}
	return &SegmentVisibilityCriteria{
		Expression: segmentVisibilityExpressionListValue(ctx, models[0].Expression, segmentVisibilityDepth, diagnostics),
	}
}

func segmentVisibilityJSONValue(raw types.String, diagnostics *diag.Diagnostics) *SegmentVisibilityCriteria {
	if raw.IsNull() || raw.IsUnknown() || raw.ValueString() == "" {
		return nil
	}
	var value SegmentVisibilityCriteria
	if err := json.Unmarshal([]byte(raw.ValueString()), &value); err != nil {
		diagnostics.AddAttributeError(path.Root("visibility_criteria_json"), "Invalid visibility criteria JSON", err.Error())
		return nil
	}
	return &value
}

// segmentVisibilityJSONState returns the state value for visibility_criteria_json. Both values are
// compared after decoding into the API struct, so formatting, key order and unset fields do not
// cause a diff, while changes made outside Terraform are detected.
func segmentVisibilityJSONState(prior types.String, criteria *SegmentVisibilityCriteria) types.String {
	fromAPI, err := json.Marshal(criteria)
	if err != nil || criteria == nil {
		return types.StringNull()
	}
	var priorCriteria SegmentVisibilityCriteria
	if err := json.Unmarshal([]byte(prior.ValueString()), &priorCriteria); err == nil {
		if normalizedPrior, err := json.Marshal(&priorCriteria); err == nil && string(normalizedPrior) == string(fromAPI) {
			return prior
		}
	}
	return types.StringValue(string(fromAPI))
}

func visibilityCriteriaConfigured(value types.List) bool {
	return !value.IsNull() && !value.IsUnknown() && len(value.Elements()) > 0
}

func visibilityCriteriaJSONConfigured(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueString() != ""
}

// segmentVisibilityExpressionListValue converts the single-element expression list at the given depth.
// The deepest level has no children attribute, so it is decoded into the leaf model.
func segmentVisibilityExpressionListValue(ctx context.Context, list types.List, depth int, diagnostics *diag.Diagnostics) *SegmentVisibilityExpression {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil
	}
	if depth <= 1 {
		var leaves []SegmentVisibilityLeafExpressionModel
		diagnostics.Append(list.ElementsAs(ctx, &leaves, false)...)
		if len(leaves) == 0 {
			return nil
		}
		return segmentVisibilityLeafValue(ctx, leaves[0].Operator, leaves[0].Attribute, leaves[0].Value, diagnostics)
	}
	var expressions []SegmentVisibilityExpressionModel
	diagnostics.Append(list.ElementsAs(ctx, &expressions, false)...)
	if len(expressions) == 0 {
		return nil
	}
	model := expressions[0]
	expression := segmentVisibilityLeafValue(ctx, model.Operator, model.Attribute, model.Value, diagnostics)
	if !model.Children.IsNull() && !model.Children.IsUnknown() && len(model.Children.Elements()) > 0 {
		var children []SegmentVisibilityCriteriaModel
		diagnostics.Append(model.Children.ElementsAs(ctx, &children, false)...)
		for _, child := range children {
			if childExpression := segmentVisibilityExpressionListValue(ctx, child.Expression, depth-1, diagnostics); childExpression != nil {
				expression.Children = append(expression.Children, childExpression)
			}
		}
	}
	return expression
}

func segmentVisibilityLeafValue(ctx context.Context, operator, attribute types.String, value types.List, diagnostics *diag.Diagnostics) *SegmentVisibilityExpression {
	expression := &SegmentVisibilityExpression{Operator: operator.ValueString(), Attribute: attribute.ValueString()}
	if !value.IsNull() && !value.IsUnknown() && len(value.Elements()) > 0 {
		var values []SegmentVisibilityValueModel
		diagnostics.Append(value.ElementsAs(ctx, &values, false)...)
		if len(values) > 0 {
			expression.Value = &SegmentVisibilityValue{Type: values[0].Type.ValueString(), Value: values[0].Value.ValueString()}
		}
	}
	return expression
}

func segmentVisibilityCriteriaState(ctx context.Context, criteria *SegmentVisibilityCriteria, diagnostics *diag.Diagnostics) types.List {
	criteriaType := segmentVisibilityCriteriaObjectType(segmentVisibilityDepth)
	if criteria == nil || criteria.Expression == nil {
		return types.ListNull(criteriaType)
	}
	model := SegmentVisibilityCriteriaModel{
		Expression: segmentVisibilityExpressionState(ctx, criteria.Expression, segmentVisibilityDepth, diagnostics),
	}
	value, diags := types.ListValueFrom(ctx, criteriaType, []SegmentVisibilityCriteriaModel{model})
	diagnostics.Append(diags...)
	return value
}

func segmentVisibilityExpressionState(ctx context.Context, expression *SegmentVisibilityExpression, depth int, diagnostics *diag.Diagnostics) types.List {
	expressionType := segmentVisibilityExpressionObjectType(depth)
	if expression == nil {
		return types.ListNull(expressionType)
	}
	operator := stringValueOrNull(expression.Operator)
	attribute := stringValueOrNull(expression.Attribute)
	value := segmentVisibilityValueState(ctx, expression.Value, diagnostics)
	if depth <= 1 {
		result, diags := types.ListValueFrom(ctx, expressionType, []SegmentVisibilityLeafExpressionModel{{
			Operator: operator, Attribute: attribute, Value: value,
		}})
		diagnostics.Append(diags...)
		return result
	}
	model := SegmentVisibilityExpressionModel{Operator: operator, Attribute: attribute, Value: value}
	childType := segmentVisibilityCriteriaObjectType(depth - 1)
	if len(expression.Children) > 0 {
		children := make([]SegmentVisibilityCriteriaModel, len(expression.Children))
		for i, child := range expression.Children {
			children[i] = SegmentVisibilityCriteriaModel{Expression: segmentVisibilityExpressionState(ctx, child, depth-1, diagnostics)}
		}
		childList, diags := types.ListValueFrom(ctx, childType, children)
		diagnostics.Append(diags...)
		model.Children = childList
	} else {
		model.Children, _ = types.ListValue(childType, []attr.Value{})
	}
	result, diags := types.ListValueFrom(ctx, expressionType, []SegmentVisibilityExpressionModel{model})
	diagnostics.Append(diags...)
	return result
}

func segmentVisibilityValueState(ctx context.Context, value *SegmentVisibilityValue, diagnostics *diag.Diagnostics) types.List {
	valueType := types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "value": types.StringType}}
	if value == nil {
		return types.ListNull(valueType)
	}
	result, diags := types.ListValueFrom(ctx, valueType, []SegmentVisibilityValueModel{{
		Type: types.StringValue(value.Type), Value: types.StringValue(value.Value),
	}})
	diagnostics.Append(diags...)
	return result
}

func segmentVisibilityCriteriaObjectType(depth int) types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"expression": types.ListType{ElemType: segmentVisibilityExpressionObjectType(depth)},
	}}
}

func segmentVisibilityExpressionObjectType(depth int) types.ObjectType {
	attributes := map[string]attr.Type{
		"operator":  types.StringType,
		"attribute": types.StringType,
		"value":     types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "value": types.StringType}}},
	}
	if depth > 1 {
		attributes["children"] = types.ListType{ElemType: segmentVisibilityCriteriaObjectType(depth - 1)}
	}
	return types.ObjectType{AttrTypes: attributes}
}

func segmentOwnerValue(ctx context.Context, owner types.List, diagnostics *diag.Diagnostics) *ObjectInfo {
	if owner.IsNull() || owner.IsUnknown() {
		return nil
	}
	var owners []OwnerModel
	diagnostics.Append(owner.ElementsAs(ctx, &owners, false)...)
	if len(owners) == 0 {
		return nil
	}
	return &ObjectInfo{ID: owners[0].ID.ValueString(), Type: owners[0].Type.ValueString(), Name: owners[0].Name.ValueString()}
}

func (r *SegmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	visibilityCriteria := segmentVisibilityValue(ctx, data.VisibilityCriteria, &resp.Diagnostics)
	if visibilityCriteria == nil {
		visibilityCriteria = segmentVisibilityJSONValue(data.VisibilityCriteriaJSON, &resp.Diagnostics)
	}
	segment := &Segment{
		Name:               data.Name.ValueString(),
		Description:        data.Description.ValueString(),
		Owner:              segmentOwnerValue(ctx, data.Owner, &resp.Diagnostics),
		VisibilityCriteria: visibilityCriteria,
		Active:             data.Active.ValueBool(),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	newSegment, err := client.CreateSegment(ctx, segment)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create segment: %s", err))
		return
	}
	data.ID = types.StringValue(newSegment.ID)
	data.Created = types.StringValue(newSegment.Created)
	data.Modified = types.StringValue(newSegment.Modified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SegmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	segment, err := client.GetSegment(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.Name = types.StringValue(segment.Name)
	if segment.Description != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(segment.Description)
	}
	data.Active = types.BoolValue(segment.Active)
	data.Created = types.StringValue(segment.Created)
	data.Modified = types.StringValue(segment.Modified)
	data.Owner = segmentOwnerState(ctx, segment.Owner, &resp.Diagnostics)
	if visibilityCriteriaJSONConfigured(data.VisibilityCriteriaJSON) {
		data.VisibilityCriteria = types.ListNull(segmentVisibilityCriteriaObjectType(segmentVisibilityDepth))
		data.VisibilityCriteriaJSON = segmentVisibilityJSONState(data.VisibilityCriteriaJSON, segment.VisibilityCriteria)
	} else {
		data.VisibilityCriteria = segmentVisibilityCriteriaState(ctx, segment.VisibilityCriteria, &resp.Diagnostics)
		data.VisibilityCriteriaJSON = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func segmentOwnerState(ctx context.Context, owner *ObjectInfo, diagnostics *diag.Diagnostics) types.List {
	ownerType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id": types.StringType, "type": types.StringType, "name": types.StringType,
	}}
	if owner == nil {
		return types.ListNull(ownerType)
	}
	owners, diags := types.ListValueFrom(ctx, ownerType, []OwnerModel{{
		ID: types.StringValue(fmt.Sprintf("%v", owner.ID)), Type: types.StringValue(owner.Type), Name: types.StringValue(owner.Name),
	}})
	diagnostics.Append(diags...)
	return owners
}

func (r *SegmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SegmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	var patches []*UpdateSegment
	replace := func(changed bool, path string, value interface{}) {
		if changed {
			patches = append(patches, &UpdateSegment{Op: "replace", Path: path, Value: value})
		}
	}
	replace(!data.Name.Equal(state.Name), "/name", data.Name.ValueString())
	replace(!data.Description.Equal(state.Description), "/description", data.Description.ValueString())
	replace(!data.Active.Equal(state.Active), "/active", data.Active.ValueBool())
	if !data.Owner.Equal(state.Owner) {
		if owner := segmentOwnerValue(ctx, data.Owner, &resp.Diagnostics); owner != nil {
			replace(true, "/owner", owner)
		} else {
			patches = append(patches, &UpdateSegment{Op: "remove", Path: "/owner"})
		}
	}
	if !data.VisibilityCriteria.Equal(state.VisibilityCriteria) || !data.VisibilityCriteriaJSON.Equal(state.VisibilityCriteriaJSON) {
		criteria := segmentVisibilityValue(ctx, data.VisibilityCriteria, &resp.Diagnostics)
		if criteria == nil {
			criteria = segmentVisibilityJSONValue(data.VisibilityCriteriaJSON, &resp.Diagnostics)
		}
		if criteria != nil {
			replace(true, "/visibilityCriteria", criteria)
		} else {
			// Criteria were removed from the configuration, so the segment no longer restricts visibility.
			patches = append(patches, &UpdateSegment{Op: "remove", Path: "/visibilityCriteria"})
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if len(patches) == 0 {
		data.Modified = state.Modified
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.UpdateSegment(ctx, data.ID.ValueString(), patches)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update segment: %s", err))
		return
	}
	// created keeps its planned (prior state) value
	data.Modified = types.StringValue(updated.Modified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SegmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteSegment(ctx, data.ID.ValueString()); err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete segment: %s", err))
	}
}

func (r *SegmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
