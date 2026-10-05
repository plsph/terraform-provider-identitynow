---
subcategory: "OAuth Client"
page_title: "IdentityNow: identitynow_oauth_client"
description: |-
  Manages an IdentityNow OAuth (API) client.
---

# identitynow_oauth_client

Manages an OAuth (API) client, for example the credentials used by a CI pipeline or an integration.

~> **Note:** The client secret is only returned by the API when the client is created. It is stored in the Terraform state as a sensitive value, so protect the state accordingly. Imported clients have no `secret`.

## Example Usage

```terraform
resource "identitynow_oauth_client" "ci" {
  name                          = "CI pipeline"
  description                   = "Used by the CI pipeline to deploy configuration"
  access_token_validity_seconds = 750
  grant_types                   = ["CLIENT_CREDENTIALS"]
  access_type                   = "OFFLINE"
  scope                         = ["sp:scopes:all"]
}

output "ci_client_secret" {
  value     = identitynow_oauth_client.ci.secret
  sensitive = true
}
```

## Arguments Reference

* `name` - (Required) Human-readable name of the API client.
* `description` - (Required) Description of the API client.
* `access_token_validity_seconds` - (Required) Number of seconds an access token generated for this client is valid for.
* `grant_types` - (Required) OAuth 2.0 grant types the client can be used with: `CLIENT_CREDENTIALS`, `AUTHORIZATION_CODE` or `REFRESH_TOKEN`.
* `access_type` - (Required) Access type, `ONLINE` or `OFFLINE`.
* `business_name` - (Optional) Name of the business the API client belongs to.
* `homepage_url` - (Optional) Homepage URL associated with the owner of the API client.
* `refresh_token_validity_seconds` - (Optional) Number of seconds a refresh token is valid for. When not set, the value chosen by IdentityNow is used.
* `redirect_uris` - (Optional) Approved redirect URIs, required for the `AUTHORIZATION_CODE` grant type.
* `enabled` - (Optional) Whether the API client is enabled. Defaults to `true`.
* `strong_auth_supported` - (Optional) Whether the API client supports strong authentication. When not set, the value chosen by IdentityNow is used.
* `claims_supported` - (Optional) Whether the API client supports the serialization of SAML claims with the `AUTHORIZATION_CODE` flow. When not set, the value chosen by IdentityNow is used.
* `type` - (Optional) Client type, `CONFIDENTIAL` or `PUBLIC`. Cannot be updated, changing it forces a new client to be created.
* `internal` - (Optional) Whether the API client can be used for requests internal to IdentityNow. Cannot be updated, changing it forces a new client to be created.
* `scope` - (Optional) Scopes of the API client. Defaults to `sp:scopes:all`, which grants all rights of the identity that creates the client. Cannot be updated, changing it forces a new client to be created.

All other arguments are updated in place.

## Attributes Reference

* `id` - OAuth client ID, used as the client ID in OAuth flows.
* `secret` - (Sensitive) Client secret. Only available for clients created by Terraform.
* `created` - Creation date.
* `modified` - Last modification date.

## Import

OAuth clients can be imported using their ID. The secret cannot be imported:

```shell
terraform import identitynow_oauth_client.example <client-id>
```
