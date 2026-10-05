---
page_title: "Provider: IdentityNow"
description: |-
  The IdentityNow Provider is used to interact with the many resources supported by [IdentityNow API](https://developer.sailpoint.com/idn/api/v3).

---

# IdentityNow Provider

The IdentityNow Provider can be used to configure infrastructure in [SailPoint IdentityNow](https://www.sailpoint.com/products/identitynow/) using the official API's. Documentation regarding the [Data Sources](https://developer.hashicorp.com/terraform/language/data-sources) and [Resources](https://developer.hashicorp.com/terraform/language/resources) supported by the IdentityNow Provider can be found in the navigation to the left.

Interested in the provider's latest features, or want to make sure you're up to date? Check out the [changelog](https://github.com/plsph/terraform-provider-identitynow/blob/master/CHANGELOG.md) for version information and release notes.

## Authenticating to IdentityNow

The IdentityNow Provider follows the [Client Credentials Grant Flow](https://developer.sailpoint.com/idn/api/authentication/#client-credentials-grant-flow), using the Client ID and Client Secret obtained from the personal access token.

Credentials are required and can be provided in one of the following ways:

* `client_id` and `client_secret` in the provider configuration,
* one or more client id and secret pairs in the `credentials` list,
* the `IDENTITYNOW_CLIENT_ID` and `IDENTITYNOW_CLIENT_SECRET` environment variables.

`credentials` takes precedence over `client_id` and `client_secret`, which may be deprecated in the future.

Due to the SailPoint API rate limit consider using multiple API clients in `credentials` to speed up Terraform execution. The provider creates a pool of up to `max_client_pool_size` API clients (`default_client_pool_size` is used as the pool size, capped at `max_client_pool_size`), uses them in turn and assigns the configured credentials to them round-robin. Each client is limited to `client_request_rate_limit` requests per second. To use all configured credentials, set the pool sizes to at least the number of `credentials`.

Requests that fail with HTTP 429 (Too Many Requests) are retried, honouring the `Retry-After` header. `GET`, `PUT` and `DELETE` requests are also retried on HTTP 502, 503 and 504. When the API responds with HTTP 401 the access token is refreshed and the request is retried once.

## Example Usage

```hcl
# We strongly recommend using the required_providers block to set the
# IdentityNow Provider source and version being used
terraform {
  required_providers {
    identitynow = {
      source  = "plsph/identitynow"
      version = "~> 0.17"
    }
  }
}

# Configure the IdentityNow Provider
provider "identitynow" {
  api_url                   = "https://<org_name>.api.identitynow.com"
  client_id                 = "<client_id>"
  client_secret             = "<client_secret>"
  max_client_pool_size      = 1
  default_client_pool_size  = 1
  client_request_rate_limit = 10
}

# An additional provider configuration using a pool of two API clients
provider "identitynow" {
  alias   = "pool"
  api_url = "https://<org_name>.api.identitynow.com"
  credentials = [
    { client_id = "<client_id1>", client_secret = "<client_secret1>" },
    { client_id = "<client_id2>", client_secret = "<client_secret2>" },
  ]
  max_client_pool_size      = 2
  default_client_pool_size  = 2
  client_request_rate_limit = 10
}

# Create a governance group
resource "identitynow_governance_group" "example" {
  name        = "Example Approvers"
  description = "Example governance group"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    name = "John Doe"
    type = "IDENTITY"
  }
}

# Create a governance group with the aliased provider configuration
resource "identitynow_governance_group" "pooled" {
  provider    = identitynow.pool
  name        = "Example Reviewers"
  description = "Example governance group created with the client pool"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    name = "John Doe"
    type = "IDENTITY"
  }
}
```

The configuration can also be provided entirely with environment variables, in which case the provider block can be left empty (`provider "identitynow" {}`):

```shell
export IDENTITYNOW_URL="https://<org_name>.api.identitynow.com"
export IDENTITYNOW_CLIENT_ID="<client_id>"
export IDENTITYNOW_CLIENT_SECRET="<client_secret>"
```

## Argument Reference

The following arguments are supported:

* `api_url` - (Optional) The URL to the IdentityNow API, e.g. `https://<org_name>.api.identitynow.com`. If no scheme is given, `https://` is prepended. Can also be set with the `IDENTITYNOW_URL` environment variable. Must be set in the configuration or in the environment.

* `client_id` - (Optional) API client ID used to authenticate with the IdentityNow API. Can also be set with the `IDENTITYNOW_CLIENT_ID` environment variable.

* `client_secret` - (Optional) API client secret used to authenticate with the IdentityNow API. Can also be set with the `IDENTITYNOW_CLIENT_SECRET` environment variable.

* `credentials` - (Optional) List of API client id and secret pairs used to authenticate with the IdentityNow API. Takes precedence over `client_id` and `client_secret`. Each element contains:
  * `client_id` - (Required) API client ID.
  * `client_secret` - (Required) API client secret.

* `max_client_pool_size` - (Optional) Maximum number of API clients in the pool used for communication with the IdentityNow API. Must be at least 1. Can also be set with the `IDENTITYNOW_MAX_POOL_SIZE` environment variable. Defaults to `1`.

* `default_client_pool_size` - (Optional) Number of API clients in the pool, capped at `max_client_pool_size`. Must be at least 1. Can also be set with the `IDENTITYNOW_DEF_POOL_SIZE` environment variable. Defaults to `1`.

* `client_request_rate_limit` - (Optional) Request rate limit in requests per second for each API client. Must be at least 1. Can also be set with the `IDENTITYNOW_CLI_RQ_RATE` environment variable. Defaults to `10`.

Values set in the provider configuration take precedence over environment variables.

## Origin

This is fork of (https://github.com/OpenAxon/terraform-provider-identitynow/) which looked abandoned to me.

## Features and Bug Requests

The IdentityNow provider's bugs and feature requests can be found in the [GitHub repo issues](https://github.com/plsph/terraform-provider-identitynow/issues).
Please avoid "me too" or "+1" comments. Instead, use a thumbs up [reaction](https://blog.github.com/2016-03-10-add-reactions-to-pull-requests-issues-and-comments/)
on enhancement requests. Provider maintainers will often prioritize work based on the number of thumbs on an issue.

Community input is appreciated on outstanding issues! We love to hear what use
cases you have for new features, and want to provide the best possible
experience for you using the IdentityNow provider.

If you have a bug or feature request without an existing issue

* if an existing resource or field is working in an unexpected way, [file a bug](https://github.com/plsph/terraform-provider-identitynow/issues/new?template=Bug_Report.yml).

* if you'd like the provider to support a new resource or field, [file an enhancement/feature request](https://github.com/plsph/terraform-provider-identitynow/issues/new?template=Feature_Request.yml).

The provider maintainers will often use the assignee field on an issue to mark
who is working on it.

* An issue assigned to an individual maintainer indicates that the maintainer is working
on the issue

* If you're interested in working on an issue please leave a comment on that issue
