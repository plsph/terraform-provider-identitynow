---
subcategory: "Authorization"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_custom_user_level"
description: |-
  Gets information about an existing IdentityNow custom user level.
---

# Data Source: identitynow_custom_user_level

Use this data source to look up a custom user level by ID.

~> **Note:** This data source uses an experimental API that may change without notice.

## Example Usage

```hcl
data "identitynow_custom_user_level" "identity_managers" {
  id = "beb02a57-010f-4c29-a6d2-fae9628bda73"
}
```

## Arguments Reference

* `id` - (Required) User level ID.

## Attributes Reference

* `name` - Name of the user level.
* `description` - Description of the user level.
* `owner` - Identity that owns the user level. Contains `id`, `type` and `name`.
* `right_sets` - IDs of the right sets assigned to the user level.
* `status` - Status of the user level, `DRAFT` or `ACTIVE`.
* `created` - Creation date.
* `modified` - Last modification date.
* `associated_identities_count` - Number of identities the user level is assigned to.
