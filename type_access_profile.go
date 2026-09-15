package main

type AccessProfile struct {
	Description             string                                `json:"description"`
	Created                 string                                `json:"created,omitempty"`
	Modified                string                                `json:"modified,omitempty"`
	Enabled                 *bool                                 `json:"enabled,omitempty"`
	Entitlements            []*ObjectInfo                         `json:"entitlements,omitempty"`
	ID                      string                                `json:"id,omitempty"`
	Name                    string                                `json:"name,omitempty"`
	AccessProfileOwner      *ObjectInfo                           `json:"owner,omitempty"`
	AccessProfileSource     *ObjectInfo                           `json:"source,omitempty"`
	Requestable             *bool                                 `json:"requestable,omitempty"`
	AccessRequestConfig     *AccessRequestConfigList              `json:"accessRequestConfig,omitempty"`
	RevocationRequestConfig *AccessProfileRevocationRequestConfig `json:"revocationRequestConfig,omitempty"`
	Segments                []string                              `json:"segments,omitempty"`
	AccessModelMetadata     *AttributeDTOList                     `json:"accessModelMetadata,omitempty"`
	ProvisioningCriteria    *ProvisioningCriteriaLevel1           `json:"provisioningCriteria,omitempty"`
	AdditionalOwners        []*AdditionalOwnerRef                 `json:"additionalOwners,omitempty"`
}

type AccessRequestConfigList struct {
	CommentsRequired           bool                        `json:"commentsRequired,omitempty"`
	DenialCommentsRequired     bool                        `json:"denialCommentsRequired,omitempty"`
	ApprovalSchemes            []*ApprovalSchemes          `json:"approvalSchemes,omitempty"`
	ReauthorizationRequired    bool                        `json:"reauthorizationRequired,omitempty"`
	RequireEndDate             bool                        `json:"requireEndDate,omitempty"`
	MaxPermittedAccessDuration *MaxPermittedAccessDuration `json:"maxPermittedAccessDuration,omitempty"`
}

type AccessProfileRevocationRequestConfig struct {
	ApprovalSchemes []*ApprovalSchemes `json:"approvalSchemes,omitempty"`
}

type AdditionalOwnerRef struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type ProvisioningCriteriaLevel1 struct {
	Operation string                        `json:"operation,omitempty"`
	Attribute string                        `json:"attribute,omitempty"`
	Value     string                        `json:"value,omitempty"`
	Children  []*ProvisioningCriteriaLevel2 `json:"children,omitempty"`
}

type ProvisioningCriteriaLevel2 struct {
	Operation string                        `json:"operation,omitempty"`
	Attribute string                        `json:"attribute,omitempty"`
	Value     string                        `json:"value,omitempty"`
	Children  []*ProvisioningCriteriaLevel3 `json:"children,omitempty"`
}

type ProvisioningCriteriaLevel3 struct {
	Operation string `json:"operation,omitempty"`
	Attribute string `json:"attribute,omitempty"`
	Value     string `json:"value,omitempty"`
}

type UpdateAccessProfile struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

type ApprovalSchemes struct {
	ApproverType string `json:"approverType"`
	ApproverId   string `json:"approverId,omitempty"`
}

type MaxPermittedAccessDuration struct {
	Value    int    `json:"value"`
	TimeUnit string `json:"timeUnit"`
}
