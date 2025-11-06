package main

import (
	"context"
	"fmt"


	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func resourceAccessProfileAttachment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAccessProfileAttachmentCreate,
		ReadContext:   resourceAccessProfileAttachmentRead,
		UpdateContext: resourceAccessProfileAttachmentUpdate,
		DeleteContext: resourceAccessProfileAttachmentDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceAccessProfileAttachmentImport,
		},

		Schema: accessProfileAttachmentFields(),
	}
}

func resourceAccessProfileAttachmentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	accessProfileAttachment, err := expandAccessProfileAttachment(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Creating Access Profile Attachment for Source App Id:", map[string]interface{}{"name": accessProfileAttachment.SourceAppId})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	newAccessProfileAttachment, err := client.UpdateAccessProfileAttachment(ctx, accessProfileAttachment, accessProfileAttachment.SourceAppId)
	if err != nil {
		return diag.FromErr(err)
	}

	err = flattenAccessProfileAttachment(d, newAccessProfileAttachment)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceAccessProfileAttachmentRead(ctx, d, m)
}

func resourceAccessProfileAttachmentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Refreshing Access Profile Attachment", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	accessProfileAttachment, err := client.GetAccessProfileAttachment(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Access ProfileAttachment not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return diag.FromErr(err)
		}
		return diag.FromErr(err)
	}

	err = flattenAccessProfileAttachment(d, accessProfileAttachment)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAccessProfileAttachmentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Updating Access Profile Attachment", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	accessProfileAttachment, err := expandAccessProfileAttachment(d)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.UpdateAccessProfileAttachment(ctx, accessProfileAttachment, accessProfileAttachment.SourceAppId)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceAccessProfileAttachmentRead(ctx, d, m)
}

func resourceAccessProfileAttachmentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Deleting Access ProfileAttachment", map[string]interface{}{"id": d.Id()})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	accessProfileAttachment, err := client.GetAccessProfileAttachment(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Access ProfileAttachment not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = client.DeleteAccessProfileAttachment(ctx, accessProfileAttachment)
	if err != nil {
		return diag.FromErr(fmt.Errorf("Error removing Access ProfileAttachment: %s", err))
	}

	d.SetId("")
	return nil
}
