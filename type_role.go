package main

type Role struct {
	Description          string          `json:"description"`
	ID                   string          `json:"id,omitempty"`
	Name                 string          `json:"name"`
	Requestable          *bool           `json:"requestable,omitempty"`
	RoleOwner            *ObjectInfo     `json:"owner,omitempty"`
	AccessProfiles       []*ObjectInfo   `json:"accessProfiles,omitempty"`
	Entitlements         []*ObjectInfo   `json:"entitlements,omitempty"`
	LegacyMembershipInfo interface{}     `json:"legacyMembershipInfo,omitempty"`
	Enabled              *bool           `json:"enabled,omitempty"`
	Segments             []interface{}   `json:"segments,omitempty"`
	Membership           *RoleMembership `json:"membership,omitempty"`
	AccessRequestConfig  struct {
		CommentsRequired       *bool         `json:"commentsRequired,omitempty"`
		DenialCommentsRequired *bool         `json:"denialCommentsRequired,omitempty"`
		ApprovalSchemes        []interface{} `json:"approvalSchemes,omitempty"`
	} `json:"accessRequestConfig,omitempty"`
	RevocationRequestConfig struct {
		ApprovalSchemes []interface{} `json:"approvalSchemes,omitempty"`
	} `json:"revocationRequestConfig,omitempty"`
}
type ObjectInfo struct {
	ID   interface{} `json:"id,omitempty"`
	Type string      `json:"type,omitempty"`
	Name string      `json:"name"`
}

type RoleMembership struct {
	Type     string                  `json:"type,omitempty"`
	Criteria *RoleMembershipCriteria `json:"criteria,omitempty"`
}

type RoleMembershipCriteria struct {
	Operation   string                    `json:"operation,omitempty"`
	Key         *RoleKey                  `json:"key,omitempty"`
	StringValue string                    `json:"stringValue,omitempty"`
	Children    []*RoleMembershipCriteria `json:"children,omitempty"`
}

type RoleKey struct {
	Type     string      `json:"type,omitempty"`
	Property interface{} `json:"property,omitempty"`
	SourceId interface{} `json:"sourceId,omitempty"`
}

type UpdateRole struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}
