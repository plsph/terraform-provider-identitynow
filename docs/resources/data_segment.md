---
subcategory: "Data Segmentation"
page_title: "IdentityNow: identitynow_data_segment"
description: |-
  Manages an IdentityNow data access segment.
---

# identitynow_data_segment

Manages a data access segment, which limits the entitlements, identities and certifications its members can see. This resource uses an experimental API (`X-SailPoint-Experimental` header), which can change without notice.

Segment changes are made to an unpublished version of the segment and only take effect once the segment is published. Set `publish = true` to publish the segment after every create and update.

## Example Usage

```terraform
resource "identitynow_data_segment" "emea" {
  name        = "EMEA"
  description = "Members only see EMEA entitlements."
  membership  = "FILTER"
  enabled     = true
  publish     = true

  member_filter_json = jsonencode({
    expression = {
      operator  = "EQUALS"
      attribute = "location"
      value = {
        type  = "STRING"
        value = "EMEA"
      }
    }
  })

  scopes_json = jsonencode([
    {
      scope      = "ENTITLEMENT"
      visibility = "FILTER"
      scopeFilter = {
        expression = {
          operator  = "EQUALS"
          attribute = "region"
          value = {
            type  = "STRING"
            value = "EMEA"
          }
        }
      }
    }
  ])
}
```

## Arguments Reference

The following arguments are supported:

* `name` - (Required) Segment business name.
* `description` - (Optional) Segment description.
* `membership` - (Optional) How members are chosen: `ALL`, `FILTER` (identities matching `member_filter_json`) or `SELECTION` (the identities in `member_selection`). When not set, the value chosen by IdentityNow is kept.
* `member_filter_json` - (Optional) Member filter for the `FILTER` membership as a JSON object with an `expression`: `operator` (`AND` or `EQUALS`), `attribute`, `value` (`type` and `value`) and one level of `children` expressions. Use `jsonencode()`.
* `member_selection` - (Optional) Identity selected as member for the `SELECTION` membership. Can be repeated. Contains:
  * `id` - (Required) Identity ID.
  * `type` - (Optional) Object type, `IDENTITY` (default).
* `scopes_json` - (Optional) Scopes of the segment as a JSON array. Each scope has a `scope` (`ENTITLEMENT`, `CERTIFICATION`, `IDENTITY` or `ENTITLEMENTREQUEST`), a `visibility` (`ALL`, `FILTER`, `SELECTION` or `UNSEGMENTED`) and a `scopeFilter` expression or a `scopeSelection` list of typed references. Use `jsonencode()`.
* `enabled` - (Optional) Whether the segment is active, inactive segments have no effect. When not set, the value chosen by IdentityNow is kept.
* `publish` - (Optional) Whether the provider publishes the segment after every create and update. Defaults to `false`, which leaves the changes unpublished. Only this segment is published (`publishAll=false`).

JSON arguments are compared with the API value on the configured fields only: formatting, key order and fields IdentityNow adds do not produce a diff.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - Data segment ID.
* `published` - Whether the segment as read from the API is published. Changes made outside Terraform are not published automatically, even with `publish = true`, until the next change of the resource.
* `created` - Creation date.
* `modified` - Last modification date.

## Update Behaviour

Updates send a JSON Patch with the changed fields only. Destroying the resource deletes the unpublished version of the segment and, when the segment was published, the published version as well.

## Import

Data segments can be imported using their ID:

```shell
terraform import identitynow_data_segment.example <data-segment-id>
```
