---
subcategory: "Segment"
page_title: "IdentityNow: Data Source: identitynow_segment"
description: |-
  Gets information about an existing IdentityNow segment.
---

# Data Source: identitynow_segment

Use this data source to look up an existing IdentityNow segment by name. The lookup fails when several segments share the name.

## Example Usage

```hcl
data "identitynow_segment" "austin" {
  name = "Austin employees"
}

output "segment_id" {
  value = data.identitynow_segment.austin.id
}
```

## Arguments Reference

* `name` - (Required) The segment business name.

## Attributes Reference

* `id` - The segment ID.
* `description` - The segment description.
* `owner` - List with the segment owner. Each element contains `id`, `type` and `name`.
* `visibility_criteria` - The segment visibility criteria as structured nested attributes: a list with an `expression` containing `operator`, `attribute`, `value` (with `type` and `value`) and `children`, each child with a nested `expression` (up to 3 levels).
* `active` - Whether the segment is active.
* `created` - The segment creation timestamp.
* `modified` - The segment modification timestamp.
