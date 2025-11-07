package main

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func dataSourceRole() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRoleRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"access_profiles": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: roleAccessProfilesFields(),
				},
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"name": {
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
			"requestable": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Getting Role data source", map[string]interface{}{"id": d.Get("id").(string)})
	client, err := meta.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := client.GetRole(ctx, d.Get("id").(string))
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Role not found in data source", map[string]interface{}{"id": d.Get("id").(string)})
			return nil
		}
		return diag.FromErr(err)
	}

	err = flattenRole(d, role)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
