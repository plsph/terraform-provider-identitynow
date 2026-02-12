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
			Optional: true,
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
			Optional:    true,
			Description: "If comment is required",
			Default:     false,
		},
		"denial_comments_required": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If denial comment is required",
			Default:     false,
		},
		"reauthorization_required": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Indicates whether reauthorization is required for the request.",
			Default:     false,
		},
		"approval_schemes": {
			Type:     schema.TypeList,
			Optional:    true,
			Elem: &schema.Resource{
				Schema: accessProfileApprovalSchemesFields(),
			},
		},
		"require_end_date": {
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
			Description: "Indicates whether the requester of the containing object must provide access end date.",
		},
		"max_permitted_access_duration": {
			Type:     schema.TypeList,
			Optional:    true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: accessProfileMaxPermittedAccessDuration(),
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

func accessProfileMaxPermittedAccessDuration() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"value": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "The numeric value representing the amount of time, which is defined in the timeUnit.",
		},
		"time_unit": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The unit of time that corresponds to the value. It defines the scale of the time period.",
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
