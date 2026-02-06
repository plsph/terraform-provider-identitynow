---
subcategory: "Access Profile"
layout: "identitynow"
page_title: "IdentityNow: Data Access Profile: identitynow_access_profile"
description: |-
  Gets information about an existing Access Profile.
---

# Data Source: identitynow_access_profile

Use this data source to access information about an existing Access Profile.

## Example Usage

```hcl
data "identitynow_access_profile" "example" {
  id = "example"
}

output "identitynow_ap_desc" {
  value = data.identitynow_access_profile.example.description
}
```

## Arguments Reference

The following arguments are supported:

* `name` - Name of the access profile.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Access profile id.

* `description` - Access profile description.

* `requestable` - Indicates whether the access profile is requestable by access request. Currently, making an access profile non-requestable is only supported for customers enabled with the new Request Center. Otherwise, attempting to create an access profile with a value false in this field results in a 400 error.

* `enabled` - Indicates whether the access profile is enabled. If it's enabled, you must include at least one entitlement.

* `entitlements` - List of entitlements associated with the access profile. If enabled is false, this can be empty. Otherwise, it must contain at least one entitlement.

* `source` - Source associated with the access profile.

* `owner` - Owner of the object.

* `access_request_config` - Access profile request configuration. Contains:

* `comments_required` - Indicates whether the requester of the containing object must provide comments justifying the request.

* `denial_comments_required` - Indicates whether an approver must provide comments when denying the request.

* `approval_schemes` - List describing the steps involved in approving the request.

