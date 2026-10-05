---
subcategory: "Governance Group"
page_title: "IdentityNow: Data Governance Group: identitynow_governance_group"
description: |-
  Gets information about an existing Governance Group.
---

# Data Source: identitynow_governance_group

Use this data source to access information about an existing Governance Group.

## Example Usage

```terraform
data "identitynow_governance_group" "example" {
  name = "Access Approvers"
}

output "identitynow_group_description" {
  value = data.identitynow_governance_group.example.description
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) Governance group name.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - ID of the governance group.

* `description` - Governance group description.

* `owner` - List with the governance group owner. Each element contains `id`, `type` and `name`.
