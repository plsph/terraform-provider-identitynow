package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func resourceGovernanceGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGovernanceGroupCreate,
		ReadContext:   resourceGovernanceGroupRead,
		UpdateContext: resourceGovernanceGroupUpdate,
		DeleteContext: resourceGovernanceGroupDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceGovernanceGroupImport,
		},

		Schema: governanceGroupFields(),
	}
}

func resourceGovernanceGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	governanceGroup, err := expandGovernanceGroup(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Creating Governance Group", map[string]interface{}{"name": governanceGroup.Name})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	newGovernanceGroup, err := client.CreateGovernanceGroup(ctx, governanceGroup)
	if err != nil {
		return diag.FromErr(err)
	}

	err = flattenGovernanceGroup(d, newGovernanceGroup)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGovernanceGroupRead(ctx, d, m)
}

func resourceGovernanceGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Refreshing Governance Group", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	governanceGroup, err := client.GetGovernanceGroup(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Governance Group not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return diag.FromErr(err)
		}
		return diag.FromErr(err)
	}

	// Add debug logging to see what fields are in the governance group
	tflog.Debug(ctx, "Governance Group API response", map[string]interface{}{
		"id":          governanceGroup.ID,
		"name":        governanceGroup.Name,
		"description": governanceGroup.Description,
		"owner":       fmt.Sprintf("%+v", governanceGroup.GovernanceGroupOwner),
	})

	err = flattenGovernanceGroup(d, governanceGroup)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGovernanceGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Updating Governance Group", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	updatedGovernanceGroup, id, err := expandUpdateGovernanceGroup(d)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.UpdateGovernanceGroup(ctx, updatedGovernanceGroup, id)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGovernanceGroupRead(ctx, d, m)
}

func resourceGovernanceGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Deleting Governance Group", map[string]interface{}{"id": d.Id()})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	governanceGroup, err := client.GetGovernanceGroup(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Governance Group not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = client.DeleteGovernanceGroup(ctx, governanceGroup)
	if err != nil {
		return diag.FromErr(fmt.Errorf("Error removing Governance Group: %s", err))
	}

	d.SetId("")
	return nil
}
