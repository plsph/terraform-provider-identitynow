package main

import (
	"context"
	"fmt"
"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func resourceSourceApp() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSourceAppCreate,
		ReadContext:   resourceSourceAppRead,
		UpdateContext: resourceSourceAppUpdate,
		DeleteContext: resourceSourceAppDelete,

                Importer: &schema.ResourceImporter{
                        StateContext: resourceSourceAppImport,
                },

		Schema: sourceAppFields(),
	}
}

func resourceSourceAppCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sourceApp, err := expandSourceApp(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Creating Source App", map[string]interface{}{"name": sourceApp.Name})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	newSourceApp, err := client.CreateSourceApp(ctx, sourceApp)
	if err != nil {
		return diag.FromErr(err)
	}

	err = flattenSourceApp(d, newSourceApp)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceSourceAppRead(ctx, d, m)
}

func resourceSourceAppRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Refreshing Source App", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	sourceApp, err := client.GetSourceApp(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Source App not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return diag.FromErr(err)
		}
		return diag.FromErr(err)
	}

	err = flattenSourceApp(d, sourceApp)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceSourceAppUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Updating Source App", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	updatedSourceApp, id, err := expandUpdateSourceApp(d)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.UpdateSourceApp(ctx, updatedSourceApp, id)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceSourceAppRead(ctx, d, m)
}

func resourceSourceAppDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Deleting Source App", map[string]interface{}{"id": d.Id()})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	sourceApp, err := client.GetSourceApp(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Source App not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = client.DeleteSourceApp(ctx, sourceApp)
	if err != nil {
		return diag.FromErr(fmt.Errorf("Error removing Source App: %s", err))
	}

	d.SetId("")
	return nil
}
