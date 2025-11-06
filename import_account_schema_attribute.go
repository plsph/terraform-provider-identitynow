package main

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAccountSchemaImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	sourceID, schemaId, err := splitAccountSchemaID(d.Id())
	if err != nil {
		return []*schema.ResourceData{}, err
	}
	d.Set("source_id", sourceID)
	d.Set("schema_id", schemaId)
	diags := resourceAccountSchemaRead(ctx, d, meta)
	if diags.HasError() {
		return []*schema.ResourceData{}, errors.New(diags[0].Summary)
	}

	return []*schema.ResourceData{d}, nil
}
