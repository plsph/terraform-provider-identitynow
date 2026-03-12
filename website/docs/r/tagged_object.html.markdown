---
subcategory: "Tagged Object"
layout: "identitynow"
page_title: "IdentityNow: identitynow_tagged_object"
description: |-
  Manages tags on any SailPoint IdentityNow resource.
---

# identitynow_tagged_object

Manages tags on any SailPoint IdentityNow resource using the v2025/tagged-objects API.

This resource allows you to add, update, and remove tags from any SailPoint object such as access profiles, roles, sources, identities, governance groups, entitlements, and applications.

## Example Usage

### Tag an Access Profile

```hcl
resource "identitynow_tagged_object" "access_profile_tags" {
  object_type = "ACCESS_PROFILE"
  object_ids  = [identitynow_access_profile.example.id]
  tags        = ["production", "finance"]
}
```

### Tag a Role

```hcl
resource "identitynow_tagged_object" "role_tags" {
  object_type = "ROLE"
  object_ids  = [identitynow_role.example.id]
  tags        = ["critical", "audit-required"]
}
```

### Tag a Source

```hcl
resource "identitynow_tagged_object" "source_tags" {
  object_type = "SOURCE"
  object_ids  = [identitynow_source.example.id]
  tags        = ["active-directory", "hr-system"]
}
```

### Tag Multiple Objects

```hcl
resource "identitynow_tagged_object" "finance_access_profiles" {
  object_type = "ACCESS_PROFILE"
  object_ids  = [
    identitynow_access_profile.ap1.id,
    identitynow_access_profile.ap2.id,
    identitynow_access_profile.ap3.id,
  ]
  tags = ["finance", "quarterly-review"]
}
```

## Arguments Reference

The following arguments are supported:

As described in (https://developer.sailpoint.com/docs/api/v2025/set-tagged-object)

* `object_type` - (Required, ForceNew) Type of the SailPoint object to tag. Supported values include: `ACCESS_PROFILE`, `ROLE`, `SOURCE`, `IDENTITY`, `GOVERNANCE_GROUP`, `ENTITLEMENT`, `APPLICATION`.

* `object_ids` - (Required) Set of IDs of the SailPoint objects to tag. All objects will receive the same tags.

* `tags` - (Required) List of tags to apply to the objects.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Tagged object ID (composed as `object_type/object_id1,object_id2,...`).

## Import

Tagged objects can be imported using the format `{object_type}/{object_id1},{object_id2},...`:

```shell
terraform import identitynow_tagged_object.example ACCESS_PROFILE/2c91808568c529c60168cca6f90c1313
```

Multiple objects:

```shell
terraform import identitynow_tagged_object.example ACCESS_PROFILE/2c91808568c529c60168cca6f90c1313,2c91808568c529c60168cca6f90c1314
```
