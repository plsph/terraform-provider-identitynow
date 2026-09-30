---
subcategory: "Tenant Settings"
layout: "identitynow"
page_title: "IdentityNow: identitynow_public_identities_config"
description: |-
  Manages the IdentityNow public identities configuration.
---

# identitynow_public_identities_config

Manages the identity attributes that are publicly visible to access request approvers and certification reviewers.

There is one configuration per tenant. Creating the resource replaces the current configuration, and destroying it only removes it from Terraform state, the configuration is left unchanged in IdentityNow.

## Example Usage

```hcl
resource "identitynow_public_identities_config" "this" {
  attribute {
    key  = "country"
    name = "Country"
  }

  attribute {
    key  = "department"
    name = "Department"
  }
}
```

## Arguments Reference

* `attribute` - (Optional) Identity attribute that is publicly visible. Up to 5 blocks can be configured. Without blocks no attributes are public. Contains:
  * `key` - (Required) Identity attribute key.
  * `name` - (Required) Identity attribute display name.

## Attributes Reference

* `id` - Always `public-identities-config`.
* `modified` - Last modification date.

## Import

The configuration can be imported with any ID:

```shell
terraform import identitynow_public_identities_config.this public-identities-config
```
