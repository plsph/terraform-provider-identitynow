package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestSegmentVisibilityCriteriaRoundTripsThreeLevels(t *testing.T) {
	ctx := context.Background()
	criteria := &SegmentVisibilityCriteria{Expression: &SegmentVisibilityExpression{
		Operator: "AND",
		Children: []*SegmentVisibilityExpression{
			{
				Operator: "OR",
				Children: []*SegmentVisibilityExpression{
					{Operator: "EQUALS", Attribute: "location", Value: &SegmentVisibilityValue{Type: "STRING", Value: "Austin"}},
				},
			},
			{Operator: "EQUALS", Attribute: "department", Value: &SegmentVisibilityValue{Type: "STRING", Value: "IT"}},
		},
	}}

	var diags diag.Diagnostics
	state := segmentVisibilityCriteriaState(ctx, criteria, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics converting to state: %v", diags)
	}
	got := segmentVisibilityValue(ctx, state, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics converting from state: %v", diags)
	}

	root := got.Expression
	if root.Operator != "AND" || root.Attribute != "" || len(root.Children) != 2 {
		t.Fatalf("unexpected root %+v", root)
	}
	leaf := root.Children[0].Children[0]
	if leaf.Operator != "EQUALS" || leaf.Attribute != "location" || leaf.Value.Value != "Austin" {
		t.Fatalf("unexpected third level expression %+v", leaf)
	}
	if root.Children[1].Value.Value != "IT" {
		t.Fatalf("unexpected second level expression %+v", root.Children[1])
	}
}

func TestSegmentVisibilityStateUsesNullForEmptyStrings(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	state := segmentVisibilityCriteriaState(ctx, &SegmentVisibilityCriteria{Expression: &SegmentVisibilityExpression{Operator: "AND"}}, &diags)
	var criteria []SegmentVisibilityCriteriaModel
	diags.Append(state.ElementsAs(ctx, &criteria, false)...)
	var expressions []SegmentVisibilityExpressionModel
	diags.Append(criteria[0].Expression.ElementsAs(ctx, &expressions, false)...)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !expressions[0].Attribute.IsNull() || !expressions[0].Value.IsNull() {
		t.Fatalf("expected null attribute and value for an AND node, got %+v", expressions[0])
	}
}
