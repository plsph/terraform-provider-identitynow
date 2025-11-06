package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Flatteners

func flattenGovernanceGroup(d *schema.ResourceData, in *GovernanceGroup) error {
	if in == nil {
		return nil
	}

	tflog.Debug(context.Background(), "Flattening Governance Group", map[string]interface{}{
		"id":          in.ID,
		"name":        in.Name,
		"description": in.Description,
		"owner":       fmt.Sprintf("%+v", in.GovernanceGroupOwner),
	})

	d.SetId(in.ID)
	d.Set("name", in.Name)
	d.Set("description", in.Description)

	if in.GovernanceGroupOwner != nil {
		v, ok := d.Get("owner").([]interface{})
		if !ok {
			v = []interface{}{}
		}
		governanceGroupOwnerList := []*GovernanceGroupOwner{in.GovernanceGroupOwner}
		d.Set("owner", flattenObjectGovernanceGroup(governanceGroupOwnerList, v))
	}
	return nil
}

func flattenObjectGovernanceGroup(in []*GovernanceGroupOwner, p []interface{}) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	out := make([]interface{}, 0, len(in))
	for i := range in {
		var obj = make(map[string]interface{})
		obj["type"] = in[i].Type
		obj["id"] = in[i].ID
		obj["name"] = in[i].Name
		out = append(out, obj)
	}
	return out
}

// Expanders

func expandGovernanceGroup(in *schema.ResourceData) (*GovernanceGroup, error) {
	obj := GovernanceGroup{}
	if in == nil {
		return nil, fmt.Errorf("[ERROR] Expanding Governance Group: Schema Resource data is nil")
	}

	obj.Name = in.Get("name").(string)
	obj.Description = in.Get("description").(string)

	if v, ok := in.Get("owner").([]interface{}); ok && len(v) > 0 {
		obj.GovernanceGroupOwner = expandObjectGovernanceGroup(v)[0]
	}

	return &obj, nil
}

func expandUpdateGovernanceGroup(in *schema.ResourceData) ([]*UpdateGovernanceGroup, interface{}, error) {
	updatableFields := []string{"name", "description", "owner"}
	var id interface{}
	if in == nil {
		return nil, nil, fmt.Errorf("[ERROR] Expanding Role: Schema Resource data is nil")
	}

	if v := in.Id(); len(v) > 0 {
		id = v
	}

	out := []*UpdateGovernanceGroup{}
	for _, field := range updatableFields {
		obj := UpdateGovernanceGroup{}
		switch field {
		case "name", "description":
			if v, ok := in.Get(fmt.Sprintf("%s", field)).(string); ok {
				obj.Op = "replace"
				obj.Path = fmt.Sprintf("/%s", field)
				obj.Value = v
			}
		case "owner":
			if v, ok := in.Get(fmt.Sprintf("%s", field)).([]interface{}); ok {
				obj.Op = "replace"
				obj.Path = fmt.Sprintf("/%s", field)
				obj.Value = v[0]
			}
		default:
			return nil, id, nil
		}
		out = append(out, &obj)
	}

	return out, id, nil
}

func expandObjectGovernanceGroup(p []interface{}) []*GovernanceGroupOwner {
	if len(p) == 0 || p[0] == nil {
		return []*GovernanceGroupOwner{}
	}
	out := make([]*GovernanceGroupOwner, 0, len(p))
	for i := range p {
		obj := GovernanceGroupOwner{}
		in := p[i].(map[string]interface{})
		obj.ID = in["id"].(string)
		obj.Name = in["name"].(string)
		obj.Type = in["type"].(string)
		out = append(out, &obj)
	}
	return out
}
