---
subcategory: "Tenant Settings"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_org_config"
description: |-
  Gets the IdentityNow org configuration.
---

# Data Source: identitynow_org_config

Use this data source to read the tenant-wide organization configuration, e.g. the time zone. To change the settings use the [identitynow_org_config](../r/org_config.html) resource.

## Example Usage

```hcl
data "identitynow_org_config" "current" {}

output "org_time_zone" {
  value = data.identitynow_org_config.current.time_zone
}
```

## Arguments Reference

This data source has no arguments.

## Attributes Reference

* `id` - Always `org-config`.
* `org_name` - Name of the org.
* `time_zone` - Time zone of the org, e.g. `Europe/Warsaw`. It determines when scheduled tasks run. Valid values are returned by the `identitynow_valid_time_zones` data source.
* `lcs_change_honors_source_enable_feature` - Whether the LCS_CHANGE_HONORS_SOURCE_ENABLE_FEATURE flag is enabled.
* `iai_enable_certification_recommendations` - Whether AI certification recommendations are enabled.
* `arm_customer_id` - Access Risk Management (ARM) customer ID.
* `arm_sap_system_id_mappings` - ARM mappings of IdentityNow source IDs to ARM system IDs, as returned by the API.
* `arm_auth` - ARM authentication string. The value is sensitive.
* `arm_db` - ARM database name.
* `arm_sso_url` - ARM SSO URL.
* `sod_report_configs_json` - Separation of duties report columns, objects with `columnName`, `required`, `included` and `order`. Required columns cannot be changed. A JSON array.
