package main

import (
"context"
"errors"

"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceSourceAppImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	diags := resourceSourceAppRead(ctx, d, meta)
	if diags.HasError() {
		return []*schema.ResourceData{}, errors.New(diags[0].Summary)
	}

	return []*schema.ResourceData{d}, nil
}
