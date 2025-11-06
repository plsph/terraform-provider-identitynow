package main

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func flattenIdentity(d *schema.ResourceData, in *Identity) error {
	if in == nil {
		return nil
	}
	d.SetId(in.ID)
	d.Set("alias", in.Alias)
	d.Set("name", in.Name)
	d.Set("description", in.Description)
	d.Set("enabled", in.Enabled)
	d.Set("is_manager", in.IsManager)
	d.Set("email_address", in.EmailAddress)
	d.Set("identity_status", in.IdentityStatus)

	if in.IdentityAttributes != nil {
		d.Set("attributes", []interface{}{flattenIdentityAttributes(in.IdentityAttributes, nil)})
	}

	return nil
}

func flattenIdentityAttributes(in *IdentityAttributes, p interface{}) interface{} {
	if in == nil {
		return nil
	}
	var obj = make(map[string]interface{})
	obj["adp_id"] = in.AdpID
	obj["lastname"] = in.LastName
	obj["firstname"] = in.FirstName
	obj["phone"] = in.Phone
	obj["user_type"] = in.UserType
	obj["uid"] = in.UID
	obj["email"] = in.Email
	obj["workday_id"] = in.WorkdayId

	return obj
}
