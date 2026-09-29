package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// jsonArrayStringValidator checks that a string attribute contains a JSON array.
type jsonArrayStringValidator struct{}

func (v jsonArrayStringValidator) Description(ctx context.Context) string {
	return "value must be a JSON array"
}

func (v jsonArrayStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonArrayStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var value interface{}
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &value); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", fmt.Sprintf("Unable to parse %s: %s", req.Path, err))
		return
	}
	if _, ok := value.([]interface{}); !ok {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", fmt.Sprintf("%s must be a JSON array", req.Path))
	}
}

// jsonObjectStringValidator checks that a string attribute contains a JSON object.
type jsonObjectStringValidator struct{}

func (v jsonObjectStringValidator) Description(ctx context.Context) string {
	return "value must be a JSON object"
}

func (v jsonObjectStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonObjectStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var value interface{}
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &value); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", fmt.Sprintf("Unable to parse %s: %s", req.Path, err))
		return
	}
	if _, ok := value.(map[string]interface{}); !ok {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", fmt.Sprintf("%s must be a JSON object", req.Path))
	}
}

// listSizeValidator checks that a list or list block has between min and max elements.
// A max of 0 means no upper bound.
type listSizeValidator struct {
	min int
	max int
}

func listSizeBetween(min, max int) listSizeValidator {
	return listSizeValidator{min: min, max: max}
}

func (v listSizeValidator) Description(ctx context.Context) string {
	if v.max == 0 {
		return fmt.Sprintf("list must contain at least %d elements", v.min)
	}
	if v.min == v.max {
		return fmt.Sprintf("list must contain exactly %d elements", v.min)
	}
	return fmt.Sprintf("list must contain between %d and %d elements", v.min, v.max)
}

func (v listSizeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v listSizeValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsUnknown() {
		return
	}
	size := len(req.ConfigValue.Elements())
	if size < v.min || (v.max > 0 && size > v.max) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid list size", fmt.Sprintf("%s, got %d.", v.Description(ctx), size))
	}
}

// int64AtLeastValidator checks that an integer attribute is at least min.
type int64AtLeastValidator struct {
	min int64
}

func (v int64AtLeastValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be at least %d", v.min)
}

func (v int64AtLeastValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v int64AtLeastValidator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueInt64() < v.min {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid value", fmt.Sprintf("%s, got %d.", v.Description(ctx), req.ConfigValue.ValueInt64()))
	}
}
