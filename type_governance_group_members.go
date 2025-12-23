package main

type GovernanceGroupMembers struct {
	GovernanceGroupId        string                   `json:"id,omitempty"`
	GovernanceGroupMembersMembers   []*GovernanceGroupMembersMembers  `json:"governanceGroupMembers,omitempty"`
}

type GovernanceGroupMembersMembers struct {
        ID   string `json:"id"`
        Name string `json:"name"`
        Type string `json:"type"`
}

type GovernanceGroupMembersResponse struct {
        ID          string `json:"id"`
        Description string `json:"description"`
        Status      int    `json:"status"`
}

