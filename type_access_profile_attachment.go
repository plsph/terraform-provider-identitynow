package main

type AccessProfileAttachment struct {
	AccessProfiles      []string                 `json:"accessProfiles,omitempty"`
	SourceAppId         string                   `json:"id,omitempty"`
}

type UpdateAccessProfileAttachment struct {
        Op    string        `json:"op"`
        Path  string        `json:"path"`
        Value []string      `json:"value"`
}

type AccessProfileFromSourceApp struct {
	Description         string                   `json:"description"`
	Enabled             *bool                    `json:"enabled,omitempty"`
	Entitlements        []string                 `json:"entitlements,omitempty"`
	ID                  string                   `json:"id,omitempty"`
	Name                string                   `json:"name,omitempty"`
}
