package main

type Segment struct {
	ID                 string      `json:"id,omitempty"`
	Name               string      `json:"name"`
	Created            string      `json:"created,omitempty"`
	Modified           string      `json:"modified,omitempty"`
	Description        string      `json:"description,omitempty"`
	Owner              *ObjectInfo `json:"owner,omitempty"`
	VisibilityCriteria interface{} `json:"visibilityCriteria,omitempty"`
	Active             bool        `json:"active"`
}

type UpdateSegment struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}
