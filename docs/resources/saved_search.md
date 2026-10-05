---
subcategory: "Search"
page_title: "IdentityNow: identitynow_saved_search"
description: |-
  Manages an IdentityNow saved search.
---

# identitynow_saved_search

Manages a saved search, which can be run on a schedule with `identitynow_scheduled_search`. The saved search is owned by the identity the provider authenticates as, and the owner cannot be changed.

## Example Usage

```terraform
resource "identitynow_saved_search" "disabled_accounts" {
  name        = "Identities with disabled accounts"
  description = "Identities that have at least one disabled account."
  indices     = ["identities"]
  query       = "@accounts(disabled:true)"
  sort        = ["displayName"]

  order_by = {
    identity = ["lastName", "firstName"]
  }

  columns_json = jsonencode({
    identity = [
      { field = "displayName", header = "Display Name" },
      { field = "email", header = "Work Email" }
    ]
  })

  filters_json = jsonencode({
    "source.name" = {
      type    = "TERMS"
      terms   = ["HR Employees"]
      exclude = false
    }
  })
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) Saved search name.
* `indices` - (Required) Search indices: `accessprofiles`, `accountactivities`, `entitlements`, `events`, `identities`, `roles` or `*`.
* `query` - (Required) Search query in Elasticsearch query string syntax, such as `@accounts(disabled:true)`.
* `description` - (Optional) Saved search description.
* `fields` - (Optional) Fields searched in a multi-field query.
* `order_by` - (Optional) Map of document type to the fields its results are sorted by. It takes precedence over `sort`.
* `sort` - (Optional) Fields the results are sorted by.
* `filters_json` - (Optional) Filters per field name as a JSON object. Each filter has a `type` (`EXISTS`, `RANGE` or `TERMS`), a `range` (`lower` and `upper` bounds with `value` and `inclusive`), `terms` and `exclude`. Use `jsonencode()`.
* `columns_json` - (Optional) Columns returned per document type as a JSON object, each column with a `field` and a `header`. Use `jsonencode()`.

JSON arguments are compared with the API value on the configured fields only: formatting, key order and fields IdentityNow adds do not produce a diff.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - Saved search ID.
* `public` - Whether the saved search is visible to others than the owner. Saved searches cannot be made public at this time, so it is always `false`.
* `owner_id` - ID of the identity that owns the saved search.
* `created` - Creation date.
* `modified` - Last modification date.

## Update Behaviour

Updates use PUT: the current saved search is read, the configured fields are replaced and the saved search is written back, so the owner is kept.

## Import

Saved searches can be imported using their ID:

```shell
terraform import identitynow_saved_search.example <saved-search-id>
```
