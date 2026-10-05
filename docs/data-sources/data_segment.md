---
subcategory: "Data Segmentation"
page_title: "IdentityNow: Data Source: identitynow_data_segment"
description: |-
  Gets information about an existing IdentityNow data access segment.
---

# Data Source: identitynow_data_segment

Use this data source to look up a data access segment by ID. It uses an experimental API (`X-SailPoint-Experimental` header).

## Example Usage

```hcl
data "identitynow_data_segment" "emea" {
  id = "ef38f943-47e9-4562-b5bb-8424a56397d8"
}
```

## Arguments Reference

* `id` - (Required) Data segment ID.

## Attributes Reference

* `name` - Segment business name.
* `description` - Segment description.
* `membership` - How members are chosen, `ALL`, `FILTER` or `SELECTION`.
* `member_filter_json` - Member filter as a JSON object.
* `member_selection` - Identities selected as members, with `id` and `type`.
* `scopes_json` - Scopes of the segment as a JSON array.
* `enabled` - Whether the segment is active.
* `published` - Whether the segment is published.
* `created` - Creation date.
* `modified` - Last modification date.
