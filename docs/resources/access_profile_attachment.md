---
subcategory: "Access Profile"
page_title: "IdentityNow: identitynow_access_profile_attachment"
description: |-
  Manages the Access Profiles attached to an IdentityNow Source App.
---

# identitynow_access_profile_attachment

Manages the Access Profiles attached to an IdentityNow Source App.

The list of access profiles is authoritative: access profiles attached to the source app outside Terraform are detached on apply. Destroying the resource detaches only the access profiles managed by the resource.

An access profile attached to a source app can't be deleted, it must be detached first.

## Example Usage

```terraform
resource "identitynow_source_app" "example" {
  name        = "Active Directory Developers"
  description = "Application for requesting developer access"

  source {
    id   = "2c9180835d191a86015d28455b4a2329"
    name = "Active Directory"
  }
}

resource "identitynow_access_profile_attachment" "example" {
  source_app_id = identitynow_source_app.example.id
  access_profiles = [
    "2c91808a7813090a017813b6301f0044",
    "2c91808a7813090a017813b6301f0045",
  ]
}
```

## Arguments Reference

The following arguments are supported:

* `source_app_id` - (Required) ID of the source app. Changing this forces a new resource to be created.

* `access_profiles` - (Required) List of IDs of the access profiles attached to the source app.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Access profile attachment ID (same as `source_app_id`).

## Import

Access profile attachments can be imported using the source app ID:

```shell
terraform import identitynow_access_profile_attachment.example <source-app-id>
```
