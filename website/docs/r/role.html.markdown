---
subcategory: "Role"
layout: "identitynow"
page_title: "IdentityNow: identitynow_role"
description: |-
  Manages an IdentityNow Role.
---

# identitynow_role

Manages an IdentityNow Role. Roles bundle access profiles, entitlements, and dimensions together and can be assigned to identities through access requests or membership criteria.

## Example Usage

### Basic Role

```hcl
resource "identitynow_role" "example" {
  name        = "Example Role"
  description = "An example role"

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

  requestable = true
  enabled     = true
}
```

### Role with Entitlements and Dimensions

```hcl
resource "identitynow_role" "advanced" {
  name        = "Advanced Role"
  description = "A role with entitlements and dimensions"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  entitlements {
    id   = "2c91808a7813090a017813b6301fabcd"
    type = "ENTITLEMENT"
    name = "Example Entitlement"
  }

  dimension_refs {
    id   = "2c91808a7813090a017813b6301f5678"
    type = "DIMENSION"
    name = "Example Dimension"
  }

  requestable = true
  enabled     = true
}
```

### Role with Standard Membership Criteria

```hcl
resource "identitynow_role" "standard_membership" {
  name        = "Department Role"
  description = "Automatically assigned based on department"

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
      operation    = "EQUALS"
      string_value = "Engineering"

      key {
        type     = "IDENTITY"
        property = "attribute.department"
      }
    }
  }

  enabled = true
}
```

### Role with Compound Membership Criteria

```hcl
resource "identitynow_role" "compound_membership" {
  name        = "Compound Membership Role"
  description = "Assigned based on multiple criteria"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  membership {
    type = "STANDARD"

    criteria {
      operation = "AND"

      children {
        operation    = "EQUALS"
        string_value = "Engineering"

        key {
          type     = "IDENTITY"
          property = "attribute.department"
        }
      }

      children {
        operation    = "EQUALS"
        string_value = "US"

        key {
          type     = "IDENTITY"
          property = "attribute.location"
        }
      }
    }
  }

  enabled = true
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) The name of the role. Changing this forces a new resource to be created.
* `description` - (Required) A description of the role.
* `owner` - (Optional) An `owner` block as defined below.
* `access_profiles` - (Optional) One or more `access_profiles` blocks as defined below.
* `entitlements` - (Optional) One or more `entitlements` blocks as defined below.
* `dimension_refs` - (Optional) One or more `dimension_refs` blocks as defined below.
* `membership` - (Optional) A `membership` block as defined below.
* `requestable` - (Optional) Whether this role is requestable via access requests.
* `enabled` - (Optional) Whether this role is enabled.

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

A `dimension_refs` block supports:

* `id` - (Required) The dimension ID.
* `type` - (Required) The type (e.g. `DIMENSION`).
* `name` - (Required) The dimension name.

---

A `membership` block supports:

* `type` - (Required) The membership type (`STANDARD` or `IDENTITY_LIST`).
* `criteria` - (Optional) A `criteria` block as defined below.

---

A `criteria` block supports:

* `operation` - (Required) The criteria operation (`EQUALS`, `NOT_EQUALS`, `CONTAINS`, `AND`, `OR`, etc.).
* `string_value` - (Optional) The value to match against.
* `key` - (Optional) A `key` block identifying the identity attribute.
* `children` - (Optional) One or more child `criteria` blocks (supports up to 3 levels of nesting).

---

A `key` block supports:

* `type` - (Required) The key type (`IDENTITY` or `ACCOUNT`).
* `property` - (Required) The identity or account attribute name (e.g. `attribute.department`).
* `source_id` - (Optional) The source ID (required when `type` is `ACCOUNT`).

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - The ID of the Role.

## Import

Roles can be imported using the `id`, e.g.

```shell
terraform import identitynow_role.example <role-id>
```
