package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSourceRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Source id",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Source name",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Source description",
			},
			"connector": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Source connector type",
			},
			"delete_threshold": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"authoritative": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if this source is authoritative",
			},
			"owner": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: sourceOwnerFields(),
				},
			},
			"schemas": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: sourceSchemaFields(),
				},
			},
			"cluster": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: sourceClusterFields(),
				},
			},
			"account_correlation_config": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: sourceAccountCorrelationConfigFields(),
				},
			},
			"connector_attributes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: sourceConnectorAttributesFields(),
				},
			},
		},
	}
}

func dataSourceSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Data source for Source ID %s", d.Get("id").(string))
	client, err := meta.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	source, err := client.GetSource(ctx, d.Get("id").(string))
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			log.Printf("[INFO] Data source for Source ID %s not found.", d.Get("id").(string))
			return nil
		}
		return diag.FromErr(err)
	}

	err = flattenSource(d, source)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
