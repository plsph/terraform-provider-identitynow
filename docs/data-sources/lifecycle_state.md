---
subcategory: "Identity Profile"
page_title: "IdentityNow: Data Source: identitynow_lifecycle_state"
description: |-
  Gets information about an existing lifecycle state of an IdentityNow identity profile.
---

# Data Source: identitynow_lifecycle_state

Use this data source to look up a lifecycle state of an identity profile by ID.

## Example Usage

```hcl
data "identitynow_lifecycle_state" "active" {
  identity_profile_id = "2b838de9-db9b-abcf-e646-d4f274ad4238"
  id                  = "ef38f94347e94562b5bb8424a56397d8"
}
```

## Arguments Reference

* `identity_profile_id` - (Required) ID of the identity profile the lifecycle state belongs to.
* `id` - (Required) Lifecycle state ID.

## Attributes Reference

* `name` - Name of the lifecycle state.
* `technical_name` - Technical name of the lifecycle state.
* `description` - Description of the lifecycle state.
* `enabled` - Whether the lifecycle state is enabled.
* `identity_state` - Identity state associated with the lifecycle state.
* `priority` - Sort order of the lifecycle state.
* `account_actions_json` - Account actions as a JSON array.
* `access_profile_ids` - Set of IDs of the access profiles granted in the lifecycle state.
* `remove_all_access_enabled` - Whether all access is marked for removal.
* `email_notification_option` - Email notifications, a list with one element with `notify_managers`, `notify_all_admins`, `notify_specific_users` and `email_address_list`.
* `identity_count` - Number of identities in the lifecycle state.
* `created` - Creation date.
* `modified` - Last modification date.
