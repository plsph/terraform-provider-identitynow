package main

type GovernanceGroup struct {
	Description          string                `json:"description"`
	ID                   string                `json:"id,omitempty"`
	Name                 string                `json:"name,omitempty"`
	GovernanceGroupOwner *GovernanceGroupOwner `json:"owner,omitempty"`
	DirectPermissions    []string              `json:"direct_permissions,omitempty"` // Ignore this field - not used in Terraform
}

type UpdateGovernanceGroup struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

type GovernanceGroupOwner struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"displayName"`
}
