---
subcategory: "Launcher"
page_title: "IdentityNow: Data Source: identitynow_launcher"
description: |-
  Gets information about an existing IdentityNow launcher.
---

# Data Source: identitynow_launcher

Use this data source to look up a launcher by ID.

## Example Usage

```terraform
data "identitynow_launcher" "onboarding" {
  id = "e3012408-8b61-4564-ad41-c5ec131c325b"
}
```

## Arguments Reference

* `id` - (Required) Launcher ID.

## Attributes Reference

* `name` - Name of the launcher.
* `description` - Description of the launcher.
* `type` - Launcher type.
* `disabled` - Whether the launcher is disabled.
* `reference` - Object the launcher starts. Contains `id` and `type`.
* `config_json` - Launcher configuration as a JSON object.
* `owner` - Owner of the launcher. Contains `id` and `type`.
* `created` - Creation date.
* `modified` - Last modification date.
