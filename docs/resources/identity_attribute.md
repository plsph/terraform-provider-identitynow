---
subcategory: "Identity Attribute"
page_title: "IdentityNow: identitynow_identity_attribute"
description: |-
  Manages an IdentityNow identity attribute.
---

# identitynow_identity_attribute

Manages an identity attribute. Identity attributes are populated by the attribute mappings of identity profiles.

## Example Usage

```terraform
resource "identitynow_identity_attribute" "cost_center" {
  name         = "costCenter"
  display_name = "Cost Center"
  type         = "string"
  searchable   = true
}
```

## Arguments Reference

* `name` - (Required) Technical name of the identity attribute. It identifies the attribute, changing this forces a new identity attribute to be created.
* `display_name` - (Optional) Business-friendly name of the identity attribute. The API fills it in when not set.
* `type` - (Optional) Type of the identity attribute, e.g. `string`. The API fills it in when not set.
* `standard` - (Optional) Whether the attribute is a standard (default) attribute.
* `multi` - (Optional) Whether the attribute is multi-valued.
* `searchable` - (Optional) Whether the attribute is searchable. Searchable attributes must not be `standard` or `multi`.
* `sources_json` - (Optional) Sources the attribute value is derived from, as a JSON array of objects with `type` (e.g. `rule`) and `properties`. The value is compared semantically.

Optional values that are not set keep the value the API returned. Updates replace the whole attribute with the values from the configuration and state.

## Attributes Reference

* `id` - Identity attribute ID, the same as `name`.
* `system` - Whether the attribute is a system attribute that has no source and is not configurable.

## Import

Identity attributes can be imported using their technical name:

```shell
terraform import identitynow_identity_attribute.example costCenter
```

~> **Note:** IdentityNow only deletes attributes whose `standard` and `system` properties are `false`. Set `standard = false` and apply before destroying a standard attribute.
