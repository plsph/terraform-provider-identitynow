---
subcategory: "Custom Forms"
layout: "identitynow"
page_title: "IdentityNow: identitynow_form_definition"
description: |-
  Manages an IdentityNow custom form definition.
---

# identitynow_form_definition

Manages an IdentityNow custom form definition.

## Example Usage

```hcl
resource "identitynow_form_definition" "access_request" {
  name        = "Access Request Justification"
  description = "Collects a business justification for an access request."

  owner {
    id   = "2c91808568c529c60168cca6f90c1313"
    type = "IDENTITY"
  }

  form_input {
    type        = "STRING"
    label       = "requestedFor"
    description = "Identity the access is requested for"
  }

  form_elements_json = jsonencode([
    {
      id          = "section1"
      elementType = "SECTION"
      config = {
        alignment  = "LEFT"
        label      = "Justification"
        labelStyle = "h2"
        showLabel  = true
        formElements = [
          {
            id          = "justification"
            key         = "justification"
            elementType = "TEXTAREA"
            config = {
              label    = "Business justification"
              helpText = "Explain why the access is needed"
              required = true
            }
            validations = [
              { validationType = "REQUIRED" }
            ]
          },
          {
            id          = "urgent"
            key         = "urgent"
            elementType = "TOGGLE"
            config = {
              label = "Urgent request"
            }
            validations = []
          }
        ]
      }
    }
  ])

  form_conditions_json = jsonencode([
    {
      ruleOperator = "AND"
      rules = [
        {
          sourceType = "ELEMENT"
          source     = "urgent"
          operator   = "EQ"
          valueType  = "BOOLEAN"
          value      = "true"
        }
      ]
      effects = [
        {
          effectType = "REQUIRE"
          config = {
            element = "justification"
          }
        }
      ]
    }
  ])
}
```

## Arguments Reference

The following arguments are supported:

As per developer guide: (https://developer.sailpoint.com/docs/api/v2026/create-form-definition)

* `name` - (Required) The name of the form definition.

* `description` - (Optional) The description of the form definition.

* `owner` - (Required) Owner of the form definition. Exactly one block is required. Contains:
  * `id` - (Required) Owner identity ID.
  * `type` - (Required) Owner type. Must be `IDENTITY`.
  * `name` - (Optional) Owner name. If omitted, the name returned by the API is not tracked.

* `form_input` - (Optional) Form inputs that must be provided when creating a form instance. Can be repeated. Contains:
  * `type` - (Required) Form input type. One of `STRING` or `ARRAY`.
  * `label` - (Optional) Form input name.
  * `description` - (Optional) Form input description.

* `form_elements_json` - (Optional) List of nested form elements as a JSON array. Root elements must have `elementType` `SECTION`, and child elements go inside the section's `config.formElements`. Use `jsonencode()` for convenience.

* `form_conditions_json` - (Optional) Conditional logic that dynamically modifies the form as the recipient interacts with it, as a JSON array. Use `jsonencode()` for convenience.

JSON arguments are compared semantically, so differences in whitespace or key order do not produce a diff.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Form definition ID.
* `form_input.*.id` - Form input identifier, assigned by the API.
* `used_by` - Systems currently using the form definition. Each item contains `id`, `type` (`WORKFLOW`, `SOURCE` or `MySailPoint`) and `name`.
* `created` - The date and time the form definition was created.
* `modified` - The date and time the form definition was last modified.

## Import

Form definitions can be imported using their ID:

```shell
terraform import identitynow_form_definition.example <form-definition-id>
```
