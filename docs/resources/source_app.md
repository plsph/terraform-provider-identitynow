---
subcategory: "Source App"
page_title: "IdentityNow: identitynow_source_app"
description: |-
  Manages an IdentityNow Source App.
---

# identitynow_source_app

Manages an IdentityNow Source App. Use `identitynow_access_profile_attachment` to attach access profiles to it.

## Example Usage

```terraform
resource "identitynow_source_app" "example" {
  name               = "Active Directory Developers"
  description        = "Application for requesting developer access"
  enabled            = true
  match_all_accounts = true

  source {
    id   = "2c9180835d191a86015d28455b4a2329"
    name = "Active Directory"
    type = "SOURCE"
  }
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) The source app name.

* `description` - (Required) The description of the source app.

* `enabled` - (Optional) Whether the source app is enabled. If not set, the value returned by the API is used.

* `match_all_accounts` - (Optional) Whether the source app matches all accounts of the source. If not set, the value returned by the API is used.

* `source` - (Optional) Account source of the source app. At most one block is allowed. Can be changed in place. Contains:
  * `id` - (Required) Source ID.
  * `name` - (Required) Source name.
  * `type` - (Optional) Source type. Defaults to `SOURCE`.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Source app ID.

## Import

Source apps can be imported using their ID:

```shell
terraform import identitynow_source_app.example <source-app-id>
```
