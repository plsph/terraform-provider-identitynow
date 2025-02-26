package main

import "github.com/hashicorp/terraform-plugin-sdk/helper/schema"

func resourceAccessProfileImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	err := resourceAccessProfileRead(d, meta)
	if err != nil {
		return []*schema.ResourceData{}, err
	}

	return []*schema.ResourceData{d}, nil
}
