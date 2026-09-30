---
subcategory: "Tenant Settings"
layout: "identitynow"
page_title: "IdentityNow: identitynow_session_config"
description: |-
  Manages the IdentityNow user session configuration.
---

# identitynow_session_config

Manages the tenant-wide idle and maximum session times of users.

There is one session configuration per tenant, it cannot be created or deleted. Creating the resource applies the configured settings to the existing session configuration, and destroying it only removes it from Terraform state: the settings are left unchanged in IdentityNow.

Only the settings present in the configuration are managed. Settings that are not configured are never changed or reset; they show the current tenant value. Removing a setting from the configuration stops managing it and leaves its current value in place. Updates are sent as JSON Patch operations for the configured settings that changed.

## Example Usage

```hcl
resource "identitynow_session_config" "this" {
  max_idle_time    = 15
  max_session_time = 480
  remember_me      = false
}
```

## Arguments Reference

All arguments are optional; settings that are not configured keep their current value.

* `max_idle_time` - (Optional) Maximum time in minutes a session can be idle.
* `max_session_time` - (Optional) Maximum session time in minutes.
* `remember_me` - (Optional) Whether 'remember me' is enabled on the login page.

## Attributes Reference

In addition to the arguments, the following attributes are exported; arguments that are not configured show the current tenant value.

* `id` - Always `session-config`.

## Import

The settings can be imported with any ID; the ID is always set to `session-config`:

```shell
terraform import identitynow_session_config.this session-config
```
