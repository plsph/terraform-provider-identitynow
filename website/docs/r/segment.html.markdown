---
subcategory: "Segment"
layout: "identitynow"
page_title: "IdentityNow: identitynow_segment"
description: |-
  Manages an IdentityNow segment.
---

# identitynow_segment

Manages an IdentityNow segment. Segment changes can take time to propagate to all identities.

The visibility criteria can be configured either with nested `visibility_criteria` blocks or as JSON with `visibility_criteria_json`, but not both. Removing the criteria from the configuration removes the visibility restriction of the segment.

## Example Usage

### Visibility Criteria as Blocks

```hcl
resource "identitynow_segment" "austin" {
  name        = "Austin employees"
  description = "Employees whose location is Austin"
  active      = true

  owner {
    id   = "2c9180a46faadee4016fb4e018c20639"
    type = "IDENTITY"
    name = "support"
  }

  visibility_criteria {
    expression {
      operator  = "EQUALS"
      attribute = "location"

      value {
        type  = "STRING"
        value = "Austin"
      }
    }
  }
}
```

### Nested Visibility Criteria

```hcl
resource "identitynow_segment" "austin_engineering" {
  name   = "Austin engineering"
  active = true

  visibility_criteria {
    expression {
      operator = "AND"

      children {
        expression {
          operator  = "EQUALS"
          attribute = "location"

          value {
            type  = "STRING"
            value = "Austin"
          }
        }
      }

      children {
        expression {
          operator  = "EQUALS"
          attribute = "department"

          value {
            type  = "STRING"
            value = "Engineering"
          }
        }
      }
    }
  }
}
```

### Visibility Criteria as JSON

```hcl
resource "identitynow_segment" "austin_json" {
  name   = "Austin employees (JSON)"
  active = true

  visibility_criteria_json = jsonencode({
    expression = {
      operator  = "EQUALS"
      attribute = "location"
      value = {
        type  = "STRING"
        value = "Austin"
      }
    }
  })
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) The segment business name.

* `description` - (Optional) The segment description.

* `active` - (Optional) Whether the segment is active. Inactive segments have no effect. Defaults to `false`.

* `owner` - (Optional) The segment owner. Contains:
  * `id` - (Required) Owner identity ID.
  * `type` - (Required) Owner type, `IDENTITY`.
  * `name` - (Required) Owner name.

* `visibility_criteria` - (Optional) Visibility criteria following the SailPoint Visibility Criteria schema. Conflicts with `visibility_criteria_json`. Contains:
  * `expression` - (Optional) Expression block. Contains:
    * `operator` - (Optional) Operator, e.g. `EQUALS`, `AND` or `OR`.
    * `attribute` - (Optional) Identity attribute to compare, for comparison operators.
    * `value` - (Optional) Value to compare with. Contains `type` (e.g. `STRING`) and `value`, both optional.
    * `children` - (Optional) Child criteria for `AND` and `OR` operators. Each `children` block contains an `expression` block with the same arguments. Supports up to 3 levels of nesting.

* `visibility_criteria_json` - (Optional) Visibility criteria as a JSON object following the SailPoint Visibility Criteria schema. Must be a JSON object. Conflicts with `visibility_criteria`. The value is compared semantically, so differences in whitespace, key order or unset fields don't produce a diff, and it is refreshed from the API, so changes made outside Terraform are detected.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - The segment ID.
* `created` - The segment creation timestamp.
* `modified` - The segment modification timestamp.

## Import

Segments can be imported using their ID:

```shell
terraform import identitynow_segment.austin <segment-id>
```
