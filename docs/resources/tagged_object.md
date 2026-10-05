---
subcategory: "Tagged Object"
page_title: "IdentityNow: identitynow_tagged_object"
description: |-
  Manages tags on any SailPoint IdentityNow resource.
---

# identitynow_tagged_object

Manages tags on any SailPoint IdentityNow resource using the v2025/tagged-objects API.

This resource allows you to add, update, and remove tags from any SailPoint object such as access profiles, roles, sources, identities, governance groups, entitlements, and applications.

The resource manages the full tag set of each object: tags set on the objects outside Terraform are replaced with the configured tags. Removing `tags` (or setting it to an empty set) clears the tags of the objects, and destroying the resource removes all tags from the objects. Manage the tags of an object with a single resource.

Tags are case-insensitive and stored in uppercase, so `production` and `PRODUCTION` are the same tag.

## Example Usage

### Tag an Access Profile

```terraform
resource "identitynow_tagged_object" "access_profile_tags" {
  object_type = "ACCESS_PROFILE"
  object_ids  = ["2c91808568c529c60168cca6f90c1313"]
  tags        = ["production", "finance"]
}
```

### Tag a Role

```terraform
resource "identitynow_role" "example" {
  name        = "Finance Role"
  description = "Access for the finance department"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "John Doe"
  }
}

resource "identitynow_tagged_object" "role_tags" {
  object_type = "ROLE"
  object_ids  = [identitynow_role.example.id]
  tags        = ["critical", "audit-required"]
}
```

### Tag a Source

```terraform
data "identitynow_source" "hr" {
  name = "Workday"
}

resource "identitynow_tagged_object" "source_tags" {
  object_type = "SOURCE"
  object_ids  = [data.identitynow_source.hr.id]
  tags        = ["authoritative", "hr-system"]
}
```

### Tag Multiple Objects

```terraform
resource "identitynow_tagged_object" "finance_access_profiles" {
  object_type = "ACCESS_PROFILE"
  object_ids = [
    "2c91808568c529c60168cca6f90c1314",
    "2c91808568c529c60168cca6f90c1315",
    "2c91808568c529c60168cca6f90c1316",
  ]
  tags = ["finance", "quarterly-review"]
}
```

## Arguments Reference

The following arguments are supported:

As described in (https://developer.sailpoint.com/docs/api/v2025/set-tagged-object)

* `object_type` - (Required) Type of the SailPoint objects to tag. Supported values include: `ACCESS_PROFILE`, `ROLE`, `SOURCE`, `IDENTITY`, `GOVERNANCE_GROUP`, `ENTITLEMENT`, `APPLICATION`. Changing this forces a new resource to be created.

* `object_ids` - (Required) Set of IDs of the SailPoint objects to tag. All objects receive the same tags.

* `tags` - (Optional) Set of tags of the objects. Tags are case-insensitive and stored in uppercase. If not set, the tags of the objects are cleared.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Tagged object ID (composed as `object_type/object_id1,object_id2,...`).

## Import

Tagged objects can be imported using the format `<object_type>/<object_id1>,<object_id2>,...`:

```shell
terraform import identitynow_tagged_object.example ACCESS_PROFILE/2c91808568c529c60168cca6f90c1313
```

Multiple objects:

```shell
terraform import identitynow_tagged_object.example ACCESS_PROFILE/2c91808568c529c60168cca6f90c1313,2c91808568c529c60168cca6f90c1314
```
