package main

import (
	"context"
	"fmt"


	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
		tflog.Info(ctx, "Getting Identity data source by alias", map[string]interface{}{"alias": alias})
		client, err := meta.(*Config).IdentityNowClient(ctx)
		if err != nil {
			return diag.FromErr(err)
		}

		identity, err := client.GetIdentityByAlias(ctx, alias)
		if err != nil {
			// non-panicking type assertion, 2nd arg is boolean indicating type match
			_, notFound := err.(*NotFoundError)
			if notFound {
				tflog.Debug(ctx, "Identity not found by alias", map[string]interface{}{"alias": alias})
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
			tflog.Debug(ctx, "Identity not found by alias", map[string]interface{}{"alias": alias})
			return nil
		}
	}

	if email != "" {
		tflog.Info(ctx, "Getting Identity data source by email", map[string]interface{}{"email": email})
		client, err := meta.(*Config).IdentityNowClient(ctx)
		if err != nil {
			return diag.FromErr(err)
		}

		identity, err := client.GetIdentityByEmail(ctx, email)
		if err != nil {
			// non-panicking type assertion, 2nd arg is boolean indicating type match
			_, notFound := err.(*NotFoundError)
			if notFound {
				tflog.Debug(ctx, "Identity not found by email", map[string]interface{}{"email": email})
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
			tflog.Debug(ctx, "Identity not found by email", map[string]interface{}{"email": email})
			return nil
		}
	}
	tflog.Debug(ctx, "Identity not found - no email nor alias match")
	return nil
}
