package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// listWithDefaultString sets attribute to value in every object of list where it is null or unknown.
// It resolves optional and computed nested attributes that the API fills with a fixed default.
func listWithDefaultString(ctx context.Context, list types.List, attribute, value string, diags *diag.Diagnostics) types.List {
	if list.IsNull() || list.IsUnknown() {
		return list
	}
	elements := make([]attr.Value, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		object, ok := element.(types.Object)
		if !ok || object.IsNull() || object.IsUnknown() {
			elements = append(elements, element)
			continue
		}
		attributes := object.Attributes()
		if current, ok := attributes[attribute]; ok && (current.IsNull() || current.IsUnknown()) {
			copied := make(map[string]attr.Value, len(attributes))
			for k, v := range attributes {
				copied[k] = v
			}
			copied[attribute] = types.StringValue(value)
			updated, d := types.ObjectValue(object.AttributeTypes(ctx), copied)
			diags.Append(d...)
			element = updated
		}
		elements = append(elements, element)
	}
	result, d := types.ListValue(list.ElementType(ctx), elements)
	diags.Append(d...)
	return result
}
