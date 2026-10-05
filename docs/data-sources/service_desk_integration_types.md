---
subcategory: "Service Desk Integration"
page_title: "IdentityNow: Data Source: identitynow_service_desk_integration_types"
description: |-
  Lists the supported IdentityNow service desk integration types.
---

# Data Source: identitynow_service_desk_integration_types

Use this data source to list the supported service desk integration types, e.g. to validate the `type` of an [identitynow_service_desk_integration](../resources/service_desk_integration).

## Example Usage

```hcl
data "identitynow_service_desk_integration_types" "all" {}

output "service_desk_integration_types" {
  value = data.identitynow_service_desk_integration_types.all.types[*].type
}
```

## Arguments Reference

This data source has no arguments.

## Attributes Reference

* `id` - Fixed identifier of the data source, `service-desk-integration-types`.
* `types` - Supported service desk integration types. Each item contains:
    * `name` - Display name of the type.
    * `type` - Type value, used as `type` of the integration.
    * `script_name` - Script name of the integration template of the type.
