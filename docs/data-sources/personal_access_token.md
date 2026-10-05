---
subcategory: "Personal Access Token"
page_title: "IdentityNow: Data Source: identitynow_personal_access_token"
description: |-
  Gets information about an existing IdentityNow personal access token.
---

# Data Source: identitynow_personal_access_token

Use this data source to look up a personal access token by name. The token secret is not available.

## Example Usage

```terraform
data "identitynow_personal_access_token" "reporting" {
  name = "reporting"
}
```

## Arguments Reference

* `name` - (Required) Token name.
* `owner_id` - (Optional) Identity ID of the token owner. Defaults to the identity the provider authenticates as. Looking up tokens of other identities requires the `idn:all-personal-access-tokens:read` right.

## Attributes Reference

* `id` - Token ID.
* `scope` - Scopes of the token.
* `access_token_validity_seconds` - Number of seconds an access token generated with this token is valid.
* `expiration_date` - Expiration date, null when the token never expires.
* `user_aware_token_never_expires` - Whether a token that never expires was acknowledged.
* `owner` - Identity that owns the token. Contains `id`, `type` and `name`.
* `managed` - Whether the token is managed by the SailPoint platform.
* `created` - Creation date.
* `last_used` - Date the token was last used to generate an access token. It is updated once a day.
