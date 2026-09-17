package main

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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SegmentResource{}
var _ resource.ResourceWithImportState = &SegmentResource{}
var _ resource.ResourceWithModifyPlan = &SegmentResource{}

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
				Computed: true,
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
				MarkdownDescription: "An optional description for the segment.",
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the segment is active. Inactive segments have no effect.",
			},
			"created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The segment creation timestamp.",
			},
			"modified": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The segment modification timestamp.",
			},
			"visibility_criteria_json": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Visibility criteria as a JSON object. Conflicts with visibility_criteria.",
			},
		},
		Blocks: map[string]schema.Block{
			"visibility_criteria": visibilityCriteriaBlock(3, false),
			"owner": schema.ListNestedBlock{
				MarkdownDescription: "The segment owner.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Required: true},
						"type": schema.StringAttribute{Required: true},
						"name": schema.StringAttribute{Required: true},
					},
				},
			},
		},
	}
}

func (r *SegmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if visibilityCriteriaConfigured(data.VisibilityCriteria) && !data.VisibilityCriteriaJSON.IsNull() && !data.VisibilityCriteriaJSON.IsUnknown() && data.VisibilityCriteriaJSON.ValueString() != "" {
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
	if computed {
		return schema.ListNestedBlock{NestedObject: object}
	}
	return schema.ListNestedBlock{NestedObject: object}
}

func visibilityExpressionBlock(depth int, computed bool) schema.Block {
	attribute := func() schema.StringAttribute {
		if computed {
			return schema.StringAttribute{Computed: true}
		}
		return schema.StringAttribute{Optional: true}
	}
	object := schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"operator":  attribute(),
			"attribute": attribute(),
		},
		Blocks: map[string]schema.Block{
			"value": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"type":  attribute(),
					"value": attribute(),
				}},
			},
		},
	}
	if depth > 1 {
		object.Blocks["children"] = schema.ListNestedBlock{
			NestedObject: schema.NestedBlockObject{Blocks: map[string]schema.Block{
				"expression": visibilityExpressionBlock(depth-1, computed),
			}},
		}
	}
	return schema.ListNestedBlock{NestedObject: object}
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
	criteria := &SegmentVisibilityCriteria{}
	if !models[0].Expression.IsNull() && len(models[0].Expression.Elements()) > 0 {
		var expressions []SegmentVisibilityExpressionModel
		diagnostics.Append(models[0].Expression.ElementsAs(ctx, &expressions, false)...)
		if len(expressions) > 0 {
			criteria.Expression = segmentVisibilityExpressionValue(ctx, expressions[0], diagnostics)
		}
	}
	return criteria
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

func visibilityCriteriaConfigured(value types.List) bool {
	return !value.IsNull() && !value.IsUnknown() && len(value.Elements()) > 0
}

func visibilityCriteriaJSONConfigured(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueString() != ""
}

func segmentVisibilityExpressionValue(ctx context.Context, model SegmentVisibilityExpressionModel, diagnostics *diag.Diagnostics) *SegmentVisibilityExpression {
	expression := &SegmentVisibilityExpression{Operator: model.Operator.ValueString(), Attribute: model.Attribute.ValueString()}
	if !model.Value.IsNull() && len(model.Value.Elements()) > 0 {
		var values []SegmentVisibilityValueModel
		diagnostics.Append(model.Value.ElementsAs(ctx, &values, false)...)
		if len(values) > 0 {
			expression.Value = &SegmentVisibilityValue{Type: values[0].Type.ValueString(), Value: values[0].Value.ValueString()}
		}
	}
	if !model.Children.IsNull() && len(model.Children.Elements()) > 0 {
		var children []SegmentVisibilityCriteriaModel
		diagnostics.Append(model.Children.ElementsAs(ctx, &children, false)...)
		for _, child := range children {
			if !child.Expression.IsNull() && len(child.Expression.Elements()) > 0 {
				var childExpressions []SegmentVisibilityExpressionModel
				diagnostics.Append(child.Expression.ElementsAs(ctx, &childExpressions, false)...)
				if len(childExpressions) > 0 {
					expression.Children = append(expression.Children, segmentVisibilityExpressionValue(ctx, childExpressions[0], diagnostics))
				}
			}
		}
	}
	return expression
}

func segmentVisibilityCriteriaState(ctx context.Context, criteria *SegmentVisibilityCriteria, diagnostics *diag.Diagnostics) types.List {
	criteriaType := segmentVisibilityCriteriaObjectType(3)
	if criteria == nil || criteria.Expression == nil {
		return types.ListNull(criteriaType)
	}
	model := SegmentVisibilityCriteriaModel{
		Expression: segmentVisibilityExpressionState(ctx, criteria.Expression, 3, diagnostics),
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
	model := SegmentVisibilityExpressionModel{
		Operator:  types.StringValue(expression.Operator),
		Attribute: types.StringValue(expression.Attribute),
		Value:     segmentVisibilityValueState(ctx, expression.Value, diagnostics),
	}
	if depth > 1 && len(expression.Children) > 0 {
		children := make([]SegmentVisibilityCriteriaModel, len(expression.Children))
		for i, child := range expression.Children {
			children[i] = SegmentVisibilityCriteriaModel{Expression: segmentVisibilityExpressionState(ctx, child, depth-1, diagnostics)}
		}
		childType := segmentVisibilityCriteriaObjectType(depth - 1)
		value, diags := types.ListValueFrom(ctx, childType, children)
		diagnostics.Append(diags...)
		model.Children = value
	} else if depth > 1 {
		model.Children, _ = types.ListValue(segmentVisibilityCriteriaObjectType(depth-1), []attr.Value{})
	}
	value, diags := types.ListValueFrom(ctx, expressionType, []SegmentVisibilityExpressionModel{model})
	diagnostics.Append(diags...)
	return value
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
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.Name = types.StringValue(segment.Name)
	data.Description = types.StringValue(segment.Description)
	data.Active = types.BoolValue(segment.Active)
	data.Created = types.StringValue(segment.Created)
	data.Modified = types.StringValue(segment.Modified)
	data.Owner = segmentOwnerState(ctx, segment.Owner, &resp.Diagnostics)
	if visibilityCriteriaJSONConfigured(data.VisibilityCriteriaJSON) {
		data.VisibilityCriteria = types.ListNull(segmentVisibilityCriteriaObjectType(3))
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

	patches := []*UpdateSegment{
		{Op: "replace", Path: "/name", Value: data.Name.ValueString()},
		{Op: "replace", Path: "/description", Value: data.Description.ValueString()},
		{Op: "replace", Path: "/active", Value: data.Active.ValueBool()},
	}
	if owner := segmentOwnerValue(ctx, data.Owner, &resp.Diagnostics); owner != nil {
		patches = append(patches, &UpdateSegment{Op: "replace", Path: "/owner", Value: owner})
	}
	criteria := segmentVisibilityValue(ctx, data.VisibilityCriteria, &resp.Diagnostics)
	if criteria == nil {
		criteria = segmentVisibilityJSONValue(data.VisibilityCriteriaJSON, &resp.Diagnostics)
	}
	if criteria != nil {
		patches = append(patches, &UpdateSegment{Op: "replace", Path: "/visibilityCriteria", Value: criteria})
	}
	if resp.Diagnostics.HasError() {
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
	data.Created = types.StringValue(updated.Created)
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
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete segment: %s", err))
	}
}

func (r *SegmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
