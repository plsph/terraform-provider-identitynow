package main

type Dimension struct {
	Description    string          `json:"description"`
	ID             string          `json:"id,omitempty"`
	Name           string          `json:"name"`
	Owner          *ObjectInfo     `json:"owner,omitempty"`
	AccessProfiles []*ObjectInfo   `json:"accessProfiles,omitempty"`
	Entitlements   []*ObjectInfo   `json:"entitlements,omitempty"`
	Membership     *RoleMembership `json:"membership,omitempty"`
}

type UpdateDimension struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}
