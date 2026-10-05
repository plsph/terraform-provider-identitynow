---
subcategory: "Password Policy"
page_title: "IdentityNow: Data Source: identitynow_password_sync_group"
description: |-
  Gets information about an existing IdentityNow password sync group.
---

# Data Source: identitynow_password_sync_group

Use this data source to look up a password sync group by ID or name. Lookups by name list all sync groups and match the name exactly.

## Example Usage

```terraform
data "identitynow_password_sync_group" "ad_and_entra" {
  name = "AD and Entra ID"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) Password sync group ID.
* `name` - (Optional) Name of the sync group.

## Attributes Reference

* `password_policy_id` - ID of the password policy of the sync group.
* `source_ids` - Set of IDs of the sources in the sync group.
* `created` - Creation date.
* `modified` - Last modification date.
