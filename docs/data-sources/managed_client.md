---
subcategory: "Managed Cluster"
page_title: "IdentityNow: Data Source: identitynow_managed_client"
description: |-
  Gets information about an existing IdentityNow managed client.
---

# Data Source: identitynow_managed_client

Use this data source to look up a managed client (virtual appliance or connector gateway) by ID. The client secret is not exposed.

## Example Usage

```hcl
data "identitynow_managed_client" "va_1" {
  id = "<MANAGED_CLIENT_ID>"
}
```

## Arguments Reference

* `id` - (Required) Managed client ID.

## Attributes Reference

* `cluster_id` - ID of the managed cluster the client belongs to.
* `name` - Name of the client.
* `description` - Description of the client.
* `type` - Client type, `VA` or `CCG`.
* `client_id` - Client ID used in API management.
* `status` - Status of the client.
* `cluster_type` - Type of the cluster the client belongs to.
* `alert_key` - Key describing any immediate client alerts.
* `api_gateway_base_url` - API gateway base URL of the client.
* `ip_address` - Public IP address of the client.
* `last_seen` - When the client was last seen by the server.
* `since_last_seen` - Milliseconds since the client last polled the server.
* `va_download_url` - Virtual appliance download URL.
* `va_version` - Version of the virtual appliance software the client runs.
* `provision_status` - Provisioning status of the client.
* `created_at` - Creation date.
* `updated_at` - Last update date.
