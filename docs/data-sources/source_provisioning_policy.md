---
subcategory: "Source"
page_title: "IdentityNow: Data Source: identitynow_source_provisioning_policy"
description: |-
  Gets information about a provisioning policy of an IdentityNow source.
---

# Data Source: identitynow_source_provisioning_policy

Use this data source to look up the provisioning policy of a source by usage type.

## Example Usage

```terraform
data "identitynow_source_provisioning_policy" "create" {
  source_id  = "2c9180835d191a86015d28455b4a2329"
  usage_type = "CREATE"
}
```

## Arguments Reference

* `source_id` - (Required) ID of the source.
* `usage_type` - (Required) Provisioning operation the policy applies to, e.g. `CREATE`.

## Attributes Reference

* `id` - ID in the format `<source_id>/<usage_type>`.
* `name` - Provisioning policy name.
* `description` - Provisioning policy description.
* `fields_json` - Policy fields as a JSON array.
