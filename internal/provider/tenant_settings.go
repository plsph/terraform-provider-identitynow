package provider

// Shared implementation of the tenant-wide singleton settings (tenant settings) resources.
//
// Every tenant setting API has a single object per tenant that is read with GET and changed with
// PUT or JSON Patch; it cannot be created or deleted. Each resource declares a tenantSettingsSpec
// with its API path, update strategy and fields. The resources follow the public_identities_config
// reference (fixed ID, Create applies the planned settings, Delete only removes the resource from
// state, Import sets the fixed ID), with one difference: only the settings present in the
// configuration are managed. All settings are Optional + Computed with UseStateForUnknown, so
// settings that are not configured show the current tenant value and are never changed or reset:
//   - PATCH APIs receive operations for the configured settings that changed only.
//   - PUT APIs receive the current object with only the configured settings replaced (putMerged),
//     or, when the request schema differs from the response schema, a body built from the
//     configured settings and the current values of the other writable settings. Secrets that
//     the API does not return cannot be sent back: when one is not configured, the update fails
//     with an error instead of clearing it.
//   - JSON Patch operations replace existing members and add missing ones.

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// tenantSettingsKind is the Terraform and API representation of a setting.
type tenantSettingsKind int

const (
	tenantSettingsBool tenantSettingsKind = iota
	tenantSettingsInt64
	// tenantSettingsInt64String is a number in Terraform that the API represents as a string.
	tenantSettingsInt64String
	tenantSettingsFloat64
	tenantSettingsString
	tenantSettingsStringList
	tenantSettingsStringMap
	// tenantSettingsJSONObject is a free-form object stored as a JSON string. Only the configured
	// keys are managed: they are merged into the current object and compared on read.
	tenantSettingsJSONObject
	// tenantSettingsJSONArray is an array stored as a JSON string and replaced as a whole.
	tenantSettingsJSONArray
)

// tenantSettingsField describes one setting of a tenant settings object.
type tenantSettingsField struct {
	// Name is the Terraform attribute name.
	Name string
	// Path is the location of the value in the API object (after View, if the spec has one).
	Path        []string
	Kind        tenantSettingsKind
	Description string
	Sensitive   bool
	// ReadOnly settings are Computed only.
	ReadOnly bool
	// Stable read-only settings never change and keep their state value in plans.
	Stable bool
	// WriteOnly settings may not be returned by the API (secrets): the known value is kept when
	// the API returns none. PUT APIs refuse to send the current object when a write-only setting
	// is not configured and the API does not return it, because the PUT would clear it.
	WriteOnly bool
	// WriteOnlyKeys are keys of a JSON object setting that the API may not return (secrets). PUT
	// APIs refuse to send the object when one of them is neither configured nor returned.
	WriteOnlyKeys []string
	// Trigger settings are one-shot requests that the API resets after use (e.g. on the next
	// pipeline run). Read keeps the known state value, so the reset is not shown as drift, and
	// the value is only sent on create and when the configured value changes.
	Trigger bool
}

// tenantSettingsUpdateMode is how a tenant settings object is updated.
type tenantSettingsUpdateMode int

const (
	// tenantSettingsPutMerged sends the current object with the configured settings replaced.
	tenantSettingsPutMerged tenantSettingsUpdateMode = iota
	// tenantSettingsPutBody sends only the writable settings: configured values, and the current
	// values of the other settings. Used when the request schema is a subset of the response.
	tenantSettingsPutBody
	// tenantSettingsPatch sends JSON Patch operations for the configured settings that changed:
	// replace for members of the current object, add for missing members.
	tenantSettingsPatch
)

// tenantSettingsChange is a configured setting that is sent to a PATCH API.
type tenantSettingsChange struct {
	Field *tenantSettingsField
	Value interface{}
}

// tenantSettingsSpec describes a tenant settings resource.
type tenantSettingsSpec struct {
	// TypeName is the Terraform type name without the provider prefix.
	TypeName string
	// ID is the fixed ID of the singleton.
	ID string
	// Title names the settings in messages, e.g. "Lockout config".
	Title       string
	Description string
	// Path is the API path, e.g. /v2026/auth-org/lockout-config.
	Path         string
	Experimental bool
	Mode         tenantSettingsUpdateMode
	// CreateWithPost creates the object with POST when GET returns 404 (PATCH APIs only).
	CreateWithPost bool
	// View converts the API object to the document the field paths refer to (PATCH APIs only).
	View func(doc map[string]interface{}) map[string]interface{}
	// PatchOps builds the JSON Patch operations for the changes. By default each change is a
	// replace operation at the field path.
	PatchOps func(ctx context.Context, c *Client, spec *tenantSettingsSpec, changes []tenantSettingsChange) ([]jsonPatchOp, error)
	Fields   []tenantSettingsField
}

func (spec *tenantSettingsSpec) requestOptions() []requestOption {
	if spec.Experimental {
		return []requestOption{withExperimental()}
	}
	return nil
}

