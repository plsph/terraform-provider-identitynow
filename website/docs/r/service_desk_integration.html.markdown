---
subcategory: "Service Desk Integration"
layout: "identitynow"
page_title: "IdentityNow: identitynow_service_desk_integration"
description: |-
  Manages an IdentityNow service desk integration.
---

# identitynow_service_desk_integration

Manages a service desk integration (SDIM). A service desk integration sends the provisioning requests of its managed sources to a service desk such as ServiceNow, where they are fulfilled manually.

Updates replace the whole integration (PUT) with the configured values.

## Example Usage

```hcl
data "identitynow_identity" "john_doe" {
  alias = "A12BCDE3F"
}

resource "identitynow_service_desk_integration" "servicenow" {
  name            = "ServiceNow"
  description     = "Creates ServiceNow tickets for Active Directory provisioning"
  type            = "ServiceNowSDIM"
  managed_sources = ["<SOURCE_ID>"]

  provisioning_config_json = jsonencode({
    noProvisioningRequests        = false
    provisioningRequestExpiration = 7
  })

  attributes_json = jsonencode({
    url      = "https://<INSTANCE>.service-now.com"
    username = "<SERVICENOW_USER>"
    password = "<SERVICENOW_PASSWORD>"
  })

  owner_ref {
    id = data.identitynow_identity.john_doe.id
  }

  cluster_ref {
    id = "<CLUSTER_ID>"
  }

  before_provisioning_rule {
    id = "<RULE_ID>"
  }
}
```

## Arguments Reference

* `name` - (Required) Unique name of the integration.
* `description` - (Required) Description of the integration.
* `type` - (Required) Service desk integration type, e.g. `ServiceNowSDIM`. The [identitynow_service_desk_integration_types](../d/service_desk_integration_types.html) data source lists the supported types.
* `attributes_json` - (Required) Integration attributes as a JSON object, e.g. the service desk URL and credentials. Use `jsonencode()` for convenience. The value is sensitive and is not shown in plans. The API does not return secrets such as passwords: keys missing from the API response, and keys the API adds, do not cause a diff, while changed values of the other keys are detected as drift.
* `provisioning_config_json` - (Optional) Provisioning configuration as a JSON object, with keys such as `managedResourceRefs`, `planInitializerScript`, `noProvisioningRequests` and `provisioningRequestExpiration`. When not set, the value returned by the API is stored and sent back on updates, so removing the argument does not clear the configuration; set it to `"{}"` to clear it. Keys added by the API, such as the read-only `universalManager`, do not cause a diff; `universalManager` is never sent to the API.
* `managed_sources` - (Optional) IDs of the sources managed by the integration. The API deprecates this field in favor of `managedResourceRefs` in `provisioning_config_json`. When not set, the value returned by the API is stored.
* `owner_ref` - (Optional) Identity that owns the integration. At most one block:
    * `id` - (Required) Identity ID.
    * `type` - (Optional) Reference type. Defaults to `IDENTITY`.
    * `name` - (Optional) Identity name. Resolved by the API when not set.
* `cluster_ref` - (Optional) Virtual appliance cluster the integration uses. At most one block with `id` (Required), `type` (Optional, defaults to `CLUSTER`) and `name` (Optional, resolved by the API).
* `before_provisioning_rule` - (Optional) Before provisioning rule of the integration. At most one block with `id` (Required), `type` (Optional, defaults to `RULE`) and `name` (Optional, resolved by the API).

In the reference blocks, a configured `type` and `name` are kept in state as long as the referenced `id` does not change, so a name that differs from the display name returned by the API does not cause a diff.

## Attributes Reference

* `id` - Service desk integration ID.
* `created` - Creation date.
* `modified` - Last modification date.

## Import

Service desk integrations can be imported using their ID:

```shell
terraform import identitynow_service_desk_integration.example <integration-id>
```

After an import, `attributes_json` contains the attributes returned by the API, without secrets. Configure the complete attributes; the next apply sends them to the API.
