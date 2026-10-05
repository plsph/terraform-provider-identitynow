package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// equalIgnoringUnknown reports whether plan equals state, treating values that are unknown in plan
// as equal. Optional and computed nested attributes are unknown in the plan whenever the resource
// changes, so a plain Equal would report every nested block with such attributes as changed.
func equalIgnoringUnknown(plan, state attr.Value) bool {
	if plan == nil || state == nil {
		return plan == state
	}
	if plan.IsUnknown() {
		return true
	}
	if plan.IsNull() || state.IsNull() || state.IsUnknown() {
		return plan.Equal(state)
	}
	switch p := plan.(type) {
	case types.List:
		s, ok := state.(types.List)
		return ok && elementsEqualIgnoringUnknown(p.Elements(), s.Elements())
	case types.Set:
		s, ok := state.(types.Set)
		return ok && elementsEqualIgnoringUnknown(p.Elements(), s.Elements())
	case types.Object:
		s, ok := state.(types.Object)
		if !ok {
			return false
		}
		stateAttributes := s.Attributes()
		for name, value := range p.Attributes() {
			if !equalIgnoringUnknown(value, stateAttributes[name]) {
				return false
			}
		}
		return len(p.Attributes()) == len(stateAttributes)
	}
	return plan.Equal(state)
}

func elementsEqualIgnoringUnknown(plan, state []attr.Value) bool {
	if len(plan) != len(state) {
		return false
	}
	for i := range plan {
		if !equalIgnoringUnknown(plan[i], state[i]) {
			return false
		}
	}
	return true
}
