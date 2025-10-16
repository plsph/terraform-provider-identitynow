package main

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// Flatteners

func flattenAccessProfileAttachment(d *schema.ResourceData, in *AccessProfileAttachment) error {
	if in == nil {
		return nil
	}

	d.SetId(in.SourceAppId)
	d.Set("source_app_id", in.SourceAppId)
	d.Set("access_profiles", toArrayInterface(in.AccessProfiles))
	return nil
}

// Expanders

func expandAccessProfileAttachment(in *schema.ResourceData) (*AccessProfileAttachment, error) {
	obj := AccessProfileAttachment{}
	if in == nil {
		return nil, fmt.Errorf("[ERROR] Expanding Access Profile Attachment: Schema Resource data is nil")
	}

	obj.SourceAppId = in.Get("source_app_id").(string)

	if v, ok := in.Get("access_profiles").([]interface{}); ok && len(v) > 0 {
		obj.AccessProfiles = toArrayString(v)
	}

	return &obj, nil
}

