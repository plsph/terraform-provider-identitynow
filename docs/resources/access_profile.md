---
subcategory: "Access Profile"
page_title: "IdentityNow: identitynow_access_profile"
description: |-
  Manages an IdentityNow Access Profile.
---

# identitynow_access_profile

Manages an IdentityNow Access Profile.

All arguments can be updated in place, including `name`, the `source` together with the entitlements of the new source, `segments`, `provisioning_criteria` and `additional_owners`. The only exception is `access_model_metadata`: the API accepts it only when the access profile is created, so changes to it on an existing access profile are not applied and a warning is shown. Recreate the access profile to change its metadata.

## Example Usage

```hcl
data "identitynow_identity" "owner" {
  alias = "john.doe"
}

data "identitynow_source" "active_directory" {
  name = "Active Directory"
}

data "identitynow_source_entitlement" "developers" {
  source_id = data.identitynow_source.active_directory.id
  name      = "Developers"
}

resource "identitynow_access_profile" "developers" {
  name        = "AD Developers"
  description = "Developer access in Active Directory"
  requestable = true
  enabled     = true
  segments    = ["f7b1b8a3-5fed-4fd4-ad29-82014e137e19"]

  owner {
    id   = data.identitynow_identity.owner.id
    name = data.identitynow_identity.owner.name
    type = "IDENTITY"
  }

  source {
    id   = data.identitynow_source.active_directory.id
    name = data.identitynow_source.active_directory.name
    type = "SOURCE"
  }

  entitlements {
    id   = data.identitynow_source_entitlement.developers.entitlements[0].id
    name = data.identitynow_source_entitlement.developers.entitlements[0].name
    type = "ENTITLEMENT"
  }

  access_request_config {
    comments_required        = true
    denial_comments_required = true
    reauthorization_required = false
    require_end_date         = true

    approval_schemes {
      approver_type = "MANAGER"
    }

    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = "46c79819-a69f-49a2-becb-12c971ae66c6"
    }

    max_permitted_access_duration {
      value     = 6
      time_unit = "MONTHS"
    }
  }

  revocation_request_config {
    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = "46c79819-a69f-49a2-becb-12c971ae66c6"
    }
  }

  additional_owners {
    id   = "46c79819-a69f-49a2-becb-12c971ae66c6"
    name = "Access Approvers"
    type = "GOVERNANCE_GROUP"
  }

  provisioning_criteria {
    operation = "OR"

    children {
      operation = "EQUALS"
      attribute = "accountType"
      value     = "developer"
    }

    children {
      operation = "EQUALS"
      attribute = "accountType"
      value     = "admin"
    }
  }
}
```

### Access Profile with Access Model Metadata

```hcl
resource "identitynow_access_profile" "with_metadata" {
  name        = "AD Operators"
  description = "Operator access in Active Directory"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    name = "John Doe"
    type = "IDENTITY"
  }

  source {
    id   = "2c9180835d191a86015d28455b4a2329"
    name = "Active Directory"
    type = "SOURCE"
  }

  entitlements {
    id   = "2c91808a7813090a017813b6301fabcd"
    name = "Operators"
  }

  access_model_metadata {
    attributes {
      key  = "iscPrivacy"
      name = "Privacy"

      values {
        value = "internal"
      }
    }
  }
}
```

## Arguments Reference

The following arguments are supported:

