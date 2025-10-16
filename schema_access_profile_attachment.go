package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func accessProfileAttachmentFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"source_app_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Source App id",
		},

		"access_profiles": {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 250,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	}
	return s
}
