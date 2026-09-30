---
subcategory: "Privilege Criteria"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_privilege_criteria"
description: |-
  Gets information about an existing IdentityNow privilege criteria.
---

# Data Source: identitynow_privilege_criteria

Use this data source to look up a privilege criteria by ID. Connector and single level criteria can be read as well as custom criteria.

## Example Usage

```hcl
data "identitynow_privilege_criteria" "admins" {
  id = "2c9180867817ac4d017817c491119a20"
}
```

## Arguments Reference

* `id` - (Required) Privilege criteria ID.

## Attributes Reference

* `source_id` - ID of the source the criteria applies to.
* `type` - Criteria type, `CUSTOM`, `CONNECTOR` or `SINGLE_LEVEL`.
* `operator` - Logical operator between the groups.
* `groups_json` - Criteria groups as a JSON array.
* `privilege_level` - Privilege level assigned by the criteria.
