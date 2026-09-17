---
subcategory: "Segment"
layout: "identitynow"
page_title: "IdentityNow: identitynow_segment"
description: |-
  Manages an IdentityNow segment.
---

# identitynow_segment

Manages an IdentityNow segment. Segment changes can take time to propagate to all identities.

## Example Usage

```hcl
resource "identitynow_segment" "austin" {
  name        = "Austin employees"
  description = "Employees whose location is Austin"
  active      = true

  owner {
    id   = "2c9180a46faadee4016fb4e018c20639"
    type = "IDENTITY"
    name = "support"
  }

  visibility_criteria {
    expression {
    operator  = "EQUALS"
    attribute = "location"

    value {
      type  = "STRING"
      value = "Austin"
    }
  }
  }
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) The segment business name.
* `description` - (Optional) The segment description.
* `owner` - (Optional) The segment owner. Supports `id`, `type`, and `name`.
* `visibility_criteria` - (Optional) Visibility criteria following the SailPoint Visibility Criteria schema. Supports `expression`, `operator`, `attribute`, `value`, and nested `children` blocks.
* `visibility_criteria_json` - (Optional) Visibility criteria as a JSON object following the SailPoint Visibility Criteria schema. Conflicts with `visibility_criteria`; configure only one.
* `active` - (Optional) Whether the segment is active. Defaults to `false`.

## Attributes Reference

* `id` - The segment ID.
* `created` - The segment creation timestamp.
* `modified` - The segment modification timestamp.

## Import

```shell
terraform import identitynow_segment.austin <segment-id>
```