---
subcategory: "Tenant Settings"
layout: "identitynow"
page_title: "IdentityNow: identitynow_role_propagation_config"
description: |-
  Manages the IdentityNow role change propagation configuration.
---

# identitynow_role_propagation_config

Manages whether role change propagation is enabled for the tenant.

There is one role propagation configuration per tenant, it cannot be created or deleted. Creating the resource applies the configured settings to the existing role propagation configuration, and destroying it only removes it from Terraform state: the settings are left unchanged in IdentityNow.

Only the settings present in the configuration are managed. Settings that are not configured are never changed or reset; they show the current tenant value. Removing a setting from the configuration stops managing it and leaves its current value in place. The API replaces the whole role propagation configuration on update, so the provider sends the configured settings together with the current values of the other settings. The API is experimental; the provider sends the `X-SailPoint-Experimental` header.

## Example Usage

```hcl
resource "identitynow_role_propagation_config" "this" {
  enabled = true
}
```

## Arguments Reference

All arguments are optional; settings that are not configured keep their current value.

* `enabled` - (Optional) Whether the role change propagation process is enabled.

## Attributes Reference

In addition to the arguments, the following attributes are exported; arguments that are not configured show the current tenant value.

* `id` - Always `role-propagation-config`.
* `enabled_date` - Time when role change propagation was last enabled.
* `created_date` - Time when the configuration was first created.
* `modified_date` - Time when the configuration was last updated.

## Import

The settings can be imported with any ID; the ID is always set to `role-propagation-config`:

```shell
terraform import identitynow_role_propagation_config.this role-propagation-config
```
