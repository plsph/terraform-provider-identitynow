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
  value = data.identitynow_source_app.description
}
```

## Arguments Reference

The following arguments are supported:

* name

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* description
* match_all_accounts
* enabled
* source
* date_created
* last_updated

