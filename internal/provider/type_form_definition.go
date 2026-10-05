package provider

import "encoding/json"

type FormDefinition struct {
	ID             string                 `json:"id,omitempty"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Owner          *FormOwner             `json:"owner,omitempty"`
	UsedBy         []*FormUsedBy          `json:"usedBy,omitempty"`
	FormInput      []*FormDefinitionInput `json:"formInput,omitempty"`
	FormElements   json.RawMessage        `json:"formElements,omitempty"`
	FormConditions json.RawMessage        `json:"formConditions,omitempty"`
	Created        string                 `json:"created,omitempty"`
	Modified       string                 `json:"modified,omitempty"`
}

type FormOwner struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type FormUsedBy struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type FormDefinitionInput struct {
	ID          string `json:"id,omitempty"`
	Type        string `json:"type,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

type ListFormDefinitionsResponse struct {
	Count   int64             `json:"count"`
	Results []*FormDefinition `json:"results"`
}

type UpdateFormDefinition struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}
