---
subcategory: "Tagged Object"
layout: "identitynow"
page_title: "IdentityNow: identitynow_tag"
description: |-
  Manages an IdentityNow tag.
---

# identitynow_tag

Manages a tag. Tags are assigned to objects with [identitynow_tagged_object](tagged_object.html).

## Example Usage

```hcl
resource "identitynow_tag" "pci" {
  name = "PCI"
}
```

## Arguments Reference

* `name` - (Required) Tag name. Tags cannot be renamed, changing this forces a new tag to be created.

## Attributes Reference

* `id` - Tag ID.
* `created` - Creation date.
* `modified` - Last modification date.
* `tag_category_refs` - Objects the tag is assigned to. Each item contains `id`, `type` and `name`.

## Import

Tags can be imported using their ID:

```shell
terraform import identitynow_tag.example <tag-id>
```
