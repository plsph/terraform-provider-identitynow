---
subcategory: "Managed Cluster"
layout: "identitynow"
page_title: "IdentityNow: identitynow_managed_cluster_type"
description: |-
  Manages an IdentityNow managed cluster type.
---

# identitynow_managed_cluster_type

Manages a managed cluster type, which defines the processes that run on the [managed clusters](managed_cluster.html) of that type for a pod or org.

Updates are sent as JSON Patch operations for the changed attributes only.

## Example Usage

```hcl
resource "identitynow_managed_cluster_type" "custom" {
  type                = "custom-cluster"
  pod                 = "<POD>"
  org                 = "<ORG>"
  managed_process_ids = ["<MANAGED_PROCESS_ID>"]
}
```

## Arguments Reference

* `type` - (Required) Name of the cluster type.
* `pod` - (Required) Pod the cluster type applies to.
* `org` - (Required) Org the cluster type applies to.
* `managed_process_ids` - (Optional) IDs of the processes that run on clusters of this type. Removing the attribute clears the list.

## Attributes Reference

* `id` - Managed cluster type ID.

## Import

Managed cluster types can be imported using their ID:

```shell
terraform import identitynow_managed_cluster_type.example <cluster-type-id>
```
