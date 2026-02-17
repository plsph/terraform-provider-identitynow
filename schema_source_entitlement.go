package main

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func sourceEntitlementFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Source entitlements name",
		},
		"source_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "source id",
		},
		"source_name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "source name",
		},
		"source_schema_object_type": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"attribute": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "attribute",
		},
		"created": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"description": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"modified": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"owner": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: sourceOwnerFields(),
			},
		},
		"privileged": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"requestable": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"value": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"direct_permissions": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	}

	return s
}
