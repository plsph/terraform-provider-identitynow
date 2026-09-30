---
subcategory: "Privilege Criteria"
layout: "identitynow"
page_title: "IdentityNow: identitynow_privilege_criteria"
description: |-
  Manages an IdentityNow custom privilege criteria.
---

# identitynow_privilege_criteria

Manages a custom privilege criteria, which assigns a privilege level to the entitlements of a source that match the criteria.

## Example Usage

```hcl
resource "identitynow_privilege_criteria" "admins" {
  source_id       = "c42c45d8d7c04d2da64d215cd8c32f21"
  operator        = "AND"
  privilege_level = "HIGH"

  groups_json = jsonencode([
    {
      operator = "OR"
      criteriaItems = [
        {
          targetType = "group"
          property   = "displayName"
          operator   = "CONTAINS"
          values     = ["admin", "superuser"]
          ignoreCase = true
        }
      ]
    }
  ])
}
```

## Arguments Reference

The following arguments are supported:

* `source_id` - (Required) ID of the source the criteria applies to.
* `operator` - (Required) Logical operator between the groups, `AND` or `OR`.
* `privilege_level` - (Required) Privilege level assigned by the criteria, `HIGH`, `MEDIUM` or `LOW`.
* `groups_json` - (Required) Criteria groups as a JSON array. Each group has an `operator` (`AND` or `OR`) between its `criteriaItems`. Each item has a `targetType` (`group`), a `property` (`displayName`, `description`, `value` or `attributes.<name>`), an `operator` (`IN`, `EQUALS`, `NOT_EQUALS`, `CONTAINS`, `DOES_NOT_CONTAIN`, `STARTS_WITH` or `ENDS_WITH`), 1 to 50 `values` and `ignoreCase`. Use `jsonencode()`. The value is compared on the configured fields only, so formatting, key order and defaults IdentityNow adds do not produce a diff.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - Privilege criteria ID.
* `type` - Criteria type, always `CUSTOM` for criteria managed with this resource.

## Update Behaviour

Updates replace the whole criteria with PUT, every field of the criteria is managed by the resource.

## Import

Custom privilege criteria can be imported using their ID:

```shell
terraform import identitynow_privilege_criteria.example <privilege-criteria-id>
```
