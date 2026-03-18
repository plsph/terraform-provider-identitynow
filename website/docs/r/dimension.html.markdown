---
subcategory: "Role"
layout: "identitynow"
page_title: "IdentityNow: identitynow_dimension"
description: |-
  Manages an IdentityNow Dimension.
---

# identitynow_dimension

Manages an IdentityNow Dimension. A dimension is a sub-division of a role that allows fine-grained access grouping.

## Example Usage

```hcl
resource "identitynow_dimension" "example" {
  role_id     = identitynow_role.example.id
  name        = "Example Dimension"
  description = "An example dimension"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  access_profiles {
    id   = "2c91808a7813090a017813b6301f0044"
    type = "ACCESS_PROFILE"
    name = "Example Access Profile"
  }

  membership {
    type = "STANDARD"

    criteria {
      operation = "EQUALS"
      string_value = "Sales"

      key {
        type     = "IDENTITY"
        property = "attribute.department"
      }
    }
  }
}
```

## Arguments Reference

The following arguments are supported:

* `role_id` - (Required) The ID of the role this dimension belongs to. Changing this forces a new resource to be created.
* `name` - (Required) The name of the dimension. Changing this forces a new resource to be created.
* `description` - (Required) A description for the dimension.
* `owner` - (Optional) An owner block as defined below.
* `access_profiles` - (Optional) One or more `access_profiles` blocks as defined below.
* `entitlements` - (Optional) One or more `entitlements` blocks as defined below.
* `membership` - (Optional) A `membership` block as defined below.

---

An `owner` block supports:

* `id` - (Required) The owner's ID.
* `type` - (Required) The owner type (e.g. `IDENTITY`).
* `name` - (Required) The owner name.

---

An `access_profiles` block supports:

* `id` - (Required) The access profile ID.
* `type` - (Required) The type (e.g. `ACCESS_PROFILE`).
* `name` - (Required) The access profile name.

---

An `entitlements` block supports:

* `id` - (Required) The entitlement ID.
* `type` - (Required) The type (e.g. `ENTITLEMENT`).
* `name` - (Required) The entitlement name.

---

A `membership` block supports:

* `type` - (Required) The membership type (`STANDARD` or `IDENTITY_LIST`).
* `criteria` - (Optional) A `criteria` block as defined below.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - The ID of the Dimension.

## Import

Dimensions can be imported using the `id`, e.g.

```shell
terraform import identitynow_dimension.example <dimension-id>
```
