package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSourceEntitlement() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSourceEntitlementRead,

		Schema: sourceEntitlementFields(),
	}
}

func dataSourceSourceEntitlementRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Getting Source Entitlements data source", map[string]interface{}{"source_id": d.Get("source_id").(string)})
	client, err := meta.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	sourceEntitlements, err := client.GetSourceEntitlement(ctx, d.Get("source_id").(string), d.Get("name").(string))
	if err != nil || len(sourceEntitlements) == 0 {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound || len(sourceEntitlements) == 0 {
			tflog.Debug(ctx, "Source not found in data source", map[string]interface{}{"source_id": d.Get("source_id").(string)})
			return nil
		}
		return diag.FromErr(err)
	}

	err = flattenSourceEntitlement(d, sourceEntitlements[0])
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
