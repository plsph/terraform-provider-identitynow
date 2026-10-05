---
subcategory: "Event Trigger"
page_title: "IdentityNow: identitynow_trigger_subscription"
description: |-
  Manages an IdentityNow event trigger subscription.
---

# identitynow_trigger_subscription

Manages a subscription to an event trigger. The subscription defines where trigger invocations are delivered: an HTTP endpoint, AWS EventBridge, a workflow, or an inline or script handler. Use the [identitynow_trigger](../data-sources/trigger) data source to look up available triggers.

## Example Usage

```hcl
variable "webhook_token" {
  type      = string
  sensitive = true
}

resource "identitynow_trigger_subscription" "identity_created_webhook" {
  name        = "Identity created webhook"
  description = "Notifies the HR integration about new identities"
  trigger_id  = "idn:identity-created"
  type        = "HTTP"
  filter      = "$[?($.attributes.department == \"Engineering\")]"

  http_config {
    url                      = "https://hooks.example.com/identity-created"
    http_dispatch_mode       = "SYNC"
    http_authentication_type = "BEARER_TOKEN"
    bearer_token             = var.webhook_token
  }
}

resource "identitynow_trigger_subscription" "identity_created_eventbridge" {
  name       = "Identity created to EventBridge"
  trigger_id = "idn:identity-created"
  type       = "EVENTBRIDGE"

  event_bridge_config {
    aws_account = "123456789012"
    aws_region  = "eu-central-1"
  }
}
```

## Arguments Reference

* `name` - (Required) Subscription name.
* `trigger_id` - (Required) ID of the trigger to subscribe to, e.g. `idn:identity-created`. The trigger of a subscription cannot be changed, changing this forces a new subscription to be created.
* `type` - (Required) Subscription type: `HTTP`, `EVENTBRIDGE`, `INLINE`, `SCRIPT` or `WORKFLOW`. `HTTP` requires an `http_config` block and `EVENTBRIDGE` requires an `event_bridge_config` block.
* `description` - (Optional) Subscription description.
* `enabled` - (Optional) Whether the subscription receives real-time trigger invocations. Test invocations are always delivered. Defaults to `true`.
* `filter` - (Optional) [JSONPath filter](https://developer.sailpoint.com/docs/extensibility/event-triggers/filtering-events); the trigger is only invoked when the expression evaluates to true.
* `response_deadline` - (Optional) Deadline for completing a `REQUEST_RESPONSE` trigger invocation as an ISO-8601 duration, e.g. `PT1H`. When not set, the API default (`PT1H`) is used.
* `workflow_config_json` - (Optional) Configuration of a `WORKFLOW` subscription as a JSON object. The v2026 API lists the field as patchable but does not document its content, so the value is passed through as is. Compared semantically, so formatting and key order do not produce a diff.
* `http_config` - (Optional) Configuration of an `HTTP` subscription. At most one block. Contains:
  * `url` - (Required) URL of the external integration.
  * `http_dispatch_mode` - (Required) HTTP response mode: `SYNC`, `ASYNC` or `DYNAMIC`.
  * `http_authentication_type` - (Optional) Authentication type: `NO_AUTH`, `BASIC_AUTH` or `BEARER_TOKEN`. Defaults to `NO_AUTH`.
  * `basic_auth_user_name` - (Optional) User name for `BASIC_AUTH`.
  * `basic_auth_password` - (Optional, Sensitive) Password for `BASIC_AUTH`.
  * `bearer_token` - (Optional, Sensitive) Token for `BEARER_TOKEN`.
* `event_bridge_config` - (Optional) Configuration of an `EVENTBRIDGE` subscription. At most one block. Contains:
  * `aws_account` - (Required) 12-digit AWS account number that has the EventBridge partner event source.
  * `aws_region` - (Required) AWS region that has the EventBridge partner event source, e.g. `us-east-1`.

The API never returns `basic_auth_password` and `bearer_token`. They are stored in the Terraform state (marked sensitive) as configured, and changes made outside Terraform are not detected.

Changes are applied with a JSON Patch of the changed fields only. Optional fields that were not set before (e.g. `event_bridge_config` when `type` changes from `HTTP` to `EVENTBRIDGE`) are set with an `add` operation, because they may be missing from the subscription and `replace` fails for a missing member.

## Attributes Reference

* `id` - Subscription ID.
* `trigger_name` - Name of the trigger.

## Import

Trigger subscriptions can be imported using their ID:

```shell
terraform import identitynow_trigger_subscription.example <subscription-id>
```

After an import the password and bearer token are not known, so the next apply sends the configured values.
