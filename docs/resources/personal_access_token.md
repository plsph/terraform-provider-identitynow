---
subcategory: "Personal Access Token"
page_title: "IdentityNow: identitynow_personal_access_token"
description: |-
  Manages an IdentityNow personal access token.
---

# identitynow_personal_access_token

Manages a personal access token (PAT). The token is owned by the identity the provider authenticates as.

~> **Note:** The token secret is only returned by the API when the token is created. It is stored in the Terraform state as a sensitive value, so protect the state accordingly. Imported tokens have no `secret`.

## Example Usage

```terraform
resource "identitynow_personal_access_token" "reporting" {
  name                          = "reporting"
  scope                         = ["sp:scopes:all"]
  access_token_validity_seconds = 3600
  expiration_date               = "2027-12-31T23:59:59Z"
}

# A token that never expires must be acknowledged explicitly.
resource "identitynow_personal_access_token" "service" {
  name                           = "service"
  user_aware_token_never_expires = true
}
```

## Arguments Reference

* `name` - (Required) Token name, unique among the tokens of the owner.
* `scope` - (Optional) Scopes of the token. Defaults to `sp:scopes:all`, all rights of the owner. Scope changes only apply to access tokens generated after the change, which can take up to 20 minutes.
* `access_token_validity_seconds` - (Optional) Number of seconds an access token generated with this token is valid, between 15 and 43200. Defaults to 43200. Cannot be updated, changing it forces a new token to be created.
* `expiration_date` - (Optional) Date and time in RFC 3339 format when the token expires. It must be in the future. When not set, the token never expires and `user_aware_token_never_expires` must be `true`. Dates that denote the same instant do not produce a diff.
* `user_aware_token_never_expires` - (Optional) Acknowledges the security implications of a token that never expires. Must be `true` when `expiration_date` is not set.

`name`, `scope`, `expiration_date` and `user_aware_token_never_expires` are updated in place.

## Attributes Reference

* `id` - Token ID, used as the client ID when requesting access tokens.
* `secret` - (Sensitive) Token secret, used as the client secret. Only available for tokens created by Terraform.
* `owner` - Identity that owns the token. Contains `id`, `type` and `name`.
* `managed` - Whether the token is managed by the SailPoint platform, for example by workflows.
* `created` - Creation date.

## Import

Personal access tokens can be imported using their ID. The API has no endpoint to read a single token, so the token is searched in the tokens of the caller and then in all tokens of the tenant, which requires the `idn:all-personal-access-tokens:read` right. The secret cannot be imported:

```shell
terraform import identitynow_personal_access_token.example <token-id>
```
