---
subcategory: "Source App"
page_title: "IdentityNow: Data Source App: identitynow_source_app"
description: |-
  Gets information about an existing Source App.
---

# Data Source: identitynow_source_app

Use this data source to access information about an existing Source App.

## Example Usage

```hcl
data "identitynow_source_app" "example" {
  name = "Active Directory Developers"
}

output "identitynow_source_app_description" {
  value = data.identitynow_source_app.example.description
}

output "identitynow_source_app_source_id" {
  value = data.identitynow_source_app.example.source[0].id
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) Name of the source app.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - The source app ID.

* `description` - The description of the source app.

* `enabled` - Whether the source app is enabled.

* `match_all_accounts` - Whether the source app matches all accounts of the source.

* `source` - List with the account source of the source app. Each element contains `id`, `type` and `name`.
