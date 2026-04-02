package main

type Workflow struct {
	ID             string              `json:"id,omitempty"`
	Name           string              `json:"name"`
	Description    string              `json:"description,omitempty"`
	Owner          *WorkflowOwner      `json:"owner,omitempty"`
	Enabled        *bool               `json:"enabled,omitempty"`
	Trigger        *WorkflowTrigger    `json:"trigger,omitempty"`
	Definition     *WorkflowDefinition `json:"definition,omitempty"`
	ExecutionCount *int                `json:"executionCount,omitempty"`
	FailureCount   *int                `json:"failureCount,omitempty"`
	Created        string              `json:"created,omitempty"`
	Modified       string              `json:"modified,omitempty"`
}

type WorkflowOwner struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

type WorkflowTrigger struct {
	Type        string      `json:"type,omitempty"`
	DisplayName string      `json:"displayName,omitempty"`
	Attributes  interface{} `json:"attributes,omitempty"`
}

type WorkflowDefinition struct {
	Start string      `json:"start,omitempty"`
	Steps interface{} `json:"steps,omitempty"`
}
