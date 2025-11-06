package main

import (
	"context"
	"fmt"
"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func resourcePasswordPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePasswordPolicyCreate,
		ReadContext:   resourcePasswordPolicyRead,
		UpdateContext: resourcePasswordPolicyUpdate,
		DeleteContext: resourcePasswordPolicyDelete,

		Schema: passwordPolicyFields(),
	}

}

func resourcePasswordPolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	passwordPolicy, err := expandPasswordPolicy(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Creating Password Policy", map[string]interface{}{"name": passwordPolicy.Name})

	c, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	newPasswordPolicy, err := c.CreatePasswordPolicy(ctx, passwordPolicy)
	if err != nil {
		return diag.FromErr(err)
	}

	err = flattenPasswordPolicy(d, newPasswordPolicy)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourcePasswordPolicyRead(ctx, d, m)

}

func resourcePasswordPolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Refreshing Password Policy", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	passwordPolicy, err := client.GetPasswordPolicy(ctx, d.Id())
	if err != nil {
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Password Policy not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = flattenPasswordPolicy(d, passwordPolicy)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourcePasswordPolicyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Updating Password Policy", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	updatedPasswordPolicy, err := expandPasswordPolicy(d)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.UpdatePasswordPolicy(ctx, updatedPasswordPolicy)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourcePasswordPolicyRead(ctx, d, m)
}

func resourcePasswordPolicyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Deleting Password Policy", map[string]interface{}{"id": d.Id()})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	passwordPolicy, err := client.GetPasswordPolicy(ctx, d.Id())
	if err != nil {
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Password Policy not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = client.DeletePasswordPolicy(ctx, passwordPolicy.ID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error removing Passwprd Policy: %s", err))
	}

	d.SetId("")
	return nil
}
