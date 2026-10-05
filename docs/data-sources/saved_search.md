---
subcategory: "Search"
page_title: "IdentityNow: Data Source: identitynow_saved_search"
description: |-
  Gets information about an existing IdentityNow saved search.
---

# Data Source: identitynow_saved_search

Use this data source to look up a saved search by ID.

## Example Usage

```hcl
data "identitynow_saved_search" "disabled_accounts" {
  id = "0de46054-fe90-434a-b84e-c6b3359d0c64"
}
```

## Arguments Reference

* `id` - (Required) Saved search ID.

## Attributes Reference

* `name` - Saved search name.
* `description` - Saved search description.
* `public` - Whether the saved search is public.
* `indices` - Search indices.
* `query` - Search query.
* `fields` - Fields searched in a multi-field query.
* `order_by` - Map of document type to sort fields.
* `sort` - Fields the results are sorted by.
* `filters_json` - Filters per field name as a JSON object.
* `columns_json` - Columns per document type as a JSON object.
* `owner_id` - ID of the identity that owns the saved search.
* `created` - Creation date.
* `modified` - Last modification date.
