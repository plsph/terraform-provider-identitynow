package main

import (
	"context"
	"log"

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
	log.Printf("[INFO] Getting Data source for Entitlements. Source ID %s", d.Get("source_id").(string))
	client, err := meta.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	sourceEntitlements, err := client.GetSourceEntitlement(ctx, d.Get("source_id").(string), d.Get("name").(string))
	if ( err != nil || len(sourceEntitlements) == 0 ) {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if ( notFound || len(sourceEntitlements) == 0 ) {
			log.Printf("[INFO] Data source for Source ID %s not found.", d.Get("source_id").(string))
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
