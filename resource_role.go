package main

import (
	"context"
	"fmt"
"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func resourceRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		UpdateContext: resourceRoleUpdate,
		DeleteContext: resourceRoleDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceRoleImport,
		},

		Schema: roleFields(),
	}
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	role, err := expandRole(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Creating Role", map[string]interface{}{"name": role.Name})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	newRole, err := client.CreateRole(ctx, role)
	if err != nil {
		return diag.FromErr(err)
	}

	err = flattenRole(d, newRole)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRoleRead(ctx, d, m)
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Refreshing Role", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := client.GetRole(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Role not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = flattenRole(d, role)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceUpdateRoleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Refreshing Role", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := client.GetRole(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Role not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = flattenRole(d, role)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceRoleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Updating Role", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Role disabled status", map[string]interface{}{"disabled": d.Get("disabled")})

	updatedRole, id, err := expandUpdateRole(d)
	tflog.Debug(ctx, "Role after expand", map[string]interface{}{"role": fmt.Sprintf("%v", updatedRole)})
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.UpdateRole(ctx, updatedRole, id)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRoleRead(ctx, d, m)
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Deleting Role", map[string]interface{}{"id": d.Id()})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := client.GetRole(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Role not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_, err = client.DeleteRole(ctx, role)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error removing Role: %s", err))
	}

	d.SetId("")
	return nil
}
