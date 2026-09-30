---
subcategory: "Data Access Security"
layout: "identitynow"
page_title: "IdentityNow: identitynow_das_task_schedule"
description: |-
  Manages an IdentityNow Data Access Security task schedule.
---

# identitynow_das_task_schedule

Manages a Data Access Security task schedule, which runs a task such as a crawl or a permission collection of a [Data Access Security application](das_application.html) on a recurring schedule.

Updates replace the whole schedule (PUT) with the configured values.

## Example Usage

```hcl
resource "identitynow_das_task_schedule" "file_share_crawl" {
  task_type_name = "Crawl"
  schedule_type  = "Weekly"
  interval       = 1
  days_of_week   = ["Monday"]
  active         = true
  application_id = 42
}
```

## Arguments Reference

* `task_type_name` - (Required) Type of the scheduled task.
* `schedule_type` - (Required) Schedule cycle, e.g. `Daily`, `Weekly` or `Manual`.
* `schedule_task_name` - (Optional) Display name of the scheduled task.
* `interval` - (Optional) Interval between runs in units of the schedule cycle, e.g. days for a daily schedule.
* `start_time` - (Optional) Start time of the schedule, in seconds since the epoch. When not set, the value returned by the API is stored and kept: the API always has a start time, so removing the argument keeps the current start time.
* `end_time` - (Optional) End time of the schedule, in seconds since the epoch.
* `days_of_week` - (Optional) Days of the week the task runs on, e.g. `Monday`.
* `active` - (Optional) Whether the schedule is active. Defaults to `false`.
* `run_after_schedule_task_id` - (Optional) ID of another schedule whose completion triggers this task.
* `application_id` - (Optional) ID of the Data Access Security application of the task, e.g. `identitynow_das_application.example.id`.

## Attributes Reference

* `id` - Schedule ID.
* `run_after_schedule_task_name` - Name of the schedule whose completion triggers this task.
* `created_by_display_name` - Display name of the user who created the schedule.
* `next_run` - Next run time, in seconds since the epoch.
* `last_run` - Last run time, in seconds since the epoch.

## Import

Data Access Security task schedules can be imported using their numeric ID:

```shell
terraform import identitynow_das_task_schedule.example <schedule-id>
```
