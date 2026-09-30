---
subcategory: "Transform"
layout: "identitynow"
page_title: "IdentityNow: identitynow_transform"
description: |-
  Manages an IdentityNow transform.
---

# identitynow_transform

Manages a transform. Transforms manipulate attribute values and are used in identity profile attribute mappings.

## Example Usage

```hcl
resource "identitynow_transform" "country_lookup" {
  name = "Country Lookup"
  type = "lookup"
  attributes_json = jsonencode({
    table = {
      US      = "United States"
      PL      = "Poland"
      default = "Unknown"
    }
  })
}
```

## Arguments Reference

* `name` - (Required) Unique name of the transform. Changing this forces a new transform to be created.
* `type` - (Required) Transform operation type, e.g. `lookup`, `concat` or `dateFormat`. See the [transform operations](https://developer.sailpoint.com/docs/extensibility/transforms/operations). Changing this forces a new transform to be created.
* `attributes_json` - (Optional) Transform attributes as a JSON object. The attributes depend on the transform type. Use `jsonencode()` for convenience. The value is compared semantically, so formatting and key order do not produce a diff.

## Attributes Reference

* `id` - Transform ID.
* `internal` - Whether this is a SailPoint internal transform.

## Import

Transforms can be imported using their ID:

```shell
terraform import identitynow_transform.example <transform-id>
```
