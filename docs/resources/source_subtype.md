---
subcategory: "Source"
page_title: "IdentityNow: identitynow_source_subtype"
description: |-
  Manages a machine account subtype of an IdentityNow source.
---

# identitynow_source_subtype

Manages a machine account subtype of a source, e.g. service accounts. The resource uses the experimental `/source-subtypes` API, which addresses subtypes by their ID and supports create, read, update and delete. The provider sends the required `X-SailPoint-Experimental` header.

Only `display_name` and `description` can be updated. Deleting a subtype also deletes its approval settings and the entitlement for machine account creation.

## Example Usage

```hcl
resource "identitynow_source_subtype" "service_account" {
  source_id      = "2c9180835d191a86015d28455b4a2329"
  technical_name = "service_account"
  display_name   = "Service Account"
  description    = "Accounts used by applications"
}
```

## Arguments Reference

* `source_id` - (Required) ID of the source the subtype belongs to. Changing this forces a new subtype to be created.
* `technical_name` - (Required) Technical name of the subtype. Changing this forces a new subtype to be created.
* `display_name` - (Required) Display name of the subtype.
* `description` - (Required) Description of the subtype.
* `type` - (Optional) Type of the subtype, `MACHINE` or unset. When not set, the value returned by IdentityNow is kept. Changing this forces a new subtype to be created.

## Attributes Reference

* `id` - Subtype ID.
* `system_managed` - Whether the subtype is managed by the system.
* `created` - Creation date.
* `modified` - Last modification date.

## Import

Source subtypes can be imported using their ID:

```shell
terraform import identitynow_source_subtype.example <subtype-id>
```
