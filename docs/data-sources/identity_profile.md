---
subcategory: "Identity Profile"
page_title: "IdentityNow: Data Source: identitynow_identity_profile"
description: |-
  Gets information about an existing IdentityNow identity profile.
---

# Data Source: identitynow_identity_profile

Use this data source to look up an identity profile by ID or name.

## Example Usage

```terraform
data "identitynow_identity_profile" "employees" {
  name = "Employees"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) Identity profile ID.
* `name` - (Optional) Identity profile name.

## Attributes Reference

* `description` - Description of the identity profile.
* `priority` - Priority of the identity profile.
* `owner` - Owner of the identity profile, a list with one element with `id`, `type` and `name`.
* `authoritative_source` - Authoritative source of the identity profile, a list with one element with `id`, `type` and `name`.
* `identity_attribute_config_enabled` - Whether the identity attribute mappings are enabled.
* `attribute_config_json` - Identity attribute mappings as a JSON array.
* `identity_refresh_required` - Whether an identity refresh is required for the profile.
* `identity_count` - Number of identities belonging to the profile.
* `has_time_based_attr` - Whether the profile has time based attributes.
* `identity_exception_report_task_result_id` - Task result ID of the last identity exception report.
* `identity_exception_report_name` - Name of the last identity exception report.
* `created` - Creation date.
* `modified` - Last modification date.
