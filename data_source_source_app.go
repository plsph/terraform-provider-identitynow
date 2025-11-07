package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceApp() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSourceAppRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Source id",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Source App name",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Source App description",
			},

			"match_all_accounts": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"source": {
				Type:     schema.TypeList,
				Computed: true,
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
		},
	}
}

func dataSourceSourceAppRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Getting Source App data source", map[string]interface{}{"name": d.Get("name").(string)})
	client, err := meta.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	sourceApp, err := client.GetSourceAppByName(ctx, d.Get("name").(string))
	if err != nil || len(sourceApp) == 0 {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Source App not found in data source", map[string]interface{}{"name": d.Get("name").(string)})
			return nil
		}
		return diag.FromErr(err)
	}

	err = flattenSourceApp(d, sourceApp[0])
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
