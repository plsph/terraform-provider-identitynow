---
subcategory: "Source App"
layout: "identitynow"
page_title: "IdentityNow: Data Source App: identitynow_source_app"
description: |-
  Gets information about an existing Source App.
---

# Data Source: identitynow_source_app

Use this data source to access information about an existing Source App.

## Example Usage

```hcl
data "identitynow_source_app" "example" {
  name = "example"
}

output "identitynow_source_description" {
  value = data.identitynow_source_app.example.description
}
```

## Arguments Reference

The following arguments are supported:

* `name` - Name of the source app.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `description` - The description of the source app.

* `enabled` - True if the source app is enabled.

* `match_all_accounts` - True if the source app match all accounts.

* `source` - The source of the source app.

* `date_created` - Time when the source app was created.

* `last_updated` - Time when the source app was last modified.

