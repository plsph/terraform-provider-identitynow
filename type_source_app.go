package main

type SourceApp struct {
	Description         string                   `json:"description"`
	Enabled             *bool                    `json:"enabled,omitempty"`
	ID                  string                   `json:"id,omitempty"`
	Name                string                   `json:"name,omitempty"`
	SourceAppSource     *ObjectInfo              `json:"accountSource,omitempty"`
	MatchAllAccounts    *bool                    `json:"matchAllAccounts,omitempty"`
}

type UpdateSourceApp struct {
	Op    string        `json:"op"`
	Path  string        `json:"path"`
	Value interface{}   `json:"value"`
}
