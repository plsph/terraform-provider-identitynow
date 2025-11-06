package main

import (
	"context"
	"fmt"


	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func resourceAccessProfile() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAccessProfileCreate,
		ReadContext:   resourceAccessProfileRead,
		UpdateContext: resourceAccessProfileUpdate,
		DeleteContext: resourceAccessProfileDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceAccessProfileImport,
		},

		Schema: accessProfileFields(),
	}
}

func resourceAccessProfileCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	accessProfile, err := expandAccessProfile(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Creating Access Profile", map[string]interface{}{"name": accessProfile.Name})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	newAccessProfile, err := client.CreateAccessProfile(ctx, accessProfile)
	if err != nil {
		return diag.FromErr(err)
	}

	err = flattenAccessProfile(d, newAccessProfile)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceAccessProfileRead(ctx, d, m)
}

func resourceAccessProfileRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Refreshing Access Profile", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	accessProfile, err := client.GetAccessProfile(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Access Profile not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return diag.FromErr(err)
		}
		return diag.FromErr(err)
	}

	err = flattenAccessProfile(d, accessProfile)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAccessProfileUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Updating Access Profile", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	updatedAccessProfile, id, err := expandUpdateAccessProfile(d)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.UpdateAccessProfile(ctx, updatedAccessProfile, id)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceAccessProfileRead(ctx, d, m)
}

func resourceAccessProfileDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Deleting Access Profile", map[string]interface{}{"id": d.Id()})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	accessProfile, err := client.GetAccessProfile(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Access Profile not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = client.DeleteAccessProfile(ctx, accessProfile)
	if err != nil {
		return diag.FromErr(fmt.Errorf("Error removing Access Profile: %s", err))
	}

	d.SetId("")
	return nil
}
