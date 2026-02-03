package main

import (
	"context"


	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func dataSourceSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSourceRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Source id",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
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
	tflog.Info(ctx, "Getting Source data source", map[string]interface{}{"name": d.Get("name").(string)})
	client, err := meta.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	source, err := client.GetSourceByName(ctx, d.Get("name").(string))
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Source not found in data source", map[string]interface{}{"name": d.Get("name").(string)})
			return nil
		}
		return diag.FromErr(err)
	}

	if len(source) > 0 {
		err = flattenSource(d, source[0])
		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	} else {
		tflog.Debug(ctx, "Source not found in data source", map[string]interface{}{"name": d.Get("name").(string)})
		return nil
	}

	return nil
}
