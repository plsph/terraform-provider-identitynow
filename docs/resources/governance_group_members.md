---
subcategory: "Governance Group"
page_title: "IdentityNow: identitynow_governance_group_members"
description: |-
  Manages the members of an IdentityNow Governance Group.
---

# identitynow_governance_group_members

Manages the members of an IdentityNow Governance Group.

The list of members is authoritative: members added to the governance group outside Terraform are shown as a difference and removed on apply. The order of the `members` blocks doesn't matter. Destroying the resource removes only the members managed by the resource from the governance group. Manage the members of a governance group with a single resource.

## Example Usage

```terraform
locals {
  owner_email   = "jane.doe@example.com"
  member_emails = ["john.doe@example.com", "mary.major@example.com"]
}

# Fetch identity details
data "identitynow_identity" "owner" {
  email_address = local.owner_email
}

data "identitynow_identity" "members" {
  for_each      = toset(local.member_emails)
  email_address = each.key
}

resource "identitynow_governance_group" "example" {
  name        = "Access Approvers"
  description = "Approves access requests"

  owner {
    id   = data.identitynow_identity.owner.id
    name = data.identitynow_identity.owner.name
    type = "IDENTITY"
  }
}

resource "identitynow_governance_group_members" "example" {
  governance_group_id = identitynow_governance_group.example.id

  dynamic "members" {
    for_each = data.identitynow_identity.members
    content {
      id   = members.value.id
      name = members.value.name
      type = "IDENTITY"
    }
  }
}
```

## Arguments Reference

The following arguments are supported:

As described in (https://developer.sailpoint.com/docs/api/v2024/create-workgroup)

* `governance_group_id` - (Required) ID of the governance group. Changing this forces a new resource to be created.

* `members` - (Optional) Members of the governance group. Can be repeated. Contains:
  * `id` - (Required) Identity ID of the member.
  * `name` - (Required) Name of the member.
  * `type` - (Optional) Member type. Defaults to `IDENTITY`.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Governance group members ID (same as `governance_group_id`).

## Import

Governance group members can be imported using the governance group ID:

```shell
terraform import identitynow_governance_group_members.example <governance-group-id>
```
