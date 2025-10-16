package main

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// Flatteners

func flattenSourceApp(d *schema.ResourceData, in *SourceApp) error {
	if in == nil {
		return nil
	}

	d.SetId(in.ID)
	d.Set("name", in.Name)
	d.Set("description", in.Description)
	d.Set("enabled", in.Enabled)
	d.Set("match_all_accounts", in.MatchAllAccounts)

	if in.SourceAppSource != nil {
		v, ok := d.Get("source").([]interface{})
		if !ok {
			v = []interface{}{}
		}
		sourceAppSourceList := []*ObjectInfo{in.SourceAppSource}
		d.Set("source", flattenObjectSourceApp(sourceAppSourceList, v))
	}
	return nil
}

func flattenObjectSourceApp(in []*ObjectInfo, p []interface{}) []interface{} {
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

func expandSourceApp(in *schema.ResourceData) (*SourceApp, error) {
	obj := SourceApp{}
	if in == nil {
		return nil, fmt.Errorf("[ERROR] Expanding Source App: Schema Resource data is nil")
	}

	obj.Name = in.Get("name").(string)
	obj.Description = in.Get("description").(string)

	if v, ok := in.Get("match_all_accounts").(bool); ok {
		obj.MatchAllAccounts = &v
	}

	if v, ok := in.Get("enabled").(bool); ok {
		obj.Enabled = &v
	}

	if v, ok := in.Get("source").([]interface{}); ok && len(v) > 0 {
		obj.SourceAppSource = expandObjectSourceApp(v)[0]
	}

	return &obj, nil
}


func expandUpdateSourceApp(in *schema.ResourceData) ([]*UpdateSourceApp, interface{}, error) {
	updatableFields := []string{"name", "description", "enabled", "matchAllAccounts"}
	updatableFieldsCodes := []string{"name", "description", "enabled", "match_all_accounts"}
	var id interface{}
	if in == nil {
		return nil, nil, fmt.Errorf("[ERROR] Expanding Source App: Schema Resource data is nil")
	}

	if v := in.Id(); len(v) > 0 {
		id = v
	}

	out := []*UpdateSourceApp{}
	for key, field := range updatableFields {
	    obj := UpdateSourceApp{}
	    switch field {
	    case "name", "description" :
		if v, ok := in.Get(fmt.Sprintf("%s", updatableFieldsCodes[key])).(string); ok {
		    obj.Op = "replace"
		    obj.Path = fmt.Sprintf("/%s", field)
		    obj.Value = v
		}
	    case "enabled", "matchAllAccounts":
		if v, ok := in.Get(fmt.Sprintf("%s", updatableFieldsCodes[key])).(bool); ok {
		    obj.Op = "replace"
		    obj.Path = fmt.Sprintf("/%s", field)
		    obj.Value = v
		}
	    default:
		return nil, id, nil
	    }
	    out = append(out, &obj)
	}

	return out, id, nil
}

func expandObjectSourceApp(p []interface{}) []*ObjectInfo {
	if len(p) == 0 || p[0] == nil {
		return []*ObjectInfo{}
	}
	out := make([]*ObjectInfo, 0, len(p))
	for i := range p {
		obj := ObjectInfo{}
		in := p[i].(map[string]interface{})
		obj.ID = in["id"].(string)
		obj.Name = in["name"].(string)
		obj.Type = in["type"].(string)
		out = append(out, &obj)
	}
	return out
}
