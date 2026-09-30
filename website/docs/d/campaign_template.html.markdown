---
subcategory: "Certification"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_campaign_template"
description: |-
  Gets information about an existing IdentityNow certification campaign template.
---

# Data Source: identitynow_campaign_template

Use this data source to look up a certification campaign template by ID or name.

## Example Usage

```hcl
data "identitynow_campaign_template" "quarterly_managers" {
  name = "Quarterly manager certification"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) Campaign template ID.
* `name` - (Optional) Campaign template name.

## Attributes Reference

* `description` - Template description.
* `deadline_duration` - Completion period of generated campaigns as an ISO-8601 duration.
* `campaign_json` - Definition of the generated campaigns as a JSON object, without read-only fields.
* `scheduled` - Whether the template has a schedule.
* `owner_ref` - Owner of the template, with `id`, `type` and `name`.
* `created` - Creation date.
* `modified` - Last modification date.
