---
subcategory: "Tenant Settings"
page_title: "IdentityNow: identitynow_reassignment_tenant_config"
description: |-
  Manages the IdentityNow tenant-wide reassignment configuration.
---

# identitynow_reassignment_tenant_config

Manages the tenant-wide work reassignment configuration, which enables or disables the reassignment configurations of identities.

There is one reassignment configuration per tenant, it cannot be created or deleted. Creating the resource applies the configured settings to the existing reassignment configuration, and destroying it only removes it from Terraform state: the settings are left unchanged in IdentityNow.

Only the settings present in the configuration are managed. Settings that are not configured are never changed or reset; they show the current tenant value. Removing a setting from the configuration stops managing it and leaves its current value in place. The API replaces the whole reassignment configuration on update, so the provider sends the configured settings together with the current values of the other settings. The API is experimental; the provider sends the `X-SailPoint-Experimental` header.

## Example Usage

```terraform
resource "identitynow_reassignment_tenant_config" "this" {
  disabled = false
}
```

## Arguments Reference

All arguments are optional; settings that are not configured keep their current value.

* `disabled` - (Optional) Whether reassignment configurations are disabled for the tenant.

## Attributes Reference

In addition to the arguments, the following attributes are exported; arguments that are not configured show the current tenant value.

* `id` - Always `reassignment-tenant-config`.
* `created` - Creation date of the configuration.
* `created_by_id` - ID of the identity that created the configuration.
* `created_by_name` - Name of the identity that created the configuration.
* `modified` - Last modification date of the configuration.
* `modified_by_id` - ID of the identity that last modified the configuration.
* `modified_by_name` - Name of the identity that last modified the configuration.

## Import

The settings can be imported with any ID; the ID is always set to `reassignment-tenant-config`:

```shell
terraform import identitynow_reassignment_tenant_config.this reassignment-tenant-config
```
