---
subcategory: "Access Profile"
layout: "identitynow"
page_title: "IdentityNow: identitynow_access_profile"
description: |-
  Manages an IdentityNow Access Profile.
---

# identitynow_access_profile

Manages an IdentityNow Access Profile.

## Example Usage

```hcl
locals {
  manager = [{
    approver_type = "MANAGER"
    approver_id   = null
  }]
  governance_group = [for elem in var.governance_groups :
    {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = elem
    }
  ]
}

resource "identitynow_access_profile" "this" {
  name        = var.name
  description = var.description
  requestable = true
  enabled     = true
 
  entitlements {
    id   = var.entitlement_id
    name = var.entitlement_name
    type = "ENTITLEMENT"
  }
 
  source {
    id   = var.src_id
    name = var.src_name
    type = "SOURCE"
  }
 
  owner {
    id   = var.owner_id
    name = var.owner_name
    type = "IDENTITY"
  }
 
  access_request_config {
    comments_required        = true
    denial_comments_required = true
    dynamic "approval_schemes" {
      for_each = concat(local.manager, local.governance_group)
      content {
        approver_type = approval_schemes.value.approver_type
        approver_id   = approval_schemes.value.approver_id
      }
    }
  }
}
```

## Arguments Reference

The following arguments are supported:

As per developer guide: (https://developer.sailpoint.com/docs/api/v3/create-access-profile)

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* id

## Timeouts

The `timeouts` block allows you to specify [timeouts](https://www.terraform.io/language/resources/syntax#operation-timeouts) for certain actions:

* `create` - (Defaults to 30 minutes) Used when creating the Access Profile.
* `read` - (Defaults to 5 minutes) Used when retrieving the Access Profile.
* `update` - (Defaults to 30 minutes) Used when updating the Access Profile.
* `delete` - (Defaults to 30 minutes) Used when deleting the Access Profile.

## Import

```
terraform import identitynow_access_profile.this [id]
```
