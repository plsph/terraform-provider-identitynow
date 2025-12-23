package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func governanceGroupMembersFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"governance_group_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Governance Group id",
		},

		"members": {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 50,
			Elem: &schema.Resource{
				Schema: governanceGroupMembersMembersFields(),
			},
		},
	}
	return s
}

func governanceGroupMembersMembersFields() map[string]*schema.Schema {
        s := map[string]*schema.Schema{
                "id": {
                        Type:        schema.TypeString,
                        Required:    true,
                        Description: "Id of member",
                },
                "name": {
                        Type:        schema.TypeString,
                        Required:    true,
                        Description: "Name of member",
                },
                "type": {
                        Type:        schema.TypeString,
                        Optional:    true,
                        Default:     "IDENTITY",
                        Description: "Type of member",
                },
        }

        return s
}
