package main

import (
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// jsonSemanticallyEqual reports whether two JSON documents decode to equal values.
func jsonSemanticallyEqual(a, b []byte) bool {
	var av, bv interface{}
	if err := json.Unmarshal(a, &av); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

// jsonStringState returns the state value for a JSON string attribute holding value from the API.
// The prior value is kept when it is semantically equal, so formatting and key order do not cause
// diffs. A missing or empty object value maps to null unless the prior value is also empty.
func jsonStringState(prior types.String, value interface{}) types.String {
	priorSet := !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() != ""
	encoded, err := json.Marshal(value)
	if err != nil {
		return prior
	}
	if value == nil || string(encoded) == "{}" || string(encoded) == "null" {
		if priorSet && (jsonSemanticallyEqual([]byte(prior.ValueString()), []byte("{}")) || prior.ValueString() == "null") {
			return prior
		}
		return types.StringNull()
	}
	if priorSet && jsonSemanticallyEqual([]byte(prior.ValueString()), encoded) {
		return prior
	}
	return types.StringValue(string(encoded))
}
