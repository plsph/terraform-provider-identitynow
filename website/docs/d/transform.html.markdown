---
subcategory: "Transform"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_transform"
description: |-
  Gets information about an existing IdentityNow transform.
---

# Data Source: identitynow_transform

Use this data source to look up a transform by ID or name.

## Example Usage

```hcl
data "identitynow_transform" "country_lookup" {
  name = "Country Lookup"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) Transform ID.
* `name` - (Optional) Transform name.

## Attributes Reference

* `type` - Transform operation type.
* `attributes_json` - Transform attributes as a JSON object.
* `internal` - Whether this is a SailPoint internal transform.
