---
subcategory: "Non-Employee"
page_title: "IdentityNow: Data Source: identitynow_non_employee_source"
description: |-
  Gets information about an IdentityNow non-employee source.
---

# Data Source: identitynow_non_employee_source

Use this data source to look up a non-employee source by ID or name.

## Example Usage

```hcl
data "identitynow_non_employee_source" "contractors" {
  name = "Contractors"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) Non-employee source ID.
* `name` - (Optional) Non-employee source name. All non-employee sources are listed and matched by exact name.

## Attributes Reference

* `source_id` - ID of the source that backs the non-employee source.
* `description` - Description of the non-employee source.
* `approvers` - IDs of the approvers, in approval order.
* `account_managers` - IDs of the account managers.
* `created` - Creation date.
* `modified` - Last modification date.
