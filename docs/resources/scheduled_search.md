---
subcategory: "Search"
page_title: "IdentityNow: identitynow_scheduled_search"
description: |-
  Manages an IdentityNow scheduled search.
---

# identitynow_scheduled_search

Manages a scheduled search, which runs a saved search on a schedule and emails the results to its recipients. The scheduled search is owned by the identity the provider authenticates as.

## Example Usage

```hcl
resource "identitynow_saved_search" "disabled_accounts" {
  name    = "Identities with disabled accounts"
  indices = ["identities"]
  query   = "@accounts(disabled:true)"
}

resource "identitynow_scheduled_search" "disabled_accounts_daily" {
  name                  = "Daily disabled accounts report"
  saved_search_id       = identitynow_saved_search.disabled_accounts.id
  enabled               = true
  email_empty_results   = false
  display_query_details = false

  schedule_json = jsonencode({
    type       = "DAILY"
    hours      = { type = "LIST", values = ["9"] }
    timeZoneId = "Europe/Warsaw"
  })

  recipient {
    id = "2c91808568c529c60168cca6f90c1313"
  }
}
```

## Arguments Reference

The following arguments are supported:

* `saved_search_id` - (Required) ID of the saved search that is run.
* `schedule_json` - (Required) Schedule as a JSON object with `type` (`DAILY`, `WEEKLY`, `MONTHLY`, `CALENDAR` or `ANNUALLY`), the `months`, `days` and `hours` selectors (each with `type` `LIST` or `RANGE`, `values` and an optional `interval`), `expiration` and `timeZoneId`. `hours` is required. Use `jsonencode()`. The value is compared on the configured fields only, so formatting, key order and fields IdentityNow adds do not produce a diff.
* `recipient` - (Required) Identity that receives the search results by email. At least one block is required. Contains:
  * `id` - (Required) Identity ID.
  * `type` - (Optional) Recipient type, `IDENTITY` (default).
* `name` - (Optional) Scheduled search name.
* `description` - (Optional) Scheduled search description.
* `enabled` - (Optional) Whether the scheduled search is enabled. When not set, the value chosen by IdentityNow is kept.
* `email_empty_results` - (Optional) Whether an email is sent when the search returns no results. When not set, the value chosen by IdentityNow is kept.
* `display_query_details` - (Optional) Whether the email includes the query and a preview of the results, which can contain personal data. When not set, the value chosen by IdentityNow is kept.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - Scheduled search ID.
* `owner` - Owner of the scheduled search, with `type` and `id`.
* `created` - Creation date.
* `modified` - Last modification date.

## Update Behaviour

Updates use PUT: the current scheduled search is read, the configured fields are replaced and the scheduled search is written back, so the owner is kept.

## Import

Scheduled searches can be imported using their ID:

```shell
terraform import identitynow_scheduled_search.example <scheduled-search-id>
```