// ---- API access ----

// tenantSettingsGet reads the settings object.
func (c *Client) tenantSettingsGet(ctx context.Context, spec *tenantSettingsSpec) (map[string]interface{}, error) {
	doc := map[string]interface{}{}
	if err := c.doJSON(ctx, http.MethodGet, spec.Path, nil, &doc, spec.requestOptions()...); err != nil {
		return nil, err
	}
	return doc, nil
}

// tenantSettingsApply saves the configured settings and returns the updated API object.
// planned holds the planned values, prior the state values (nil on create) and configured the
// names of the settings present in the configuration.
func (c *Client) tenantSettingsApply(ctx context.Context, spec *tenantSettingsSpec, planned, prior map[string]attr.Value, configured map[string]bool, create bool) (map[string]interface{}, error) {
	var fields []*tenantSettingsField
	for i := range spec.Fields {
		f := &spec.Fields[i]
		if !f.ReadOnly && configured[f.Name] && planned[f.Name] != nil && !planned[f.Name].IsNull() && !planned[f.Name].IsUnknown() {
			if f.Trigger && !create && prior[f.Name] != nil && planned[f.Name].Equal(prior[f.Name]) {
				// Sending an unchanged one-shot request again would repeat it.
				continue
			}
			fields = append(fields, f)
		}
	}
	var out map[string]interface{}
	var err error
	switch spec.Mode {
	case tenantSettingsPatch:
		out, err = c.tenantSettingsPatchApply(ctx, spec, fields, planned, prior, create)
	case tenantSettingsPutBody:
		out, err = c.tenantSettingsPutBodyApply(ctx, spec, fields, planned)
	default:
		out, err = c.tenantSettingsPutMergedApply(ctx, spec, fields, planned)
	}
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		// Nothing was sent, or the API returned no body.
		return c.tenantSettingsGet(ctx, spec)
	}
	return out, nil
}

