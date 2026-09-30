---
subcategory: "Search"
layout: "identitynow"
page_title: "IdentityNow: identitynow_search_attribute_config"
description: |-
  Manages an IdentityNow extended account search attribute.
---

# identitynow_search_attribute_config

Manages an extended account search attribute. It promotes an account attribute of one or more sources to a searchable account attribute.

~> **Note:** This resource uses an experimental IdentityNow API, which can change without notice.

The API creates the configuration asynchronously. After the create request, the provider reads the configuration until it exists, for up to 60 seconds. If it does not become available in time, the apply fails and the resource is marked as tainted, so the next apply replaces it.

## Example Usage

```hcl
resource "identitynow_search_attribute_config" "employee_number" {
  name         = "employeeNumber"
  display_name = "Employee Number"
  application_attributes = {
    "2c9180835d191a86015d28455b4a2329" = "employeeID"
    "2c918083746f642c01746f990884012a" = "empNo"
  }
}
```

## Arguments Reference

* `name` - (Required) Name of the search attribute. Give it a unique name that is not used by an account or source attribute, or in the account schema of any current or future source.
* `display_name` - (Required) Display name of the search attribute.
* `application_attributes` - (Required) Map of source ID to the name of the account attribute promoted to the search attribute.

All arguments can be updated in place. Renaming changes the `id`.

## Attributes Reference

* `id` - Configuration ID, the same as `name`.

## Import

Search attribute configurations can be imported using their name:

```shell
terraform import identitynow_search_attribute_config.example employeeNumber
```
