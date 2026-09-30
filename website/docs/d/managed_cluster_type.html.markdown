---
subcategory: "Managed Cluster"
layout: "identitynow"
page_title: "IdentityNow: Data Source: identitynow_managed_cluster_type"
description: |-
  Gets information about an existing IdentityNow managed cluster type.
---

# Data Source: identitynow_managed_cluster_type

Use this data source to look up a managed cluster type by ID.

## Example Usage

```hcl
data "identitynow_managed_cluster_type" "custom" {
  id = "<CLUSTER_TYPE_ID>"
}
```

## Arguments Reference

* `id` - (Required) Managed cluster type ID.

## Attributes Reference

* `type` - Name of the cluster type.
* `pod` - Pod the cluster type applies to.
* `org` - Org the cluster type applies to.
* `managed_process_ids` - IDs of the processes that run on clusters of this type.
