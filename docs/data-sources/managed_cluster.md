---
subcategory: "Managed Cluster"
page_title: "IdentityNow: Data Source: identitynow_managed_cluster"
description: |-
  Gets information about an existing IdentityNow managed cluster.
---

# Data Source: identitynow_managed_cluster

Use this data source to look up a managed cluster by ID or name, e.g. to reference the virtual appliance cluster of a source.

## Example Usage

```terraform
data "identitynow_managed_cluster" "va_cluster" {
  name = "Primary VA Cluster"
}
```

## Arguments Reference

Exactly one of the following must be set:

* `id` - (Optional) Managed cluster ID.
* `name` - (Optional) Managed cluster name.

## Attributes Reference

* `type` - Cluster type.
* `description` - Description of the cluster.
* `configuration` - Map of all cluster configuration entries.
* `pod` - Pod of the cluster.
* `org` - Org (tenant) of the cluster.
* `client_type` - Type of the clients of the cluster.
* `ccg_version` - CCG version used by the cluster.
* `pinned_config` - Whether the cluster configuration is pinned.
* `operational` - Whether the cluster is operational.
* `status` - Cluster status.
* `public_key` - Public key of the cluster.
* `public_key_thumbprint` - Public key thumbprint of the cluster.
* `public_key_certificate` - Public key certificate of the cluster.
* `alert_key` - Key describing any immediate cluster alerts.
* `client_ids` - IDs of the clients of the cluster.
* `service_count` - Number of services bound to the cluster.
* `created_at` - Creation date.
* `updated_at` - Last update date.
* `current_installed_release_version` - Release installed on the cluster.
* `consolidated_health_indicators_status` - Consolidated health status of the cluster.
