---
subcategory: "Source"
page_title: "IdentityNow: Data Source: identitynow_source_schedule"
description: |-
  Gets information about an aggregation schedule of an IdentityNow source.
---

# Data Source: identitynow_source_schedule

Use this data source to look up an aggregation schedule of a source by type.

## Example Usage

```terraform
data "identitynow_source_schedule" "account_aggregation" {
  source_id = "2c9180835d191a86015d28455b4a2329"
  type      = "ACCOUNT_AGGREGATION"
}
```

## Arguments Reference

* `source_id` - (Required) ID of the source.
* `type` - (Required) Schedule type, `ACCOUNT_AGGREGATION` or `GROUP_AGGREGATION`.

## Attributes Reference

* `id` - ID in the format `<source_id>/<type>`.
* `cron_expression` - Cron expression of the schedule.
