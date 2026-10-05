---
subcategory: "Custom Forms"
page_title: "IdentityNow: Data Source: identitynow_form_definition"
description: |-
  Gets information about an existing IdentityNow custom form definition.
---

# Data Source: identitynow_form_definition

Use this data source to look up an existing IdentityNow custom form definition by name.

## Example Usage

```hcl
data "identitynow_form_definition" "access_request" {
  name = "Access Request Justification"
}

output "form_definition_id" {
  value = data.identitynow_form_definition.access_request.id
}
```

## Arguments Reference

* `name` - (Required) The name of the form definition.

## Attributes Reference

* `id` - The form definition ID.
* `description` - The form definition description.
* `owner` - The form definition owner. Contains `id`, `type` and `name`.
* `form_input` - Form inputs required when creating a form instance. Each item contains `id`, `type`, `label` and `description`.
* `form_elements_json` - The form elements as a JSON array.
* `form_conditions_json` - The form conditions as a JSON array.
* `used_by` - Systems currently using the form definition. Each item contains `id`, `type` and `name`.
* `created` - The date and time the form definition was created.
* `modified` - The date and time the form definition was last modified.
