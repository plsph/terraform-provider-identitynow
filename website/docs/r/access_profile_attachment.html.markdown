---
subcategory: "Access Profile"
layout: "identitynow"
page_title: "IdentityNow: identitynow_access_profile_attachment"
description: |-
  Manages an IdentityNow Source App's Access Profiles attachment.
---

# identitynow_access_profile_attachment

Manages an IdentityNow Source App's Access Profiles attachment.
Access Profile attached to Source App cannot be deleted, it must be detached first.

## Example Usage

```hcl
resource "identitynow_access_profile_attachment" "example" {
  source_app_id = identitynow_source_app.example.id
  access_profiles = [ "example_id0", "example_id1" ]
}
```

## Arguments Reference

The following arguments are supported:

* `source_app_id` - Id of source app.
* `access_profiles`- List of access profiles attached to source app.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/resource/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the Source App.
* `read` - (Defaults to 5 minutes) Used when retrieving the Source App.
* `update` - (Defaults to 30 minutes) Used when updating the Source App.
* `delete` - (Defaults to 30 minutes) Used when deleting the Source App.
