---
subcategory: "Segment"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_segment"
description: |-
  Gets information about an existing IdentityNow segment.
---

# Data Source: identitynow_segment

Use this data source to look up an existing IdentityNow segment by name.

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
* `owner` - The segment owner.
* `visibility_criteria_json` - The segment visibility criteria encoded as JSON.
* `active` - Whether the segment is active.
* `created` - The segment creation timestamp.
* `modified` - The segment modification timestamp.