---
subcategory: "Access Model Metadata"
page_title: "IdentityNow: identitynow_access_model_metadata_attribute"
description: |-
  Manages an IdentityNow access model metadata attribute.
---

# identitynow_access_model_metadata_attribute

Manages an access model metadata attribute and its values. Metadata attribute values can be assigned to access items such as entitlements, access profiles and roles.

~> **Note:** The IdentityNow API cannot delete metadata attributes. Destroying this resource only removes it from Terraform state and leaves the attribute and its values in IdentityNow.

## Example Usage

```hcl
resource "identitynow_access_model_metadata_attribute" "data_sensitivity" {
  name         = "Data Sensitivity"
  description  = "Sensitivity of the data the access grants"
  type         = "governance"
  object_types = ["all"]
  multiselect  = false

  values {
    value = "public"
    name  = "Public"
  }

  values {
    value = "confidential"
    name  = "Confidential"
  }
}
```

## Arguments Reference

* `name` - (Required) Display name of the attribute.
* `key` - (Optional) Unique technical name of the attribute. The API derives it from `name` when not set. It cannot be changed after creation.
* `type` - (Optional) Type of the attribute, `custom` or `governance`. The API sets a default when not set. It cannot be changed after creation.
* `object_types` - (Optional) Object types the attribute values can be applied to, `all` or `entitlement`. The API sets a default when not set. It cannot be changed after creation.
* `status` - (Optional) Status of the attribute, e.g. `active`. The API sets a default when not set. It cannot be changed after creation.
* `description` - (Optional) Description of the attribute.
* `multiselect` - (Optional) Whether an object can have multiple values of the attribute. The API defaults it to `false`.
* `values` - (Optional) Allowed values of the attribute, one block per value. Updates replace the whole list of values. It supports:
    * `value` - (Required) Unique technical name of the value.
    * `name` - (Required) Display name of the value.
    * `status` - (Optional) Status of the value, e.g. `active`. The API sets a default when not set.

Only `name`, `description`, `multiselect` and `values` can be updated. Since the API cannot delete metadata attributes, the resource cannot be replaced either: changing `key`, `type`, `status` or `object_types` of an existing attribute makes the plan fail with an error. Revert the change, or change the attribute in the IdentityNow UI and update the configuration to match, or create an attribute with another `key` in a new resource.

## Attributes Reference

* `id` - Attribute ID, the same as `key`.

## Import

Access model metadata attributes can be imported using their key:

```shell
terraform import identitynow_access_model_metadata_attribute.example <key>
```
