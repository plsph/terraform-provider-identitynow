---
subcategory: "SOD Policy"
layout: "identitynow"
page_title: "IdentityNow: identitynow_sod_policy_schedule"
description: |-
  Manages the violation report schedule of an IdentityNow SOD policy.
---

# identitynow_sod_policy_schedule

Manages the schedule on which the violation report of a separation of duties (SOD) policy is run and emailed to its recipients. A policy has at most one schedule, so the schedule is identified by the policy ID. Every create and update replaces the whole schedule.

## Example Usage

```hcl
resource "identitynow_sod_policy_schedule" "weekly" {
  policy_id           = "0f11f2a4-7c94-4bf3-a2bd-742580fe3bde"
  name                = "Weekly SOD report"
  description         = "Sent every Monday at 8 AM"
  email_empty_results = false

  schedule_json = jsonencode({
    type       = "WEEKLY"
    days       = { type = "LIST", values = ["MON"] }
    hours      = { type = "LIST", values = ["8"] }
    timeZoneId = "Europe/Warsaw"
  })

  recipient {
    id = "2c91808568c529c60168cca6f90c1313"
  }
}
```

## Arguments Reference

The following arguments are supported:

* `policy_id` - (Required) ID of the scheduled SOD policy. Changing this forces a new schedule to be created.
* `schedule_json` - (Required) Schedule as a JSON object with `type` (`DAILY`, `WEEKLY`, `MONTHLY`, `CALENDAR` or `ANNUALLY`), the `months`, `days` and `hours` selectors (each with `type` `LIST` or `RANGE`, `values` and an optional `interval`), `expiration` and `timeZoneId`. `hours` is required. Use `jsonencode()`. The value is compared on the configured fields only, so formatting, key order and fields IdentityNow adds do not produce a diff.
* `name` - (Optional) Schedule name.
* `description` - (Optional) Schedule description.
* `email_empty_results` - (Optional) Whether the report is emailed when it has no results. When not set, the value chosen by IdentityNow is kept.
* `recipient` - (Optional) Identity that receives the violation report. Can be repeated. Contains:
  * `id` - (Required) Identity ID.
  * `type` - (Optional) Recipient type, `IDENTITY` (default).
  * `name` - (Optional) Identity display name. When omitted, the name resolved by the API is not tracked.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - SOD policy ID.
* `creator_id` - ID of the identity that created the schedule.
* `modifier_id` - ID of the identity that last modified the schedule.
* `created` - Creation date.
* `modified` - Last modification date.

## Import

SOD policy schedules can be imported using the SOD policy ID:

```shell
terraform import identitynow_sod_policy_schedule.example <sod-policy-id>
```
