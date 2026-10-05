---
subcategory: "Tenant Settings"
page_title: "IdentityNow: identitynow_campaign_reports_config"
description: |-
  Manages the IdentityNow certification campaign reports configuration.
---

# identitynow_campaign_reports_config

Manages the identity attributes that certification campaign reports show as custom columns.

There is one campaign reports configuration per tenant, it cannot be created or deleted. Creating the resource applies the configured settings to the existing campaign reports configuration, and destroying it only removes it from Terraform state: the settings are left unchanged in IdentityNow.

Only the settings present in the configuration are managed. Settings that are not configured are never changed or reset; they show the current tenant value. Removing a setting from the configuration stops managing it and leaves its current value in place. The API replaces the whole campaign reports configuration on update, so the provider reads the current campaign reports configuration, replaces the configured settings and sends it back.

## Example Usage

```terraform
resource "identitynow_campaign_reports_config" "this" {
  identity_attribute_columns = ["department", "location"]
}
```

## Arguments Reference

All arguments are optional; settings that are not configured keep their current value.

* `identity_attribute_columns` - (Optional) Identity attributes added as custom columns to campaign reports.

## Attributes Reference

In addition to the arguments, the following attributes are exported; arguments that are not configured show the current tenant value.

* `id` - Always `campaign-reports-config`.

## Import

The settings can be imported with any ID; the ID is always set to `campaign-reports-config`:

```shell
terraform import identitynow_campaign_reports_config.this campaign-reports-config
```
