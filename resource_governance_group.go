package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
)

func resourceGovernanceGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceGovernanceGroupCreate,
		Read:   resourceGovernanceGroupRead,
		Update: resourceGovernanceGroupUpdate,
		Delete: resourceGovernanceGroupDelete,

                Importer: &schema.ResourceImporter{
                        State: resourceGovernanceGroupImport,
                },

		Schema: governanceGroupFields(),
	}
}

func resourceGovernanceGroupCreate(d *schema.ResourceData, m interface{}) error {
	governanceGroup, err := expandGovernanceGroup(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Creating Governance Group %s", governanceGroup.Name)

	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	newGovernanceGroup, err := client.CreateGovernanceGroup(context.Background(), governanceGroup)
	if err != nil {
		return err
	}

	err = flattenGovernanceGroup(d, newGovernanceGroup)
	if err != nil {
		return err
	}

	return resourceGovernanceGroupRead(d, m)
}

func resourceGovernanceGroupRead(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] Refreshing Governance Group ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	governanceGroup, err := client.GetGovernanceGroup(context.Background(), d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Governance Group ID %s not found.", d.Id())
			d.SetId("")
			return err
		}
		return err
	}

	err = flattenGovernanceGroup(d, governanceGroup)
	if err != nil {
		return err
	}

	return nil
}

func resourceGovernanceGroupUpdate(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] Updating Governance Group ID %s", d.Id())
	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	updatedGovernanceGroup, id, err := expandUpdateGovernanceGroup(d)
	if err != nil {
		return err
	}

	_, err = client.UpdateGovernanceGroup(context.Background(), updatedGovernanceGroup, id)
	if err != nil {
		return err
	}

	return resourceGovernanceGroupRead(d, m)
}

func resourceGovernanceGroupDelete(d *schema.ResourceData, m interface{}) error {
	log.Printf("[INFO] Deleting Governance Group ID %s", d.Id())

	client, err := m.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	governanceGroup, err := client.GetGovernanceGroup(context.Background(), d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Governance Group ID %s not found.", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	err = client.DeleteGovernanceGroup(context.Background(), governanceGroup)
	if err != nil {
		return fmt.Errorf("Error removing Governance Group: %s", err)
	}

	d.SetId("")
	return nil
}
