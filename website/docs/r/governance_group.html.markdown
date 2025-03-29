---
subcategory: "Governance Group"
layout: "identitynow"
page_title: "IdentityNow: identitynow_governance_group"
description: |-
  Manages an IdentityNow Governance Group.
---

# identitynow_governance_group

Manages an IdentityNow Governance Group.

## Example Usage

```hcl
resource "identitynow_governance_group" "this" {
    name        = var.name
    description = var.description
    owner {
      id   = var.owner_id
      name = var.owner_name
      type = "IDENTITY"
    }
}
```

## Arguments Reference

The following arguments are supported:

As described in (https://developer.sailpoint.com/docs/api/v2024/create-workgroup)

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* id

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/regovernance_groups/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the Governance Group.
* `read` - (Defaults to 5 minutes) Used when retrieving the Governance Group.
* `update` - (Defaults to 30 minutes) Used when updating the Governance Group.
* `delete` - (Defaults to 30 minutes) Used when deleting the Governance Group.

## Import

```
terraform import identitynow_governance_group.this [id]
```
