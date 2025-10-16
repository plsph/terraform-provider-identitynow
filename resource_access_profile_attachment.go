package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
)

func resourceAccessProfileAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAccessProfileAttachmentCreate,
		Read:   resourceAccessProfileAttachmentRead,
		Update: resourceAccessProfileAttachmentUpdate,
		Delete: resourceAccessProfileAttachmentDelete,

                Importer: &schema.ResourceImporter{
                        State: resourceAccessProfileAttachmentImport,
                },

		Schema: accessProfileAttachmentFields(),
	}
}

func resourceAccessProfileAttachmentCreate(d *schema.ResourceData, m interface{}) error {
	accessProfileAttachment, err := expandAccessProfileAttachment(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Creating Access Profile Attachment for Source App Id: %s", accessProfileAttachment.SourceAppId)

	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	newAccessProfileAttachment, err := client.UpdateAccessProfileAttachment(context.Background(), accessProfileAttachment, accessProfileAttachment.SourceAppId)
	if err != nil {
		return err
	}

	err = flattenAccessProfileAttachment(d, newAccessProfileAttachment)
	if err != nil {
		return err
	}

	return resourceAccessProfileAttachmentRead(d, m)
}

func resourceAccessProfileAttachmentRead(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] Refreshing Access Profile Attachment ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	accessProfileAttachment, err := client.GetAccessProfileAttachment(context.Background(), d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Access ProfileAttachment ID %s not found.", d.Id())
			d.SetId("")
			return err
		}
		return err
	}

	err = flattenAccessProfileAttachment(d, accessProfileAttachment)
	if err != nil {
		return err
	}

	return nil
}

func resourceAccessProfileAttachmentUpdate(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] Updating Access Profile Attachment ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	accessProfileAttachment, err := expandAccessProfileAttachment(d)
	if err != nil {
		return err
	}

	_, err = client.UpdateAccessProfileAttachment(context.Background(), accessProfileAttachment, accessProfileAttachment.SourceAppId)
	if err != nil {
		return err
	}

	return resourceAccessProfileAttachmentRead(d, m)
}

func resourceAccessProfileAttachmentDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] Deleting Access ProfileAttachment ID %s", d.Id())

	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	accessProfileAttachment, err := client.GetAccessProfileAttachment(context.Background(), d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Access ProfileAttachment ID %s not found.", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	err = client.DeleteAccessProfileAttachment(context.Background(), accessProfileAttachment)
	if err != nil {
		return fmt.Errorf("Error removing Access ProfileAttachment: %s", err)
	}

	d.SetId("")
	return nil
}
