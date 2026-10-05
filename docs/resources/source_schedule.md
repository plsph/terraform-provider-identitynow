---
subcategory: "Source"
page_title: "IdentityNow: identitynow_source_schedule"
description: |-
  Manages an aggregation schedule of an IdentityNow source.
---

# identitynow_source_schedule

Manages an account or group aggregation schedule of a source with the v2026 source schedules API. It uses the v2026 source ID and is the successor of [identitynow_schedule_account_aggregation](schedule_account_aggregation), which uses the legacy numeric source ID. Do not manage the account aggregation schedule of a source with both resources.

Destroying the resource deletes the schedule, so the source is no longer aggregated automatically. If the source already has a schedule of the type, import it instead of creating it.

## Example Usage

```terraform
resource "identitynow_source_schedule" "account_aggregation" {
  source_id       = "2c9180835d191a86015d28455b4a2329"
  type            = "ACCOUNT_AGGREGATION"
  cron_expression = "0 0 12 1/1 * ? *" # every day at 12:00
}
```

## Arguments Reference

* `source_id` - (Required) ID of the source. Changing this forces a new schedule to be created.
* `type` - (Required) Schedule type, `ACCOUNT_AGGREGATION` or `GROUP_AGGREGATION`. The type cannot be changed, changing this forces a new schedule to be created.
* `cron_expression` - (Required) Cron expression of the schedule, e.g. `0 0 12 1/1 * ? *` for every day at 12:00. Days of the week are 1-7 (Sunday-Saturday).

## Attributes Reference

* `id` - Resource ID in the format `<source_id>/<type>`.

## Import

Source schedules can be imported using the source ID and the schedule type:

```shell
terraform import identitynow_source_schedule.example <source-id>/ACCOUNT_AGGREGATION
```
