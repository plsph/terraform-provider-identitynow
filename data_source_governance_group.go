package main

import (
	"context"


	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func dataSourceGovernanceGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGovernanceGroupRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Source id",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Governance Group name",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Governance Group description",
			},

			"source": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: governanceGroupSourceFields(),
				},
			},

		},
	}
}

func dataSourceGovernanceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Getting Governance Group data source", map[string]interface{}{"name": d.Get("name").(string)})
	client, err := meta.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	governanceGroup, err := client.GetGovernanceGroupByName(ctx, d.Get("name").(string))
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Governance Group not found in data source", map[string]interface{}{"name": d.Get("name").(string)})
			return nil
		}
		return diag.FromErr(err)
	}

	if len(governanceGroup) > 0 {
		err = flattenGovernanceGroup(d, governanceGroup[0])
		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	} else {
		tflog.Debug(ctx, "Governance Group not found", map[string]interface{}{"name": d.Get("name").(string)})
		return nil
	}
	return nil
}
