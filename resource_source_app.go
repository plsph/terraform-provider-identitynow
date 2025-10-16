package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
)

func resourceSourceApp() *schema.Resource {
	return &schema.Resource{
		Create: resourceSourceAppCreate,
		Read:   resourceSourceAppRead,
		Update: resourceSourceAppUpdate,
		Delete: resourceSourceAppDelete,

                Importer: &schema.ResourceImporter{
                        State: resourceSourceAppImport,
                },

		Schema: sourceAppFields(),
	}
}

func resourceSourceAppCreate(d *schema.ResourceData, m interface{}) error {
	sourceApp, err := expandSourceApp(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Creating Source App %s", sourceApp.Name)

	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	newSourceApp, err := client.CreateSourceApp(context.Background(), sourceApp)
	if err != nil {
		return err
	}

	err = flattenSourceApp(d, newSourceApp)
	if err != nil {
		return err
	}

	return resourceSourceAppRead(d, m)
}

func resourceSourceAppRead(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] Refreshing Source App ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	sourceApp, err := client.GetSourceApp(context.Background(), d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Source App ID %s not found.", d.Id())
			d.SetId("")
			return err
		}
		return err
	}

	err = flattenSourceApp(d, sourceApp)
	if err != nil {
		return err
	}

	return nil
}

func resourceSourceAppUpdate(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] Updating Source App ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	updatedSourceApp, id, err := expandUpdateSourceApp(d)
	if err != nil {
		return err
	}

	_, err = client.UpdateSourceApp(context.Background(), updatedSourceApp, id)
	if err != nil {
		return err
	}

	return resourceSourceAppRead(d, m)
}

func resourceSourceAppDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] Deleting Source App ID %s", d.Id())

	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	sourceApp, err := client.GetSourceApp(context.Background(), d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Source App ID %s not found.", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	err = client.DeleteSourceApp(context.Background(), sourceApp)
	if err != nil {
		return fmt.Errorf("Error removing Source App: %s", err)
	}

	d.SetId("")
	return nil
}
