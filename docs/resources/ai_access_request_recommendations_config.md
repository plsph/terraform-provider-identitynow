---
subcategory: "Tenant Settings"
page_title: "IdentityNow: identitynow_ai_access_request_recommendations_config"
description: |-
  Manages the IdentityNow AI access request recommendations configuration.
---

# identitynow_ai_access_request_recommendations_config

Manages the tenant-wide configuration of AI access request recommendations.

There is one access request recommendations configuration per tenant, it cannot be created or deleted. Creating the resource applies the configured settings to the existing access request recommendations configuration, and destroying it only removes it from Terraform state: the settings are left unchanged in IdentityNow.

Only the settings present in the configuration are managed. Settings that are not configured are never changed or reset; they show the current tenant value. Removing a setting from the configuration stops managing it and leaves its current value in place. The API replaces the whole access request recommendations configuration on update, so the provider reads the current access request recommendations configuration, replaces the configured settings and sends it back. The API is experimental; the provider sends the `X-SailPoint-Experimental` header.

## Example Usage

```hcl
resource "identitynow_ai_access_request_recommendations_config" "this" {
  score_threshold           = 0.5
  restriction_attribute     = "location"
  use_restriction_attribute = true
}
```

## Arguments Reference

All arguments are optional; settings that are not configured keep their current value.

* `score_threshold` - (Optional) Value the internal calculations must exceed to recommend access.
* `start_date_attribute` - (Optional) Identity attribute with the start date of identities.
* `restriction_attribute` - (Optional) Identity attribute recommendations are restricted to.
* `use_restriction_attribute` - (Optional) Whether only `restriction_attribute` is used to make recommendations.
* `mover_attribute` - (Optional) Identity attribute that tells whether an identity is a mover.
* `joiner_attribute` - (Optional) Identity attribute that tells whether an identity is a joiner.

## Attributes Reference

In addition to the arguments, the following attributes are exported; arguments that are not configured show the current tenant value.

* `id` - Always `ai-access-request-recommendations-config`.

## Import

The settings can be imported with any ID; the ID is always set to `ai-access-request-recommendations-config`:

```shell
terraform import identitynow_ai_access_request_recommendations_config.this ai-access-request-recommendations-config
```
