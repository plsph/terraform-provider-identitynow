---
subcategory: "Event Trigger"
page_title: "IdentityNow: Data Source: identitynow_trigger"
description: |-
  Gets information about an IdentityNow event trigger.
---

# Data Source: identitynow_trigger

Use this data source to look up an event trigger available in the tenant by ID or name, e.g. to reference it in an [identitynow_trigger_subscription](../resources/trigger_subscription).

## Example Usage

```terraform
data "identitynow_trigger" "identity_created" {
  id = "idn:identity-created"
}

data "identitynow_trigger" "access_request_submitted" {
  name = "Access Request Submitted"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) Trigger ID, e.g. `idn:identity-created`.
* `name` - (Optional) Trigger name. All triggers are listed and matched by exact name.

## Attributes Reference

* `type` - Trigger type: `REQUEST_RESPONSE` or `FIRE_AND_FORGET`.
* `description` - Trigger description.
* `input_schema` - JSON schema of the payload sent by the trigger to subscribers.
* `example_input_json` - Example payload sent by the trigger, as a JSON document.
