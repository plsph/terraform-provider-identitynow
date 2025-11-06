package main

import (
	"context"
	"fmt"
"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
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

	log.Printf("[INFO] Creating Role %s", role.Name)

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
	log.Printf("[INFO] Refreshing Role ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := client.GetRole(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Role ID %s not found.", d.Id())
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
	log.Printf("[INFO] Refreshing Role ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := client.GetRole(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Role ID %s not found.", d.Id())
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
	log.Printf("[INFO] Updating Role ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("disabled in role: %s\n", d.Get("disabled"))

	updatedRole, id, err := expandUpdateRole(d)
	log.Printf("role after expand: %v\n", updatedRole)
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
	log.Printf("[INFO] Deleting Role ID %s", d.Id())

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := client.GetRole(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Role ID %s not found.", d.Id())
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
