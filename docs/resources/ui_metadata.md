---
subcategory: "Tenant Settings"
page_title: "IdentityNow: identitynow_ui_metadata"
description: |-
  Manages the IdentityNow tenant UI metadata.
---

# identitynow_ui_metadata

Manages the tenant-wide user interface metadata: the labels of the login page and the domains that may embed IdentityNow in an iframe. Changes can take up to 5 minutes to take effect.

There is one UI metadata per tenant, it cannot be created or deleted. Creating the resource applies the configured settings to the existing UI metadata, and destroying it only removes it from Terraform state: the settings are left unchanged in IdentityNow.

Only the settings present in the configuration are managed. Settings that are not configured are never changed or reset; they show the current tenant value. Removing a setting from the configuration stops managing it and leaves its current value in place. The API replaces the whole UI metadata on update, so the provider sends the configured settings together with the current values of the other settings. The API is experimental; the provider sends the `X-SailPoint-Experimental` header.

## Example Usage

```hcl
resource "identitynow_ui_metadata" "this" {
  username_label      = "Work email"
  username_empty_text = "Please provide your work email address"
}
```

## Arguments Reference

All arguments are optional; settings that are not configured keep their current value.

* `iframe_white_list` - (Optional) Space-separated domains that may embed the non-authenticated parts of IdentityNow (e.g. password reset) in an iframe.
* `username_label` - (Optional) Label of the username field on the login page.
* `username_empty_text` - (Optional) Placeholder text of the username field on the login page.

## Attributes Reference

In addition to the arguments, the following attributes are exported; arguments that are not configured show the current tenant value.

* `id` - Always `ui-metadata`.

## Import

The settings can be imported with any ID; the ID is always set to `ui-metadata`:

```shell
terraform import identitynow_ui_metadata.this ui-metadata
```
