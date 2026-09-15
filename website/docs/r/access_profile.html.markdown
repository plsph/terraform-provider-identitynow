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
  name        = "example"
  description = "example"
  requestable = true
  enabled     = true
  segments    = ["f7b1b8a3-5fed-4fd4-ad29-82014e137e19"]

  entitlements {
    id   = "example id"
    name = "example name"
    type = "ENTITLEMENT"
  }

  source {
    id   = "example id"
    name = "example source name"
    type = "SOURCE"
  }

  owner {
    id   = "example id"
    name = "example owner name"
    type = "IDENTITY"
  }

  access_request_config {
    comments_required        = true
    denial_comments_required = true
    reauthorization_required = true
    require_end_date        = true

    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = "46c79819-a69f-49a2-becb-12c971ae66c6"
    }

    max_permitted_access_duration {
      value    = 6
      time_unit = "MONTHS"
    }
  }

  revocation_request_config {
    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = "46c79819-a69f-49a2-becb-12c971ae66c6"
    }
  }
}
```

## Arguments Reference

The following arguments are supported:

As per developer guide: (https://developer.sailpoint.com/docs/api/v3/create-access-profile)

* `name` - Access profile name.

* `description` - Access profile description.

* `requestable` - Indicates whether the access profile is requestable by access request. Currently, making an access profile non-requestable is only supported for customers enabled with the new Request Center. Otherwise, attempting to create an access profile with a value false in this field results in a 400 error.

* `enabled` - Indicates whether the access profile is enabled. If it's enabled, you must include at least one entitlement.

* `entitlements` - List of entitlements associated with the access profile. If enabled is false, this can be empty. Otherwise, it must contain at least one entitlement.

* `source` - Source associated with the access profile.

* `owner` - Owner of the object.

* `access_request_config` - Access profile request configuration. Contains:

* `comments_required` - Indicates whether the requester must provide comments justifying the request.

* `denial_comments_required` - Indicates whether an approver must provide comments when denying the request.

* `reauthorization_required` - Indicates whether reauthorization is required.

* `require_end_date` - Indicates whether the requester must provide an access end date.

* `max_permitted_access_duration` - Maximum access duration the requester can grant.

* `approval_schemes` - List describing the approval steps.

* `revocation_request_config` - Revocation approval configuration. Contains an `approval_schemes` block.

* `segments` - List of segment IDs assigned to the access profile.

* `access_model_metadata` - Optional metadata attributes for the access profile.

* `provisioning_criteria` - Optional criteria used to choose the account to provision.

* `additional_owners` - Optional additional identity or governance group owners.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Access profile id.

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
