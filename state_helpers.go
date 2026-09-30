package main

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// optionalStringState returns the state value of an optional string read from the API. An empty
// API value keeps a null prior value, so an unset attribute does not show a diff.
func optionalStringState(prior types.String, value string) types.String {
	if value == "" && prior.IsNull() {
		return types.StringNull()
	}
	return types.StringValue(value)
}

// optionalBoolState returns the state value of an optional boolean read from the API. A false or
// missing API value keeps a null prior value, so an unset attribute does not show a diff.
func optionalBoolState(prior types.Bool, value *bool) types.Bool {
	if (value == nil || !*value) && prior.IsNull() {
		return types.BoolNull()
	}
	return types.BoolValue(value != nil && *value)
}

// optionalInt64State returns the state value of an optional integer read from the API. A zero or
// missing API value keeps a null prior value, so an unset attribute does not show a diff.
func optionalInt64State(prior types.Int64, value *int64) types.Int64 {
	if (value == nil || *value == 0) && prior.IsNull() {
		return types.Int64Null()
	}
	if value == nil {
		return types.Int64Value(0)
	}
	return types.Int64Value(*value)
}

// computedStringFromAPI resolves an optional and computed string after apply: a known planned
// value is kept, an unknown one is taken from the API.
func computedStringFromAPI(planned types.String, value string) types.String {
	if planned.IsUnknown() {
		return types.StringValue(value)
	}
	return planned
}

// stringPointer returns a pointer to a known string value, or nil for null and unknown values.
func stringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueString()
	return &v
}

// int64Pointer returns a pointer to a known integer value, or nil for null and unknown values.
func int64Pointer(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueInt64()
	return &v
}

// patchBuilder collects JSON Patch operations for the attributes that changed between plan and state.
type patchBuilder struct {
	ops []jsonPatchOp
}

// replaceIfChanged adds a replace operation when planned differs from prior. Values that are
// unknown in the plan are ignored, see equalIgnoringUnknown.
func (b *patchBuilder) replaceIfChanged(planned, prior attr.Value, path string, value interface{}) {
	if !equalIgnoringUnknown(planned, prior) {
		b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: path, Value: value})
	}
}

// replaceOrRemoveIfChanged adds a replace operation when planned differs from prior, or a remove
// operation when the planned value is null and the prior value is set.
func (b *patchBuilder) replaceOrRemoveIfChanged(planned, prior attr.Value, path string, value interface{}) {
	if equalIgnoringUnknown(planned, prior) {
		return
	}
	if planned.IsNull() {
		if !prior.IsNull() {
			b.ops = append(b.ops, jsonPatchOp{Op: "remove", Path: path})
		}
		return
	}
	b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: path, Value: value})
}

// singletonDeleteWarning reports that destroying a tenant setting only removes it from state.
func singletonDeleteWarning(resp *resource.DeleteResponse, name string) {
	resp.Diagnostics.AddWarning(
		name+" left in place",
		"Tenant settings cannot be deleted. The resource was removed from Terraform state and the current settings were left unchanged in IdentityNow.",
	)
}
