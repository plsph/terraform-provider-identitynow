---
subcategory: "Certification"
layout: "identitynow"
page_title: "IdentityNow: identitynow_campaign_template_schedule"
description: |-
  Manages the schedule of an IdentityNow certification campaign template.
---

# identitynow_campaign_template_schedule

Manages the schedule that generates campaigns from a certification campaign template. A template has at most one schedule, so the schedule is identified by the template ID. Every create and update replaces the whole schedule.

## Example Usage

```hcl
resource "identitynow_campaign_template" "quarterly_managers" {
  name        = "Quarterly manager certification"
  description = "Managers review the access of their reports every quarter."

  campaign_json = jsonencode({
    name        = "Manager certification"
    description = "Review the access of your reports"
    type        = "MANAGER"
  })
}

# Runs at 9 AM on the first day of January, April, July and October.
resource "identitynow_campaign_template_schedule" "quarterly_managers" {
  campaign_template_id = identitynow_campaign_template.quarterly_managers.id
  type                 = "ANNUALLY"
  time_zone_id         = "Europe/Warsaw"

  months {
    type     = "LIST"
    values   = ["1"]
    interval = 3
  }

  days {
    type   = "LIST"
    values = ["1"]
  }

  hours {
    type   = "LIST"
    values = ["9"]
  }
}
```

## Arguments Reference

The following arguments are supported:

* `campaign_template_id` - (Required) ID of the scheduled campaign template. Changing this forces a new schedule to be created.
* `type` - (Required) Schedule cadence: `WEEKLY`, `MONTHLY`, `ANNUALLY` or `CALENDAR`. All periods smaller than the cadence can be selected.
* `hours` - (Required) Active hours (`0`-`23`). Exactly one block is required.
* `days` - (Optional) Active days: days of the week (`1`-`7`) for `WEEKLY`, days of the month (`1`-`31`, `L` for the last day) for `MONTHLY` and `ANNUALLY`, ISO-8601 dates for `CALENDAR`. At most one block.
* `months` - (Optional) Active months (`1`-`12`), only valid for `ANNUALLY` schedules. At most one block.
* `expiration` - (Optional) Date and time (ISO-8601) after which the schedule no longer runs.
* `time_zone_id` - (Optional) Time zone the schedule runs in, such as `America/New_York`. When not set, the value chosen by IdentityNow is kept.

The `months`, `days` and `hours` blocks contain:

* `type` - (Required) `LIST` (distinct values) or `RANGE` (two values, the inclusive start and end of the range).
* `values` - (Required) Selected values, as strings.
* `interval` - (Optional) Interval between the selected values, for example `3` with hour `8` runs every three hours from 8 AM.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - Campaign template ID.

## Import

Campaign template schedules can be imported using the campaign template ID:

```shell
terraform import identitynow_campaign_template_schedule.example <campaign-template-id>
```