func (c *Client) tenantSettingsPatchApply(ctx context.Context, spec *tenantSettingsSpec, fields []*tenantSettingsField, planned, prior map[string]attr.Value, create bool) (map[string]interface{}, error) {
	opts := spec.requestOptions()
	var current map[string]interface{}
	if create && spec.CreateWithPost {
		var err error
		current, err = c.tenantSettingsGet(ctx, spec)
		if isNotFound(err) {
			body := map[string]interface{}{}
			for _, f := range fields {
				value, err := tenantSettingsToAPI(f, planned[f.Name])
				if err != nil {
					return nil, err
				}
				body = tenantSettingsSetAt(body, f.Path, value, false).(map[string]interface{})
			}
			out := map[string]interface{}{}
			if err := c.doJSON(ctx, http.MethodPost, spec.Path, body, &out, opts...); err != nil {
				return nil, err
			}
			return out, nil
		}
		if err != nil {
			return nil, err
		}
	}
	var b patchBuilder
	var changes []tenantSettingsChange
	for _, f := range fields {
		var old attr.Value
		if !create {
			old = prior[f.Name]
		}
		if old != nil && tenantSettingsJSONEqual(f, planned[f.Name], old) {
			continue
		}
		value, err := tenantSettingsToAPI(f, planned[f.Name])
		if err != nil {
			return nil, err
		}
		before := len(b.ops)
		b.replaceIfChanged(planned[f.Name], old, tenantSettingsPointer(f.Path), value)
		if len(b.ops) > before {
			changes = append(changes, tenantSettingsChange{Field: f, Value: value})
		}
	}
	if len(changes) == 0 {
		return nil, nil
	}
	var ops []jsonPatchOp
	if spec.PatchOps != nil {
		var err error
		if ops, err = spec.PatchOps(ctx, c, spec, changes); err != nil {
			return nil, err
		}
	} else {
		// A replace operation fails when the member does not exist (RFC 6902), and the APIs omit
		// some unset settings, so members missing from the current object are added instead.
		if current == nil {
			var err error
			if current, err = c.tenantSettingsGet(ctx, spec); err != nil {
				return nil, err
			}
		}
		view := spec.view(current)
		for _, change := range changes {
			ops = append(ops, tenantSettingsPatchOp(view, change.Field.Path, change.Value))
		}
	}
	out := map[string]interface{}{}
	if err := c.doJSON(ctx, http.MethodPatch, spec.Path, ops, &out, append(opts, withJSONPatch())...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) tenantSettingsPutMergedApply(ctx context.Context, spec *tenantSettingsSpec, fields []*tenantSettingsField, planned map[string]attr.Value) (map[string]interface{}, error) {
	if len(fields) == 0 {
		return nil, nil
	}
	var current map[string]interface{}
	for _, f := range fields {
		if len(f.Path) > 1 || f.Kind == tenantSettingsJSONObject || spec.hasWriteOnly() {
			// Nested and merged settings need the current value of their top-level key, and the
			// write-only settings are checked against the current object.
			var err error
			if current, err = c.tenantSettingsGet(ctx, spec); err != nil {
				return nil, err
			}
			break
		}
	}
	if err := tenantSettingsCheckWriteOnly(spec, fields, planned, current); err != nil {
		return nil, err
	}
	managed := map[string]interface{}{}
	for _, f := range fields {
		value, err := tenantSettingsToAPI(f, planned[f.Name])
		if err != nil {
			return nil, err
		}
		top := f.Path[0]
		base, ok := managed[top]
		if !ok {
			base = current[top]
		}
		managed[top] = tenantSettingsSetAt(base, f.Path[1:], value, f.Kind == tenantSettingsJSONObject)
	}
	out := map[string]interface{}{}
	if err := c.putMerged(ctx, spec.Path, managed, &out, spec.requestOptions()...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) tenantSettingsPutBodyApply(ctx context.Context, spec *tenantSettingsSpec, fields []*tenantSettingsField, planned map[string]attr.Value) (map[string]interface{}, error) {
	if len(fields) == 0 {
		return nil, nil
	}
	configured := map[string]bool{}
	for _, f := range fields {
		configured[f.Name] = true
	}
	var current map[string]interface{}
	for i := range spec.Fields {
		f := &spec.Fields[i]
		if !f.ReadOnly && !configured[f.Name] || f.Kind == tenantSettingsJSONObject {
			var err error
			if current, err = c.tenantSettingsGet(ctx, spec); err != nil {
				return nil, err
			}
			break
		}
	}
	if err := tenantSettingsCheckWriteOnly(spec, fields, planned, current); err != nil {
		return nil, err
	}
	body := map[string]interface{}{}
	for i := range spec.Fields {
		f := &spec.Fields[i]
		if f.ReadOnly {
			continue
		}
		existing, found := tenantSettingsLookup(current, f.Path)
		if !configured[f.Name] {
			if found {
				body = tenantSettingsSetAt(body, f.Path, existing, false).(map[string]interface{})
			}
			continue
		}
		value, err := tenantSettingsToAPI(f, planned[f.Name])
		if err != nil {
			return nil, err
		}
		if f.Kind == tenantSettingsJSONObject {
			value = tenantSettingsMergeJSON(existing, value)
		}
		body = tenantSettingsSetAt(body, f.Path, value, false).(map[string]interface{})
	}
	out := map[string]interface{}{}
	if err := c.doJSON(ctx, http.MethodPut, spec.Path, body, &out, spec.requestOptions()...); err != nil {
		return nil, err
	}
	return out, nil
}

// hasWriteOnly reports whether the spec has writable settings that the API may not return.
func (spec *tenantSettingsSpec) hasWriteOnly() bool {
	for i := range spec.Fields {
		f := &spec.Fields[i]
		if !f.ReadOnly && (f.WriteOnly || len(f.WriteOnlyKeys) > 0) {
			return true
		}
	}
	return false
}

// tenantSettingsMissingValue reports whether a value is not usable as the current value of a
// secret: missing, null or an empty string.
func tenantSettingsMissingValue(value interface{}, found bool) bool {
	return !found || value == nil || value == ""
}

// tenantSettingsCheckWriteOnly returns an error when a PUT built from current would clear a secret:
// a write-only setting (or write-only key of a JSON object setting) that is not configured and
// that the API does not return would be sent back empty. fields are the configured settings that
// are sent, current is the object returned by GET.
func tenantSettingsCheckWriteOnly(spec *tenantSettingsSpec, fields []*tenantSettingsField, planned map[string]attr.Value, current map[string]interface{}) error {
	configured := map[string]bool{}
	for _, f := range fields {
		configured[f.Name] = true
	}
	var missing []string
	for i := range spec.Fields {
		f := &spec.Fields[i]
		if f.ReadOnly || (!f.WriteOnly && len(f.WriteOnlyKeys) == 0) {
			continue
		}
		existing, found := tenantSettingsLookup(current, f.Path)
		value := existing
		if configured[f.Name] {
			if f.Kind != tenantSettingsJSONObject {
				continue
			}
			// Configured JSON objects are merged into the current object.
			configuredValue, err := tenantSettingsToAPI(f, planned[f.Name])
			if err != nil {
				return err
			}
			value, found = tenantSettingsMergeJSON(existing, configuredValue), true
		} else if f.WriteOnly && tenantSettingsMissingValue(existing, found) {
			missing = append(missing, "`"+f.Name+"`")
			continue
		}
		object, _ := value.(map[string]interface{})
		for _, key := range f.WriteOnlyKeys {
			item, ok := object[key]
			if tenantSettingsMissingValue(item, ok) {
				missing = append(missing, fmt.Sprintf("the `%s` key of `%s`", key, f.Name))
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("configure %s: the API does not return the current secret, and the update is a PUT of the whole %s, which would clear it",
		strings.Join(missing, " and "), spec.Title)
}

// tenantSettingsPatchOp returns the JSON Patch operation that sets the value at p in the current
// object. Existing members are replaced; a missing member is added (add creates or replaces an
// object member), together with the missing intermediate objects.
func tenantSettingsPatchOp(current map[string]interface{}, p []string, value interface{}) jsonPatchOp {
	var node interface{} = current
	for i, key := range p {
		m, ok := node.(map[string]interface{})
		if !ok {
			// The parent exists but is not an object (null): replace it with an object.
			return jsonPatchOp{Op: "replace", Path: tenantSettingsPointer(p[:i]), Value: tenantSettingsSetAt(nil, p[i:], value, false)}
		}
		child, found := m[key]
		if !found {
			return jsonPatchOp{Op: "add", Path: tenantSettingsPointer(p[:i+1]), Value: tenantSettingsSetAt(nil, p[i+1:], value, false)}
		}
		node = child
	}
	return jsonPatchOp{Op: "replace", Path: tenantSettingsPointer(p), Value: value}
}

// ---- JSON document helpers ----

// tenantSettingsPointer returns the JSON Pointer of a field path.
func tenantSettingsPointer(p []string) string {
	escaped := make([]string, len(p))
	for i, s := range p {
		escaped[i] = strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
	}
	return "/" + strings.Join(escaped, "/")
}

// tenantSettingsLookup returns the value at p in doc.
func tenantSettingsLookup(doc interface{}, p []string) (interface{}, bool) {
	current := doc
	for _, key := range p {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		if current, ok = m[key]; !ok {
			return nil, false
		}
	}
	return current, true
}

// tenantSettingsSetAt returns base with value set at p. Missing or non-object intermediate values
// are replaced with objects. With merge the value is merged into the existing value.
func tenantSettingsSetAt(base interface{}, p []string, value interface{}, merge bool) interface{} {
	if len(p) == 0 {
		if merge {
			return tenantSettingsMergeJSON(base, value)
		}
		return value
	}
	m, ok := base.(map[string]interface{})
	if !ok || m == nil {
		m = map[string]interface{}{}
	}
	m[p[0]] = tenantSettingsSetAt(m[p[0]], p[1:], value, merge)
	return m
}

// tenantSettingsMergeJSON merges configured into current: objects are merged key by key, other
// values are replaced.
func tenantSettingsMergeJSON(current, configured interface{}) interface{} {
	c, ok1 := current.(map[string]interface{})
	n, ok2 := configured.(map[string]interface{})
	if !ok1 || !ok2 {
		return configured
	}
	out := make(map[string]interface{}, len(c)+len(n))
	for k, v := range c {
		out[k] = v
	}
	for k, v := range n {
		out[k] = tenantSettingsMergeJSON(c[k], v)
	}
	return out
}

// tenantSettingsProjectJSON returns the part of the API value that corresponds to the keys of the
// prior value, so keys that are not configured do not show a diff. Arrays of the same length are
// projected element by element (the API adds keys to array elements too). Keys the API does not
// return keep a prior false, 0, "" or empty value, as for jsonContains. With keepMissing, keys the
// API does not return keep their prior value (secrets).
func tenantSettingsProjectJSON(prior, api interface{}, keepMissing bool) interface{} {
	if p, ok := prior.([]interface{}); ok {
		a, ok := api.([]interface{})
		if !ok || len(a) != len(p) {
			return api
		}
		out := make([]interface{}, len(a))
		for i := range a {
			out[i] = tenantSettingsProjectJSON(p[i], a[i], keepMissing)
		}
		return out
	}
	p, ok1 := prior.(map[string]interface{})
	a, ok2 := api.(map[string]interface{})
	if !ok1 || !ok2 {
		return api
	}
	out := map[string]interface{}{}
	for k, pv := range p {
		if av, ok := a[k]; ok && av != nil {
			out[k] = tenantSettingsProjectJSON(pv, av, keepMissing)
		} else if keepMissing || jsonContains(nil, pv) {
			out[k] = pv
		} else if ok {
			out[k] = nil
		}
	}
	return out
}

// ---- value conversion ----

// tenantSettingsToAPI converts a known Terraform value to its API representation.
func tenantSettingsToAPI(f *tenantSettingsField, v attr.Value) (interface{}, error) {
	switch f.Kind {
	case tenantSettingsBool:
		return v.(types.Bool).ValueBool(), nil
	case tenantSettingsInt64:
		return v.(types.Int64).ValueInt64(), nil
	case tenantSettingsInt64String:
		return strconv.FormatInt(v.(types.Int64).ValueInt64(), 10), nil
	case tenantSettingsFloat64:
		return v.(types.Float64).ValueFloat64(), nil
	case tenantSettingsString:
		return v.(types.String).ValueString(), nil
	case tenantSettingsStringList:
		values := []string{}
		for _, e := range v.(types.List).Elements() {
			if s, ok := e.(types.String); ok {
				values = append(values, s.ValueString())
			}
		}
		return values, nil
	case tenantSettingsStringMap:
		values := map[string]string{}
		for k, e := range v.(types.Map).Elements() {
			if s, ok := e.(types.String); ok {
				values[k] = s.ValueString()
			}
		}
		return values, nil
	case tenantSettingsJSONObject, tenantSettingsJSONArray:
		var value interface{}
		if err := json.Unmarshal([]byte(v.(types.String).ValueString()), &value); err != nil {
			return nil, fmt.Errorf("%s is not valid JSON: %w", f.Name, err)
		}
		return value, nil
	}
	return nil, fmt.Errorf("unsupported setting kind for %s", f.Name)
}

// tenantSettingsJSONEqual reports whether two JSON string values are semantically equal.
func tenantSettingsJSONEqual(f *tenantSettingsField, a, b attr.Value) bool {
	if f.Kind != tenantSettingsJSONObject && f.Kind != tenantSettingsJSONArray {
		return false
	}
	as, ok1 := a.(types.String)
	bs, ok2 := b.(types.String)
	if !ok1 || !ok2 || as.IsNull() || bs.IsNull() || as.IsUnknown() || bs.IsUnknown() {
		return false
	}
	return jsonSemanticallyEqual([]byte(as.ValueString()), []byte(bs.ValueString()))
}

func tenantSettingsKnown(v attr.Value) bool {
	return v != nil && !v.IsNull() && !v.IsUnknown()
}

func tenantSettingsNumber(raw interface{}) (float64, bool) {
	switch n := raw.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	}
	return 0, false
}

func tenantSettingsStringOf(raw interface{}) string {
	switch v := raw.(type) {
	case string:
		return v
	case bool, float64, json.Number:
		return fmt.Sprint(v)
	}
	encoded, _ := json.Marshal(raw)
	return string(encoded)
}

// tenantSettingsFromAPI returns the state value of a setting read from the API. prior is the
// current state value, or nil when there is none (import, data sources, unknown planned values).
func tenantSettingsFromAPI(f *tenantSettingsField, prior attr.Value, raw interface{}, found bool) attr.Value {
	missing := !found || raw == nil
	if f.WriteOnly && tenantSettingsKnown(prior) && (missing || raw == "") && f.Kind != tenantSettingsJSONObject {
		return prior
	}
	if f.Trigger && tenantSettingsKnown(prior) {
		// The API resets one-shot requests after use: keep the known value.
		return prior
	}
	switch f.Kind {
	case tenantSettingsBool:
		switch b := raw.(type) {
		case bool:
			return types.BoolValue(b)
		case string:
			if parsed, err := strconv.ParseBool(b); err == nil {
				return types.BoolValue(parsed)
			}
		}
		if missing {
			// The API omits false values of boolean settings, which all default to false.
			return types.BoolValue(false)
		}
		return types.BoolNull()
	case tenantSettingsInt64, tenantSettingsInt64String:
		if n, ok := tenantSettingsNumber(raw); ok {
			return types.Int64Value(int64(n))
		}
		return types.Int64Null()
	case tenantSettingsFloat64:
		n, ok := tenantSettingsNumber(raw)
		if !ok {
			return types.Float64Null()
		}
		// The API stores single precision numbers: keep the prior value when it is the same number.
		if p, isFloat := prior.(types.Float64); isFloat && tenantSettingsKnown(p) && math.Abs(p.ValueFloat64()-n) <= 1e-6*math.Max(1, math.Abs(n)) {
			return p
		}
		return types.Float64Value(n)
	case tenantSettingsString:
		if missing {
			if p, ok := prior.(types.String); ok && tenantSettingsKnown(p) && p.ValueString() == "" {
				return p
			}
			return types.StringNull()
		}
		return types.StringValue(tenantSettingsStringOf(raw))
	case tenantSettingsStringList:
		items, ok := raw.([]interface{})
		if !ok {
			if p, isList := prior.(types.List); isList && tenantSettingsKnown(p) && len(p.Elements()) == 0 {
				return p
			}
			return types.ListNull(types.StringType)
		}
		values := make([]attr.Value, 0, len(items))
		for _, item := range items {
			values = append(values, types.StringValue(tenantSettingsStringOf(item)))
		}
		list := types.ListValueMust(types.StringType, values)
		// The lists are sets of values: keep the prior order when only the order differs.
		if p, isList := prior.(types.List); isList && tenantSettingsKnown(p) && tenantSettingsSameElements(p.Elements(), values) {
			return p
		}
		return list
	case tenantSettingsStringMap:
		items, ok := raw.(map[string]interface{})
		if !ok {
			if p, isMap := prior.(types.Map); isMap && tenantSettingsKnown(p) && len(p.Elements()) == 0 {
				return p
			}
			return types.MapNull(types.StringType)
		}
		values := make(map[string]attr.Value, len(items))
		for k, item := range items {
			values[k] = types.StringValue(tenantSettingsStringOf(item))
		}
		return types.MapValueMust(types.StringType, values)
	case tenantSettingsJSONObject:
		p, _ := prior.(types.String)
		if !tenantSettingsKnown(p) {
			return jsonStringState(types.StringNull(), raw)
		}
		var priorValue interface{}
		if err := json.Unmarshal([]byte(p.ValueString()), &priorValue); err != nil {
			return jsonStringState(types.StringNull(), raw)
		}
		if missing {
			if f.WriteOnly {
				return p
			}
			return jsonStringState(p, nil)
		}
		if jsonContains(raw, priorValue) {
			// Every configured key has the configured value; the API may add keys, also to array elements.
			return p
		}
		return jsonStringState(p, tenantSettingsProjectJSON(priorValue, raw, f.WriteOnly))
	case tenantSettingsJSONArray:
		p, _ := prior.(types.String)
		if p.IsUnknown() {
			p = types.StringNull()
		}
		if missing {
			return formDefinitionJSONState(p, nil)
		}
		encoded, err := json.Marshal(raw)
		if err != nil {
			return types.StringNull()
		}
		return formDefinitionJSONState(p, encoded)
	}
	return nil
}

func tenantSettingsSameElements(a, b []attr.Value) bool {
	if len(a) != len(b) {
		return false
	}
	as := make([]string, 0, len(a))
	bs := make([]string, 0, len(b))
	for i := range a {
		as = append(as, a[i].String())
		bs = append(bs, b[i].String())
	}
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

func (spec *tenantSettingsSpec) view(doc map[string]interface{}) map[string]interface{} {
	if spec.View != nil {
		return spec.View(doc)
	}
	return doc
}

// tenantSettingsRefresh returns the state values read from the API object. Configured values that
// are equivalent to the API values are kept.
func tenantSettingsRefresh(spec *tenantSettingsSpec, prior map[string]attr.Value, doc map[string]interface{}) map[string]attr.Value {
	view := spec.view(doc)
	values := make(map[string]attr.Value, len(spec.Fields))
	for i := range spec.Fields {
		f := &spec.Fields[i]
		raw, found := tenantSettingsLookup(view, f.Path)
		values[f.Name] = tenantSettingsFromAPI(f, prior[f.Name], raw, found)
	}
	return values
}

// tenantSettingsResolve returns the state values after apply: planned values are kept, unknown
// values are taken from the API object.
func tenantSettingsResolve(spec *tenantSettingsSpec, planned map[string]attr.Value, doc map[string]interface{}) map[string]attr.Value {
	view := spec.view(doc)
	values := make(map[string]attr.Value, len(spec.Fields))
	for i := range spec.Fields {
		f := &spec.Fields[i]
		if v := planned[f.Name]; v != nil && !v.IsUnknown() {
			values[f.Name] = v
			continue
		}
		raw, found := tenantSettingsLookup(view, f.Path)
		values[f.Name] = tenantSettingsFromAPI(f, nil, raw, found)
	}
	return values
}

// ---- Terraform plumbing ----

type tenantSettingsGetter interface {
	GetAttribute(ctx context.Context, p path.Path, target interface{}) diag.Diagnostics
}

type tenantSettingsSetter interface {
	SetAttribute(ctx context.Context, p path.Path, value interface{}) diag.Diagnostics
}

// tenantSettingsValues reads the setting values of a plan, state or configuration.
func tenantSettingsValues(ctx context.Context, g tenantSettingsGetter, spec *tenantSettingsSpec, diags *diag.Diagnostics) map[string]attr.Value {
	values := make(map[string]attr.Value, len(spec.Fields))
	for i := range spec.Fields {
		f := &spec.Fields[i]
		p := path.Root(f.Name)
		switch f.Kind {
		case tenantSettingsBool:
			var v types.Bool
			diags.Append(g.GetAttribute(ctx, p, &v)...)
			values[f.Name] = v
		case tenantSettingsInt64, tenantSettingsInt64String:
			var v types.Int64
			diags.Append(g.GetAttribute(ctx, p, &v)...)
			values[f.Name] = v
		case tenantSettingsFloat64:
			var v types.Float64
			diags.Append(g.GetAttribute(ctx, p, &v)...)
			values[f.Name] = v
		case tenantSettingsStringList:
			var v types.List
			diags.Append(g.GetAttribute(ctx, p, &v)...)
			values[f.Name] = v
		case tenantSettingsStringMap:
			var v types.Map
			diags.Append(g.GetAttribute(ctx, p, &v)...)
			values[f.Name] = v
		default:
			var v types.String
			diags.Append(g.GetAttribute(ctx, p, &v)...)
			values[f.Name] = v
		}
	}
	return values
}

// tenantSettingsSetValues writes the fixed ID and the setting values to a state.
func tenantSettingsSetValues(ctx context.Context, s tenantSettingsSetter, spec *tenantSettingsSpec, values map[string]attr.Value, diags *diag.Diagnostics) {
	diags.Append(s.SetAttribute(ctx, path.Root("id"), spec.ID)...)
	for i := range spec.Fields {
		f := &spec.Fields[i]
		diags.Append(s.SetAttribute(ctx, path.Root(f.Name), values[f.Name])...)
	}
}

func tenantSettingsConfigured(values map[string]attr.Value) map[string]bool {
	configured := make(map[string]bool, len(values))
	for name, v := range values {
		configured[name] = v != nil && !v.IsNull()
	}
	return configured
}

func tenantSettingsFieldDescription(spec *tenantSettingsSpec, f *tenantSettingsField, resourceSchema bool) string {
	writable := resourceSchema && !f.ReadOnly
	// PUT APIs refuse to clear secrets that the API does not return (see tenantSettingsCheckWriteOnly).
	put := spec.Mode != tenantSettingsPatch
	description := f.Description + "."
	switch f.Kind {
	case tenantSettingsJSONObject:
		if writable {
			description += " A JSON object, e.g. built with `jsonencode`. Only the configured keys are managed, they are merged into the current object."
		} else {
			description += " A JSON object."
		}
	case tenantSettingsJSONArray:
		if writable {
			description += " A JSON array, e.g. built with `jsonencode`."
		} else {
			description += " A JSON array."
		}
	}
	if f.Sensitive {
		description += " The value is sensitive."
	}
	if writable && f.WriteOnly {
		if put && f.Kind != tenantSettingsJSONObject {
			description += " When the API does not return the value, the known value is kept in state, and it must be configured, because the update would clear it."
		} else {
			description += " When the API does not return the value, the known value is kept in state."
		}
	}
	if writable && put {
		for _, key := range f.WriteOnlyKeys {
			description += " When the API does not return `" + key + "`, it must be configured here, because the update would clear it."
		}
	}
	if writable && f.Trigger {
		description += " One-shot request that the API resets after use: the state keeps the configured value, and the value is only sent on create and when the configured value changes."
	}
	if writable {
		description += " When not configured, the current tenant value is kept and shown."
	}
	return description
}

func tenantSettingsResourceSchema(spec *tenantSettingsSpec) schema.Schema {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "Always `" + spec.ID + "`.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}
	for i := range spec.Fields {
		f := &spec.Fields[i]
		optional := !f.ReadOnly
		keep := !f.ReadOnly || f.Stable
		description := tenantSettingsFieldDescription(spec, f, true)
		switch f.Kind {
		case tenantSettingsBool:
			a := schema.BoolAttribute{MarkdownDescription: description, Optional: optional, Computed: true, Sensitive: f.Sensitive}
			if keep {
				a.PlanModifiers = []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}
			}
			attributes[f.Name] = a
		case tenantSettingsInt64, tenantSettingsInt64String:
			a := schema.Int64Attribute{MarkdownDescription: description, Optional: optional, Computed: true, Sensitive: f.Sensitive}
			if keep {
				a.PlanModifiers = []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}
			}
			if optional {
				a.Validators = []validator.Int64{int64AtLeastValidator{min: 0}}
			}
			attributes[f.Name] = a
		case tenantSettingsFloat64:
			a := schema.Float64Attribute{MarkdownDescription: description, Optional: optional, Computed: true, Sensitive: f.Sensitive}
			if keep {
				a.PlanModifiers = []planmodifier.Float64{float64planmodifier.UseStateForUnknown()}
			}
			attributes[f.Name] = a
		case tenantSettingsStringList:
			a := schema.ListAttribute{MarkdownDescription: description, ElementType: types.StringType, Optional: optional, Computed: true, Sensitive: f.Sensitive}
			if keep {
				a.PlanModifiers = []planmodifier.List{listplanmodifier.UseStateForUnknown()}
			}
			attributes[f.Name] = a
		case tenantSettingsStringMap:
			a := schema.MapAttribute{MarkdownDescription: description, ElementType: types.StringType, Optional: optional, Computed: true, Sensitive: f.Sensitive}
			if keep {
				a.PlanModifiers = []planmodifier.Map{mapplanmodifier.UseStateForUnknown()}
			}
			attributes[f.Name] = a
		default:
			a := schema.StringAttribute{MarkdownDescription: description, Optional: optional, Computed: true, Sensitive: f.Sensitive}
			if keep {
				a.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
			}
			if optional && f.Kind == tenantSettingsJSONObject {
				a.Validators = []validator.String{jsonObjectStringValidator{}}
			}
			if optional && f.Kind == tenantSettingsJSONArray {
				a.Validators = []validator.String{jsonArrayStringValidator{}}
			}
			attributes[f.Name] = a
		}
	}
	return schema.Schema{MarkdownDescription: spec.Description, Attributes: attributes}
}

func tenantSettingsDataSourceSchema(spec *tenantSettingsSpec, description string) dsschema.Schema {
	attributes := map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{MarkdownDescription: "Always `" + spec.ID + "`.", Computed: true},
	}
	for i := range spec.Fields {
		f := &spec.Fields[i]
		d := tenantSettingsFieldDescription(spec, f, false)
		switch f.Kind {
		case tenantSettingsBool:
			attributes[f.Name] = dsschema.BoolAttribute{MarkdownDescription: d, Computed: true, Sensitive: f.Sensitive}
		case tenantSettingsInt64, tenantSettingsInt64String:
			attributes[f.Name] = dsschema.Int64Attribute{MarkdownDescription: d, Computed: true, Sensitive: f.Sensitive}
		case tenantSettingsFloat64:
			attributes[f.Name] = dsschema.Float64Attribute{MarkdownDescription: d, Computed: true, Sensitive: f.Sensitive}
		case tenantSettingsStringList:
			attributes[f.Name] = dsschema.ListAttribute{MarkdownDescription: d, ElementType: types.StringType, Computed: true, Sensitive: f.Sensitive}
		case tenantSettingsStringMap:
			attributes[f.Name] = dsschema.MapAttribute{MarkdownDescription: d, ElementType: types.StringType, Computed: true, Sensitive: f.Sensitive}
		default:
			attributes[f.Name] = dsschema.StringAttribute{MarkdownDescription: d, Computed: true, Sensitive: f.Sensitive}
		}
	}
	return dsschema.Schema{MarkdownDescription: description, Attributes: attributes}
}

// tenantSettingsResource is the resource implementation shared by all tenant settings.
type tenantSettingsResource struct {
	spec   *tenantSettingsSpec
	client *Config
}

var _ resource.Resource = &tenantSettingsResource{}
var _ resource.ResourceWithImportState = &tenantSettingsResource{}

func (r *tenantSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.spec.TypeName
}

func (r *tenantSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = tenantSettingsResourceSchema(r.spec)
}

func (r *tenantSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// apply saves the configured settings. The settings always exist, so Create and Update both
// update them; Create has no prior values and sends every configured setting.
func (r *tenantSettingsResource) apply(ctx context.Context, planned, prior, config map[string]attr.Value, create bool, diags *diag.Diagnostics) map[string]attr.Value {
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		diags.AddError("Client Error", err.Error())
		return nil
	}
	doc, err := client.tenantSettingsApply(ctx, r.spec, planned, prior, tenantSettingsConfigured(config), create)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to update %s: %s", strings.ToLower(r.spec.Title), err))
		return nil
	}
	return tenantSettingsResolve(r.spec, planned, doc)
}

func (r *tenantSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	planned := tenantSettingsValues(ctx, req.Plan, r.spec, &resp.Diagnostics)
	config := tenantSettingsValues(ctx, req.Config, r.spec, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	values := r.apply(ctx, planned, nil, config, true, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.Raw = req.Plan.Raw.Copy()
	tenantSettingsSetValues(ctx, &resp.State, r.spec, values, &resp.Diagnostics)
}

func (r *tenantSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	prior := tenantSettingsValues(ctx, req.State, r.spec, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	doc, err := client.tenantSettingsGet(ctx, r.spec)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read %s: %s", strings.ToLower(r.spec.Title), err))
		return
	}
	tenantSettingsSetValues(ctx, &resp.State, r.spec, tenantSettingsRefresh(r.spec, prior, doc), &resp.Diagnostics)
}

func (r *tenantSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	planned := tenantSettingsValues(ctx, req.Plan, r.spec, &resp.Diagnostics)
	prior := tenantSettingsValues(ctx, req.State, r.spec, &resp.Diagnostics)
	config := tenantSettingsValues(ctx, req.Config, r.spec, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	values := r.apply(ctx, planned, prior, config, false, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.Raw = req.Plan.Raw.Copy()
	tenantSettingsSetValues(ctx, &resp.State, r.spec, values, &resp.Diagnostics)
}

func (r *tenantSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	singletonDeleteWarning(resp, r.spec.Title)
}

func (r *tenantSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), r.spec.ID)...)
}

// tenantSettingsDataSource reads all settings of a tenant settings object.
type tenantSettingsDataSource struct {
	spec        *tenantSettingsSpec
	typeName    string
	description string
	client      *Config
}

var _ datasource.DataSource = &tenantSettingsDataSource{}

func (d *tenantSettingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.typeName
}

func (d *tenantSettingsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = tenantSettingsDataSourceSchema(d.spec, d.description)
}

func (d *tenantSettingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *Config, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *tenantSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	doc, err := client.tenantSettingsGet(ctx, d.spec)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read %s: %s", strings.ToLower(d.spec.Title), err))
		return
	}
	tenantSettingsSetValues(ctx, &resp.State, d.spec, tenantSettingsRefresh(d.spec, nil, doc), &resp.Diagnostics)
}
