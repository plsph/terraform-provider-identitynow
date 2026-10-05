---
subcategory: "Search"
page_title: "IdentityNow: Data Source: identitynow_search_attribute_config"
description: |-
  Gets information about an existing IdentityNow extended account search attribute.
---

# Data Source: identitynow_search_attribute_config

Use this data source to look up an extended account search attribute by name.

~> **Note:** This data source uses an experimental IdentityNow API, which can change without notice.

## Example Usage

```terraform
data "identitynow_search_attribute_config" "employee_number" {
  name = "employeeNumber"
}
```

## Arguments Reference

* `name` - (Required) Name of the search attribute.

## Attributes Reference

* `id` - Configuration ID, the same as `name`.
* `display_name` - Display name of the search attribute.
* `application_attributes` - Map of source ID to the name of the promoted account attribute.
