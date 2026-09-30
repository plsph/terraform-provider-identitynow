---
subcategory: "SOD Policy"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_sod_policy"
description: |-
  Gets information about an existing IdentityNow SOD policy.
---

# Data Source: identitynow_sod_policy

Use this data source to look up a separation of duties (SOD) policy by ID or name.

## Example Usage

```hcl
data "identitynow_sod_policy" "payables_receivables" {
  name = "Payables vs Receivables"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) SOD policy ID.
* `name` - (Optional) SOD policy name.

## Attributes Reference

* `description` - Policy description.
* `owner_ref` - Owner of the policy, with `id`, `type` and `name`.
* `type` - Policy type, `GENERAL` or `CONFLICTING_ACCESS_BASED`.
* `policy_query` - Search query of the policy.
* `conflicting_access_criteria_json` - Conflicting access criteria as a JSON object.
* `external_policy_reference` - Reference to an external policy.
* `compensating_controls` - Compensating (mitigating) controls.
* `correction_advice` - Advice on how to correct a violation.
* `state` - Whether the policy is `ENFORCED` or `NOT_ENFORCED`.
* `tags` - Tags of the policy.
* `scheduled` - Whether the policy is scheduled.
* `violation_owner_assignment_config` - Who owns the violations of the policy, with `assignment_rule` and `owner_ref` (`id`, `type` and `name`).
* `creator_id` - ID of the identity that created the policy.
* `modifier_id` - ID of the identity that last modified the policy.
* `created` - Creation date.
* `modified` - Last modification date.
