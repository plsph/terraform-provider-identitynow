---
subcategory: "Governance Group"
layout: "identitynow"
page_title: "IdentityNow: identitynow_governance_group_members"
description: |-
  Manages an IdentityNow Governance Group Members.
---

# identitynow_governance_group_members

Manages an IdentityNow Governance Group Members.

## Example Usage

```hcl
locals {
emails = [ "some list of emails", ... ]
}

#here we fetch identity details
data "identitynow_identity" "example" {
for_each = toset(local.emails)
  email_address = each.key
}

resource "identitynow_governance_group" "example" {
  name        = "governance_group_name"
  description = "example"
  owner {
    id   = data.identitynow_identity.example["example owner email"].id
    name = data.identitynow_identity.example["example owner email"].name
    type = "IDENTITY"
  }
}

#Here goes nasty toset() trick to get members in "id" order as sailpoint backend keeps them this way.
resource "identitynow_governance_group_members" "example" {
  governance_group_id = identitynow_governance_group.example.id
  dynamic "members" {
    for_each = toset([for identity in data.identitynow_identity.example : identity.id])
    content {
      name = [for identity in data.identitynow_identity.example : identity.name if identity.id == members.value][0]
      id   = members.value
      type = "IDENTITY"
    }
  }
}

#Here will be ethernal drift if you don't order these emails with alphabetic order based on sailpoint assigned ids.
resource "identitynow_governance_group_members" "example2" {
  governance_group_id = identitynow_governance_group.example.id
  members {
    type = "IDENTITY"
    id   = data.identitynow_identity.example["example member email"].id
    name = data.identitynow_identity.example["example member email"].name
  }
  members {
    type = "IDENTITY"
    id   = data.identitynow_identity.example["example member email"].id
    name = data.identitynow_identity.example["example member email"].name
  }
}
```

## Arguments Reference

The following arguments are supported:

As described in (https://developer.sailpoint.com/docs/api/v2024/create-workgroup)

* `governance_group_id` - id of Governance Group

* `members` - For each member map of type, id and name.

## Attributes Reference

Arguments listed above.

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/regovernance_groups/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 1 minute) Used when creating the Governance Group Members.
* `read` - (Defaults to 1 minute) Used when retrieving the Governance Group Members.
* `update` - (Defaults to 1 minute) Used when updating the Governance Group Members.
* `delete` - (Defaults to 1 minute) Used when deleting the Governance Group Members.

## Import

```
terraform import identitynow_governance_group_members.this [id of governance group]
```
