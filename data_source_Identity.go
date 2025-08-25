package main

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
	"fmt"
)

func dataSourceIdentity() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIdentityRead,

		Schema: identityFields(),
	}
}

func dataSourceIdentityRead(d *schema.ResourceData, meta interface{}) error {
	alias, _ :=d.Get("alias").(string)
	email, _ :=d.Get("email_address").(string)

	if alias != "" && email != "" {
        return fmt.Errorf("only one of 'alias' or 'email_address' must be set")
	}

	if alias == "" && email == "" {
	return fmt.Errorf("one of 'alias' or 'email_address' must be set")
	}

	if alias != "" {
		log.Printf("[INFO] Getting Data source for Identity. Identity alias %s", alias)
		client, err := meta.(*Config).IdentityNowClient()
		if err != nil {
			return err
		}

		identity, err := client.GetIdentityByAlias(context.Background(), alias)
		if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
			_, notFound := err.(*NotFoundError)
			if notFound {
				log.Printf("[INFO] Data source for Identity alias %s not found.", alias)
				return nil
			}
			return err
		}
		if len(identity)>0 {
			return flattenIdentity(d, identity[0])
		} else {
			log.Printf("[INFO] Data source for Identity alias %s not found.", alias)
			return nil
		}
	}

	if email != "" {
		log.Printf("[INFO] Getting Data source for Identity. Identity email %s", email)
		client, err := meta.(*Config).IdentityNowClient()
		if err != nil {
			return err
		}

		identity, err := client.GetIdentityByEmail(context.Background(), email)
		if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
			_, notFound := err.(*NotFoundError)
			if notFound {
				log.Printf("[INFO] Data source for Identity email %s not found.", email)
				return nil
			}
			return err
		}
		if len(identity)>0 {
			return flattenIdentity(d, identity[0])
		} else {
			log.Printf("[INFO] Data source for Identity email %s not found.", email)
			return nil
		}
	}
	log.Printf("[INFO] Data source for Identity not found. No email nor alias match.")
	return nil
}
