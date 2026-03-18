---
subcategory: "Role"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_role"
description: |-
  Gets information about an existing Role.
---

# Data Source: identitynow_role

Use this data source to access information about an existing Role.

## Example Usage

```hcl
data "identitynow_role" "example" {
  id = "2c91808a7813090a017813b6301f1234"
}

output "identitynow_role_name" {
  value = data.identitynow_role.example.name
}
```

## Arguments Reference

The following arguments are supported:

* `id` - (Required) The ID of the Role.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `name` - The name of the role.
* `description` - The description of the role.
* `owner` - The owner of the role. Each element contains `id`, `type`, and `name`.
* `access_profiles` - The access profiles assigned to this role. Each element contains `id`, `type`, and `name`.
* `requestable` - Whether the role is requestable.
* `enabled` - Whether the role is enabled.
