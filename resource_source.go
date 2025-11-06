package main

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceSource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSourceCreate,
		ReadContext:   resourceSourceRead,
		UpdateContext: resourceSourceUpdate,
		DeleteContext: resourceSourceDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceSourceImport,
		},

		Schema: sourceFields(),
	}
}

func resourceSourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	source, err := expandSource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Creating Source %s", source.Name)

	c, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	newSource, err := c.CreateSource(ctx, source)
	if err != nil {
		return diag.FromErr(err)
	}

	err = flattenSource(d, newSource)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceSourceRead(ctx, d, m)
}

func resourceSourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[INFO] Refreshing source ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	source, err := client.GetSource(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Source ID %s not found.", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = flattenSource(d, source)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceSourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[INFO] Updating Source ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	updatedSource, err := expandSource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.UpdateSource(ctx, updatedSource)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceSourceRead(ctx, d, m)
}

func resourceSourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Printf("[INFO] Deleting Source ID %s", d.Id())

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	source, err := client.GetSource(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Source ID %s not found.", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = client.DeleteSource(ctx, source)
	if err != nil {
		return diag.FromErr(fmt.Errorf("Error removing Source: %s", err))
	}

	d.SetId("")
	return nil
}
