package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// objectInfoObjectType is the state type of an object reference with id, type and name.
var objectInfoObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":   types.StringType,
	"type": types.StringType,
	"name": types.StringType,
}}

// objectInfoListState converts an optional object reference to a single-element list, or null.
func objectInfoListState(ctx context.Context, info *ObjectInfo, diags *diag.Diagnostics) types.List {
	if info == nil {
		return types.ListNull(objectInfoObjectType)
	}
	return objectInfoSliceState(ctx, []*ObjectInfo{info}, diags)
}

// objectInfoSliceState converts object references to a list.
func objectInfoSliceState(ctx context.Context, infos []*ObjectInfo, diags *diag.Diagnostics) types.List {
	models := make([]OwnerModel, 0, len(infos))
	for _, info := range infos {
		id := ""
		if info.ID != nil {
			id = fmt.Sprintf("%v", info.ID)
		}
		models = append(models, OwnerModel{
			ID:   types.StringValue(id),
			Type: types.StringValue(info.Type),
			Name: types.StringValue(info.Name),
		})
	}
	list, d := types.ListValueFrom(ctx, objectInfoObjectType, models)
	diags.Append(d...)
	return list
}
