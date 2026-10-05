---
subcategory: "Search"
page_title: "IdentityNow: Data Source: identitynow_scheduled_search"
description: |-
  Gets information about an existing IdentityNow scheduled search.
---

# Data Source: identitynow_scheduled_search

Use this data source to look up a scheduled search by ID.

## Example Usage

```hcl
data "identitynow_scheduled_search" "daily" {
  id = "4dc7ca01-0ca6-4c12-9e4b-4b5fed1a1a3b"
}
```

## Arguments Reference

* `id` - (Required) Scheduled search ID.

## Attributes Reference

* `name` - Scheduled search name.
* `description` - Scheduled search description.
* `saved_search_id` - ID of the saved search that is run.
* `schedule_json` - Schedule as a JSON object.
* `recipient` - Identities that receive the search results, with `type` and `id`.
* `enabled` - Whether the scheduled search is enabled.
* `email_empty_results` - Whether an email is sent when the search returns no results.
* `display_query_details` - Whether the email includes the query and a preview of the results.
* `owner` - Owner of the scheduled search, with `type` and `id`.
* `created` - Creation date.
* `modified` - Last modification date.
