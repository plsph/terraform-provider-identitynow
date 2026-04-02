---
subcategory: "Workflow"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_workflow"
description: |-
  Gets information about an existing Workflow.
---

# Data Source: identitynow_workflow

Use this data source to access information about an existing Workflow.

## Example Usage

```hcl
data "identitynow_workflow" "example" {
  name = "Send Email on Manager Change"
}

output "workflow_id" {
  value = data.identitynow_workflow.example.id
}

output "workflow_enabled" {
  value = data.identitynow_workflow.example.enabled
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) The name of the workflow.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - The ID of the workflow.

* `description` - Description of the workflow.

* `enabled` - Whether the workflow is enabled.

* `owner` - Owner of the workflow. Contains:
  * `id` - Owner identity ID.
  * `type` - Owner type.
  * `name` - Owner name.

* `trigger` - Trigger configuration. Contains:
  * `type` - Trigger type (EVENT, SCHEDULED, or EXTERNAL).
  * `display_name` - Trigger display name.
  * `attributes_json` - Trigger attributes as a JSON string.

* `definition` - Workflow definition. Contains:
  * `start` - The name of the starting step.
  * `steps_json` - Workflow steps as a JSON string.

* `created` - The date and time the workflow was created.

* `modified` - The date and time the workflow was modified.
