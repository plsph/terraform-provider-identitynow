---
subcategory: "Role"
page_title: "IdentityNow: identitynow_role"
description: |-
  Manages an IdentityNow Role.
---

# identitynow_role

Manages an IdentityNow Role. Roles bundle access profiles, entitlements, and dimensions together and can be assigned to identities through access requests or membership criteria.

All arguments, including `name`, can be updated in place. Removing `description` or `membership` from the configuration clears them in IdentityNow.

## Example Usage

### Basic Role

```terraform
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

```terraform
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

```terraform
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
      key         = "iscPrivacy"
      name        = "Privacy"
      multiselect = false
      status      = "active"
      type        = "custom"

      values {
        value  = "public"
        name   = "Public"
        status = "active"
      }

      values {
        value  = "internal"
        name   = "Internal"
        status = "active"
      }
    }
  }

  requestable = true
  enabled     = true
}
```

### Role with Access Request Config

```terraform
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

```terraform
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

```terraform
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

```terraform
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

### Role with Multi-Value Membership Criteria

```terraform
resource "identitynow_role" "multivalue_membership" {
  name        = "Multi-Value Membership Role"
  description = "Assigned based on multiple job codes"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  membership {
    type = "STANDARD"

    criteria {
      operation = "OR"

      children {
        operation = "AND"

        children {
          operation = "EQUALS"
          values    = ["active"]

          key {
            type     = "IDENTITY"
            property = "attribute.cloudLifecycleState"
          }
        }

        children {
          operation = "EQUALS"
          values    = ["G8244", "G8243", "G8242", "G6644"]

          key {
            type     = "IDENTITY"
            property = "attribute.jobcode"
          }
        }
      }
    }
  }

  enabled = true
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the role. Can be updated in place.

### Optional

- `access_model_metadata` (Block List) Defines access model metadata for this role. (see [below for nested schema](#nestedblock--access_model_metadata))
- `access_profiles` (Block List) Access profiles assigned to this role. (see [below for nested schema](#nestedblock--access_profiles))
- `access_request_config` (Block List) Configures the approval process for access requests. (see [below for nested schema](#nestedblock--access_request_config))
- `description` (String) A description of the role. Removing it clears the description.
- `dimensional` (Boolean) Whether this role is dimensional. If not set, the value returned by the API is used.
- `enabled` (Boolean) Whether this role is enabled. If not set, the value returned by the API is used.
- `entitlements` (Block List) Entitlements assigned to this role. (see [below for nested schema](#nestedblock--entitlements))
- `membership` (Block List) Role membership definition. Defines how identities are assigned to this role. Removing it clears the membership criteria. (see [below for nested schema](#nestedblock--membership))
- `owner` (Block List) Role owner. Exactly one block is required. (see [below for nested schema](#nestedblock--owner))
- `requestable` (Boolean) Whether this role is requestable via access requests. If not set, the value returned by the API is used.

### Read-Only

- `id` (String) The ID of the role.

<a id="nestedblock--access_model_metadata"></a>
### Nested Schema for `access_model_metadata`

Optional:

- `attributes` (Block List) Metadata attributes. Each `attributes` block corresponds to an `AccessModelMetadataAttribute` in the IdentityNow API. (see [below for nested schema](#nestedblock--access_model_metadata--attributes))

<a id="nestedblock--access_model_metadata--attributes"></a>
### Nested Schema for `access_model_metadata.attributes`

Required:

- `key` (String) The unique identifier for the metadata type (e.g. `iscPrivacy`).
- `name` (String) The human readable name of the metadata attribute.

Optional:

- `description` (String) The description of the metadata attribute. If not set, the value returned by the API is used.
- `multiselect` (Boolean) Whether multiple values can be selected for the metadata attribute. If not set, the value returned by the API is used.
- `object_types` (Block List) One or more `object_types` blocks, each with a required `value` naming an object type the metadata attribute applies to (e.g. `role`). (see [below for nested schema](#nestedblock--access_model_metadata--attributes--object_types))
- `status` (String) The status of the metadata attribute (e.g. `active`). If not set, the value returned by the API is used.
- `type` (String) The type of the metadata attribute (e.g. `custom`). If not set, the value returned by the API is used.
- `values` (Block List) Values assigned to this metadata attribute. (see [below for nested schema](#nestedblock--access_model_metadata--attributes--values))

<a id="nestedblock--access_model_metadata--attributes--object_types"></a>
### Nested Schema for `access_model_metadata.attributes.object_types`

Required:

- `value` (String) An object type the metadata attribute applies to (e.g. `role`).


<a id="nestedblock--access_model_metadata--attributes--values"></a>
### Nested Schema for `access_model_metadata.attributes.values`

Required:

- `name` (String) The human readable name of the value.
- `value` (String) The metadata value.

Optional:

- `status` (String) The status of the value (e.g. `active`). If not set, the value returned by the API is used.




<a id="nestedblock--access_profiles"></a>
### Nested Schema for `access_profiles`

Required:

- `id` (String) The access profile ID.
- `name` (String) The access profile name.
- `type` (String) The type (e.g. `ACCESS_PROFILE`).


<a id="nestedblock--access_request_config"></a>
### Nested Schema for `access_request_config`

Optional:

- `approval_schemes` (Block List) Approval schemes for this role. (see [below for nested schema](#nestedblock--access_request_config--approval_schemes))
- `comments_required` (Boolean) Whether comments are required when requesting access. If not set, the value returned by the API is used.
- `denial_comments_required` (Boolean) Whether comments are required when denying access. If not set, the value returned by the API is used.
- `dimension_schema` (Block List) A `dimension_schema` block for dimension-specific approval configuration. (see [below for nested schema](#nestedblock--access_request_config--dimension_schema))
- `form_definition_id` (String) ID of the form definition presented to the requester during the access request.

<a id="nestedblock--access_request_config--approval_schemes"></a>
### Nested Schema for `access_request_config.approval_schemes`

Required:

- `approver_type` (String) The type of approver (e.g. `APP_OWNER`, `MANAGER`, `GOVERNANCE_GROUP`).

Optional:

- `approver_id` (String) The ID of the approver (required when `approver_type` is `GOVERNANCE_GROUP`).


<a id="nestedblock--access_request_config--dimension_schema"></a>
### Nested Schema for `access_request_config.dimension_schema`

Optional:

- `dimension_attributes` (Block List) Dimension attributes that define this dimension. (see [below for nested schema](#nestedblock--access_request_config--dimension_schema--dimension_attributes))

<a id="nestedblock--access_request_config--dimension_schema--dimension_attributes"></a>
### Nested Schema for `access_request_config.dimension_schema.dimension_attributes`

Required:

- `name` (String) The attribute name.

Optional:

- `derived` (Boolean) Whether the attribute is derived. If not set, the value returned by the API is used.
- `display_name` (String) The display name of the attribute. If not set, the value returned by the API is used.




<a id="nestedblock--entitlements"></a>
### Nested Schema for `entitlements`

Required:

- `id` (String) The entitlement ID.
- `name` (String) The entitlement name.
- `type` (String) The type (e.g. `ENTITLEMENT`).


<a id="nestedblock--membership"></a>
### Nested Schema for `membership`

Required:

- `type` (String) The membership type (`STANDARD` or `IDENTITY_LIST`).

Optional:

- `criteria` (Block List) Membership criteria. (see [below for nested schema](#nestedblock--membership--criteria))

<a id="nestedblock--membership--criteria"></a>
### Nested Schema for `membership.criteria`

Required:

- `operation` (String) The criteria operation (`EQUALS`, `NOT_EQUALS`, `CONTAINS`, `AND`, `OR`, etc.).

Optional:

- `children` (Block List) One or more child `criteria` blocks (supports up to 3 levels of nesting). (see [below for nested schema](#nestedblock--membership--criteria--children))
- `key` (Block List) A `key` block identifying the identity attribute. (see [below for nested schema](#nestedblock--membership--criteria--key))
- `string_value` (String) A single value to match against.
- `values` (List of String) A list of values to match against. Use this when the criteria should match any of multiple values.

<a id="nestedblock--membership--criteria--children"></a>
### Nested Schema for `membership.criteria.children`

Required:

- `operation` (String) The criteria operation (`EQUALS`, `NOT_EQUALS`, `CONTAINS`, `AND`, `OR`, etc.).

Optional:

- `children` (Block List) One or more child `criteria` blocks on the third and last nesting level, which cannot have children of their own. (see [below for nested schema](#nestedblock--membership--criteria--children--children))
- `key` (Block List) A `key` block identifying the identity attribute. (see [below for nested schema](#nestedblock--membership--criteria--children--key))
- `string_value` (String) A single value to match against.
- `values` (List of String) A list of values to match against. Use this when the criteria should match any of multiple values.

<a id="nestedblock--membership--criteria--children--children"></a>
### Nested Schema for `membership.criteria.children.children`

Required:

- `operation` (String) The criteria operation (`EQUALS`, `NOT_EQUALS`, `CONTAINS`, `AND`, `OR`, etc.).

Optional:

- `key` (Block List) A `key` block identifying the identity attribute. (see [below for nested schema](#nestedblock--membership--criteria--children--children--key))
- `string_value` (String) A single value to match against.
- `values` (List of String) A list of values to match against. Use this when the criteria should match any of multiple values.

<a id="nestedblock--membership--criteria--children--children--key"></a>
### Nested Schema for `membership.criteria.children.children.key`

Required:

- `property` (String) The identity or account attribute name (e.g. `attribute.department`).
- `type` (String) The key type (`IDENTITY` or `ACCOUNT`).

Optional:

- `source_id` (String) The source ID (required when `type` is `ACCOUNT`).



<a id="nestedblock--membership--criteria--children--key"></a>
### Nested Schema for `membership.criteria.children.key`

Required:

- `property` (String) The identity or account attribute name (e.g. `attribute.department`).
- `type` (String) The key type (`IDENTITY` or `ACCOUNT`).

Optional:

- `source_id` (String) The source ID (required when `type` is `ACCOUNT`).



<a id="nestedblock--membership--criteria--key"></a>
### Nested Schema for `membership.criteria.key`

Required:

- `property` (String) The identity or account attribute name (e.g. `attribute.department`).
- `type` (String) The key type (`IDENTITY` or `ACCOUNT`).

Optional:

- `source_id` (String) The source ID (required when `type` is `ACCOUNT`).




<a id="nestedblock--owner"></a>
### Nested Schema for `owner`

Required:

- `id` (String) The owner's ID.
- `name` (String) The owner name.
- `type` (String) The owner type (e.g. `IDENTITY`).

## Import

Roles can be imported using the `id`, e.g.

```shell
terraform import identitynow_role.example <role-id>
```
