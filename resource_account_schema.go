package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAccountSchema() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAccountSchemaCreate,
		ReadContext:   resourceAccountSchemaRead,
		UpdateContext: resourceAccountSchemaUpdate,
		DeleteContext: resourceAccountSchemaDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceAccountSchemaImport,
		},

		Schema: accountSchemaFields(),
	}
}

func resourceAccountSchemaCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	accountSchema, err := expandAccountSchema(d)
	if err != nil {
		return diag.FromErr(err)
	}

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	newAccountSchema, err := client.GetAccountSchema(ctx, accountSchema.SourceID, accountSchema.ID)
	if err != nil {
		// Handle NotFoundError and other errors as before
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Source not found", map[string]interface{}{"source_id": accountSchema.SourceID})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	newAccountSchema.SourceID = accountSchema.SourceID

	// Use a map to track seen attributes based on a combination of fields
	seen := make(map[string]bool)
	result := []*AccountSchemaAttribute{}

	// Iterate over accountSchema.Attributes to filter out duplicates
	for _, attribute := range accountSchema.Attributes {
		// Create a unique key for each attribute based on important fields (e.g., "name")
		key := attribute.Name // or use a combination of "Name" and other unique fields if necessary
		if _, ok := seen[key]; !ok {
			seen[key] = true
			result = append(result, attribute)
		}
	}

	// Update the newAccountSchema with the filtered attributes
	newAccountSchema.Attributes = result
	newAccountSchema.ID = accountSchema.ID

	tflog.Info(ctx, "Creating Account Schema Attribute", map[string]interface{}{"source_id": newAccountSchema.SourceID})

	accountSchemaResponse, err := client.UpdateAccountSchema(ctx, newAccountSchema)
	if err != nil {
		return diag.FromErr(err)
	}

	accountSchemaResponse.SourceID = accountSchema.SourceID

	err = flattenAccountSchema(d, accountSchemaResponse)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceAccountSchemaRead(ctx, d, m)
}

func resourceAccountSchemaRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sourceId := d.Get("source_id").(string)
	schemaId := d.Get("schema_id").(string)
	attrName := d.Get("name").(string)
	tflog.Info(ctx, "Refreshing Account Schema", map[string]interface{}{"source_id": sourceId})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	accountSchema, err := client.GetAccountSchema(ctx, sourceId, schemaId)
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Source not found", map[string]interface{}{"source_id": sourceId})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if accountSchema.Attributes == nil {
		tflog.Debug(ctx, "Attribute not found in Account Schema", map[string]interface{}{"attribute": attrName})
		d.SetId("")
	}

	accountSchema.SourceID = sourceId
	err = flattenAccountSchema(d, accountSchema)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAccountSchemaUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	updatedAccountSchema, err := expandAccountSchema(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Updating Account Schema attribute", map[string]interface{}{"attribute": d.Get("name").(string), "source_id": d.Get("source_id").(string)})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.UpdateAccountSchema(ctx, updatedAccountSchema)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceAccountSchemaRead(ctx, d, m)
}

func resourceAccountSchemaDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sourceId := d.Get("source_id").(string)
	schemaId := d.Get("schema_id").(string)
	name := d.Get("name").(string)
	tflog.Info(ctx, "Deleting Account Schema attribute", map[string]interface{}{"attribute": name, "source_id": sourceId})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	accountSchema, err := client.GetAccountSchema(ctx, sourceId, schemaId)
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Source not found", map[string]interface{}{"source_id": sourceId})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if accountSchema.Attributes == nil {
		tflog.Debug(ctx, "Attribute not found in Account Schema", map[string]interface{}{"attribute": name})
		d.SetId("")
	}

	accountSchema.SourceID = sourceId

	err = client.DeleteAccountSchema(ctx, accountSchema)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error removing Account Schema from source %s. Error: %s", sourceId, err))
	}

	d.SetId("")
	return nil
}
