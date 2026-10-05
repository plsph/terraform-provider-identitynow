---
subcategory: "Non-Employee"
page_title: "IdentityNow: identitynow_non_employee_schema_attribute"
description: |-
  Manages a custom schema attribute of an IdentityNow non-employee source.
---

# identitynow_non_employee_schema_attribute

Manages a custom schema attribute of a non-employee source. Every non-employee source has 8 mandatory attributes, and up to 10 custom attributes can be added.

## Example Usage

```hcl
resource "identitynow_non_employee_source" "contractors" {
  name        = "Contractors"
  description = "External contractors"

  owner {
    id = "2c9180867624cbd7017642d8c8c81f67"
  }
}

resource "identitynow_non_employee_schema_attribute" "cost_center" {
  non_employee_source_id = identitynow_non_employee_source.contractors.id
  technical_name         = "costCenter"
  label                  = "Cost center"
  help_text              = "Cost center that pays the contractor"
  placeholder            = "CC-0000"
  required               = true
}
```

## Arguments Reference

* `non_employee_source_id` - (Required) ID of the non-employee source. Changing this forces a new attribute to be created.
* `technical_name` - (Required) Technical name of the attribute, unique per source. It cannot be changed, changing this forces a new attribute to be created.
* `label` - (Required) Label displayed in the UI.
* `type` - (Optional) Attribute type. Only `TEXT` is supported for custom attributes, which is the default. Changing this forces a new attribute to be created.
* `help_text` - (Optional) Help text displayed in the UI.
* `placeholder` - (Optional) Hint text shown in the empty input field.
* `required` - (Optional) Whether the attribute is required for all non-employees of the source. Defaults to `false`.

`label`, `help_text`, `placeholder` and `required` are changed with a JSON Patch of the changed fields.

## Attributes Reference

* `id` - Schema attribute ID.
* `system` - Whether this is a mandatory system attribute.
* `created` - Creation date.
* `modified` - Last modification date.

## Import

Schema attributes can be imported using the non-employee source ID and the attribute ID separated by `/`:

```shell
terraform import identitynow_non_employee_schema_attribute.example <non-employee-source-id>/<attribute-id>
```
