package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

	priorUpper, d := upperStringSet(ctx, v.SetValue.Elements())
	diags.Append(d...)
	newUpper, d := upperStringSet(ctx, newSetValue.SetValue.Elements())
	diags.Append(d...)
	if diags.HasError() {
		return false, diags
	}
	return sameStringSet(priorUpper, newUpper), diags
}

// upperStringSet returns the upper-cased string values of elements as a set.
func upperStringSet(ctx context.Context, elements []attr.Value) (map[string]struct{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make(map[string]struct{}, len(elements))
	for _, e := range elements {
		if sv, ok := e.(basetypes.StringValuable); ok {
			s, d := sv.ToStringValue(ctx)
			diags.Append(d...)
			result[strings.ToUpper(s.ValueString())] = struct{}{}
		}
	}
	return result, diags
}

func sameStringSet(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}
