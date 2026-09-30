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

// jsonIsEmpty reports whether a decoded JSON value is null, an empty object or an empty array.
func jsonIsEmpty(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return true
	case map[string]interface{}:
		return len(v) == 0
	case []interface{}:
		return len(v) == 0
	}
	return false
}

// jsonIsZeroScalar reports whether a decoded JSON value is false, 0 or an empty string.
func jsonIsZeroScalar(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return !v
	case float64:
		return v == 0
	case json.Number:
		f, err := v.Float64()
		return err == nil && f == 0
	case string:
		return v == ""
	}
	return false
}

// jsonContains reports whether the decoded API value contains everything in the
// decoded configured value: every configured object key must be present with a matching value,
// arrays must have the same length and matching elements. Keys the API adds (defaults, read-only
// fields, resolved names) are ignored. A configured null, false, 0, "" or empty array/object
// matches a missing (or null) value, since APIs omit keys at their default values.
func jsonContains(api, configured interface{}) bool {
	if api == nil && jsonIsZeroScalar(configured) {
		return true
	}
	switch c := configured.(type) {
	case nil:
		return api == nil
	case map[string]interface{}:
		if len(c) == 0 && api == nil {
			return true
		}
		a, ok := api.(map[string]interface{})
		if !ok {
			return false
		}
		for key, value := range c {
			if !jsonContains(a[key], value) {
				return false
			}
		}
		return true
	case []interface{}:
		if len(c) == 0 && api == nil {
			return true
		}
		a, ok := api.([]interface{})
		if !ok || len(a) != len(c) {
			return false
		}
		for i := range c {
			if !jsonContains(a[i], c[i]) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(api, configured)
}

// jsonSubsetState returns the state value of a JSON attribute holding value from the
// API. The prior value is kept when the API value contains it (see jsonContains), so
// formatting, key order and fields added by the API do not cause diffs, while changes of configured
// fields are detected. An empty API value maps to null unless the prior value is also empty.
func jsonSubsetState(prior types.String, value interface{}) types.String {
	priorSet := !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() != ""
	var priorValue interface{}
	priorValid := priorSet && json.Unmarshal([]byte(prior.ValueString()), &priorValue) == nil
	if jsonIsEmpty(value) {
		if priorValid && jsonIsEmpty(priorValue) {
			return prior
		}
		return types.StringNull()
	}
	if priorValid && jsonContains(value, priorValue) {
		return prior
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return prior
	}
	return types.StringValue(string(encoded))
}

// jsonSubsetStateRaw is jsonSubsetState for a raw JSON API value.
func jsonSubsetStateRaw(prior types.String, raw json.RawMessage) types.String {
	var value interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &value); err != nil {
			return types.StringValue(string(raw))
		}
	}
	if prior.IsUnknown() {
		prior = types.StringNull()
	}
	return jsonSubsetState(prior, value)
}
