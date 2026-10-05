---
subcategory: "Access Model Metadata"
page_title: "IdentityNow: Data Source: identitynow_access_model_metadata_attribute"
description: |-
  Gets information about an existing IdentityNow access model metadata attribute.
---

# Data Source: identitynow_access_model_metadata_attribute

Use this data source to look up an access model metadata attribute and its values by key.

## Example Usage

```terraform
data "identitynow_access_model_metadata_attribute" "privacy" {
  key = "iscPrivacy"
}
```

## Arguments Reference

* `key` - (Required) Technical name of the attribute.

## Attributes Reference

* `id` - Attribute ID, the same as `key`.
* `name` - Display name of the attribute.
* `type` - Type of the attribute.
* `status` - Status of the attribute.
* `object_types` - Object types the attribute values can be applied to.
* `description` - Description of the attribute.
* `multiselect` - Whether an object can have multiple values of the attribute.
* `values` - Allowed values of the attribute, each with `value`, `name` and `status`.
