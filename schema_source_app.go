package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func sourceAppFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Source App name",
		},
		"description": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Source App description",
		},

		"match_all_accounts": {
			Type:     schema.TypeBool,
			Optional: true,
		},

		"enabled": {
			Type:     schema.TypeBool,
			Optional: true,
		},

		"source": {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: sourceAppSourceFields(),
			},
		},

		"date_created": {
			Type:     schema.TypeString,
			Computed: true,
		},

		"last_updated": {
			Type:     schema.TypeString,
			Computed: true,
		},
	}
	return s
}

func sourceAppSourceFields() map[string]*schema.Schema {
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
