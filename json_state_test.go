package main

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestJSONContains(t *testing.T) {
	decode := func(s string) interface{} {
		var v interface{}
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			t.Fatalf("invalid JSON %s: %s", s, err)
		}
		return v
	}
	for _, tc := range []struct {
		api, configured string
		want            bool
	}{
		{`{"a": 1, "b": 2}`, `{"a": 1}`, true},
		{`{"a": 1}`, `{"a": 2}`, false},
		{`{"a": [{"x": 1, "y": null}]}`, `{"a": [{"x": 1}]}`, true},
		{`{"a": [{"x": 1}, {"x": 2}]}`, `{"a": [{"x": 1}]}`, false},
		// The API omits keys at their default values: a configured false, 0, "", null or empty
		// collection matches a missing or null key.
		{`{}`, `{"enabled": false, "count": 0, "name": "", "none": null, "list": [], "map": {}}`, true},
		{`{"enabled": null, "count": null, "name": null}`, `{"enabled": false, "count": 0, "name": ""}`, true},
		{`{"a": [{"x": 1}]}`, `{"a": [{"x": 1, "required": false}]}`, true},
		// Values that are not defaults must be present.
		{`{}`, `{"enabled": true}`, false},
		{`{}`, `{"count": 1}`, false},
		{`{}`, `{"name": "x"}`, false},
		{`{}`, `{"list": [1]}`, false},
		// A present value must still match.
		{`{"enabled": true}`, `{"enabled": false}`, false},
		{`{"count": 5}`, `{"count": 0}`, false},
		{`{"name": "x"}`, `{"name": ""}`, false},
	} {
		if got := jsonContains(decode(tc.api), decode(tc.configured)); got != tc.want {
			t.Errorf("jsonContains(%s, %s) = %v, want %v", tc.api, tc.configured, got, tc.want)
		}
	}
	if !jsonContains(nil, json.Number("0")) || jsonContains(nil, json.Number("1")) {
		t.Errorf("json.Number zero values must match a missing value")
	}
}

func TestJSONSubsetStateKeepsConfiguredDefaults(t *testing.T) {
	prior := types.StringValue(`{"name": "x", "enabled": false}`)
	if got := jsonSubsetState(prior, map[string]interface{}{"name": "x", "extra": 1}); !got.Equal(prior) {
		t.Fatalf("a configured false omitted by the API must keep the prior value, got %s", got)
	}
	if got := jsonSubsetState(prior, map[string]interface{}{"name": "x", "enabled": true}); got.Equal(prior) {
		t.Fatalf("a changed value must be drift")
	}
}
