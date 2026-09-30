---
subcategory: "Service Desk Integration"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_service_desk_integration"
description: |-
  Gets information about an existing IdentityNow service desk integration.
---

# Data Source: identitynow_service_desk_integration

Use this data source to look up a service desk integration by ID or name.

## Example Usage

```hcl
data "identitynow_service_desk_integration" "servicenow" {
  name = "ServiceNow"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) Service desk integration ID.
* `name` - (Optional) Service desk integration name.

## Attributes Reference

* `description` - Description of the integration.
* `type` - Service desk integration type.
* `managed_sources` - IDs of the sources managed by the integration.
* `provisioning_config_json` - Provisioning configuration as a JSON object.
* `attributes_json` - Integration attributes as a JSON object, as returned by the API. The value is sensitive.
* `owner_ref` - Identity that owns the integration, a list with at most one item with `id`, `type` and `name`.
* `cluster_ref` - Virtual appliance cluster of the integration, a list with at most one item with `id`, `type` and `name`.
* `before_provisioning_rule` - Before provisioning rule, a list with at most one item with `id`, `type` and `name`.
* `created` - Creation date.
* `modified` - Last modification date.
