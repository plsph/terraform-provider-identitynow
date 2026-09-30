---
subcategory: "Non-Employee"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_non_employee_schema_attribute"
description: |-
  Gets information about a schema attribute of an IdentityNow non-employee source.
---

# Data Source: identitynow_non_employee_schema_attribute

Use this data source to look up a schema attribute of a non-employee source by ID or technical name. Mandatory system attributes, e.g. `firstName`, can be looked up too.

## Example Usage

```hcl
data "identitynow_non_employee_schema_attribute" "first_name" {
  non_employee_source_id = "ef38f94347e94562b5bb8424a56397d8"
  technical_name         = "firstName"
}
```

## Arguments Reference

* `non_employee_source_id` - (Required) ID of the non-employee source.

Exactly one of the following must be set:

* `id` - (Optional) Schema attribute ID.
* `technical_name` - (Optional) Technical name of the attribute.

## Attributes Reference

* `type` - Attribute type: `TEXT`, `DATE` or `IDENTITY`.
* `label` - Label displayed in the UI.
* `help_text` - Help text displayed in the UI.
* `placeholder` - Hint text shown in the empty input field.
* `required` - Whether the attribute is required for all non-employees.
* `system` - Whether this is a mandatory system attribute.
* `created` - Creation date.
* `modified` - Last modification date.
