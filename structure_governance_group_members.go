package main

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Flatteners

func flattenGovernanceGroupMembers(d *schema.ResourceData, in *GovernanceGroupMembers) error {
	if in == nil {
		return nil
	}

	d.SetId(in.GovernanceGroupId)
	d.Set("governance_group_id", in.GovernanceGroupId)
	if in.GovernanceGroupMembersMembers != nil {
		v, ok := d.Get("members").([]interface{})
		if !ok {
			v = []interface{}{}
		}
		//governanceGroupMembersMembers := []*GovernanceGroupMembersMembers{in.GovernanceGroupMembersMembers}
		//d.Set("members", flattenObjectGovernanceGroupMembers(governanceGroupMembersMembers, v))
		d.Set("members", flattenObjectGovernanceGroupMembers(in.GovernanceGroupMembersMembers, v))
	}
	return nil
}

func flattenObjectGovernanceGroupMembers(in []*GovernanceGroupMembersMembers, p []interface{}) []interface{} {
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

func expandGovernanceGroupMembers(in *schema.ResourceData) (*GovernanceGroupMembers, error) {
	obj := GovernanceGroupMembers{}
	if in == nil {
		return nil, fmt.Errorf("[ERROR] Expanding Governance Group Members: Schema Resource data is nil")
	}

	obj.GovernanceGroupId = in.Get("governance_group_id").(string)

	if v, ok := in.Get("members").([]interface{}); ok && len(v) > 0 {
		obj.GovernanceGroupMembersMembers = expandObjectGovernanceGroupMembers(v)
	}

	return &obj, nil
}

func expandObjectGovernanceGroupMembers(p []interface{}) []*GovernanceGroupMembersMembers {
	if len(p) == 0 || p[0] == nil {
		return []*GovernanceGroupMembersMembers{}
	}
	out := make([]*GovernanceGroupMembersMembers, 0, len(p))
	for i := range p {
		obj := GovernanceGroupMembersMembers{}
		in := p[i].(map[string]interface{})
		obj.ID = in["id"].(string)
		obj.Name = in["name"].(string)
		obj.Type = in["type"].(string)
		out = append(out, &obj)
	}
	return out
}
