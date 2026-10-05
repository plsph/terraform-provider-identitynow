---
subcategory: "Access Profile"
page_title: "IdentityNow: Data Access Profile: identitynow_access_profile"
description: |-
  Gets information about an existing Access Profile.
---

# Data Source: identitynow_access_profile

Use this data source to access information about an existing Access Profile.

## Example Usage

```terraform
data "identitynow_access_profile" "example" {
  name = "AD Developers"
}

output "identitynow_ap_desc" {
  value = data.identitynow_access_profile.example.description
}

output "identitynow_ap_segments" {
  value = data.identitynow_access_profile.example.segments
}

output "identitynow_ap_source_id" {
  value = data.identitynow_access_profile.example.source[0].id
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) Name of the access profile.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Access profile ID.

* `description` - Access profile description.

* `requestable` - Whether the access profile is requestable by access request.

* `enabled` - Whether the access profile is enabled.

* `segments` - List of segment IDs assigned to the access profile.

* `owner` - List with the owner of the access profile. Each element contains `id`, `type` and `name`.

* `source` - List with the source associated with the access profile. Each element contains `id`, `type` and `name`.

* `additional_owners` - Additional identity or governance group owners. Each element contains `id`, `type` and `name`.

* `access_request_config` - List with the access request configuration. Each element contains:
  * `comments_required` - Whether the requester must provide comments justifying the request.
  * `denial_comments_required` - Whether an approver must provide comments when denying the request.
  * `reauthorization_required` - Whether reauthorization is required.
  * `require_end_date` - Whether the requester must provide an access end date.
  * `form_definition_id` - ID of the form definition presented to the requester during the access request.

* `revocation_request_config` - List with the revocation request configuration. Each element contains:
  * `approval_schemes` - List of the approver types of the revocation approval steps, e.g. `["MANAGER"]`.

* `access_model_metadata` - Access model metadata of the access profile. Each element contains `attributes`, a list of metadata attributes with `key`, `name`, `multiselect`, `status`, `type`, `description`, `object_types` (each with `value`) and `values` (each with `value`, `name` and `status`).

* `provisioning_criteria` - Criteria used to choose the account to provision. Each element contains `operation`, `attribute`, `value` and `children`, the child criteria with the same attributes (up to 3 levels).
