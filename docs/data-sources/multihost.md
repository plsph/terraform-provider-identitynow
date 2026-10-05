---
subcategory: "Multi-Host Integration"
page_title: "IdentityNow: Data Source: identitynow_multihost"
description: |-
  Gets information about an existing IdentityNow Multi-Host Integration.
---

# Data Source: identitynow_multihost

Use this data source to look up a Multi-Host Integration by ID.

## Example Usage

```terraform
data "identitynow_multihost" "sql_servers" {
  id = "2c91808568c529c60168cca6f90c1324"
}
```

## Arguments Reference

* `id` - (Required) Multi-Host Integration ID.

## Attributes Reference

* `name` - Name of the Multi-Host Integration.
* `description` - Description of the Multi-Host Integration.
* `connector` - Connector script name.
* `connector_attributes_json` - All connector attributes as a JSON object. This value is sensitive, since it may contain credentials.
* `max_sources_per_agg_group` - Maximum number of sources per aggregation group.
* `max_allowed_sources` - Maximum number of sources in the Multi-Host Integration.
* `type` - Type of system managed.
* `owner` - Owner identity, a list with one element with `id` and `name`.
* `cluster` - Virtual appliance cluster, a list with at most one element with `id` and `name`.
* `management_workgroup` - Management governance group, a list with at most one element with `id` and `name`.
* `created` - Creation date.
* `modified` - Last modification date.
