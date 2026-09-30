---
subcategory: "Authorization"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_authorization_right_sets"
description: |-
  Lists the IdentityNow right sets that can be assigned to custom user levels.
---

# Data Source: identitynow_authorization_right_sets

Use this data source to list the right sets that can be assigned to [identitynow_custom_user_level](../r/custom_user_level.html). The hierarchy is flattened: each right set is followed by its children.

~> **Note:** This data source uses an experimental API that may change without notice.

## Example Usage

```hcl
data "identitynow_authorization_right_sets" "identity" {
  category = "identity"
}

locals {
  top_level_identity_right_set_ids = [for r in data.identitynow_authorization_right_sets.identity.right_sets : r.id if r.depth == 0]
}
```

## Arguments Reference

* `category` - (Optional) Only return right sets of this category, e.g. `identity`.

## Attributes Reference

* `right_sets` - Right sets. Each item contains:
  * `id` - Right set ID, used in `right_sets` of `identitynow_custom_user_level`.
  * `name` - Right set name.
  * `description` - Right set description.
  * `category` - Right set category.
  * `parent_id` - ID of the parent right set, null for top-level right sets.
  * `ancestor_id` - ID of the top-level ancestor right set.
  * `depth` - Depth in the hierarchy, 0 for top-level right sets.
  * `children_ids` - IDs of the child right sets.
