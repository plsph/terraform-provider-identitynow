---
subcategory: "Password Policy"
layout: "identitynow"
page_title: "IdentityNow: identitynow_password_sync_group"
description: |-
  Manages an IdentityNow password sync group.
---

# identitynow_password_sync_group

Manages a password sync group, a set of sources that share the same password. Updates replace the whole group, so unset arguments are cleared in IdentityNow.

## Example Usage

```hcl
resource "identitynow_password_sync_group" "ad_and_entra" {
  name               = "AD and Entra ID"
  password_policy_id = "2c91808d744ba0ce01746f93b6204199"
  source_ids = [
    "2c9180835d191a86015d28455b4a2329",
    "2c918085744ba0ce01746f93b6204200",
  ]
}
```

## Arguments Reference

* `name` - (Required) Name of the sync group.
* `password_policy_id` - (Optional) ID of the password policy of the sync group.
* `source_ids` - (Optional) Set of IDs of the password managed sources in the sync group.

## Attributes Reference

* `id` - Password sync group ID.
* `created` - Creation date.
* `modified` - Last modification date.

## Import

Password sync groups can be imported using their ID:

```shell
terraform import identitynow_password_sync_group.example <sync-group-id>
```
