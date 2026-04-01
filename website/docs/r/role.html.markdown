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

  requestable = true
  enabled     = true
  dimensional = true
}
```

### Role with Access Model Metadata

```hcl
resource "identitynow_role" "with_metadata" {
  name        = "Metadata Role"
  description = "A role with access model metadata"

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

  access_model_metadata {
    attributes {
      key  = "iscPrivacy"
      name = "Privacy"

      values {
        value  = "public"
        name   = "Public"
        status = "active"
      }
    }
  }

  requestable = true
  enabled     = true
}
```

### Role with Access Request Config

```hcl
resource "identitynow_role" "with_approval" {
  name        = "Approval Role"
  description = "A role with access request configuration"

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

  access_request_config {
    comments_required        = true
    denial_comments_required = true

    approval_schemes {
      approver_type = "MANAGER"
    }

    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = "2c91808a7813090a017813b6301faaaa"
    }
  }

  requestable = true
  enabled     = true
}
```

### Dimensional Role with Dimension-Specific Approval Schemas

```hcl
resource "identitynow_role" "dimensional_approval" {
  name        = "Dimensional Approval Role"
  description = "A dimensional role with per-dimension approval schemas"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  access_request_config {
    comments_required        = true
    denial_comments_required = false

    approval_schemes {
      approver_type = "MANAGER"
    }

    dimension_schema {
      dimension_attributes {
        name         = "eqLocation"
        display_name = "EQ Location"
        derived      = true
      }
    }
  }

  requestable = true
  enabled     = true
  dimensional = true
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
* `description` - (Optional) A description of the role.
* `owner` - (Optional) An `owner` block as defined below.
* `access_profiles` - (Optional) One or more `access_profiles` blocks as defined below.
* `entitlements` - (Optional) One or more `entitlements` blocks as defined below.
* `access_model_metadata` - (Optional) An `access_model_metadata` block as defined below. Defines access model metadata for this role.
* `access_request_config` - (Optional) An `access_request_config` block as defined below. Configures the approval process for access requests.
* `membership` - (Optional) A `membership` block as defined below.
* `requestable` - (Optional) Whether this role is requestable via access requests.
* `enabled` - (Optional) Whether this role is enabled.
* `dimensional` - (Optional) Whether this role is dimensional.

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

An `access_model_metadata` block supports:

* `attributes` - (Optional) One or more `attributes` blocks as defined below.

---

An `attributes` block (within `access_model_metadata`) supports:

* `key` - (Required) The unique identifier for the metadata type (e.g. `iscPrivacy`).
* `name` - (Required) The human readable name of the metadata attribute.
* `values` - (Optional) One or more `values` blocks as defined below.

---

A `values` block (within `attributes`) supports:

* `value` - (Required) The metadata value.
* `name` - (Required) The human readable name of the value.
* `status` - (Optional) The status of the value (e.g. `active`).

---

An `access_request_config` block supports:

* `comments_required` - (Optional) Whether comments are required when requesting access.
* `denial_comments_required` - (Optional) Whether comments are required when denying access.
* `approval_schemes` - (Optional) One or more `approval_schemes` blocks as defined below.
* `dimension_schema` - (Optional) A `dimension_schema` block for dimension-specific approval configuration.

---

An `approval_schemes` block (within `access_request_config`) supports:

* `approver_type` - (Required) The type of approver (e.g. `APP_OWNER`, `MANAGER`, `GOVERNANCE_GROUP`).
* `approver_id` - (Optional) The ID of the approver (required when `approver_type` is `GOVERNANCE_GROUP`).

---

A `dimension_schema` block supports:

* `dimension_attributes` - (Optional) One or more `dimension_attributes` blocks as defined below.

---

A `dimension_attributes` block supports:

* `name` - (Required) The attribute name.
* `display_name` - (Required) The display name of the attribute.
* `derived` - (Required) Whether the attribute is derived.

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
