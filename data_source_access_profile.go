package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAccessProfile() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAccessProfileRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Source id",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Access Profile name",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Access Profile description",
			},

			"source": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: accessProfileSourceFields(),
				},
			},

			"owner": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: sourceOwnerFields(),
				},
			},

			"entitlements": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: accessProfileEntitlementsFields(),
				},
			},

			"enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"requestable": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceAccessProfileRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Getting Access Profile data source", map[string]interface{}{"name": d.Get("name").(string)})
	client, err := meta.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	accessProfile, err := client.GetAccessProfileByName(ctx, d.Get("name").(string))
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Access Profile not found in data source", map[string]interface{}{"name": d.Get("name").(string)})
			return nil
		}
		return diag.FromErr(err)
	}

	if len(accessProfile) > 0 {
		err = flattenAccessProfile(d, accessProfile[0])
		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	} else {
		tflog.Debug(ctx, "Access Profile not found", map[string]interface{}{"name": d.Get("name").(string)})
		return nil
	}
	return nil
}
