---
subcategory: "Source"
page_title: "IdentityNow: Data Source: identitynow_source"
description: |-
  Gets information about an existing Source.
---

# Data Source: identitynow_source

Use this data source to access information about an existing Source.

## Example Usage

```terraform
data "identitynow_source" "example" {
  name = "Active Directory"
}

output "identitynow_source_description" {
  value = data.identitynow_source.example.description
}

output "identitynow_source_owner_name" {
  value = data.identitynow_source.example.owner[0].name
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) Name of the source.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Source ID.

* `description` - Source description.

* `connector` - Connector script name, e.g. `active-directory`.

* `authoritative` - Whether the source is authoritative.

* `delete_threshold` - Maximum percentage of accounts that can be deleted during an aggregation.

* `owner` - List with the owner of the source. Each element contains `id`, `type` and `name`.

* `cluster` - List with the virtual appliance cluster of the source, empty when the source has no cluster. Each element contains `id`, `type` and `name`.
