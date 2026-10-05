---
subcategory: "OAuth Client"
page_title: "IdentityNow: Data Source: identitynow_oauth_client"
description: |-
  Gets information about an existing IdentityNow OAuth (API) client.
---

# Data Source: identitynow_oauth_client

Use this data source to look up an OAuth (API) client by ID. The client secret is not available.

## Example Usage

```terraform
data "identitynow_oauth_client" "ci" {
  id = "2c9180835d2e5168015d32f890ca1581"
}
```

## Arguments Reference

* `id` - (Required) OAuth client ID.

## Attributes Reference

* `name` - Human-readable name of the API client.
* `description` - Description of the API client.
* `business_name` - Name of the business the API client belongs to.
* `homepage_url` - Homepage URL associated with the owner of the API client.
* `access_token_validity_seconds` - Number of seconds an access token is valid for.
* `refresh_token_validity_seconds` - Number of seconds a refresh token is valid for.
* `redirect_uris` - Approved redirect URIs.
* `grant_types` - OAuth 2.0 grant types the client can be used with.
* `access_type` - Access type, `ONLINE` or `OFFLINE`.
* `type` - Client type, `CONFIDENTIAL` or `PUBLIC`.
* `internal` - Whether the API client can be used for requests internal to IdentityNow.
* `enabled` - Whether the API client is enabled.
* `strong_auth_supported` - Whether the API client supports strong authentication.
* `claims_supported` - Whether the API client supports the serialization of SAML claims.
* `scope` - Scopes of the API client.
* `created` - Creation date.
* `modified` - Last modification date.
* `last_used` - Date the client was last used to generate an access token. It is updated once a day.
