---
subcategory: "Role"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_dimension"
description: |-
  Gets information about an existing Dimension.
---

# Data Source: identitynow_dimension

Use this data source to access information about an existing Dimension. A dimension is a sub-division of a role that allows fine-grained access grouping.

## Example Usage

```hcl
data "identitynow_dimension" "example" {
  id      = "2c91808a7813090a017813b6301f1234"
  role_id = "2c91808a7813090a017813b6301fabcd"
}

output "identitynow_dimension_name" {
  value = data.identitynow_dimension.example.name
}
```

## Arguments Reference

The following arguments are supported:

* `id` - (Required) The ID of the Dimension.
* `role_id` - (Required) The ID of the role this dimension belongs to.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `name` - The name of the dimension.
* `description` - The description of the dimension.
* `owner` - The owner of the dimension.
* `access_profiles` - The access profiles assigned to this dimension.
* `entitlements` - The entitlements assigned to this dimension.
