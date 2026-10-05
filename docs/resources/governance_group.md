---
subcategory: "Governance Group"
page_title: "IdentityNow: identitynow_governance_group"
description: |-
  Manages an IdentityNow Governance Group.
---

# identitynow_governance_group

Manages an IdentityNow Governance Group. Use `identitynow_governance_group_members` to manage its members.

## Example Usage

```terraform
resource "identitynow_governance_group" "this" {
  name        = "Access Approvers"
  description = "Approves access requests"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    name = "John Doe"
    type = "IDENTITY"
  }
}
```

## Arguments Reference

The following arguments are supported:

As described in (https://developer.sailpoint.com/docs/api/v2024/create-workgroup)

* `name` - (Required) Governance group name.

* `description` - (Required) Governance group description.

* `owner` - (Required) Governance group owner. Exactly one block is required. Contains:
  * `id` - (Required) Owner identity ID.
  * `name` - (Required) Owner name.
  * `type` - (Optional) Owner type. Defaults to `IDENTITY`.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Governance group ID.

## Import

Governance groups can be imported using their ID:

```shell
terraform import identitynow_governance_group.this <governance-group-id>
```
