package main

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func identityFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"alias": {
			Type:     schema.TypeString,
			Optional: true,
		},

		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Identity name",
		},
		"description": {
			Type:     schema.TypeString,
			Computed: true,
		},

		"email_address": {
			Type:     schema.TypeString,
			Optional: true,
		},

		"enabled": {
			Type:     schema.TypeBool,
			Computed: true,
		},

		"is_manager": {
			Type:     schema.TypeBool,
			Computed: true,
		},

		"identity_status": {
			Type:     schema.TypeString,
			Computed: true,
		},

		"attributes": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: identityAttributesFields(),
			},
		},
	}
	return s
}

func identityAttributesFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"adp_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"lastname": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"firstname": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"phone": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"user_type": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"uid": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"email": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"workday_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}
	return s
}
