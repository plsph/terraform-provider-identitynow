package main

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

// Flatteners

func flattenSourceEntitlement(d *schema.ResourceData, in *SourceEntitlement) error {
	if in == nil {
		return nil
	}
	d.SetId(in.ID)
	d.Set("source_id", in.Source.ID)
	d.Set("source_name", in.Source.Name)
	d.Set("attribute", in.Attribute)
	d.Set("created", in.Created)
	d.Set("description", in.Description)
	d.Set("direct_permissions", toArrayString(in.DirectPermissions))
	d.Set("name", in.Name)
	d.Set("modified", in.Modified)
	if in.Owner != nil {
		v, ok := d.Get("owner").([]interface{})
		if !ok {
			v = []interface{}{}
		}

		d.Set("owner", flattenSourceOwner(in.Owner, v))
	}
	d.Set("privileged", in.Privileged)
	d.Set("source_schema_object_type", in.SourceSchemaObjectType)
	d.Set("value", in.Value)

	return nil
}

func getEntitlement(entitlements []*SourceEntitlement, name string) *SourceEntitlement {
	for i := range entitlements {
		if entitlements[i].Name == name {
			return entitlements[i]
		}
	}
	return nil
}

