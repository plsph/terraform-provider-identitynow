package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// --- Custom String Type (case-insensitive element) ---

// CaseInsensitiveStringType is a custom string type that compares values case-insensitively.
type CaseInsensitiveStringType struct {
	basetypes.StringType
}

func (t CaseInsensitiveStringType) Equal(o attr.Type) bool {
	other, ok := o.(CaseInsensitiveStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t CaseInsensitiveStringType) String() string {
	return "CaseInsensitiveStringType"
}

func (t CaseInsensitiveStringType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return CaseInsensitiveStringValue{StringValue: in}, nil
}

func (t CaseInsensitiveStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}
	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}
	return stringValuable, nil
}

func (t CaseInsensitiveStringType) ValueType(ctx context.Context) attr.Value {
	return CaseInsensitiveStringValue{}
}

// CaseInsensitiveStringValue is a custom string value that implements semantic equality
// by comparing strings case-insensitively.
type CaseInsensitiveStringValue struct {
	basetypes.StringValue
}

func (v CaseInsensitiveStringValue) Equal(o attr.Value) bool {
	other, ok := o.(CaseInsensitiveStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v CaseInsensitiveStringValue) Type(ctx context.Context) attr.Type {
	return CaseInsensitiveStringType{}
}

func (v CaseInsensitiveStringValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(CaseInsensitiveStringValue)
	if !ok {
		diags.AddError("Semantic Equality Check Error",
			"An unexpected value type was received.\n"+
				fmt.Sprintf("Expected: CaseInsensitiveStringValue, Got: %T", newValuable))
		return false, diags
	}

	return strings.EqualFold(v.ValueString(), newValue.ValueString()), diags
}

// --- Custom Set Type (case-insensitive set comparison) ---

// CaseInsensitiveStringSetType is a custom set type that compares elements case-insensitively.
type CaseInsensitiveStringSetType struct {
	basetypes.SetType
}

func (t CaseInsensitiveStringSetType) Equal(o attr.Type) bool {
	other, ok := o.(CaseInsensitiveStringSetType)
	if !ok {
		return false
	}
	return t.SetType.Equal(other.SetType)
}

func (t CaseInsensitiveStringSetType) String() string {
	return "CaseInsensitiveStringSetType"
}

func (t CaseInsensitiveStringSetType) ValueFromSet(ctx context.Context, in basetypes.SetValue) (basetypes.SetValuable, diag.Diagnostics) {
	return CaseInsensitiveStringSetValue{SetValue: in}, nil
}

func (t CaseInsensitiveStringSetType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.SetType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	setValue, ok := attrValue.(basetypes.SetValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}
	setValuable, diags := t.ValueFromSet(ctx, setValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting SetValue to SetValuable: %v", diags)
	}
	return setValuable, nil
}

func (t CaseInsensitiveStringSetType) ValueType(ctx context.Context) attr.Value {
	return CaseInsensitiveStringSetValue{}
}

// CaseInsensitiveStringSetValue is a custom set value that implements SetSemanticEquals.
// Two sets are semantically equal if they contain the same elements compared case-insensitively.
type CaseInsensitiveStringSetValue struct {
	basetypes.SetValue
}

func (v CaseInsensitiveStringSetValue) Equal(o attr.Value) bool {
	other, ok := o.(CaseInsensitiveStringSetValue)
	if !ok {
		return false
	}
	return v.SetValue.Equal(other.SetValue)
}

func (v CaseInsensitiveStringSetValue) Type(ctx context.Context) attr.Type {
	return CaseInsensitiveStringSetType{
		SetType: basetypes.SetType{ElemType: CaseInsensitiveStringType{}},
	}
}

func (v CaseInsensitiveStringSetValue) SetSemanticEquals(ctx context.Context, newValuable basetypes.SetValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newSetValue, ok := newValuable.(CaseInsensitiveStringSetValue)
	if !ok {
		diags.AddError("Semantic Equality Check Error",
			fmt.Sprintf("Expected CaseInsensitiveStringSetValue, Got: %T", newValuable))
		return false, diags
	}

	priorElems := v.SetValue.Elements()
	newElems := newSetValue.SetValue.Elements()

	if len(priorElems) != len(newElems) {
		return false, diags
	}

	// Build uppercase set of one side
	priorUpper := make(map[string]struct{}, len(priorElems))
	for _, e := range priorElems {
		if sv, ok := e.(basetypes.StringValuable); ok {
			s, d := sv.ToStringValue(ctx)
			diags.Append(d...)
			if diags.HasError() {
				return false, diags
			}
			priorUpper[strings.ToUpper(s.ValueString())] = struct{}{}
		}
	}

	for _, e := range newElems {
		if sv, ok := e.(basetypes.StringValuable); ok {
			s, d := sv.ToStringValue(ctx)
			diags.Append(d...)
			if diags.HasError() {
				return false, diags
			}
			if _, found := priorUpper[strings.ToUpper(s.ValueString())]; !found {
				return false, diags
			}
		}
	}

	return true, diags
}

// --- Plan Modifier ---

// useStateForCaseInsensitiveSet is a plan modifier that prevents drift caused by
// case differences between config and state. When the planned set (from config)
// is case-insensitively equal to the prior state, it uses the prior state value.
// Terraform Core accepts plan == prior for Optional+Computed attributes.
type useStateForCaseInsensitiveSet struct{}

func UseStateForCaseInsensitiveSet() planmodifier.Set {
	return useStateForCaseInsensitiveSet{}
}

func (m useStateForCaseInsensitiveSet) Description(ctx context.Context) string {
	return "Uses the prior state value when the planned set differs only in case."
}

func (m useStateForCaseInsensitiveSet) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForCaseInsensitiveSet) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	// During Create there is no prior state, let the plan be the config value.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if resp.PlanValue.IsNull() || resp.PlanValue.IsUnknown() {
		return
	}

	planElems := resp.PlanValue.Elements()
	stateElems := req.StateValue.Elements()

	if len(planElems) != len(stateElems) {
		return
	}

	stateUpper := make(map[string]struct{}, len(stateElems))
	for _, e := range stateElems {
		if sv, ok := e.(basetypes.StringValuable); ok {
			s, d := sv.ToStringValue(ctx)
			resp.Diagnostics.Append(d...)
			if resp.Diagnostics.HasError() {
				return
			}
			stateUpper[strings.ToUpper(s.ValueString())] = struct{}{}
		}
	}

	for _, e := range planElems {
		if sv, ok := e.(basetypes.StringValuable); ok {
			s, d := sv.ToStringValue(ctx)
			resp.Diagnostics.Append(d...)
			if resp.Diagnostics.HasError() {
				return
			}
			if _, found := stateUpper[strings.ToUpper(s.ValueString())]; !found {
				return // Real difference, keep plan from config
			}
		}
	}

	// Sets are case-insensitively equal; use prior state to prevent drift.
	resp.PlanValue = req.StateValue
}

// useStateForCaseInsensitiveString is a plan modifier that prevents drift caused
// by case differences between config and state for individual string attributes.
type useStateForCaseInsensitiveString struct{}

func UseStateForCaseInsensitiveString() planmodifier.String {
	return useStateForCaseInsensitiveString{}
}

func (m useStateForCaseInsensitiveString) Description(ctx context.Context) string {
	return "Uses the prior state value when the planned string differs only in case."
}

func (m useStateForCaseInsensitiveString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForCaseInsensitiveString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if resp.PlanValue.IsNull() || resp.PlanValue.IsUnknown() {
		return
	}

	if strings.EqualFold(req.StateValue.ValueString(), resp.PlanValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}
