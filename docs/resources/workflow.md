---
subcategory: "Workflow"
page_title: "IdentityNow: identitynow_workflow"
description: |-
  Manages an IdentityNow Workflow.
---

# identitynow_workflow

Manages an IdentityNow Workflow.

Workflows can be created with `enabled = true`: the workflow is created disabled and enabled afterwards. When an enabled workflow is destroyed, it is disabled first and then deleted.

## Example Usage

### Basic Workflow with Event Trigger

```hcl
resource "identitynow_workflow" "email_on_manager_change" {
  name        = "Send Email on Manager Change"
  description = "Send an email to the identity when their manager attribute changes."
  enabled     = false

  owner {
    id   = "2c91808568c529c60168cca6f90c1313"
    type = "IDENTITY"
    name = "William Wilson"
  }

  trigger {
    type = "EVENT"
    attributes_json = jsonencode({
      id         = "idn:identity-attributes-changed"
      "filter.$" = "$.changes[?(@.attribute == 'manager')]"
    })
  }

  definition {
    start = "Send Email"
    steps_json = jsonencode({
      "Send Email" = {
        actionId = "sp:send-email"
        attributes = {
          body             = "This is a test"
          from             = "sailpoint@sailpoint.com"
          "recipientId.$"  = "$.identity.id"
          subject          = "test"
        }
        nextStep     = "success"
        selectResult = null
        type         = "ACTION"
      }
      "success" = {
        type = "success"
      }
    })
  }
}
```

### Scheduled Workflow

```hcl
resource "identitynow_workflow" "scheduled_report" {
  name        = "Weekly Compliance Report"
  description = "Generate a weekly compliance report."
  enabled     = false

  owner {
    id   = "2c91808568c529c60168cca6f90c1313"
    type = "IDENTITY"
    name = "William Wilson"
  }

  trigger {
    type = "SCHEDULED"
    attributes_json = jsonencode({
      cronString = "0 0 9 ? * MON"
    })
  }

  definition {
    start = "Generate Report"
    steps_json = jsonencode({
      "Generate Report" = {
        actionId = "sp:generate-report"
        attributes = {
          reportType = "compliance"
        }
        nextStep = "success"
        type     = "ACTION"
      }
      "success" = {
        type = "success"
      }
    })
  }
}
```

### External Trigger Workflow

```hcl
resource "identitynow_workflow" "external_trigger" {
  name        = "External Trigger Workflow"
  description = "A workflow triggered externally via API."
  enabled     = false

  owner {
    id   = "2c91808568c529c60168cca6f90c1313"
    type = "IDENTITY"
    name = "William Wilson"
  }

  trigger {
    type = "EXTERNAL"
  }

  definition {
    start = "Process Request"
    steps_json = jsonencode({
      "Process Request" = {
        actionId = "sp:send-email"
        attributes = {
          body    = "External trigger received"
          from    = "sailpoint@sailpoint.com"
          subject = "External Trigger"
        }
        nextStep = "success"
        type     = "ACTION"
      }
      "success" = {
        type = "success"
      }
    })
  }
}
```

## Arguments Reference

The following arguments are supported:

As per developer guide: (https://developer.sailpoint.com/docs/api/v2025/create-workflow)

* `name` - (Required) The name of the workflow.

* `description` - (Optional) Description of what the workflow accomplishes.

* `enabled` - (Optional) Enable or disable the workflow. If not set, the workflow is created disabled and the value returned by the API is used afterwards.

* `owner` - (Optional) Owner of the workflow. The IdentityNow API requires an owner. Contains:
  * `id` - (Required) Owner identity ID.
  * `type` - (Required) Owner type (e.g. `IDENTITY`).
  * `name` - (Required) Owner name.

* `trigger` - (Optional) Trigger configuration for the workflow. Contains:
  * `type` - (Required) Trigger type. One of `EVENT`, `SCHEDULED`, or `EXTERNAL`.
  * `display_name` - (Optional) Display name for the trigger.
  * `attributes_json` - (Optional) Trigger attributes as a JSON object string. Use `jsonencode()` for convenience.

* `definition` - (Optional) Workflow definition. Contains:
  * `start` - (Required) The name of the starting step. Must be the name of a step in `steps_json`.
  * `steps_json` - (Required) Workflow steps as a JSON object string, keyed by step name. Use `jsonencode()` for convenience.

`steps_json` and `attributes_json` must be JSON objects. They are compared semantically, so differences in whitespace or key order don't produce a diff.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Workflow ID (UUID).

## Import

Workflows can be imported using their ID:

```shell
terraform import identitynow_workflow.example <workflow-id>
```
