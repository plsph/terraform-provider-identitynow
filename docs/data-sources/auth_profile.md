---
subcategory: "Auth Profile"
page_title: "IdentityNow: Data Source: identitynow_auth_profile"
description: |-
  Gets information about an existing IdentityNow authentication profile.
---

# Data Source: identitynow_auth_profile

Use this data source to look up an authentication profile by ID or name.

~> **Note:** This data source uses an experimental IdentityNow API, which can change without notice. The list of auth profiles only contains IDs, so a lookup by name reads every auth profile of the tenant.

## Example Usage

```terraform
data "identitynow_auth_profile" "default" {
  name = "Default"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) Auth profile ID.
* `name` - (Optional) Auth profile name.

## Attributes Reference

* `type` - Type of the auth profile: `BLOCK`, `MFA`, `NON_PTA` or `PTA`.
* `off_network` - Whether access from off network is blocked.
* `untrusted_geography` - Whether access from untrusted geographies is blocked.
* `application_id` - Application ID.
* `application_name` - Application name.
* `strong_auth_login` - Whether strong authentication is enabled.
