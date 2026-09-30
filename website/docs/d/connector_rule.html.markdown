---
subcategory: "Connector"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_connector_rule"
description: |-
  Gets information about an existing IdentityNow connector rule.
---

# Data Source: identitynow_connector_rule

Use this data source to look up a connector rule by ID or name. Lookups by name list all connector rules and match the name exactly.

## Example Usage

```hcl
data "identitynow_connector_rule" "before_create" {
  name = "AD Before Create"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) Connector rule ID.
* `name` - (Optional) Rule name.

## Attributes Reference

* `description` - Description of the rule's purpose.
* `type` - Rule type.
* `signature_json` - Function signature of the rule as a JSON object.
* `source_code` - Code of the rule, a list with one element:
  * `version` - Version of the code.
  * `script` - The BeanShell code.
* `attributes_json` - Rule attributes as a JSON object.
* `created` - Creation date.
* `modified` - Last modification date.
