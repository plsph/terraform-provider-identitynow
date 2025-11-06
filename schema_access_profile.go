package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func accessProfileFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Access Profile name",
		},
		"description": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Access Profile description",
		},

		"source": {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: accessProfileSourceFields(),
			},
		},

		"owner": {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: sourceOwnerFields(),
			},
		},

		"entitlements": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Resource{
				Schema: accessProfileEntitlementsFields(),
			},
		},

		"enabled": {
			Type:     schema.TypeBool,
			Optional: true,
		},

		"requestable": {
			Type:     schema.TypeBool,
			Optional: true,
		},

		"access_request_config": {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: accessProfileAccessRequestConfigFields(),
			},
		},
	}
	return s
}

func accessProfileSourceFields() map[string]*schema.Schema {
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

func accessProfileAccessRequestConfigFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"comments_required": {
			Type:        schema.TypeBool,
			Required:    true,
			Description: "If comment is required",
		},
		"denial_comments_required": {
			Type:        schema.TypeBool,
			Required:    true,
			Description: "If denial comment is required",
		},
		"approval_schemes": {
			Type:     schema.TypeList,
			Required: true,
			Elem: &schema.Resource{
				Schema: accessProfileApprovalSchemesFields(),
			},
		},
	}

	return s
}

func accessProfileApprovalSchemesFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"approver_type": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Type of approver",
		},
		"approver_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Id of approver",
			Default:     "",
		},
	}

	return s
}

func accessProfileEntitlementsFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Id of entitlement",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of entitlement",
		},
		"type": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "ENTITLEMENT",
			Description: "Type of entitlement",
		},
	}

	return s
}
