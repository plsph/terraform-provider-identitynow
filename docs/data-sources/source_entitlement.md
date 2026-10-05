---
subcategory: "Source"
page_title: "IdentityNow: Data Source: identitynow_source_entitlement"
description: |-
  Gets information about entitlements of an existing Source.
---

# Data Source: identitynow_source_entitlement

Use this data source to look up the entitlements of a source by name. Entitlements are only available after the source has been aggregated.

All entitlements of the source with the given name are returned in the `entitlements` list. The list is empty when no entitlement matches.

## Example Usage

```hcl
data "identitynow_source" "active_directory" {
  name = "Active Directory"
}

data "identitynow_source_entitlement" "developers" {
  source_id = data.identitynow_source.active_directory.id
  name      = "Developers"
}

output "developers_entitlement_id" {
  value = data.identitynow_source_entitlement.developers.entitlements[0].id
}

output "all_matching_entitlement_ids" {
  value = data.identitynow_source_entitlement.developers.entitlements[*].id
}
```

## Arguments Reference

The following arguments are supported:

* `source_id` - (Required) ID of the source.

* `name` - (Required) Name of the entitlement.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `entitlements` - List of the matching entitlements. Each element contains:
  * `id` - Entitlement ID.
  * `name` - Entitlement name.
  * `description` - Entitlement description.
  * `attribute` - Name of the account attribute the entitlement is derived from, e.g. `memberOf`.
  * `value` - Value of the entitlement, e.g. the group distinguished name.
  * `source_schema_object_type` - Schema object type of the entitlement, e.g. `group`.
  * `privileged` - Whether the entitlement is privileged.
  * `requestable` - Whether the entitlement is requestable.
  * `created` - The date and time the entitlement was created.
  * `modified` - The date and time the entitlement was last modified.
  * `owner` - List with the owner of the entitlement, null when the entitlement has no owner. Each element contains `id`, `type` and `name`.
  * `direct_permissions` - Direct permissions of the entitlement, each formatted as a string.
