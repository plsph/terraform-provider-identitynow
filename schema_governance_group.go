package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func governanceGroupFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Governance Group name",
		},
		"description": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Governance Group description",
		},

		"owner": {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: governanceGroupSourceFields(),
			},
		},
	}
	return s
}

func governanceGroupSourceFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Id of source",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of source",
		},
		"type": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "SOURCE",
			Description: "Type of source",
		},
	}

	return s
}
