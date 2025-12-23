package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func resourceGovernanceGroupMembers() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGovernanceGroupMembersCreate,
		ReadContext:   resourceGovernanceGroupMembersRead,
		UpdateContext: resourceGovernanceGroupMembersUpdate,
		DeleteContext: resourceGovernanceGroupMembersDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceGovernanceGroupMembersImport,
		},

		Schema: governanceGroupMembersFields(),
	}
}

func resourceGovernanceGroupMembersCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	governanceGroupMembers, err := expandGovernanceGroupMembers(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Creating Governance Group Members for Governance Group Id:", map[string]interface{}{"name": governanceGroupMembers.GovernanceGroupId})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	newGovernanceGroupMembers, err := client.CreateGovernanceGroupMembers(ctx, governanceGroupMembers, governanceGroupMembers.GovernanceGroupId)
	if err != nil {
		return diag.FromErr(err)
	}

	err = flattenGovernanceGroupMembers(d, newGovernanceGroupMembers)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGovernanceGroupMembersRead(ctx, d, m)
}

func resourceGovernanceGroupMembersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Refreshing Governance Group Members", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	governanceGroupMembers, err := client.GetGovernanceGroupMembers(ctx, d.Id())
	if err != nil {
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Governance Group Members not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return diag.FromErr(err)
		}
		return diag.FromErr(err)
	}

	err = flattenGovernanceGroupMembers(d, governanceGroupMembers)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGovernanceGroupMembersUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Updating Governance Group Members", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	governanceGroupMembers, err := expandGovernanceGroupMembers(d)
	if err != nil {
		return diag.FromErr(err)
	}

	governanceGroupMembersActual, err := client.GetGovernanceGroupMembers(ctx, d.Id())
	if err != nil {
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Governance Group Members not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return diag.FromErr(err)
		}
		return diag.FromErr(err)
	}

	_, err = client.UpdateGovernanceGroupMembers(ctx, governanceGroupMembers, governanceGroupMembersActual, governanceGroupMembers.GovernanceGroupId)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGovernanceGroupMembersRead(ctx, d, m)
}

func resourceGovernanceGroupMembersDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Deleting Governance Group Members", map[string]interface{}{"id": d.Id()})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	governanceGroupMembers, err := client.GetGovernanceGroupMembers(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Governance Group Members not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = client.DeleteGovernanceGroupMembers(ctx, governanceGroupMembers)
	if err != nil {
		return diag.FromErr(fmt.Errorf("Error removing Governance Group Members: %s", err))
	}

	d.SetId("")
	return nil
}
