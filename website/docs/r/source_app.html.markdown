---
subcategory: "Source App"
layout: "identitynow"
page_title: "IdentityNow: identitynow_source_app"
description: |-
  Manages an IdentityNow Source App.
---

# identitynow_source_app

Manages an IdentityNow Source App.

## Example Usage

```hcl
resource_app "identitynow_source_app" "example" {
  description = "example"
  enabled = true
  match_all_accounts = true
  name = "example"
  source {
      id = "some_id"
      name = "example"
      type = "SOURCE"
  } 
}
```

## Arguments Reference

The following arguments are supported:

* `name` - The source app name.

* `description` - The description of the source app.

* `enabled` - True if the source app is enabled.

* `match_all_accounts` - True if the source app match all accounts.

* `source` - The source of the source app.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `date_created` - Time when the source app was created.

* `last_updated` - Time when the source app was last modified.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/resource/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the Source App.
* `read` - (Defaults to 5 minutes) Used when retrieving the Source App.
* `update` - (Defaults to 30 minutes) Used when updating the Source App.
* `delete` - (Defaults to 30 minutes) Used when deleting the Source App.

## Import

* terraform import identitynow_source_app.example [id]
