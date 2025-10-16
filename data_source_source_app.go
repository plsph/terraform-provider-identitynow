package main

import (
	"context"
	"log"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceApp() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceSourceAppRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed: true,
				Description: "Source id",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Source App name",
			},
			"description": {
				Type:        schema.TypeString,
				Computed: true,
				Description: "Source App description",
			},

			"match_all_accounts": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"source": {
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: sourceAppSourceFields(),
				},
			},

			"date_created": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"last_updated": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceSourceAppRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Data source for Source App Name %s", d.Get("name").(string))
	client, err := meta.(*Config).IdentityNowClient()
	if err != nil {
		return err
	}

	sourceApp, err := client.GetSourceAppByName(context.Background(), d.Get("name").(string))
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Data source for Source App Name %s not found.", d.Get("name").(string))
			return nil
		}
		return err
	}

	return flattenSourceApp(d, sourceApp[0])
}
