---
subcategory: "Source"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_source_subtype"
description: |-
  Gets information about a machine account subtype of an IdentityNow source.
---

# Data Source: identitynow_source_subtype

Use this data source to look up a machine account subtype by ID, or by source ID and technical name. It uses the experimental `/source-subtypes` API.

## Example Usage

```hcl
data "identitynow_source_subtype" "service_account" {
  source_id      = "2c9180835d191a86015d28455b4a2329"
  technical_name = "service_account"
}
```

## Arguments Reference

Set either `id`, or both `source_id` and `technical_name`:

* `id` - (Optional) Subtype ID.
* `source_id` - (Optional) ID of the source.
* `technical_name` - (Optional) Technical name of the subtype.

## Attributes Reference

* `display_name` - Display name of the subtype.
* `description` - Description of the subtype.
* `type` - Type of the subtype, `MACHINE` or null.
* `system_managed` - Whether the subtype is managed by the system.
* `created` - Creation date.
* `modified` - Last modification date.
