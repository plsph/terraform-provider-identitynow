---
subcategory: "Source"
layout: "identitynow"
page_title: "IdentityNow: identitynow_account_schema"
description: |-
  Manages the attributes of an existing account schema of an IdentityNow Source.
---

# identitynow_account_schema

Manages the settings and attributes of an existing schema of a source, e.g. the `account` or `group` schema.

The resource does not create schemas: the schema must already exist, which is usually the case after the source has been created and its connection has been tested. The schema ID can be found with the [list schemas on source](https://developer.sailpoint.com/docs/api/v3/get-source-schemas) API.

When at least one `attributes` block is configured, the list of attributes is authoritative: attributes of the schema that are not configured are removed from the schema. Without `attributes` blocks the existing attributes are left unchanged. Fields of existing attributes that are not managed by Terraform, such as `nativeName`, are kept.

Destroying the resource only removes it from the Terraform state. The schema and its attributes are left in place in IdentityNow and a warning is shown.

## Example Usage

### Manage schema settings only

```hcl
resource "identitynow_account_schema" "active_directory_account" {
  source_id          = "2c9180835d191a86015d28455b4a2329"
  schema_id          = "2c9180835d191a86015d28455b4a2330"
  identity_attribute = "distinguishedName"
  display_attribute  = "sAMAccountName"
}
```

### Manage the full list of attributes

```hcl
resource "identitynow_account_schema" "hr_account" {
  source_id          = "2c9180835d191a86015d28455b4a2329"
  schema_id          = "2c9180835d191a86015d28455b4a2331"
  identity_attribute = "employeeId"
  display_attribute  = "employeeId"

  # All attributes of the schema must be listed, unlisted attributes are removed.
  attributes {
    name        = "employeeId"
    type        = "STRING"
    description = "Employee ID"
  }

  attributes {
    name        = "email"
    type        = "STRING"
    description = "Work email address"
  }

  attributes {
    name            = "groups"
    type            = "STRING"
    description     = "Groups of the account"
    is_multi_valued = true
    is_entitlement  = true
    is_group        = true

    schema {
      id   = "2c9180835d191a86015d28455b4a2332"
      name = "group"
      type = "CONNECTOR_SCHEMA"
    }
  }
}
```

## Arguments Reference

The following arguments are supported:

As per developer guide: (https://developer.sailpoint.com/docs/api/v3/put-source-schema)

* `source_id` - (Required) ID of the source the schema belongs to. Changing this forces a new resource to be created.

* `schema_id` - (Required) ID of the existing schema. Changing this forces a new resource to be created.

* `name` - (Optional) Name of the schema, e.g. `account`. The name can't be changed after the schema is created. If not set, the existing name is used.

* `native_object_type` - (Optional) Name of the object type on the source system, e.g. `User`. If not set, the existing value is kept.

* `identity_attribute` - (Optional) Name of the attribute that uniquely identifies an account on the source. If not set, the existing value is kept.

* `display_attribute` - (Optional) Name of the attribute used to display the account. If not set, the existing value is kept.

* `hierarchy_attribute` - (Optional) Name of the attribute used to build group hierarchies. If not set, the existing value is kept.

* `include_permissions` - (Optional) Whether permissions are included in the schema. If not set, the existing value is kept.

* `attributes` - (Optional) Attributes of the schema. When at least one block is configured the list is authoritative. Can be repeated. Contains:
  * `name` - (Required) Attribute name.
  * `type` - (Optional) Attribute type, e.g. `STRING`, `LONG`, `INT`, `BOOLEAN` or `DATE`. If not set, the existing value is kept.
  * `description` - (Optional) Attribute description. If not set, the existing value is kept.
  * `is_multi_valued` - (Optional) Whether the attribute is multi-valued. If not set, the existing value is kept.
  * `is_entitlement` - (Optional) Whether the attribute is an entitlement. If not set, the existing value is kept.
  * `is_group` - (Optional) Whether the attribute refers to a group. If not set, the existing value is kept.
  * `schema` - (Optional) Reference to the schema of the objects the attribute refers to, e.g. the `group` schema for a group membership attribute. Contains:
    * `id` - (Required) Schema ID.
    * `name` - (Required) Schema name.
    * `type` - (Required) Reference type, `CONNECTOR_SCHEMA`.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Schema ID (same as `schema_id`).

* `created` - The date and time the schema was created.

* `modified` - The date and time the schema was last modified.

## Import

Account schemas can be imported using the source ID and the schema ID separated by `/`:

```shell
terraform import identitynow_account_schema.example <source-id>/<schema-id>
```

When imported, all existing attributes of the schema are read into the state. Configure all of them as `attributes` blocks to manage them, or leave out the `attributes` blocks to keep them unmanaged.
