---
subcategory: "Governance Group"
layout: "identitynow"
page_title: "IdentityNow: Data Governance Group: identitynow_governance_group"
description: |-
  Gets information about an existing Governance Group.
---

# Data Source: identitynow_governance_group

Use this data source to access information about an existing Governance Group.

## Example Usage

```hcl
data "identitynow_governance_group" "example" {
  id = "example"
}

output "identitynow_group_description" {
  value = data.identitynow_governance_group.example.description
}
```

## Arguments Reference

The following arguments are supported:

* `name` - Governance group name.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Id of the governance group.

* `description` - Governance group description.

* `owner` - Governance group owner.

