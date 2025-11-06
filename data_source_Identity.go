package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceIdentity() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceIdentityRead,

		Schema: identityFields(),
	}
}

func dataSourceIdentityRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	alias, _ := d.Get("alias").(string)
	email, _ := d.Get("email_address").(string)

	if alias != "" && email != "" {
		return diag.FromErr(fmt.Errorf("only one of 'alias' or 'email_address' must be set"))
	}

	if alias == "" && email == "" {
		return diag.FromErr(fmt.Errorf("one of 'alias' or 'email_address' must be set"))
	}

	if alias != "" {
		log.Printf("[INFO] Getting Data source for Identity. Identity alias %s", alias)
		client, err := meta.(*Config).IdentityNowClient(ctx)
		if err != nil {
			return diag.FromErr(err)
		}

		identity, err := client.GetIdentityByAlias(ctx, alias)
		if err != nil {
			// non-panicking type assertion, 2nd arg is boolean indicating type match
			_, notFound := err.(*NotFoundError)
			if notFound {
				log.Printf("[INFO] Data source for Identity alias %s not found.", alias)
				return nil
			}
			return diag.FromErr(err)
		}
		if len(identity) > 0 {
			err = flattenIdentity(d, identity[0])
			if err != nil {
				return diag.FromErr(err)
			}
			return nil
		} else {
			log.Printf("[INFO] Data source for Identity alias %s not found.", alias)
			return nil
		}
	}

	if email != "" {
		log.Printf("[INFO] Getting Data source for Identity. Identity email %s", email)
		client, err := meta.(*Config).IdentityNowClient(ctx)
		if err != nil {
			return diag.FromErr(err)
		}

		identity, err := client.GetIdentityByEmail(ctx, email)
		if err != nil {
			// non-panicking type assertion, 2nd arg is boolean indicating type match
			_, notFound := err.(*NotFoundError)
			if notFound {
				log.Printf("[INFO] Data source for Identity email %s not found.", email)
				return nil
			}
			return diag.FromErr(err)
		}
		if len(identity) > 0 {
			err = flattenIdentity(d, identity[0])
			if err != nil {
				return diag.FromErr(err)
			}
			return nil
		} else {
			log.Printf("[INFO] Data source for Identity email %s not found.", email)
			return nil
		}
	}
	log.Printf("[INFO] Data source for Identity not found. No email nor alias match.")
	return nil
}