As per developer guide: (https://developer.sailpoint.com/docs/api/v3/create-access-profile)

* `name` - (Required) Access profile name.

* `description` - (Required) Access profile description.

* `requestable` - (Optional) Whether the access profile is requestable by access request. Making an access profile non-requestable is only supported for tenants enabled with the new Request Center. If not set, the value returned by the API is used.

* `enabled` - (Optional) Whether the access profile is enabled. An enabled access profile must include at least one entitlement. If not set, the value returned by the API is used.

* `segments` - (Optional) Set of segment IDs assigned to the access profile. The order is not significant.

* `owner` - (Optional) Owner of the access profile. Contains:
  * `id` - (Required) Owner identity ID.
  * `name` - (Required) Owner name.
  * `type` - (Required) Owner type, `IDENTITY`.

* `source` - (Optional) Source associated with the access profile. Contains:
  * `id` - (Required) Source ID.
  * `name` - (Required) Source name.
  * `type` - (Required) Source type, `SOURCE`.

* `entitlements` - (Optional) Entitlements of the source assigned to the access profile. Can be repeated. If `enabled` is false this can be empty, otherwise at least one entitlement is required. Contains:
  * `id` - (Required) Entitlement ID.
  * `name` - (Required) Entitlement name. The configured casing is kept when the API name differs only in case.
  * `type` - (Optional) Entitlement type. Defaults to `ENTITLEMENT`.

* `access_request_config` - (Optional) Access request configuration. Contains:
  * `comments_required` - (Optional) Whether the requester must provide comments justifying the request. Defaults to `false`.
  * `denial_comments_required` - (Optional) Whether an approver must provide comments when denying the request. Defaults to `false`.
  * `reauthorization_required` - (Optional) Whether reauthorization is required for the request. Defaults to `false`.
  * `require_end_date` - (Optional) Whether the requester must provide an access end date. Defaults to `false`.
  * `form_definition_id` - (Optional) ID of the form definition presented to the requester during the access request. The v2026 API documents this field only for roles, so it relies on the tenant accepting it for access profiles.
  * `approval_schemes` - (Optional) Approval steps of the request, in order. Can be repeated. Contains:
    * `approver_type` - (Required) Type of approver, e.g. `APP_OWNER`, `OWNER`, `SOURCE_OWNER`, `MANAGER` or `GOVERNANCE_GROUP`.
    * `approver_id` - (Optional) ID of the approver, required when `approver_type` is `GOVERNANCE_GROUP`. Defaults to an empty string.
  * `max_permitted_access_duration` - (Optional) Maximum access duration the requester can request. Contains:
    * `value` - (Required) Amount of time.
    * `time_unit` - (Required) Unit of time, e.g. `DAYS`, `WEEKS` or `MONTHS`.

* `revocation_request_config` - (Optional) Revocation request configuration. Contains:
  * `approval_schemes` - (Optional) Approval steps of the revocation request, in order. Can be repeated. Contains:
    * `approver_type` - (Required) Type of approver, e.g. `APP_OWNER`, `OWNER`, `SOURCE_OWNER`, `MANAGER` or `GOVERNANCE_GROUP`.
    * `approver_id` - (Optional) ID of the approver, required when `approver_type` is `GOVERNANCE_GROUP`.

* `additional_owners` - (Optional) Additional identity or governance group owners of the access profile. Can be repeated. Contains:
  * `id` - (Required) ID of the identity or governance group.
  * `type` - (Required) Owner type, `IDENTITY` or `GOVERNANCE_GROUP`.
  * `name` - (Optional) Owner name.

* `provisioning_criteria` - (Optional) Criteria used to choose the account the access profile is provisioned to when an identity has several accounts on the source. Contains:
  * `operation` - (Required) Operation, e.g. `EQUALS`, `NOT_EQUALS`, `CONTAINS`, `HAS`, `AND` or `OR`.
  * `attribute` - (Optional) Account attribute to compare, for comparison operations.
  * `value` - (Optional) Value to compare the attribute with, for comparison operations.
  * `children` - (Optional) Child criteria for `AND` and `OR` operations, with the same arguments. Supports up to 3 levels of nesting.

* `access_model_metadata` - (Optional) Access model metadata of the access profile. Only applied when the access profile is created. Contains:
  * `attributes` - (Optional) Metadata attributes. Can be repeated. Contains:
    * `key` - (Required) Unique identifier of the metadata type, e.g. `iscPrivacy`.
    * `name` - (Required) Human readable name of the metadata attribute.
    * `multiselect` - (Optional) Whether multiple values can be selected. If not set, the value returned by the API is used.
    * `status` - (Optional) Status of the metadata attribute, e.g. `active`. If not set, the value returned by the API is used.
    * `type` - (Optional) Type of the metadata attribute, e.g. `governance` or `custom`. If not set, the value returned by the API is used.
    * `description` - (Optional) Description of the metadata attribute. If not set, the value returned by the API is used.
    * `object_types` - (Optional) Object types the metadata attribute applies to. Can be repeated. Contains:
      * `value` - (Required) Object type, e.g. `entitlement`.
    * `values` - (Optional) Values assigned to the metadata attribute. Can be repeated. Contains:
      * `value` - (Required) The metadata value.
      * `name` - (Optional) Human readable name of the value.
      * `status` - (Optional) Status of the value, e.g. `active`. If not set, the value returned by the API is used.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Access profile ID.

## Import

Access profiles can be imported using their ID:

```shell
terraform import identitynow_access_profile.example <access-profile-id>
```
