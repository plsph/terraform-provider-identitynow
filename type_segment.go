package main

type Segment struct {
	ID                 string                     `json:"id,omitempty"`
	Name               string                     `json:"name"`
	Created            string                     `json:"created,omitempty"`
	Modified           string                     `json:"modified,omitempty"`
	Description        string                     `json:"description,omitempty"`
	Owner              *ObjectInfo                `json:"owner,omitempty"`
	VisibilityCriteria *SegmentVisibilityCriteria `json:"visibilityCriteria,omitempty"`
	Active             bool                       `json:"active"`
}

type SegmentVisibilityCriteria struct {
	Expression *SegmentVisibilityExpression `json:"expression,omitempty"`
}

type SegmentVisibilityExpression struct {
	Operator  string                         `json:"operator,omitempty"`
	Attribute string                         `json:"attribute,omitempty"`
	Value     *SegmentVisibilityValue        `json:"value,omitempty"`
	Children  []*SegmentVisibilityExpression `json:"children,omitempty"`
}

type SegmentVisibilityValue struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type UpdateSegment struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}
